# Changelog

All notable changes to this project will be documented in this file.

> [!Note]
> This project is still pre-release, so the changelog currently tracks the evolving `main` branch under `Unreleased`. Once tagged releases begin, entries will move to versioned sections.

The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project intends to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) for tagged releases.

## [Unreleased]

### Added

- Initial Go port of NSSM with a dedicated `nssmr` CLI and Windows service host.
- Core service management commands: `install`, `remove`, `start`, `stop`, `restart`, `status`, `statuscode`, `list`, `get`, `set`, `reset`, and `dump`.
- Extended control commands for parity with classic NSSM workflows: `pause`, `continue`, `rotate`, and `processes`.
- Registry-backed configuration persistence using the familiar `Parameters` layout.
- Restart policy support for `AppExit`, `AppRestartDelay`, and `AppThrottle`.
- Environment handling for `AppEnvironment` replacement and `AppEnvironmentExtra` merging.
- Process runtime settings covering `AppDirectory`, `AppParameters`, `AppStdin`, `AppStdout`, `AppStderr`, `AppNoConsole`, and `AppKillProcessTree`.
- Windows-specific process controls for `AppPriority`, `AppAffinity`, and the legacy `AppStopMethod*` settings.
- Hook event support through `AppEvents`.
- Output rotation and timestamped logging through the `AppRotate*` and `AppTimestampLog` settings.
- Native service metadata and account support, including display name, description, startup type, dependencies, and `ObjectName`.
- Process tree inspection for managed services.
- Windows build automation via `Makefile` targets and GitHub Actions artifacts for `windows/amd64` and `windows/arm64`.
- Compatibility and architecture documentation for the port.

### Changed

- Declared the legacy GUI installer/editor out of scope for the current porting phase so the project can focus on CLI and service-runtime parity first.
- Consolidated local developer build steps around `make build`, `make test`, and `make build-windows`.

### Fixed

- Reworked the window-enumeration callback state handling to avoid unsafe pointer misuse warnings on `windows/amd64` during `go vet`.

## Bootstrap History

- `2026-03-13`: Repository initialized with the first working port slice.
- `2026-03-13`: Added process priority, affinity, hooks, log rotation, service account support, process inspection, and Windows control enhancements.
- `2026-03-13`: Added build automation and refreshed the project documentation.
