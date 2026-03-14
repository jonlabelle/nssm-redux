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
	if err := Apply(&service, SettingAppPriority, "", []string{"HIGH_PRIORITY_CLASS"}, false); err != nil {
		t.Fatalf("Apply(AppPriority) error = %v", err)
	}
	if err := Apply(&service, SettingAppAffinity, "", []string{"0-2,4"}, false); err != nil {
		t.Fatalf("Apply(AppAffinity) error = %v", err)
	}
	if err := Apply(&service, SettingAppStopMethodSkip, "", []string{"3"}, false); err != nil {
		t.Fatalf("Apply(AppStopMethodSkip) error = %v", err)
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

	values, err = Read(service, SettingAppPriority, "")
	if err != nil {
		t.Fatalf("Read(AppPriority) error = %v", err)
	}
	if len(values) != 1 || values[0] != "HIGH_PRIORITY_CLASS" {
		t.Fatalf("Read(AppPriority) = %#v, want [HIGH_PRIORITY_CLASS]", values)
	}

	values, err = Read(service, SettingAppAffinity, "")
	if err != nil {
		t.Fatalf("Read(AppAffinity) error = %v", err)
	}
	if len(values) != 1 || values[0] != "0-2,4" {
		t.Fatalf("Read(AppAffinity) = %#v, want [0-2,4]", values)
	}

	values, err = Read(service, SettingAppStopMethodSkip, "")
	if err != nil {
		t.Fatalf("Read(AppStopMethodSkip) error = %v", err)
	}
	if len(values) != 1 || values[0] != "3" {
		t.Fatalf("Read(AppStopMethodSkip) = %#v, want [3]", values)
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
