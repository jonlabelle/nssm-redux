# Compatibility

`nssmr` intentionally starts with the highest-value NSSM features first: CLI management, SCM integration, registry-backed settings, and child-process supervision.

## Command Coverage

| Area                         | Coverage                                                                       | Notes                                                                                                                                          |
| ---------------------------- | ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Service creation and control | `install`, `remove`, `start`, `stop`, `restart`, `pause`, `continue`, `rotate` | See [command-reference.md](command-reference.md). `pause` and `continue` are throttle-window controls, not general process suspend and resume. |
| Inspection and export        | `status`, `statuscode`, `list`, `processes`, `dump`                            | `dump` emits `install` plus follow-up `set` and `reset` commands.                                                                              |
| Configuration editing        | `get`, `set`, `reset`                                                          | See [settings-reference.md](settings-reference.md).                                                                                            |

## Settings Coverage

| Area                           | Supported settings                                                                                                                                                         |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Process launch and environment | `Application`, `AppParameters`, `AppDirectory`, `AppEnvironment`, `AppEnvironmentExtra`, `AppNoConsole`, `AppKillProcessTree`                                              |
| Restart and process control    | `AppExit`, `AppRestartDelay`, `AppThrottle`, `AppPriority`, `AppAffinity`, `AppStopMethodSkip`, `AppStopMethodConsole`, `AppStopMethodWindow`, `AppStopMethodThreads`      |
| Hooks                          | `AppEvents`                                                                                                                                                                |
| Logging and I/O                | `AppRotateFiles`, `AppRotateOnline`, `AppRotateSeconds`, `AppRotateBytes`, `AppRotateBytesHigh`, `AppRotateDelay`, `AppTimestampLog`, `AppStdin`, `AppStdout`, `AppStderr` |
| SCM metadata and identity      | `DisplayName`, `Description`, `Start`, `DependOnService`, `ObjectName`                                                                                                     |

## Behavior Notes

| Topic                       | Current behavior                                                                                                                                                                                              |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Registry layout             | Managed-service settings live under `HKLM\SYSTEM\CurrentControlSet\Services\<name>\Parameters`.                                                                                                               |
| `AppExit`                   | Accepts `Restart`, `Ignore`, and `Exit`. `Suicide` is accepted as an alias for `Exit` when importing legacy values.                                                                                           |
| `dump`                      | Recreates the current stored configuration. It does not try to reconstruct the original shell quoting for `AppParameters`.                                                                                    |
| Environment handling        | `AppEnvironment` replaces the base environment. `AppEnvironmentExtra` is layered on top of the chosen base.                                                                                                   |
| `AppEvents`                 | Uses the original `Event/Action` syntax such as `Start/Pre` and `Rotate/Post`. Hooks run through `cmd.exe /d /s /c` with NSSM-style hook variables.                                                           |
| Priority and affinity       | `AppPriority` accepts the classic Win32 priority class names. `AppAffinity` accepts NSSM CPU list syntax such as `0-2,4`.                                                                                     |
| Stop sequence               | `AppStopMethodSkip` plus `AppStopMethodConsole`, `AppStopMethodWindow`, and `AppStopMethodThreads` control the staged stop sequence: `CTRL_C_EVENT`, window close messages, thread `WM_QUIT`, then terminate. |
| Log rotation                | `AppRotateFiles` rotates existing redirected output on service start. `AppRotateOnline` enables runtime rotation for `rotate` requests and size-triggered rollover while the service is running.              |
| Timestamped logs            | `AppTimestampLog=1` prefixes redirected log lines with `YYYY-MM-DD HH:MM:SS.mmm: ` timestamps.                                                                                                                |
| `pause` and `continue`      | They are meaningful during throttle backoff. `continue` cancels the remaining delay and retries the launch.                                                                                                   |
| `processes`                 | Walks the current process tree and prints the executable path for each visible descendant.                                                                                                                    |
| `ObjectName`                | Can read, set, and reset the service account. Built-in accounts and `NT Service\<name>` need no password; custom accounts do.                                                                                 |
| `AppStdout` and `AppStderr` | Use a smaller Go-specific file-opening surface rather than every legacy NSSM `CreateFile` tuning knob.                                                                                                        |
| `AppKillProcessTree`        | Uses a Windows Job Object so child descendants are terminated when the primary process exits or when stop escalation reaches the terminate phase.                                                             |

## Out of scope

- GUI install/edit/remove dialogs
