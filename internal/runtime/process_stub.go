//go:build !windows

package runtime

import (
	"errors"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
)

// Logger is the logging interface used by the runtime package.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Result describes a completed process run.
type Result struct {
	ExitCode int
	Runtime  time.Duration
	Err      error
}

// Process is a platform-specific managed process.
type Process struct{}

// ErrUnsupported indicates the runtime can only run on Windows.
var ErrUnsupported = errors.New("nssmr runtime is only available on windows")

// Start returns an unsupported error on non-Windows platforms.
func Start(_ config.Service, _ Logger) (*Process, error) {
	return nil, ErrUnsupported
}

// Wait returns a closed channel on non-Windows platforms.
func (p *Process) Wait() <-chan Result {
	ch := make(chan Result)
	close(ch)
	return ch
}

// Stop returns an unsupported error on non-Windows platforms.
func (p *Process) Stop() error {
	return ErrUnsupported
}
