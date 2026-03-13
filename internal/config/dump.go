package config

import (
	"strconv"

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
