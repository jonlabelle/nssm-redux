package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jonlabelle/nssm-redux/internal/config"
	"github.com/jonlabelle/nssm-redux/internal/scm"
	"github.com/jonlabelle/nssm-redux/internal/support"
	"github.com/jonlabelle/nssm-redux/internal/svcwrap"
)

// Version is set at build time.
var Version = "dev"

// Run executes the nssmr CLI.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	_ = stderr

	if len(args) == 0 {
		_, err := fmt.Fprintln(stdout, usageText)
		return err
	}

	switch strings.ToLower(args[0]) {
	case "help", "--help", "-h", "/?":
		_, err := fmt.Fprintln(stdout, usageText)
		return err

	case "version", "--version", "-v", "/version":
		_, err := fmt.Fprintf(stdout, "nssmr %s\n", Version)
		return err

	case "install":
		if len(args) < 3 {
			return usageError("install requires <service> <application>")
		}
		cfg := config.Default(args[1])
		cfg.Executable = args[2]
		if len(args) > 3 {
			cfg.Arguments = strings.Join(args[3:], " ")
		}
		cfg.Normalize()
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable path: %w", err)
		}
		if err := scm.Install(self, cfg); err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "installed %s\n", cfg.Name)
		return err

	case "remove":
		if len(args) < 2 {
			return usageError("remove requires <service>")
		}
		if err := scm.Remove(args[1]); err != nil {
			return err
		}
		_, err := fmt.Fprintf(stdout, "removed %s\n", args[1])
		return err

	case "start":
		if len(args) < 2 {
			return usageError("start requires <service>")
		}
		return scm.Start(args[1])

	case "stop":
		if len(args) < 2 {
			return usageError("stop requires <service>")
		}
		return scm.Stop(args[1])

	case "restart":
		if len(args) < 2 {
			return usageError("restart requires <service>")
		}
		return scm.Restart(args[1])

	case "pause":
		if len(args) < 2 {
			return usageError("pause requires <service>")
		}
		return scm.Pause(args[1])

	case "continue":
		if len(args) < 2 {
			return usageError("continue requires <service>")
		}
		return scm.Continue(args[1])

	case "rotate":
		if len(args) < 2 {
			return usageError("rotate requires <service>")
		}
		return scm.Rotate(args[1])

	case "status":
		if len(args) < 2 {
			return usageError("status requires <service>")
		}
		status, err := scm.Query(args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, status.State)
		return err

	case "statuscode":
		if len(args) < 2 {
			return usageError("statuscode requires <service>")
		}
		status, err := scm.Query(args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%d\n", status.StateCode)
		return err

	case "list":
		services, err := scm.ListManaged()
		if err != nil {
			return err
		}
		for _, service := range services {
			if _, err := fmt.Fprintln(stdout, service); err != nil {
				return err
			}
		}
		return nil

	case "processes":
		if len(args) < 2 {
			return usageError("processes requires <service>")
		}
		for i, service := range args[1:] {
			tree, err := scm.ProcessTreeForService(service)
			if err != nil {
				return err
			}
			if len(args[1:]) > 1 {
				if _, err := fmt.Fprintln(stdout, tree.Service); err != nil {
					return err
				}
			}
			for _, line := range scm.FormatProcessTree(tree) {
				if _, err := fmt.Fprintln(stdout, line); err != nil {
					return err
				}
			}
			if i < len(args[1:])-1 {
				if _, err := fmt.Fprintln(stdout); err != nil {
					return err
				}
			}
		}
		return nil

	case "get":
		return runGet(stdout, args[1:])

	case "set":
		return runSet(args[1:], false)

	case "reset", "unset":
		return runSet(args[1:], true)

	case "dump":
		if len(args) < 2 {
			return usageError("dump requires <service>")
		}
		cfg, err := scm.Load(args[1])
		if err != nil {
			return err
		}
		targetName := ""
		if len(args) > 2 {
			targetName = args[2]
		}
		commands, err := config.DumpCommands("nssmr", cfg, targetName)
		if err != nil {
			return err
		}
		if account, err := scm.GetObjectName(args[1]); err == nil {
			if targetName == "" {
				targetName = args[1]
			}
			commands = append(commands, dumpObjectNameCommand(targetName, account))
		}
		for _, command := range commands {
			if _, err := fmt.Fprintln(stdout, command); err != nil {
				return err
			}
		}
		return nil

	case "service":
		if len(args) < 2 {
			return usageError("service requires <service>")
		}
		return svcwrap.Run(ctx, args[1])
	}

	return usageError(fmt.Sprintf("unknown command %q", args[0]))
}

