package runtime

import (
	"testing"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
)

func TestDecideRestartFastFailure(t *testing.T) {
	t.Parallel()

	service := config.Default("svc")
	service.Executable = `C:\app.exe`

	decision := DecideRestart(service, 500*time.Millisecond, 0, 1)
	if decision.Action != config.ExitActionRestart {
		t.Fatalf("Action = %q, want %q", decision.Action, config.ExitActionRestart)
	}
	if decision.Delay != 2*time.Second {
		t.Fatalf("Delay = %v, want 2s", decision.Delay)
	}
	if decision.ConsecutiveFast != 1 {
		t.Fatalf("ConsecutiveFast = %d, want 1", decision.ConsecutiveFast)
	}
}

func TestDecideRestartStableRun(t *testing.T) {
	t.Parallel()

	service := config.Default("svc")
	service.Executable = `C:\app.exe`
	service.RestartDelay = 3 * time.Second

	decision := DecideRestart(service, 10*time.Second, 4, 0)
	if decision.Delay != 3*time.Second {
		t.Fatalf("Delay = %v, want 3s", decision.Delay)
	}
	if decision.ConsecutiveFast != 0 {
		t.Fatalf("ConsecutiveFast = %d, want 0", decision.ConsecutiveFast)
	}
}
