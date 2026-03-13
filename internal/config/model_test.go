package config

import "testing"

func TestDefault(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	if service.Name != "svc" {
		t.Fatalf("Name = %q, want svc", service.Name)
	}
	if service.DisplayName != "svc" {
		t.Fatalf("DisplayName = %q, want svc", service.DisplayName)
	}
	if service.Startup != StartupAutomatic {
		t.Fatalf("Startup = %q, want %q", service.Startup, StartupAutomatic)
	}
	if service.DefaultExitAction != ExitActionRestart {
		t.Fatalf("DefaultExitAction = %q, want %q", service.DefaultExitAction, ExitActionRestart)
	}
	if !service.KillProcessTree {
		t.Fatal("KillProcessTree = false, want true")
	}
}

func TestValidateRejectsMissingExecutable(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	if err := service.Validate(); err == nil {
		t.Fatal("Validate() succeeded, want error")
	}
}

func TestExitActionForFallsBackToDefault(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	service.DefaultExitAction = ExitActionIgnore
	service.ExitActions[5] = ExitActionExit

	if got := service.ExitActionFor(5); got != ExitActionExit {
		t.Fatalf("ExitActionFor(5) = %q, want %q", got, ExitActionExit)
	}
	if got := service.ExitActionFor(7); got != ExitActionIgnore {
		t.Fatalf("ExitActionFor(7) = %q, want %q", got, ExitActionIgnore)
	}
}
