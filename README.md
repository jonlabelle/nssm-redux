# nssmr (Non-Sucking Service Manager _Redux_)

[![ci](https://github.com/jonlabelle/nssm-redux/actions/workflows/ci.yml/badge.svg)](https://github.com/jonlabelle/nssm-redux/actions/workflows/ci.yml)
[![code-ql](https://github.com/jonlabelle/nssm-redux/actions/workflows/codeql.yml/badge.svg)](https://github.com/jonlabelle/nssm-redux/actions/workflows/codeql.yml)
[![release](https://github.com/jonlabelle/nssm-redux/actions/workflows/release.yml/badge.svg)](https://github.com/jonlabelle/nssm-redux/actions/workflows/release.yml)

> Currently a work in progress, but early feedback is welcome! See the [compatibility notes](docs/compatibility.md) for details on the current scope and design decisions.

`nssmr` is a Go port of [NSSM](https://nssm.cc), the Non-Sucking Service Manager for Windows.

This repository is intentionally starting with a strong CLI and service-runtime core instead of trying to port the legacy GUI first. The current codebase already installs and runs arbitrary executables as Windows services, persists settings in the familiar `Parameters` registry layout, and ships CI/release automation for Windows binaries.

## Status

`nssmr` is an early CLI-first Go port of NSSM focused on Windows service installation, configuration, and runtime supervision.

The current milestone covers the core management commands, registry-compatible `Parameters` storage, restart policy, hooks, process controls, and log rotation.

The legacy GUI is intentionally out of scope for now. See the [compatibility notes](docs/compatibility.md) for detailed parity coverage and current gaps.

## Quick Start

`nssmr` wraps an existing executable and runs it as a Windows service. In the examples below, `worker.exe` is your application, not part of `nssmr`.

1. Install a service for your application:

   ```bash
   nssmr install MyService "C:\apps\worker.exe" --config "C:\apps\worker.yml"
   ```

   Everything after the executable is stored as `AppParameters`.

2. Configure the working directory, logs, and startup behavior:

   ```bash
   nssmr set MyService AppDirectory "C:\apps"
   nssmr set MyService AppStdout "C:\logs\worker.out.log"
   nssmr set MyService AppStderr "C:\logs\worker.err.log"
   nssmr set MyService DisplayName "My Worker Service"
   nssmr set MyService Start SERVICE_DELAYED_AUTO_START
   ```

3. Start the service and inspect the stored configuration:

   ```bash
   nssmr start MyService
   nssmr status MyService
   nssmr get MyService AppParameters
   ```

## More Configuration Examples

After install, you can layer on more advanced behavior:

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

> [!NOTE]
> The `service` subcommand is the internal SCM entrypoint used by the installed Windows service and is not intended for normal interactive use.

## Build

Source builds currently require Go `1.26.1` or newer, matching [go.mod](go.mod).

```bash
make test
make build
```

Build Windows artifacts:

```bash
make build-windows
```

This writes the host binary to `bin/` and the Windows release artifacts to:

- `dist/nssmr-windows-amd64.exe`
- `dist/nssmr-windows-arm64.exe`

> [!Note]
> You can build on non-Windows hosts and run most tests, but the `install` command, service control, and the managed-process runtime only work on Windows.

## Docs

- [Changelog](CHANGELOG.md)
- [Documentation index](docs/README.md)
- [Compatibility and parity notes](docs/compatibility.md)

## Credits

- [NSSM](https://nssm.cc) for the original design and registry model

## License

[MIT](LICENSE)
