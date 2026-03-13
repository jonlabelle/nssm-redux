package config

import "testing"

func TestDumpCommands(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	service.Executable = `C:\Program Files\App\app.exe`
	service.Arguments = `--flag "hello world"`
	service.Description = "Example service"
	service.DefaultExitAction = ExitActionIgnore

	commands, err := DumpCommands("nssmr", service, "")
	if err != nil {
		t.Fatalf("DumpCommands() error = %v", err)
	}
	if len(commands) < 3 {
		t.Fatalf("len(commands) = %d, want >= 3", len(commands))
	}
	wantInstall := `nssmr install svc "C:\Program Files\App\app.exe"`
	if commands[0] != wantInstall {
		t.Fatalf("commands[0] = %q, want %q", commands[0], wantInstall)
	}
}
