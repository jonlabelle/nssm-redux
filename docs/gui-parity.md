# GUI Feature Parity Checklist

This page compares the functional settings exposed by classic NSSM's GUI with the current `nssmr` CLI and runtime. It does not track the old install/edit/remove dialogs themselves, which remain intentionally out of scope.

## Managed-Service Features

| Legacy GUI area                                                     | Status in `nssmr` | Notes                                                                                                                                         |
| ------------------------------------------------------------------- | ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Application path, arguments, and startup directory                  | Covered           | Use `install`, `Application`, `AppParameters`, and `AppDirectory`.                                                                            |
| Display name, description, and startup type                         | Covered           | Use `DisplayName`, `Description`, and `Start`.                                                                                                |
| Service account selection                                           | Covered           | `ObjectName` supports `LocalSystem`, built-in service accounts, virtual accounts, and custom credentials.                                     |
| Priority, affinity, and console visibility                          | Covered           | Use `AppPriority`, `AppAffinity`, and `AppNoConsole`.                                                                                         |
| Stop sequence and grace periods                                     | Covered           | Use `AppStopMethodSkip`, `AppStopMethodConsole`, `AppStopMethodWindow`, and `AppStopMethodThreads`.                                           |
| Restart policy and throttle delay                                   | Covered           | Use `AppExit`, `AppRestartDelay`, and `AppThrottle`. `AppExit` is richer than the GUI because it supports per-exit-code overrides.            |
| Redirected stdin/stdout/stderr                                      | Covered           | Use `AppStdin`, `AppStdout`, and `AppStderr`.                                                                                                 |
| Log rotation, age and size thresholds, rotate delay, and timestamps | Covered           | Use `AppRotateFiles`, `AppRotateOnline`, `AppRotateSeconds`, `AppRotateBytes`, `AppRotateBytesHigh`, `AppRotateDelay`, and `AppTimestampLog`. |
| Environment replacement and environment overlay                     | Covered           | Use `AppEnvironment` and `AppEnvironmentExtra`.                                                                                               |
| Lifecycle hooks                                                     | Covered           | Use `AppEvents` with classic keys such as `Start/Pre` and `Rotate/Post`.                                                                      |
| Service dependencies                                                | Covered in part   | `DependOnService` is supported.                                                                                                               |

## Current Functional Gaps

| Legacy GUI feature                                          | Status in `nssmr` | Notes                                                                                                                                                                  |
| ----------------------------------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Interactive desktop service (`SERVICE_INTERACTIVE_PROCESS`) | Missing           | Classic NSSM could enable desktop interaction for `LocalSystem` services from the Logon tab. `nssmr` does not currently expose a `Type` setting or an equivalent flag. |
| Redirect hook output through the service log streams        | Missing           | The GUI exposed a "Redirect output from hooks" option. `nssmr` runs hooks via `cmd.exe`, but does not currently wire hook stdout/stderr to `AppStdout` or `AppStderr`. |
| Truncate or replace redirected stdout/stderr files on open  | Missing           | Classic NSSM's Rotation tab could switch stdout/stderr to `CREATE_ALWAYS`. `nssmr` currently uses append-oriented file opening for redirected output.                  |
| Dependency groups (`DependOnGroup`)                         | Missing           | Classic NSSM supported both `DependOnService` and `DependOnGroup`. `nssmr` currently exposes only `DependOnService`.                                                   |
| Editing non-managed or native services                      | Missing           | NSSM's GUI could inspect and edit some properties of services it did not create. `nssmr` only manages services that have an `nssmr` `Parameters` tree.                 |

## Summary

For NSSM-managed services, `nssmr` already covers most of the old GUI's high-value configuration surface. The main functional gaps are interactive services, hook output redirection, truncate-on-open log handling, dependency groups, and native-service editing.
