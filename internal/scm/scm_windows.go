//go:build windows

package scm

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const eventSource = "nssmr"

// ErrNotManaged indicates the named service is not managed by nssmr.
var ErrNotManaged = errors.New("service is not managed by nssmr")

// ServiceStatus is the normalized service status returned by Query.
type ServiceStatus struct {
	Name                    string
	State                   string
	StateCode               uint32
	ProcessID               uint32
	Win32ExitCode           uint32
	ServiceSpecificExitCode uint32
}

func Install(selfPath string, service config.Service) error {
	if err := service.Validate(); err != nil {
		return err
	}

	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { _ = manager.Disconnect() }()

	startType, delayed, err := mapStartup(service.Startup)
	if err != nil {
		return err
	}

	cfg := mgr.Config{
		DisplayName:      service.DisplayName,
		Description:      service.Description,
		StartType:        startType,
		ErrorControl:     mgr.ErrorNormal,
		Dependencies:     append([]string(nil), service.Dependencies...),
		DelayedAutoStart: delayed,
		SidType:          windows.SERVICE_SID_TYPE_UNRESTRICTED,
	}

	svcHandle, err := manager.CreateService(service.Name, selfPath, cfg, "service", service.Name)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer func() { _ = svcHandle.Close() }()

	if err := saveParameters(service); err != nil {
		_ = svcHandle.Delete()
		return err
	}

	_ = ensureEventSource()
	return nil
}

func Load(name string) (config.Service, error) {
	manager, serviceHandle, scmConfig, err := openService(name)
	if err != nil {
		return config.Service{}, err
	}
	defer func() { _ = manager.Disconnect() }()
	defer func() { _ = serviceHandle.Close() }()

	service := config.Default(name)
	service.DisplayName = scmConfig.DisplayName
	service.Description = scmConfig.Description
	service.Dependencies = append([]string(nil), scmConfig.Dependencies...)
	service.Startup = unmapStartup(scmConfig)

	if err := loadParameters(&service); err != nil {
		return config.Service{}, err
	}

	service.Normalize()
	return service, nil
}

func Save(service config.Service) error {
	if err := service.Validate(); err != nil {
		return err
	}

	manager, serviceHandle, scmConfig, err := openService(service.Name)
	if err != nil {
		return err
	}
	defer func() { _ = manager.Disconnect() }()
	defer func() { _ = serviceHandle.Close() }()

	startType, delayed, err := mapStartup(service.Startup)
	if err != nil {
		return err
	}

	scmConfig.DisplayName = service.DisplayName
	scmConfig.Description = service.Description
	scmConfig.StartType = startType
	scmConfig.DelayedAutoStart = delayed
	scmConfig.Dependencies = append([]string(nil), service.Dependencies...)

	if err := serviceHandle.UpdateConfig(scmConfig); err != nil {
		return fmt.Errorf("update service config: %w", err)
	}

	return saveParameters(service)
}

func Remove(name string) error {
	_ = Stop(name)

	manager, serviceHandle, _, err := openService(name)
	if err != nil {
		return err
	}
	defer func() { _ = manager.Disconnect() }()
	defer func() { _ = serviceHandle.Close() }()

	if err := serviceHandle.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}

	return deleteTree(registry.LOCAL_MACHINE, parametersKeyPath(name))
}

func Start(name string) error {
	manager, serviceHandle, _, err := openService(name)
	if err != nil {
		return err
	}
	defer func() { _ = manager.Disconnect() }()
	defer func() { _ = serviceHandle.Close() }()

	status, err := serviceHandle.Query()
	if err == nil && status.State == svc.Running {
		return nil
	}

	if err := serviceHandle.Start(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}

	return waitForState(serviceHandle, svc.Running)
}

