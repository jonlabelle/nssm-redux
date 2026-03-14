//go:build windows

package svcwrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
	"github.com/jonlabelle/nssm-redux/internal/runtime"
	"github.com/jonlabelle/nssm-redux/internal/support"
)

const (
	hookVersion       = "1"
	hookAbortExitCode = 99
	hookTimeout       = 30 * time.Second
)

type hookRunner struct {
	cfg          config.Service
	serviceStart time.Time
	state        *serviceState
	logger       *serviceLogger
}

type serviceState struct {
	lastControl         string
	startRequestedCount int
	startCount          int
	exitCount           int
	throttleCount       int
	lastExitCode        int
	lastRuntime         time.Duration
}

func (h *hookRunner) RunSync(hook config.Hook, trigger string, process *runtime.Process) (int, error) {
	return h.run(hook, trigger, process)
}

func (h *hookRunner) RunAsync(hook config.Hook, trigger string, process *runtime.Process) {
	if !h.hasHook(hook) {
		return
	}
	go func() {
		if exitCode, err := h.run(hook, trigger, process); err != nil {
			h.logger.Warnf("hook %s failed: %v", hook, err)
		} else if exitCode != 0 {
			h.logger.Warnf("hook %s exited with code %d", hook, exitCode)
		}
	}()
}

func (h *hookRunner) hasHook(hook config.Hook) bool {
	command, ok := h.cfg.Hooks[hook]
	return ok && strings.TrimSpace(command) != ""
}

func (h *hookRunner) run(hook config.Hook, trigger string, process *runtime.Process) (int, error) {
	command, ok := h.cfg.Hooks[hook]
	if !ok || strings.TrimSpace(command) == "" {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), hookTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cmd.exe", "/d", "/s", "/c", command)
	cmd.Dir = h.cfg.Directory
	cmd.Env = append(
		support.MergeEnvironment(os.Environ(), h.cfg.Environment, h.cfg.EnvironmentExtra),
		h.hookEnv(hook, trigger, process)...,
	)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start hook %s: %w", hook, err)
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("wait for hook %s: %w", hook, err)
	}
	return 0, nil
}

func (h *hookRunner) hookEnv(hook config.Hook, trigger string, process *runtime.Process) []string {
	info := hook.Info()
	self, _ := os.Executable()
	appPID := ""
	appRuntime := ""
	if process != nil && process.PID() != 0 {
		appPID = strconv.Itoa(process.PID())
		appRuntime = strconv.FormatInt(time.Since(process.StartedAt()).Milliseconds(), 10)
	} else if h.state.lastRuntime > 0 {
		appRuntime = strconv.FormatInt(h.state.lastRuntime.Milliseconds(), 10)
	}
	exitCode := ""
	if process == nil && h.state.exitCount > 0 {
		exitCode = strconv.Itoa(h.state.lastExitCode)
	}

	commandLine := support.JoinCommandLine([]string{h.cfg.Executable})
	if strings.TrimSpace(h.cfg.Arguments) != "" {
		commandLine += " " + h.cfg.Arguments
	}

	env := []string{
		"NSSM_HOOK_VERSION=" + hookVersion,
		"NSSM_EXE=" + self,
		"NSSM_CONFIGURATION=",
		"NSSM_VERSION=dev",
		"NSSM_BUILD_DATE=",
		"NSSM_PID=" + strconv.Itoa(os.Getpid()),
		"NSSM_DEADLINE=" + strconv.FormatInt(hookTimeout.Milliseconds(), 10),
		"NSSM_SERVICE_NAME=" + h.cfg.Name,
		"NSSM_SERVICE_DISPLAYNAME=" + h.cfg.DisplayName,
		"NSSM_COMMAND_LINE=" + commandLine,
		"NSSM_APPLICATION_PID=" + appPID,
		"NSSM_EVENT=" + info.Event,
		"NSSM_ACTION=" + info.Action,
		"NSSM_TRIGGER=" + trigger,
		"NSSM_LAST_CONTROL=" + h.state.lastControl,
		"NSSM_START_REQUESTED_COUNT=" + strconv.Itoa(h.state.startRequestedCount),
		"NSSM_START_COUNT=" + strconv.Itoa(h.state.startCount),
		"NSSM_THROTTLE_COUNT=" + strconv.Itoa(h.state.throttleCount),
		"NSSM_EXIT_COUNT=" + strconv.Itoa(h.state.exitCount),
		"NSSM_EXITCODE=" + exitCode,
		"NSSM_RUNTIME=" + strconv.FormatInt(time.Since(h.serviceStart).Milliseconds(), 10),
		"NSSM_APPLICATION_RUNTIME=" + appRuntime,
	}
	return env
}
