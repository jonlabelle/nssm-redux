package config

import (
	"strconv"
	"strings"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/support"
)

// DumpCommands renders a service as CLI commands.
func DumpCommands(program string, service Service, targetName string) ([]string, error) {
	service.Normalize()
	if err := service.Validate(); err != nil {
		return nil, err
	}
	if program == "" {
		program = "nssmr"
	}
	if targetName == "" {
		targetName = service.Name
	}

	commands := []string{
		support.JoinCommandLine([]string{program, "install", targetName, service.Executable}),
	}

	add := func(args ...string) {
		commands = append(commands, support.JoinCommandLine(args))
	}

	defaults := Default(service.Name)
	if service.Arguments != "" {
		add(program, "set", targetName, string(SettingAppParameters), service.Arguments)
	}
	if service.Directory != "" && service.Directory != defaults.Directory {
		add(program, "set", targetName, string(SettingAppDirectory), service.Directory)
	}
	if len(service.Environment) > 0 {
		args := []string{program, "set", targetName, string(SettingAppEnvironment)}
		args = append(args, service.Environment...)
		add(args...)
	}
	if len(service.EnvironmentExtra) > 0 {
		args := []string{program, "set", targetName, string(SettingAppEnvironmentExtra)}
		args = append(args, service.EnvironmentExtra...)
		add(args...)
	}
	if service.DefaultExitAction != defaults.DefaultExitAction {
		add(program, "set", targetName, string(SettingAppExit), "Default", string(service.DefaultExitAction))
	}
	for _, code := range service.SortedExitCodes() {
		add(program, "set", targetName, string(SettingAppExit), strconv.Itoa(code), string(service.ExitActions[code]))
	}
	if service.RestartDelay != defaults.RestartDelay {
		add(program, "set", targetName, string(SettingAppRestartDelay), Milliseconds(service.RestartDelay))
	}
	if service.ThrottleDelay != defaults.ThrottleDelay {
		add(program, "set", targetName, string(SettingAppThrottle), Milliseconds(service.ThrottleDelay))
	}
	if service.Priority != defaults.Priority {
		add(program, "set", targetName, string(SettingAppPriority), string(service.Priority))
	}
	if service.Affinity != 0 {
		add(program, "set", targetName, string(SettingAppAffinity), FormatAffinityMask(service.Affinity))
	}
	if service.StopMethodSkip != defaults.StopMethodSkip {
		add(program, "set", targetName, string(SettingAppStopMethodSkip), strconv.FormatUint(uint64(service.StopMethodSkip), 10))
	}
	if service.StopConsoleDelay != defaults.StopConsoleDelay {
		add(program, "set", targetName, string(SettingAppStopMethodConsole), Milliseconds(service.StopConsoleDelay))
	}
	if service.StopWindowDelay != defaults.StopWindowDelay {
		add(program, "set", targetName, string(SettingAppStopMethodWindow), Milliseconds(service.StopWindowDelay))
	}
	if service.StopThreadsDelay != defaults.StopThreadsDelay {
		add(program, "set", targetName, string(SettingAppStopMethodThreads), Milliseconds(service.StopThreadsDelay))
	}
	for _, hook := range SupportedHooks() {
		if command := service.Hooks[hook]; strings.TrimSpace(command) != "" {
			add(program, "set", targetName, string(SettingAppEvents), string(hook), command)
		}
	}
	if service.Logging.Enabled != defaults.Logging.Enabled {
		add(program, "set", targetName, string(SettingAppRotateFiles), boolString(service.Logging.Enabled))
	}
	if service.Logging.Online != defaults.Logging.Online {
		add(program, "set", targetName, string(SettingAppRotateOnline), boolString(service.Logging.Online))
	}
	if service.Logging.AgeThreshold != defaults.Logging.AgeThreshold {
		add(program, "set", targetName, string(SettingAppRotateSeconds), strconv.FormatInt(int64(service.Logging.AgeThreshold/time.Second), 10))
	}
	if service.Logging.SizeLow() != 0 {
		add(program, "set", targetName, string(SettingAppRotateBytes), strconv.FormatUint(uint64(service.Logging.SizeLow()), 10))
	}
	if service.Logging.SizeHigh() != 0 {
		add(program, "set", targetName, string(SettingAppRotateBytesHigh), strconv.FormatUint(uint64(service.Logging.SizeHigh()), 10))
	}
	if service.Logging.RotateDelay != defaults.Logging.RotateDelay {
		add(program, "set", targetName, string(SettingAppRotateDelay), Milliseconds(service.Logging.RotateDelay))
	}
	if service.Logging.TimestampLog != defaults.Logging.TimestampLog {
		add(program, "set", targetName, string(SettingAppTimestampLog), boolString(service.Logging.TimestampLog))
	}
	if service.StdinPath != "" {
		add(program, "set", targetName, string(SettingAppStdin), service.StdinPath)
	}
	if service.StdoutPath != "" {
		add(program, "set", targetName, string(SettingAppStdout), service.StdoutPath)
	}
	if service.StderrPath != "" {
		add(program, "set", targetName, string(SettingAppStderr), service.StderrPath)
	}
	if service.NoConsole != defaults.NoConsole {
		add(program, "set", targetName, string(SettingAppNoConsole), boolString(service.NoConsole))
	}
	if service.KillProcessTree != defaults.KillProcessTree {
		add(program, "set", targetName, string(SettingAppKillProcessTree), boolString(service.KillProcessTree))
	}
	if service.DisplayName != "" && service.DisplayName != defaults.DisplayName {
		add(program, "set", targetName, string(SettingDisplayName), service.DisplayName)
	}
	if service.Description != "" {
		add(program, "set", targetName, string(SettingDescription), service.Description)
	}
	if service.Startup != defaults.Startup {
		add(program, "set", targetName, string(SettingStart), string(service.Startup))
	}
	if len(service.Dependencies) > 0 {
		args := []string{program, "set", targetName, string(SettingDependOnService)}
		args = append(args, service.Dependencies...)
		add(args...)
	}

	return commands, nil
}
