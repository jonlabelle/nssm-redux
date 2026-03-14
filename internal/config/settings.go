package config

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Setting identifies a supported nssmr setting name.
type Setting string

const (
	SettingApplication          Setting = "Application"
	SettingAppParameters        Setting = "AppParameters"
	SettingAppDirectory         Setting = "AppDirectory"
	SettingAppEnvironment       Setting = "AppEnvironment"
	SettingAppEnvironmentExtra  Setting = "AppEnvironmentExtra"
	SettingAppExit              Setting = "AppExit"
	SettingAppRestartDelay      Setting = "AppRestartDelay"
	SettingAppThrottle          Setting = "AppThrottle"
	SettingAppPriority          Setting = "AppPriority"
	SettingAppAffinity          Setting = "AppAffinity"
	SettingAppStopMethodSkip    Setting = "AppStopMethodSkip"
	SettingAppStopMethodConsole Setting = "AppStopMethodConsole"
	SettingAppStopMethodWindow  Setting = "AppStopMethodWindow"
	SettingAppStopMethodThreads Setting = "AppStopMethodThreads"
	SettingAppStdin             Setting = "AppStdin"
	SettingAppStdout            Setting = "AppStdout"
	SettingAppStderr            Setting = "AppStderr"
	SettingAppNoConsole         Setting = "AppNoConsole"
	SettingAppKillProcessTree   Setting = "AppKillProcessTree"
	SettingDisplayName          Setting = "DisplayName"
	SettingDescription          Setting = "Description"
	SettingStart                Setting = "Start"
	SettingDependOnService      Setting = "DependOnService"
)

// SettingSpec describes the CLI shape for a setting.
type SettingSpec struct {
	Name                Setting
	AdditionalMandatory bool
	MultiValue          bool
}

var settingSpecs = []SettingSpec{
	{Name: SettingApplication},
	{Name: SettingAppParameters},
	{Name: SettingAppDirectory},
	{Name: SettingAppEnvironment, MultiValue: true},
	{Name: SettingAppEnvironmentExtra, MultiValue: true},
	{Name: SettingAppExit, AdditionalMandatory: true},
	{Name: SettingAppRestartDelay},
	{Name: SettingAppThrottle},
	{Name: SettingAppPriority},
	{Name: SettingAppAffinity},
	{Name: SettingAppStopMethodSkip},
	{Name: SettingAppStopMethodConsole},
	{Name: SettingAppStopMethodWindow},
	{Name: SettingAppStopMethodThreads},
	{Name: SettingAppStdin},
	{Name: SettingAppStdout},
	{Name: SettingAppStderr},
	{Name: SettingAppNoConsole},
	{Name: SettingAppKillProcessTree},
	{Name: SettingDisplayName},
	{Name: SettingDescription},
	{Name: SettingStart},
	{Name: SettingDependOnService, MultiValue: true},
}

// SupportedSettings returns the supported settings in CLI order.
func SupportedSettings() []SettingSpec {
	return slices.Clone(settingSpecs)
}

// LookupSetting finds a supported setting by name.
func LookupSetting(raw string) (SettingSpec, bool) {
	for _, spec := range settingSpecs {
		if strings.EqualFold(string(spec.Name), raw) {
			return spec, true
		}
	}
	return SettingSpec{}, false
}