func Stop(name string) error {
	manager, serviceHandle, _, err := openService(name)
	if err != nil {
		return err
	}
	defer func() { _ = manager.Disconnect() }()
	defer func() { _ = serviceHandle.Close() }()

	status, err := serviceHandle.Query()
	if err == nil && status.State == svc.Stopped {
		return nil
	}

	if _, err := serviceHandle.Control(svc.Stop); err != nil && err != windows.ERROR_SERVICE_NOT_ACTIVE {
		return fmt.Errorf("stop service: %w", err)
	}

	return waitForState(serviceHandle, svc.Stopped)
}

func Restart(name string) error {
	if err := Stop(name); err != nil {
		return err
	}
	return Start(name)
}

func Query(name string) (ServiceStatus, error) {
	manager, serviceHandle, _, err := openService(name)
	if err != nil {
		return ServiceStatus{}, err
	}
	defer func() { _ = manager.Disconnect() }()
	defer func() { _ = serviceHandle.Close() }()

	status, err := serviceHandle.Query()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("query service: %w", err)
	}

	return ServiceStatus{
		Name:                    name,
		State:                   stateName(status.State),
		StateCode:               uint32(status.State),
		ProcessID:               status.ProcessId,
		Win32ExitCode:           status.Win32ExitCode,
		ServiceSpecificExitCode: status.ServiceSpecificExitCode,
	}, nil
}

func ListManaged() ([]string, error) {
	manager, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { _ = manager.Disconnect() }()

	services, err := manager.ListServices()
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	managed := make([]string, 0, len(services))
	for _, name := range services {
		ok, err := isManaged(name)
		if err != nil {
			return nil, err
		}
		if ok {
			managed = append(managed, name)
		}
	}
	return managed, nil
}

func ensureEventSource() error {
	if err := eventlog.InstallAsEventCreate(eventSource, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		return err
	}
	return nil
}

func openService(name string) (*mgr.Mgr, *mgr.Service, mgr.Config, error) {
	manager, err := mgr.Connect()
	if err != nil {
		return nil, nil, mgr.Config{}, fmt.Errorf("connect to service manager: %w", err)
	}

	serviceHandle, err := manager.OpenService(name)
	if err != nil {
		manager.Disconnect()
		return nil, nil, mgr.Config{}, fmt.Errorf("open service %s: %w", name, err)
	}

	scmConfig, err := serviceHandle.Config()
	if err != nil {
		serviceHandle.Close()
		manager.Disconnect()
		return nil, nil, mgr.Config{}, fmt.Errorf("query service config: %w", err)
	}

	return manager, serviceHandle, scmConfig, nil
}

func waitForState(serviceHandle *mgr.Service, want svc.State) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, err := serviceHandle.Query()
		if err != nil {
			return err
		}
		if status.State == want {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", stateName(want))
}

func mapStartup(startup config.StartupType) (uint32, bool, error) {
	switch startup {
	case config.StartupAutomatic:
		return mgr.StartAutomatic, false, nil
	case config.StartupDelayed:
		return mgr.StartAutomatic, true, nil
	case config.StartupManual:
		return mgr.StartManual, false, nil
	case config.StartupDisabled:
		return mgr.StartDisabled, false, nil
	default:
		return 0, false, fmt.Errorf("unsupported startup type %q", startup)
	}
}

func unmapStartup(cfg mgr.Config) config.StartupType {
	switch cfg.StartType {
	case mgr.StartManual:
		return config.StartupManual
	case mgr.StartDisabled:
		return config.StartupDisabled
	case mgr.StartAutomatic:
		if cfg.DelayedAutoStart {
			return config.StartupDelayed
		}
		return config.StartupAutomatic
	default:
		return config.StartupAutomatic
	}
}

func stateName(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "SERVICE_STOPPED"
	case svc.StartPending:
		return "SERVICE_START_PENDING"
	case svc.StopPending:
		return "SERVICE_STOP_PENDING"
	case svc.Running:
		return "SERVICE_RUNNING"
	case svc.ContinuePending:
		return "SERVICE_CONTINUE_PENDING"
	case svc.PausePending:
		return "SERVICE_PAUSE_PENDING"
	case svc.Paused:
		return "SERVICE_PAUSED"
	default:
		return fmt.Sprintf("SERVICE_STATE_%d", uint32(state))
	}
}

