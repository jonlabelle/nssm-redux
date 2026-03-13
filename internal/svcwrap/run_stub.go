//go:build !windows

package svcwrap

import (
	"context"
	"errors"
)

// ErrUnsupported indicates service hosting requires Windows.
var ErrUnsupported = errors.New("windows service hosting is only available on windows")

// Run starts the SCM-facing service host.
func Run(_ context.Context, _ string) error {
	return ErrUnsupported
}
