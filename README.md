# nssmr (Non-Sucking Service Manager _Redux_)

> Currently a work in progress, but early feedback is welcome! See the [compatibility notes](docs/compatibility.md) for details on the current scope and design decisions.

`nssmr` is a Windows-focused Go port of [NSSM](https://nssm.cc), the Non-Sucking Service Manager.

This repository is intentionally starting with a strong CLI and service-runtime core instead of trying to port the legacy GUI first. The current codebase already installs and runs arbitrary executables as Windows services, persists settings in the familiar `Parameters` registry layout, and ships CI/release automation for Windows binaries.

## Current scope

Implemented in this first port slice:

- Install, remove, start, stop, restart, pause, continue, rotate, status, list, processes, get, set, reset, and dump commands
- Windows service hosting through `golang.org/x/sys/windows/svc`
- Registry-backed managed-service settings compatible with the original `Parameters` layout
- Restart policy with `AppExit`, `AppRestartDelay`, and `AppThrottle`
- `AppEnvironment` replacement plus `AppEnvironmentExtra` merging
- `AppDirectory`, `AppParameters`, `AppStdin`, `AppStdout`, `AppStderr`, `AppNoConsole`, and `AppKillProcessTree`
- `AppPriority`, `AppAffinity`, and the legacy `AppStopMethod*` stop controls
- Hook events under `AppEvents`
- Output rotation plus timestamped log streaming through the `AppRotate*` and `AppTimestampLog` settings
- Native service metadata updates for display name, description, startup type, dependencies, and service account
- Tagged GitHub Actions releases for `windows/amd64` and `windows/arm64`

Out of scope for now:

- Legacy GUI installer/editor

## Build

```bash
go test ./...
go build ./cmd/nssmr
```

Cross-build examples:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/nssmr-amd64.exe ./cmd/nssmr
GOOS=windows GOARCH=arm64 go build -o dist/nssmr-arm64.exe ./cmd/nssmr
```

## Usage

Install a service:

```bash
nssmr install MyService "C:\apps\worker.exe" --listen :8080
```

Update settings after install:

```bash
nssmr set MyService AppDirectory "C:\apps"
nssmr set MyService AppStdout "C:\logs\worker.out.log"
nssmr set MyService AppStderr "C:\logs\worker.err.log"
nssmr set MyService AppEnvironment "ENV=prod" "PORT=8080"
nssmr set MyService AppEvents Start/Pre "C:\hooks\before-start.cmd"
nssmr set MyService AppRotateFiles 1
nssmr set MyService AppRotateOnline 1
nssmr set MyService AppTimestampLog 1
nssmr set MyService AppPriority ABOVE_NORMAL_PRIORITY_CLASS
nssmr set MyService AppAffinity 0-3
nssmr set MyService AppStopMethodSkip 0
nssmr set MyService ObjectName "NT AUTHORITY\LocalService"
nssmr set MyService Start SERVICE_DELAYED_AUTO_START
```

Inspect or export configuration:

```bash
nssmr get MyService AppParameters
nssmr processes MyService
nssmr rotate MyService
nssmr dump MyService
```

Runtime note:

- The `service` subcommand is the internal SCM entrypoint used by the installed Windows service and is not intended for normal interactive use.

## Docs

- [Documentation index](docs/README.md)
- [Compatibility and parity notes](docs/compatibility.md)

## Credits

- [NSSM](https://nssm.cc) for the original design and registry model

## License

MIT. See [LICENSE](LICENSE).