func saveParameters(service config.Service) error {
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, parametersKeyPath(service.Name), registry.SET_VALUE|registry.CREATE_SUB_KEY|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open parameters registry: %w", err)
	}
	defer func() { _ = key.Close() }()

	if err := setExpandString(key, string(config.SettingApplication), service.Executable); err != nil {
		return err
	}
	if err := setExpandString(key, string(config.SettingAppParameters), service.Arguments); err != nil {
		return err
	}
	if err := setExpandString(key, string(config.SettingAppDirectory), service.Directory); err != nil {
		return err
	}
	if err := setStrings(key, string(config.SettingAppEnvironment), service.Environment); err != nil {
		return err
	}
	if err := setStrings(key, string(config.SettingAppEnvironmentExtra), service.EnvironmentExtra); err != nil {
		return err
	}
	if err := setExpandString(key, string(config.SettingAppStdin), service.StdinPath); err != nil {
		return err
	}
	if err := setExpandString(key, string(config.SettingAppStdout), service.StdoutPath); err != nil {
		return err
	}
	if err := setExpandString(key, string(config.SettingAppStderr), service.StderrPath); err != nil {
		return err
	}
	if err := setMilliseconds(key, string(config.SettingAppRestartDelay), service.RestartDelay, 0); err != nil {
		return err
	}
	if err := setMilliseconds(key, string(config.SettingAppThrottle), service.ThrottleDelay, 1500*time.Millisecond); err != nil {
		return err
	}
	if err := setPriorityValue(key, string(config.SettingAppPriority), service.Priority, config.PriorityNormal); err != nil {
		return err
	}
	if err := setStringValue(key, string(config.SettingAppAffinity), config.FormatAffinityMask(service.Affinity)); err != nil {
		return err
	}
	if err := setDefaultedDWord(key, string(config.SettingAppStopMethodSkip), uint32(service.StopMethodSkip), 0); err != nil {
		return err
	}
	if err := setMilliseconds(key, string(config.SettingAppStopMethodConsole), service.StopConsoleDelay, 1500*time.Millisecond); err != nil {
		return err
	}
	if err := setMilliseconds(key, string(config.SettingAppStopMethodWindow), service.StopWindowDelay, 1500*time.Millisecond); err != nil {
		return err
	}
	if err := setMilliseconds(key, string(config.SettingAppStopMethodThreads), service.StopThreadsDelay, 1500*time.Millisecond); err != nil {
		return err
	}
	if err := saveHooks(service); err != nil {
		return err
	}
	if err := setDefaultedBool(key, string(config.SettingAppRotateFiles), service.Logging.Enabled, false); err != nil {
		return err
	}
	if err := setDefaultedBool(key, string(config.SettingAppRotateOnline), service.Logging.Online, false); err != nil {
		return err
	}
	if err := setSeconds(key, string(config.SettingAppRotateSeconds), service.Logging.AgeThreshold, 0); err != nil {
		return err
	}
	if err := setDefaultedDWord(key, string(config.SettingAppRotateBytes), service.Logging.SizeLow(), 0); err != nil {
		return err
	}
	if err := setDefaultedDWord(key, string(config.SettingAppRotateBytesHigh), service.Logging.SizeHigh(), 0); err != nil {
		return err
	}
	if err := setMilliseconds(key, string(config.SettingAppRotateDelay), service.Logging.RotateDelay, 0); err != nil {
		return err
	}
	if err := setDefaultedBool(key, string(config.SettingAppTimestampLog), service.Logging.TimestampLog, false); err != nil {
		return err
	}
	if err := setDefaultedBool(key, string(config.SettingAppNoConsole), service.NoConsole, false); err != nil {
		return err
	}
	if err := setDefaultedBool(key, string(config.SettingAppKillProcessTree), service.KillProcessTree, true); err != nil {
		return err
	}

	if err := saveExitActions(service); err != nil {
		return err
	}

	return nil
}