// Apply mutates the service configuration by applying a setting value.
func Apply(service *Service, setting Setting, additional string, values []string, reset bool) error {
	defaults := Default(service.Name)

	switch setting {
	case SettingApplication:
		if reset {
			service.Executable = ""
			service.Directory = ""
			return nil
		}
		service.Executable = strings.Join(values, " ")
		if service.Directory == "" {
			service.Directory = filepathDir(service.Executable)
		}
		return nil

	case SettingAppParameters:
		if reset {
			service.Arguments = ""
			return nil
		}
		service.Arguments = strings.Join(values, " ")
		return nil

	case SettingAppDirectory:
		if reset {
			service.Directory = ""
			return nil
		}
		service.Directory = strings.Join(values, " ")
		return nil

	case SettingAppEnvironment:
		if reset {
			service.Environment = nil
			return nil
		}
		service.Environment = append([]string(nil), values...)
		return nil

	case SettingAppEnvironmentExtra:
		if reset {
			service.EnvironmentExtra = nil
			return nil
		}
		service.EnvironmentExtra = append([]string(nil), values...)
		return nil

	case SettingAppExit:
		return applyExitSetting(service, additional, values, reset)

	case SettingAppRestartDelay:
		if reset {
			service.RestartDelay = defaults.RestartDelay
			return nil
		}
		d, err := parseDurationValue(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.RestartDelay = d
		return nil

	case SettingAppThrottle:
		if reset {
			service.ThrottleDelay = defaults.ThrottleDelay
			return nil
		}
		d, err := parseDurationValue(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.ThrottleDelay = d
		return nil

	case SettingAppPriority:
		if reset {
			service.Priority = defaults.Priority
			return nil
		}
		priority, err := ParsePriorityClass(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.Priority = priority
		return nil

	case SettingAppAffinity:
		if reset {
			service.Affinity = 0
			return nil
		}
		raw := singleValue(values)
		if strings.TrimSpace(raw) == "" {
			return fmt.Errorf("parse %s: value is required", setting)
		}
		mask, err := ParseAffinityMask(raw)
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.Affinity = mask
		return nil

	case SettingAppStopMethodSkip:
		if reset {
			service.StopMethodSkip = defaults.StopMethodSkip
			return nil
		}
		mask, err := parseStopMethodSkip(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.StopMethodSkip = mask
		return nil

	case SettingAppStopMethodConsole:
		if reset {
			service.StopConsoleDelay = defaults.StopConsoleDelay
			return nil
		}
		d, err := parseDurationValue(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.StopConsoleDelay = d
		return nil

	case SettingAppStopMethodWindow:
		if reset {
			service.StopWindowDelay = defaults.StopWindowDelay
			return nil
		}
		d, err := parseDurationValue(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.StopWindowDelay = d
		return nil

	case SettingAppStopMethodThreads:
		if reset {
			service.StopThreadsDelay = defaults.StopThreadsDelay
			return nil
		}
		d, err := parseDurationValue(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.StopThreadsDelay = d
		return nil

	case SettingAppStdin:
		if reset {
			service.StdinPath = ""
			return nil
		}
		service.StdinPath = strings.Join(values, " ")
		return nil

	case SettingAppStdout:
		if reset {
			service.StdoutPath = ""
			return nil
		}
		service.StdoutPath = strings.Join(values, " ")
		return nil

	case SettingAppStderr:
		if reset {
			service.StderrPath = ""
			return nil
		}
		service.StderrPath = strings.Join(values, " ")
		return nil

	case SettingAppNoConsole:
		if reset {
			service.NoConsole = defaults.NoConsole
			return nil
		}
		b, err := parseBoolValue(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.NoConsole = b
		return nil

	case SettingAppKillProcessTree:
		if reset {
			service.KillProcessTree = defaults.KillProcessTree
			return nil
		}
		b, err := parseBoolValue(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.KillProcessTree = b
		return nil

	case SettingDisplayName:
		if reset {
			service.DisplayName = defaults.DisplayName
			return nil
		}
		service.DisplayName = strings.Join(values, " ")
		return nil

	case SettingDescription:
		if reset {
			service.Description = ""
			return nil
		}
		service.Description = strings.Join(values, " ")
		return nil

	case SettingStart:
		if reset {
			service.Startup = defaults.Startup
			return nil
		}
		startup, err := parseStartup(singleValue(values))
		if err != nil {
			return fmt.Errorf("parse %s: %w", setting, err)
		}
		service.Startup = startup
		return nil

	case SettingDependOnService:
		if reset {
			service.Dependencies = nil
			return nil
		}
		service.Dependencies = append([]string(nil), values...)
		return nil
	}

	return fmt.Errorf("unsupported setting %q", setting)
}

// Read returns the printable value for a setting.
func Read(service Service, setting Setting, additional string) ([]string, error) {
	switch setting {
	case SettingApplication:
		return []string{service.Executable}, nil
	case SettingAppParameters:
		return []string{service.Arguments}, nil
	case SettingAppDirectory:
		return []string{service.Directory}, nil
	case SettingAppEnvironment:
		return append([]string(nil), service.Environment...), nil
	case SettingAppEnvironmentExtra:
		return append([]string(nil), service.EnvironmentExtra...), nil
	case SettingAppExit:
		return []string{string(resolveExitSetting(service, additional))}, nil
	case SettingAppRestartDelay:
		return []string{Milliseconds(service.RestartDelay)}, nil
	case SettingAppThrottle:
		return []string{Milliseconds(service.ThrottleDelay)}, nil
	case SettingAppPriority:
		return []string{string(service.Priority)}, nil
	case SettingAppAffinity:
		return []string{FormatAffinityMask(service.Affinity)}, nil
	case SettingAppStopMethodSkip:
		return []string{strconv.FormatUint(uint64(service.StopMethodSkip), 10)}, nil
	case SettingAppStopMethodConsole:
		return []string{Milliseconds(service.StopConsoleDelay)}, nil
	case SettingAppStopMethodWindow:
		return []string{Milliseconds(service.StopWindowDelay)}, nil
	case SettingAppStopMethodThreads:
		return []string{Milliseconds(service.StopThreadsDelay)}, nil
	case SettingAppStdin:
		return []string{service.StdinPath}, nil
	case SettingAppStdout:
		return []string{service.StdoutPath}, nil
	case SettingAppStderr:
		return []string{service.StderrPath}, nil
	case SettingAppNoConsole:
		return []string{boolString(service.NoConsole)}, nil
	case SettingAppKillProcessTree:
		return []string{boolString(service.KillProcessTree)}, nil
	case SettingDisplayName:
		return []string{service.DisplayName}, nil
	case SettingDescription:
		return []string{service.Description}, nil
	case SettingStart:
		return []string{string(service.Startup)}, nil
	case SettingDependOnService:
		return append([]string(nil), service.Dependencies...), nil
	}

	return nil, fmt.Errorf("unsupported setting %q", setting)
}

func applyExitSetting(service *Service, additional string, values []string, reset bool) error {
	if service.ExitActions == nil {
		service.ExitActions = make(map[int]ExitAction)
	}

	defaultKey := isDefaultExitKey(additional)
	if reset {
		if defaultKey {
			service.DefaultExitAction = ExitActionRestart
			return nil
		}

		code, err := strconv.Atoi(additional)
		if err != nil {
			return fmt.Errorf("parse exit code: %w", err)
		}
		delete(service.ExitActions, code)
		return nil
	}

	action, err := parseExitAction(singleValue(values))
	if err != nil {
		return err
	}

	if defaultKey {
		service.DefaultExitAction = action
		return nil
	}

	code, err := strconv.Atoi(additional)
	if err != nil {
		return fmt.Errorf("parse exit code: %w", err)
	}
	service.ExitActions[code] = action
	return nil
}

func resolveExitSetting(service Service, additional string) ExitAction {
	if isDefaultExitKey(additional) {
		if service.DefaultExitAction == "" {
			return ExitActionRestart
		}
		return service.DefaultExitAction
	}

	code, err := strconv.Atoi(additional)
	if err != nil {
		return service.DefaultExitAction
	}

	if action, ok := service.ExitActions[code]; ok {
		return action
	}
	return service.DefaultExitAction
}

func parseDurationValue(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, fmt.Errorf("value is required")
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Duration(ms) * time.Millisecond, nil
	}
	return time.ParseDuration(raw)
}

func parseBoolValue(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
}

func parseExitAction(raw string) (ExitAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "restart":
		return ExitActionRestart, nil
	case "ignore":
		return ExitActionIgnore, nil
	case "exit", "suicide":
		return ExitActionExit, nil
	default:
		return "", fmt.Errorf("invalid exit action %q", raw)
	}
}

func parseStartup(raw string) (StartupType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "service_auto_start", "service_auto", "auto", "automatic":
		return StartupAutomatic, nil
	case "service_delayed_auto_start", "delayed", "delayed-auto":
		return StartupDelayed, nil
	case "service_demand_start", "manual", "demand":
		return StartupManual, nil
	case "service_disabled", "disabled":
		return StartupDisabled, nil
	default:
		return "", fmt.Errorf("invalid startup type %q", raw)
	}
}

func parseStopMethodSkip(raw string) (StopMethodSkip, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 0, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid stop method skip mask %q", raw)
	}

	mask := StopMethodSkip(value)
	if mask&^StopMethodAll != 0 {
		return 0, fmt.Errorf("unsupported stop method skip mask %#x", value)
	}
	return mask, nil
}

func singleValue(values []string) string {
	return strings.Join(values, " ")
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func isDefaultExitKey(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return trimmed == "" || strings.EqualFold(trimmed, "default") || trimmed == "*"
}

func filepathDir(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}
