package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultThrottleDelay = 1500 * time.Millisecond
)

// ExitAction controls how nssmr reacts when the managed process exits.
type ExitAction string

const (
	ExitActionRestart ExitAction = "Restart"
	ExitActionIgnore  ExitAction = "Ignore"
	ExitActionExit    ExitAction = "Exit"
)

// StartupType mirrors the service start types used by Windows SCM.
type StartupType string

const (
	StartupAutomatic StartupType = "SERVICE_AUTO_START"
	StartupDelayed   StartupType = "SERVICE_DELAYED_AUTO_START"
	StartupManual    StartupType = "SERVICE_DEMAND_START"
	StartupDisabled  StartupType = "SERVICE_DISABLED"
)

// Service models the supported nssmr configuration surface.
type Service struct {
	Name              string
	DisplayName       string
	Description       string
	Executable        string
	Arguments         string
	Directory         string
	Environment       []string
	EnvironmentExtra  []string
	Dependencies      []string
	Priority          PriorityClass
	Affinity          AffinityMask
	StdinPath         string
	StdoutPath        string
	StderrPath        string
	Startup           StartupType
	RestartDelay      time.Duration
	ThrottleDelay     time.Duration
	StopMethodSkip    StopMethodSkip
	StopConsoleDelay  time.Duration
	StopWindowDelay   time.Duration
	StopThreadsDelay  time.Duration
	DefaultExitAction ExitAction
	ExitActions       map[int]ExitAction
	NoConsole         bool
	KillProcessTree   bool
}

// Default returns the default service configuration for a service name.
func Default(name string) Service {
	return Service{
		Name:              name,
		DisplayName:       name,
		Priority:          PriorityNormal,
		Startup:           StartupAutomatic,
		ThrottleDelay:     defaultThrottleDelay,
		StopConsoleDelay:  defaultStopMethodDelay,
		StopWindowDelay:   defaultStopMethodDelay,
		StopThreadsDelay:  defaultStopMethodDelay,
		DefaultExitAction: ExitActionRestart,
		ExitActions:       make(map[int]ExitAction),
		KillProcessTree:   true,
	}
}

// Clone returns a deep copy of the service configuration.
func (s Service) Clone() Service {
	clone := s
	clone.Environment = append([]string(nil), s.Environment...)
	clone.EnvironmentExtra = append([]string(nil), s.EnvironmentExtra...)
	clone.Dependencies = append([]string(nil), s.Dependencies...)
	clone.ExitActions = make(map[int]ExitAction, len(s.ExitActions))
	for code, action := range s.ExitActions {
		clone.ExitActions[code] = action
	}
	return clone
}

// Normalize fills in derived defaults.
func (s *Service) Normalize() {
	if s.DisplayName == "" {
		s.DisplayName = s.Name
	}
	if s.Startup == "" {
		s.Startup = StartupAutomatic
	}
	if s.Priority == "" {
		s.Priority = PriorityNormal
	}
	if s.ThrottleDelay < 0 {
		s.ThrottleDelay = 0
	}
	if s.RestartDelay < 0 {
		s.RestartDelay = 0
	}
	if s.DefaultExitAction == "" {
		s.DefaultExitAction = ExitActionRestart
	}
	if s.ExitActions == nil {
		s.ExitActions = make(map[int]ExitAction)
	}
	if s.Directory == "" && s.Executable != "" {
		s.Directory = filepath.Dir(s.Executable)
	}
}

// Validate validates the service configuration.
func (s *Service) Validate() error {
	s.Normalize()

	if strings.TrimSpace(s.Name) == "" {
		return errors.New("service name is required")
	}
	if strings.TrimSpace(s.Executable) == "" {
		return errors.New("application path is required")
	}
	if !validStartupType(s.Startup) {
		return fmt.Errorf("unsupported startup type %q", s.Startup)
	}
	if !validPriorityClass(s.Priority) {
		return fmt.Errorf("unsupported priority class %q", s.Priority)
	}
	if !validExitAction(s.DefaultExitAction) {
		return fmt.Errorf("unsupported default exit action %q", s.DefaultExitAction)
	}
	if s.StopMethodSkip&^StopMethodAll != 0 {
		return fmt.Errorf("unsupported stop method skip mask %#x", uint32(s.StopMethodSkip))
	}
	if s.StopConsoleDelay < 0 || s.StopWindowDelay < 0 || s.StopThreadsDelay < 0 {
		return errors.New("stop method delays must be non-negative")
	}
	for code, action := range s.ExitActions {
		if code < 0 {
			return fmt.Errorf("exit action code %d is invalid", code)
		}
		if !validExitAction(action) {
			return fmt.Errorf("unsupported exit action %q for code %d", action, code)
		}
	}
	for _, entry := range append([]string{}, append(s.Environment, s.EnvironmentExtra...)...) {
		if _, _, ok := strings.Cut(entry, "="); !ok {
			return fmt.Errorf("environment entry %q must use KEY=VALUE format", entry)
		}
	}
	return nil
}

// ExitActionFor resolves the action for a process exit code.
func (s Service) ExitActionFor(code int) ExitAction {
	if action, ok := s.ExitActions[code]; ok {
		return action
	}
	if s.DefaultExitAction == "" {
		return ExitActionRestart
	}
	return s.DefaultExitAction
}

// SortedExitCodes returns configured exit-action codes in ascending order.
func (s Service) SortedExitCodes() []int {
	codes := make([]int, 0, len(s.ExitActions))
	for code := range s.ExitActions {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	return codes
}

// Milliseconds renders a duration as a whole number of milliseconds.
func Milliseconds(d time.Duration) string {
	return fmt.Sprintf("%d", d.Milliseconds())
}

func validExitAction(action ExitAction) bool {
	switch action {
	case ExitActionRestart, ExitActionIgnore, ExitActionExit:
		return true
	default:
		return false
	}
}

func validStartupType(startup StartupType) bool {
	switch startup {
	case StartupAutomatic, StartupDelayed, StartupManual, StartupDisabled:
		return true
	default:
		return false
	}
}
