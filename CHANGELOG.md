# Changelog

## 1.0.0 (2026-06-12)

### Bug Fixes

- incorrect conversion between integer types ([7fafca3](https://github.com/jonlabelle/nssm-redux/commit/7fafca3a6ed2a731612add3ea1efb2341b43630b))
- Update Go setup action to use the latest version ([d4f1889](https://github.com/jonlabelle/nssm-redux/commit/d4f18897974f839b3ea611231f9d7754f3266abb))
- **versioninfo:** correctly handle commit hashes ([4d13ca6](https://github.com/jonlabelle/nssm-redux/commit/4d13ca697b206c48cf07c63b85a3ba2c0d515c29))

### Features

- add build script for managing Go tasks and builds ([9ffa6c5](https://github.com/jonlabelle/nssm-redux/commit/9ffa6c5b2003038136c1e19cbac5fec674fc4d2f))
- Add changelog and update documentation links ([90a1be0](https://github.com/jonlabelle/nssm-redux/commit/90a1be09421b4dc41b540f75008265e587ff343c))
- add CodeQL analysis workflow for automated code scanning ([64b95e8](https://github.com/jonlabelle/nssm-redux/commit/64b95e80800e6bc7052459fc527ee407bacc3df2))
- add Go Vulnerability Check workflow and update VSCode settings ([43a3f89](https://github.com/jonlabelle/nssm-redux/commit/43a3f89c9995333916bbc3f0cf940eb6e4569861))
- add issue templates for bug reports, documentation issues, and feature requests ([de075db](https://github.com/jonlabelle/nssm-redux/commit/de075db81e8a4985d978f385b21589119e37eef3))
- Add Makefile for build automation and update README with build instructions ([e60ec41](https://github.com/jonlabelle/nssm-redux/commit/e60ec41552efa310a35800909c45b282f4295a8e))
- add process priority and affinity settings ([b267be6](https://github.com/jonlabelle/nssm-redux/commit/b267be69d2c2629c98208f3366dd6aaaa1d2626a))
- add security policy documentation ([df272bc](https://github.com/jonlabelle/nssm-redux/commit/df272bccbe31cc4df1d268d13926e6c982791031))
- Add support for log rotation and process management on Windows ([2995966](https://github.com/jonlabelle/nssm-redux/commit/2995966ffe752a600fe133839b930d7beb0fd86b))
- add VS Code configuration files for Go development ([985e27c](https://github.com/jonlabelle/nssm-redux/commit/985e27caf52a1ac55957c463d9f55ff8f16f58e2))
- **build.ps1:** add script docs ([35c317a](https://github.com/jonlabelle/nssm-redux/commit/35c317a64be878cab47563c9e912e778762a4fee))
- **codeql:** enable github actions scanning ([6790664](https://github.com/jonlabelle/nssm-redux/commit/6790664392f16210b287816799b3466e1ee80fa5))
- enable manual triggering of CodeQL workflow ([07137e4](https://github.com/jonlabelle/nssm-redux/commit/07137e4884decda8a76d7fbe172c27c09cbdd114))
- enhance documentation with command and settings references ([c1225be](https://github.com/jonlabelle/nssm-redux/commit/c1225beaf32fb4b1b8667b447e453f9df0dd7ddd))
- Enhance window signal handling with registration and lookup functions ([3a2c8ec](https://github.com/jonlabelle/nssm-redux/commit/3a2c8ec516e57d73900f2f94e802789522dd4e2e))
- implement Windows VERSIONINFO embedding and build tool for cross-compilation ([5f53228](https://github.com/jonlabelle/nssm-redux/commit/5f53228839b050027bd7ad72514ce0c47d7ec7e4))
- semantic release ([#4](https://github.com/jonlabelle/nssm-redux/issues/4)) ([c5deac1](https://github.com/jonlabelle/nssm-redux/commit/c5deac1a310028f37c460e03fa93d4380e9f939d))

## 0.0.0 (2026-03-01)

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

### Bootstrap History

- `2026-03-13`: Repository initialized with the first working port slice.
- `2026-03-13`: Added process priority, affinity, hooks, log rotation, service account support, process inspection, and Windows control enhancements.
- `2026-03-13`: Added build automation and refreshed the project documentation.
