//go:build windows

package svcwrap

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
	"github.com/jonlabelle/nssm-redux/internal/runtime"
	"github.com/jonlabelle/nssm-redux/internal/scm"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	rotateControl          = svc.Cmd(128)
	powerStatusChangeEvent = 0x000A
	powerResumeAutoEvent   = 0x0012
)

// Run starts the SCM-facing service host.
func Run(ctx context.Context, name string) error {
	handler := &serviceHandler{ctx: ctx, name: name}

	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("detect service session: %w", err)
	}
	if !isService {
		return debug.Run(name, handler)
	}
	return svc.Run(name, handler)
}

type serviceHandler struct {
	ctx  context.Context
	name string
}

func (h *serviceHandler) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	logger := newLogger()
	defer logger.Close()

	cfg, err := scm.Load(h.name)
	if err != nil {
		logger.Errorf("load service configuration: %v", err)
		return false, 1
	}
	if err := cfg.Validate(); err != nil {
		logger.Errorf("validate service configuration: %v", err)
		return false, 2
	}

	state := &serviceState{}
	hooks := &hookRunner{
		cfg:          cfg,
		serviceStart: time.Now(),
		state:        state,
		logger:       logger,
	}

	accepts := svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPowerEvent
	status := svc.Status{State: svc.StartPending}
	changes <- status

	state.startRequestedCount++
	if exitCode, err := hooks.RunSync(config.HookStartPre, "START", nil); err != nil {
		logger.Errorf("start pre-hook failed: %v", err)
		return false, 3
	} else if exitCode == hookAbortExitCode {
		logger.Errorf("start pre-hook aborted service start")
		return false, 5
	} else if exitCode != 0 {
		logger.Warnf("start pre-hook exited with code %d", exitCode)
	}

	process, err := runtime.Start(cfg, logger)
	if err != nil {
		logger.Errorf("start managed process: %v", err)
		return false, 3
	}
	state.startCount++
	state.throttleCount = 0

	status = svc.Status{State: svc.Running, Accepts: accepts}
	changes <- status
	hooks.RunAsync(config.HookStartPost, "START", process)

	results := process.Wait()
	stopDone := make(chan stopOutcome, 1)
	consecutiveFast := 0
	stopping := false

	beginStop := func(reason, trigger string) {
		if stopping {
			return
		}
		stopping = true
		state.lastControl = trigger
		if exitCode, err := hooks.RunSync(config.HookStopPre, trigger, process); err != nil {
			logger.Warnf("stop pre-hook failed: %v", err)
		} else if exitCode != 0 {
			logger.Warnf("stop pre-hook exited with code %d", exitCode)
		}
		results = nil
		status = svc.Status{State: svc.StopPending, WaitHint: waitHintFor(cfg.StopWaitHint() + time.Second)}
		changes <- status
		if process == nil {
			stopDone <- stopOutcome{}
			return
		}
		go func(proc *runtime.Process) {
			detached, err := proc.Stop()
			stopDone <- stopOutcome{detached: detached, err: err}
		}(process)
		if reason != "" {
			logger.Infof("%s", reason)
		}
	}

	for {
		select {
		case <-h.ctx.Done():
			beginStop("service context cancelled", "STOP")

		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				changes <- status

			case svc.Stop, svc.Shutdown:
				beginStop("service stop received", controlName(request.Cmd))

			case rotateControl:
				state.lastControl = controlName(request.Cmd)
				if exitCode, err := hooks.RunSync(config.HookRotatePre, state.lastControl, process); err != nil {
					logger.Warnf("rotate pre-hook failed: %v", err)
				} else if exitCode != 0 {
					logger.Warnf("rotate pre-hook exited with code %d", exitCode)
				}
				if process != nil {
					if err := process.Rotate(); err != nil {
						logger.Warnf("rotate managed logs: %v", err)
					}
				}
				hooks.RunAsync(config.HookRotatePost, state.lastControl, process)

			case svc.PowerEvent:
				state.lastControl = controlName(request.Cmd)
				switch request.EventType {
				case powerResumeAutoEvent:
					hooks.RunAsync(config.HookPowerResume, state.lastControl, process)
				case powerStatusChangeEvent:
					hooks.RunAsync(config.HookPowerChange, state.lastControl, process)
				}
			}

		case stop := <-stopDone:
			if stop.err != nil {
				logger.Warnf("stop managed process: %v", stop.err)
			}
			if stop.detached {
				logger.Warnf("managed process is still running because AppStopMethodSkip disables terminate")
			}
			return false, 0

		case result, ok := <-results:
			if !ok {
				return false, 0
			}

			if result.Err != nil {
				logger.Warnf("managed process exited with error: %v", result.Err)
			}
			state.exitCount++
			state.lastExitCode = result.ExitCode
			state.lastRuntime = result.Runtime
			hooks.RunAsync(config.HookExitPost, "", nil)

			decision := runtime.DecideRestart(cfg, result.Runtime, consecutiveFast, result.ExitCode)
			consecutiveFast = decision.ConsecutiveFast
			state.throttleCount = consecutiveFast

			switch decision.Action {
			case config.ExitActionIgnore:
				logger.Warnf("managed process exited with code %d; leaving service running", result.ExitCode)
				process = nil
				results = nil

			case config.ExitActionExit:
				logger.Infof("managed process exited with code %d; stopping service", result.ExitCode)
				if result.ExitCode < 0 {
					return false, 1
				}
				return false, uint32(result.ExitCode)

			case config.ExitActionRestart:
				logger.Infof("managed process exited with code %d; restarting after %s", result.ExitCode, decision.Delay.Truncate(time.Millisecond))
				stop, resetThrottle := h.waitForRestartWindow(requests, changes, &status, decision.Delay, logger, hooks, decision.ConsecutiveFast > 0, nil, accepts)
				if resetThrottle {
					consecutiveFast = 0
					state.throttleCount = 0
				}
				if stop {
					return false, 0
				}

				state.startRequestedCount++
				if exitCode, err := hooks.RunSync(config.HookStartPre, "START", nil); err != nil {
					logger.Errorf("start pre-hook failed: %v", err)
					return false, 4
				} else if exitCode == hookAbortExitCode {
					logger.Errorf("start pre-hook aborted service restart")
					return false, 5
				} else if exitCode != 0 {
					logger.Warnf("start pre-hook exited with code %d", exitCode)
				}
				process, err = runtime.Start(cfg, logger)
				if err != nil {
					logger.Errorf("restart managed process: %v", err)
					return false, 4
				}
				state.startCount++
				state.throttleCount = 0
				results = process.Wait()
				status = svc.Status{State: svc.Running, Accepts: accepts}
				changes <- status
				hooks.RunAsync(config.HookStartPost, "START", process)
			}
		}
	}
}

