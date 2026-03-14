package config

import "testing"

func TestDumpCommands(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	service.Executable = `C:\Program Files\App\app.exe`
	service.Arguments = `--flag "hello world"`
	service.Description = "Example service"
	service.DefaultExitAction = ExitActionIgnore
	service.Priority = PriorityHigh
	service.Affinity = 0b10111

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
	foundPriority := false
	foundAffinity := false
	for _, command := range commands {
		if command == `nssmr set svc AppPriority HIGH_PRIORITY_CLASS` {
			foundPriority = true
		}
		if command == `nssmr set svc AppAffinity 0-2,4` {
			foundAffinity = true
		}
	}
	if !foundPriority {
		t.Fatalf("DumpCommands() missing AppPriority command: %#v", commands)
	}
	if !foundAffinity {
		t.Fatalf("DumpCommands() missing AppAffinity command: %#v", commands)
	}
}
