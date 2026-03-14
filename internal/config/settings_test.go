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
	if err := Apply(&service, SettingAppEvents, "Start/Pre", []string{`C:\hooks\pre.cmd`}, false); err != nil {
		t.Fatalf("Apply(AppEvents) error = %v", err)
	}
	if err := Apply(&service, SettingAppRotateFiles, "", []string{"1"}, false); err != nil {
		t.Fatalf("Apply(AppRotateFiles) error = %v", err)
	}
	if err := Apply(&service, SettingAppRotateSeconds, "", []string{"60"}, false); err != nil {
		t.Fatalf("Apply(AppRotateSeconds) error = %v", err)
	}
	if err := Apply(&service, SettingAppRotateBytes, "", []string{"1024"}, false); err != nil {
		t.Fatalf("Apply(AppRotateBytes) error = %v", err)
	}
	if err := Apply(&service, SettingAppTimestampLog, "", []string{"1"}, false); err != nil {
		t.Fatalf("Apply(AppTimestampLog) error = %v", err)
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

	values, err = Read(service, SettingAppEvents, "Start/Pre")
	if err != nil {
		t.Fatalf("Read(AppEvents) error = %v", err)
	}
	if len(values) != 1 || values[0] != `C:\hooks\pre.cmd` {
		t.Fatalf("Read(AppEvents) = %#v, want [C:\\hooks\\pre.cmd]", values)
	}

	values, err = Read(service, SettingAppRotateFiles, "")
	if err != nil {
		t.Fatalf("Read(AppRotateFiles) error = %v", err)
	}
	if len(values) != 1 || values[0] != "1" {
		t.Fatalf("Read(AppRotateFiles) = %#v, want [1]", values)
	}

	values, err = Read(service, SettingAppRotateSeconds, "")
	if err != nil {
		t.Fatalf("Read(AppRotateSeconds) error = %v", err)
	}
	if len(values) != 1 || values[0] != "60" {
		t.Fatalf("Read(AppRotateSeconds) = %#v, want [60]", values)
	}

	values, err = Read(service, SettingAppRotateBytes, "")
	if err != nil {
		t.Fatalf("Read(AppRotateBytes) error = %v", err)
	}
	if len(values) != 1 || values[0] != "1024" {
		t.Fatalf("Read(AppRotateBytes) = %#v, want [1024]", values)
	}

	values, err = Read(service, SettingAppTimestampLog, "")
	if err != nil {
		t.Fatalf("Read(AppTimestampLog) error = %v", err)
	}
	if len(values) != 1 || values[0] != "1" {
		t.Fatalf("Read(AppTimestampLog) = %#v, want [1]", values)
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