type stopOutcome struct {
	detached bool
	err      error
}

func (h *serviceHandler) waitForRestartWindow(
	requests <-chan svc.ChangeRequest,
	changes chan<- svc.Status,
	status *svc.Status,
	delay time.Duration,
	logger *serviceLogger,
	hooks *hookRunner,
	throttled bool,
	process *runtime.Process,
	baseAccepts svc.Accepted,
) (bool, bool) {
	if delay <= 0 {
		return false, false
	}

	waitHint := uint32(delay.Milliseconds() + 1000)
	if delay > time.Duration(math.MaxUint32)*time.Millisecond {
		waitHint = math.MaxUint32
	}
	state := svc.StartPending
	accepts := baseAccepts
	if throttled {
		state = svc.Paused
		accepts |= svc.AcceptPauseAndContinue
	}
	*status = svc.Status{State: state, Accepts: accepts, WaitHint: waitHint}
	changes <- *status

	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-h.ctx.Done():
			if exitCode, err := hooks.RunSync(config.HookStopPre, "STOP", process); err != nil {
				logger.Warnf("stop pre-hook failed: %v", err)
			} else if exitCode != 0 {
				logger.Warnf("stop pre-hook exited with code %d", exitCode)
			}
			return true, false
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				changes <- *status
			case svc.Continue:
				if throttled {
					hooks.state.lastControl = controlName(request.Cmd)
					logger.Infof("service continue received during restart delay")
					*status = svc.Status{State: svc.ContinuePending, Accepts: baseAccepts, WaitHint: waitHintFor(2 * time.Second)}
					changes <- *status
					return false, true
				}
			case svc.Pause:
				changes <- *status
			case svc.Stop, svc.Shutdown:
				logger.Infof("service stop received during restart delay")
				hooks.state.lastControl = controlName(request.Cmd)
				if exitCode, err := hooks.RunSync(config.HookStopPre, hooks.state.lastControl, process); err != nil {
					logger.Warnf("stop pre-hook failed: %v", err)
				} else if exitCode != 0 {
					logger.Warnf("stop pre-hook exited with code %d", exitCode)
				}
				return true, false
			case svc.PowerEvent:
				hooks.state.lastControl = controlName(request.Cmd)
				switch request.EventType {
				case powerResumeAutoEvent:
					hooks.RunAsync(config.HookPowerResume, hooks.state.lastControl, process)
				case powerStatusChangeEvent:
					hooks.RunAsync(config.HookPowerChange, hooks.state.lastControl, process)
				}
			}
		case <-timer.C:
			return false, false
		}
	}
}

type serviceLogger struct {
	event    *eventlog.Log
	fallback *log.Logger
}

func newLogger() *serviceLogger {
	eventLog, err := eventlog.Open("nssmr")
	if err != nil {
		eventLog = nil
	}
	return &serviceLogger{
		event:    eventLog,
		fallback: log.New(os.Stderr, "nssmr: ", log.LstdFlags),
	}
}

func (l *serviceLogger) Close() {
	if l.event != nil {
		_ = l.event.Close()
	}
}

func (l *serviceLogger) Infof(format string, args ...any) {
	l.logf("info", format, args...)
}

func (l *serviceLogger) Warnf(format string, args ...any) {
	l.logf("warn", format, args...)
}

func (l *serviceLogger) Errorf(format string, args ...any) {
	l.logf("error", format, args...)
}

func (l *serviceLogger) logf(level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	switch level {
	case "info":
		if l.event != nil {
			_ = l.event.Info(1, message)
			return
		}
	case "warn":
		if l.event != nil {
			_ = l.event.Warning(1, message)
			return
		}
	case "error":
		if l.event != nil {
			_ = l.event.Error(1, message)
			return
		}
	}
	l.fallback.Printf("%s: %s", level, message)
}

func waitHintFor(delay time.Duration) uint32 {
	if delay <= 0 {
		return 1000
	}
	if delay > time.Duration(math.MaxUint32)*time.Millisecond {
		return math.MaxUint32
	}
	return uint32(delay.Milliseconds())
}

func controlName(cmd svc.Cmd) string {
	switch cmd {
	case svc.Stop:
		return "STOP"
	case svc.Shutdown:
		return "SHUTDOWN"
	case svc.Pause:
		return "PAUSE"
	case svc.Continue:
		return "CONTINUE"
	case svc.PowerEvent:
		return "POWEREVENT"
	case rotateControl:
		return "ROTATE"
	default:
		return fmt.Sprintf("CONTROL_%d", uint32(cmd))
	}
}