func runGet(stdout io.Writer, args []string) error {
	if len(args) < 2 {
		return usageError("get requires <service> <setting>")
	}

	if strings.EqualFold(args[1], "ObjectName") {
		account, err := scm.GetObjectName(args[0])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, account)
		return err
	}

	spec, ok := config.LookupSetting(args[1])
	if !ok {
		return usageError(fmt.Sprintf("unknown setting %q", args[1]))
	}

	additional := ""
	if spec.AdditionalMandatory {
		if len(args) < 3 {
			return usageError(fmt.Sprintf("%s requires an additional key", spec.Name))
		}
		additional = args[2]
	}

	cfg, err := scm.Load(args[0])
	if err != nil {
		return err
	}
	values, err := config.Read(cfg, spec.Name, additional)
	if err != nil {
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintln(stdout, value); err != nil {
			return err
		}
	}
	return nil
}

func runSet(args []string, reset bool) error {
	if len(args) < 2 {
		return usageError("set/reset require <service> <setting>")
	}

	if strings.EqualFold(args[1], "ObjectName") {
		if reset {
			return scm.ResetObjectName(args[0])
		}
		if len(args) < 3 {
			return usageError("ObjectName requires a username")
		}
		password := ""
		if len(args) > 3 {
			password = strings.Join(args[3:], " ")
		}
		return scm.SetObjectName(args[0], args[2], password)
	}

	spec, ok := config.LookupSetting(args[1])
	if !ok {
		return usageError(fmt.Sprintf("unknown setting %q", args[1]))
	}

	additional := ""
	valuesStart := 2
	if spec.AdditionalMandatory {
		if len(args) < 3 {
			return usageError(fmt.Sprintf("%s requires an additional key", spec.Name))
		}
		additional = args[2]
		valuesStart = 3
	}

	cfg, err := scm.Load(args[0])
	if err != nil {
		return err
	}
	if err := config.Apply(&cfg, spec.Name, additional, args[valuesStart:], reset); err != nil {
		return err
	}
	return scm.Save(cfg)
}

func dumpObjectNameCommand(serviceName, account string) string {
	if strings.TrimSpace(account) == "" || strings.EqualFold(account, "LocalSystem") || strings.EqualFold(account, "NT AUTHORITY\\System") {
		return support.JoinCommandLine([]string{"nssmr", "reset", serviceName, "ObjectName"})
	}
	if strings.HasPrefix(strings.ToLower(account), strings.ToLower("NT Service\\")) {
		return support.JoinCommandLine([]string{"nssmr", "set", serviceName, "ObjectName", account})
	}
	if strings.EqualFold(account, "NT AUTHORITY\\LocalService") || strings.EqualFold(account, "NT AUTHORITY\\NetworkService") {
		return support.JoinCommandLine([]string{"nssmr", "set", serviceName, "ObjectName", account})
	}
	return support.JoinCommandLine([]string{"nssmr", "set", serviceName, "ObjectName", account, "****"})
}

func usageError(message string) error {
	if message == "" {
		return fmt.Errorf("%s", usageText)
	}
	return fmt.Errorf("%s\n\n%s", message, usageText)
}

const usageText = `nssmr manages Windows services that wrap an existing executable.

Usage:
  nssmr install <service> <application> [arguments...]
  nssmr remove <service>
  nssmr start|stop|restart|pause|continue|rotate <service>
  nssmr status|statuscode <service>
  nssmr list
  nssmr processes <service> [service...]
  nssmr get <service> <setting> [additional]
  nssmr set <service> <setting> [additional] [value...]
  nssmr reset|unset <service> <setting> [additional]
  nssmr dump <service> [new-service-name]
  nssmr service <service>
  nssmr version
  nssmr help

Commands:
  install/remove    Create or delete a managed service.
  start/stop/...    Control a service through the Windows SCM.
  status/list/...   Inspect service state, process trees, or exported config.
  get/set/reset     Read or update NSSM-compatible settings.
  dump              Emit commands that recreate the current configuration.
  service           Internal SCM entrypoint used by installed services.

Notes:
  install stores everything after <application> as AppParameters.
  AppExit and AppEvents require an [additional] key such as Default, 2, or Start/Pre.
  Quote paths with spaces and any argument your shell would otherwise split.

Examples:
  nssmr install MyService "C:\apps\worker.exe" --config "C:\apps\worker.yml"
  nssmr set MyService AppDirectory "C:\apps"
  nssmr set MyService AppStdout "C:\logs\worker.out.log"
  nssmr set MyService AppEnvironment "ENV=prod" "PORT=8080"
  nssmr set MyService AppEvents Start/Pre "C:\hooks\before-start.cmd"
  nssmr set MyService Start SERVICE_DELAYED_AUTO_START
  nssmr start MyService
  nssmr get MyService AppParameters
  nssmr dump MyService CloneService`
