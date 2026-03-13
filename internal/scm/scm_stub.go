//go:build !windows

package scm

import (
	"errors"

	"github.com/jonlabelle/nssm-redux/internal/config"
)

// ErrUnsupported indicates SCM operations require Windows.
var ErrUnsupported = errors.New("service management is only available on windows")

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

func Install(_ string, _ config.Service) error {
	return ErrUnsupported
}

func Load(_ string) (config.Service, error) {
	return config.Service{}, ErrUnsupported
}

func Save(_ config.Service) error {
	return ErrUnsupported
}

func Remove(_ string) error {
	return ErrUnsupported
}

func Start(_ string) error {
	return ErrUnsupported
}

func Stop(_ string) error {
	return ErrUnsupported
}

func Restart(_ string) error {
	return ErrUnsupported
}

func Query(_ string) (ServiceStatus, error) {
	return ServiceStatus{}, ErrUnsupported
}

func ListManaged() ([]string, error) {
	return nil, ErrUnsupported
}
