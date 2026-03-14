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

	accepts := svc.AcceptStop | svc.AcceptShutdown
	status := svc.Status{State: svc.StartPending}
	changes <- status

	process, err := runtime.Start(cfg, logger)
	if err != nil {
		logger.Errorf("start managed process: %v", err)
		return false, 3
	}

	status = svc.Status{State: svc.Running, Accepts: accepts}
	changes <- status

	results := process.Wait()
	stopDone := make(chan stopOutcome, 1)
	consecutiveFast := 0
	stopping := false

	beginStop := func(reason string) {
		if stopping {
			return
		}
		stopping = true
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
			beginStop("service context cancelled")

		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				changes <- status

			case svc.Stop, svc.Shutdown:
				beginStop("service stop received")
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

			decision := runtime.DecideRestart(cfg, result.Runtime, consecutiveFast, result.ExitCode)
			consecutiveFast = decision.ConsecutiveFast

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
				if stop := h.waitForRestartWindow(requests, changes, &status, decision.Delay, logger); stop {
					return false, 0
				}

				process, err = runtime.Start(cfg, logger)
				if err != nil {
					logger.Errorf("restart managed process: %v", err)
					return false, 4
				}
				results = process.Wait()
				status = svc.Status{State: svc.Running, Accepts: accepts}
				changes <- status
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
) bool {
	if delay <= 0 {
		return false
	}

	waitHint := uint32(delay.Milliseconds() + 1000)
	if delay > time.Duration(math.MaxUint32)*time.Millisecond {
		waitHint = math.MaxUint32
	}
	*status = svc.Status{State: svc.StartPending, WaitHint: waitHint}
	changes <- *status

	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return true
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				changes <- *status
			case svc.Stop, svc.Shutdown:
				logger.Infof("service stop received during restart delay")
				return true
			}
		case <-timer.C:
			return false
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
		l.event.Close()
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