func loadParameters(service *config.Service) error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, parametersKeyPath(service.Name), registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return ErrNotManaged
		}
		return fmt.Errorf("open parameters registry: %w", err)
	}
	defer func() { _ = key.Close() }()

	executable, _, err := key.GetStringValue(string(config.SettingApplication))
	if err != nil {
		if err == registry.ErrNotExist {
			return ErrNotManaged
		}
		return fmt.Errorf("read application: %w", err)
	}
	service.Executable = executable

	service.Arguments = getStringValue(key, string(config.SettingAppParameters))
	service.Directory = getStringValue(key, string(config.SettingAppDirectory))
	service.StdinPath = getStringValue(key, string(config.SettingAppStdin))
	service.StdoutPath = getStringValue(key, string(config.SettingAppStdout))
	service.StderrPath = getStringValue(key, string(config.SettingAppStderr))
	service.Environment = getStringsValue(key, string(config.SettingAppEnvironment))
	service.EnvironmentExtra = getStringsValue(key, string(config.SettingAppEnvironmentExtra))
	service.RestartDelay = getMillisecondsValue(key, string(config.SettingAppRestartDelay), 0)
	service.ThrottleDelay = getMillisecondsValue(key, string(config.SettingAppThrottle), 1500*time.Millisecond)
	service.Priority = getPriorityValue(key, string(config.SettingAppPriority), config.PriorityNormal)
	service.Affinity = getAffinityValue(key, string(config.SettingAppAffinity))
	service.StopMethodSkip = getStopMethodValue(key, string(config.SettingAppStopMethodSkip), 0)
	service.StopConsoleDelay = getMillisecondsValue(key, string(config.SettingAppStopMethodConsole), 1500*time.Millisecond)
	service.StopWindowDelay = getMillisecondsValue(key, string(config.SettingAppStopMethodWindow), 1500*time.Millisecond)
	service.StopThreadsDelay = getMillisecondsValue(key, string(config.SettingAppStopMethodThreads), 1500*time.Millisecond)
	service.Logging.Enabled = getBoolValue(key, string(config.SettingAppRotateFiles), false)
	service.Logging.Online = getBoolValue(key, string(config.SettingAppRotateOnline), false)
	service.Logging.AgeThreshold = getSecondsValue(key, string(config.SettingAppRotateSeconds), 0)
	service.Logging.SizeBytes = uint64(getDWordValue(key, string(config.SettingAppRotateBytes), 0))
	service.Logging.SizeBytes |= uint64(getDWordValue(key, string(config.SettingAppRotateBytesHigh), 0)) << 32
	service.Logging.RotateDelay = getMillisecondsValue(key, string(config.SettingAppRotateDelay), 0)
	service.Logging.TimestampLog = getBoolValue(key, string(config.SettingAppTimestampLog), false)
	service.NoConsole = getBoolValue(key, string(config.SettingAppNoConsole), false)
	service.KillProcessTree = getBoolValue(key, string(config.SettingAppKillProcessTree), true)

	if err := loadHooks(service); err != nil {
		return err
	}
	if err := loadExitActions(service); err != nil {
		return err
	}

	return nil
}

func saveExitActions(service config.Service) error {
	path := exitActionsKeyPath(service.Name)
	if err := deleteTree(registry.LOCAL_MACHINE, path); err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, path, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open exit action registry: %w", err)
	}
	defer func() { _ = key.Close() }()

	defaultAction := service.DefaultExitAction
	if defaultAction == "" {
		defaultAction = config.ExitActionRestart
	}
	if err := key.SetStringValue("", string(defaultAction)); err != nil {
		return fmt.Errorf("write default exit action: %w", err)
	}

	for _, code := range service.SortedExitCodes() {
		if err := key.SetStringValue(strconv.Itoa(code), string(service.ExitActions[code])); err != nil {
			return fmt.Errorf("write exit action for %d: %w", code, err)
		}
	}

	return nil
}

