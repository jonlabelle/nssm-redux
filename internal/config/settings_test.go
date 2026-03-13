package config

import "testing"

func TestApplyAndRead(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	service.Executable = `C:\app.exe`

	if err := Apply(&service, SettingAppRestartDelay, "", []string{"5s"}, false); err != nil {
		t.Fatalf("Apply(AppRestartDelay) error = %v", err)
	}
	if err := Apply(&service, SettingAppExit, "12", []string{"Ignore"}, false); err != nil {
		t.Fatalf("Apply(AppExit) error = %v", err)
	}

	values, err := Read(service, SettingAppRestartDelay, "")
	if err != nil {
		t.Fatalf("Read(AppRestartDelay) error = %v", err)
	}
	if len(values) != 1 || values[0] != "5000" {
		t.Fatalf("Read(AppRestartDelay) = %#v, want [5000]", values)
	}

	values, err = Read(service, SettingAppExit, "12")
	if err != nil {
		t.Fatalf("Read(AppExit) error = %v", err)
	}
	if len(values) != 1 || values[0] != "Ignore" {
		t.Fatalf("Read(AppExit) = %#v, want [Ignore]", values)
	}
}

func TestApplyReset(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	service.Executable = `C:\app.exe`
	service.ThrottleDelay = 10

	if err := Apply(&service, SettingAppThrottle, "", nil, true); err != nil {
		t.Fatalf("Apply(reset AppThrottle) error = %v", err)
	}
	if service.ThrottleDelay != Default("svc").ThrottleDelay {
		t.Fatalf("ThrottleDelay = %v, want %v", service.ThrottleDelay, Default("svc").ThrottleDelay)
	}
}