func loadExitActions(service *config.Service) error {
	service.DefaultExitAction = config.ExitActionRestart
	service.ExitActions = make(map[int]config.ExitAction)

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, exitActionsKeyPath(service.Name), registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return fmt.Errorf("open exit action registry: %w", err)
	}
	defer func() { _ = key.Close() }()

	if value, _, err := key.GetStringValue(""); err == nil {
		action, err := parseExitAction(value)
		if err == nil {
			service.DefaultExitAction = action
		}
	}

	names, err := key.ReadValueNames(-1)
	if err != nil {
		return fmt.Errorf("list exit action values: %w", err)
	}
	for _, name := range names {
		if name == "" {
			continue
		}
		code, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		action, err := parseExitAction(value)
		if err != nil {
			continue
		}
		service.ExitActions[code] = action
	}

	return nil
}

func isManaged(name string) (bool, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, parametersKeyPath(name), registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, fmt.Errorf("open parameters registry: %w", err)
	}
	defer func() { _ = key.Close() }()

	_, _, err = key.GetStringValue(string(config.SettingApplication))
	if err == nil {
		return true, nil
	}
	if err == registry.ErrNotExist {
		return false, nil
	}
	return false, fmt.Errorf("read application value: %w", err)
}

func parametersKeyPath(name string) string {
	return `SYSTEM\CurrentControlSet\Services\` + name + `\Parameters`
}

func exitActionsKeyPath(name string) string {
	return parametersKeyPath(name) + `\` + string(config.SettingAppExit)
}

func hooksKeyPath(name string) string {
	return parametersKeyPath(name) + `\` + string(config.SettingAppEvents)
}

func deleteTree(root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return fmt.Errorf("open registry tree %s: %w", path, err)
	}

	subkeys, err := key.ReadSubKeyNames(-1)
	key.Close()
	if err != nil {
		return fmt.Errorf("list subkeys for %s: %w", path, err)
	}

	for _, subkey := range subkeys {
		if err := deleteTree(root, path+`\`+subkey); err != nil {
			return err
		}
	}

	if err := registry.DeleteKey(root, path); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("delete registry key %s: %w", path, err)
	}
	return nil
}

func setExpandString(key registry.Key, name, value string) error {
	if value == "" {
		return deleteValueIfExists(key, name)
	}
	if err := key.SetExpandStringValue(name, value); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func setStringValue(key registry.Key, name, value string) error {
	if value == "" {
		return deleteValueIfExists(key, name)
	}
	if err := key.SetStringValue(name, value); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func setStrings(key registry.Key, name string, value []string) error {
	if len(value) == 0 {
		return deleteValueIfExists(key, name)
	}
	if err := key.SetStringsValue(name, value); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func setDefaultedDWord(key registry.Key, name string, value, defaultValue uint32) error {
	if value == defaultValue {
		return deleteValueIfExists(key, name)
	}
	if err := key.SetDWordValue(name, value); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func setPriorityValue(key registry.Key, name string, value, defaultValue config.PriorityClass) error {
	if value == "" {
		value = config.PriorityNormal
	}
	if defaultValue == "" {
		defaultValue = config.PriorityNormal
	}
	return setDefaultedDWord(key, name, value.WindowsValue(), defaultValue.WindowsValue())
}

func setMilliseconds(key registry.Key, name string, value, defaultValue time.Duration) error {
	if value == defaultValue {
		return deleteValueIfExists(key, name)
	}

	if value < 0 || value > time.Duration(math.MaxUint32)*time.Millisecond {
		return fmt.Errorf("%s is out of range", name)
	}

	if err := key.SetDWordValue(name, uint32(value.Milliseconds())); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func setSeconds(key registry.Key, name string, value, defaultValue time.Duration) error {
	if value == defaultValue {
		return deleteValueIfExists(key, name)
	}
	if value < 0 || value > time.Duration(math.MaxUint32)*time.Second {
		return fmt.Errorf("%s is out of range", name)
	}
	if err := key.SetDWordValue(name, uint32(value/time.Second)); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func setDefaultedBool(key registry.Key, name string, value, defaultValue bool) error {
	if value == defaultValue {
		return deleteValueIfExists(key, name)
	}

	var raw uint32
	if value {
		raw = 1
	}
	if err := key.SetDWordValue(name, raw); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func deleteValueIfExists(key registry.Key, name string) error {
	if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("delete %s: %w", name, err)
	}
	return nil
}

func getStringValue(key registry.Key, name string) string {
	value, _, err := key.GetStringValue(name)
	if err != nil {
		return ""
	}
	return value
}

func getStringsValue(key registry.Key, name string) []string {
	value, _, err := key.GetStringsValue(name)
	if err != nil {
		return nil
	}
	return value
}

func getMillisecondsValue(key registry.Key, name string, defaultValue time.Duration) time.Duration {
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return defaultValue
	}
	return time.Duration(value) * time.Millisecond
}

func getSecondsValue(key registry.Key, name string, defaultValue time.Duration) time.Duration {
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return defaultValue
	}
	return time.Duration(value) * time.Second
}

func getBoolValue(key registry.Key, name string, defaultValue bool) bool {
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return defaultValue
	}
	return value != 0
}

func getDWordValue(key registry.Key, name string, defaultValue uint32) uint32 {
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return defaultValue
	}
	return uint32(value)
}

func getPriorityValue(key registry.Key, name string, defaultValue config.PriorityClass) config.PriorityClass {
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return defaultValue
	}
	priority, ok := config.PriorityClassFromWindowsValue(uint32(value))
	if !ok {
		return defaultValue
	}
	return priority
}

func getAffinityValue(key registry.Key, name string) config.AffinityMask {
	value, _, err := key.GetStringValue(name)
	if err != nil {
		return 0
	}
	mask, err := config.ParseAffinityMask(value)
	if err != nil {
		return 0
	}
	return mask
}

func getStopMethodValue(key registry.Key, name string, defaultValue config.StopMethodSkip) config.StopMethodSkip {
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return defaultValue
	}
	mask := config.StopMethodSkip(value)
	if mask&^config.StopMethodAll != 0 {
		return defaultValue
	}
	return mask
}

func saveHooks(service config.Service) error {
	path := hooksKeyPath(service.Name)
	if err := deleteTree(registry.LOCAL_MACHINE, path); err != nil {
		return err
	}
	if len(service.Hooks) == 0 {
		return nil
	}

	for _, hook := range config.SupportedHooks() {
		command, ok := service.Hooks[hook]
		if !ok || strings.TrimSpace(command) == "" {
			continue
		}

		info := hook.Info()
		key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, path+`\`+info.Event, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("open hook registry: %w", err)
		}
		if err := key.SetExpandStringValue(info.Action, command); err != nil {
			key.Close()
			return fmt.Errorf("write hook %s: %w", hook, err)
		}
		key.Close()
	}
	return nil
}

func loadHooks(service *config.Service) error {
	service.Hooks = make(map[config.Hook]string)

	for _, hook := range config.SupportedHooks() {
		info := hook.Info()
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, hooksKeyPath(service.Name)+`\`+info.Event, registry.QUERY_VALUE)
		if err != nil {
			if err == registry.ErrNotExist {
				continue
			}
			return fmt.Errorf("open hook registry: %w", err)
		}
		value, _, err := key.GetStringValue(info.Action)
		key.Close()
		if err != nil {
			if err == registry.ErrNotExist {
				continue
			}
			return fmt.Errorf("read hook %s: %w", hook, err)
		}
		if strings.TrimSpace(value) != "" {
			service.Hooks[hook] = value
		}
	}
	return nil
}

func parseExitAction(raw string) (config.ExitAction, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "restart":
		return config.ExitActionRestart, nil
	case "ignore":
		return config.ExitActionIgnore, nil
	case "exit", "suicide":
		return config.ExitActionExit, nil
	default:
		return "", fmt.Errorf("invalid exit action %q", raw)
	}
}
