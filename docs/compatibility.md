# Compatibility

`nssmr` intentionally starts with the highest-value NSSM features first: CLI management, SCM integration, registry-backed settings, and child-process supervision.

## Supported commands

- `install`
- `remove`
- `start`
- `stop`
- `restart`
- `pause`
- `continue`
- `rotate`
- `status`
- `statuscode`
- `list`
- `processes`
- `get`
- `set`
- `reset`
- `dump`

## Supported settings

- `Application`
- `AppParameters`
- `AppDirectory`
- `AppEnvironment`
- `AppEnvironmentExtra`
- `AppExit`
- `AppRestartDelay`
- `AppThrottle`
- `AppPriority`
- `AppAffinity`
- `AppStopMethodSkip`
- `AppStopMethodConsole`
- `AppStopMethodWindow`
- `AppStopMethodThreads`
- `AppEvents`
- `AppRotateFiles`
- `AppRotateOnline`
- `AppRotateSeconds`
- `AppRotateBytes`
- `AppRotateBytesHigh`
- `AppRotateDelay`
- `AppTimestampLog`
- `AppStdin`
- `AppStdout`
- `AppStderr`
- `AppNoConsole`
- `AppKillProcessTree`
- `DisplayName`
- `Description`
- `Start`
- `DependOnService`
- `ObjectName`

## Behavior notes

- Managed-service settings live under `HKLM\SYSTEM\CurrentControlSet\Services\<name>\Parameters`.
- `AppExit` accepts `Restart`, `Ignore`, and `Exit`. `Suicide` is accepted as an alias for `Exit` when importing legacy values.
- `dump` currently emits `install` plus follow-up `set` commands. It does not try to re-tokenize `AppParameters` back into the original argv shape.
- `AppEnvironment` replaces the base environment. `AppEnvironmentExtra` is layered on top of the chosen base.
- `AppEvents` uses the original `Event/Action` syntax such as `Start/Pre` and `Rotate/Post`. Hooks run via `cmd.exe /c` with NSSM-compatible environment variables.
- `AppPriority` accepts the classic Win32 priority class names. `AppAffinity` accepts NSSM's CPU list syntax such as `0-2,4`.
- `AppStopMethodSkip` plus `AppStopMethodConsole`, `AppStopMethodWindow`, and `AppStopMethodThreads` control the staged stop sequence: `CTRL_C_EVENT`, window close messages, thread `WM_QUIT`, then terminate.
- `AppRotateFiles` rotates existing redirected output on service start. `AppRotateOnline` enables runtime rotation for `rotate` control requests and size-triggered rollover while the service is running.
- `AppTimestampLog=1` prefixes redirected log lines with `YYYY-MM-DD HH:MM:SS.mmm: ` timestamps.
- `continue` is meaningful during throttle backoff: when repeated fast failures put the service into `SERVICE_PAUSED`, a continue request cancels the remaining delay and retries the launch.
- `pause` is exposed for SCM parity, but like classic NSSM it is only meaningful in the paused-throttle window rather than as a general process suspend feature.
- `processes` walks the service's current process tree and prints the executable path for each visible descendant.
- `ObjectName` can read, set, and reset the service account. Custom accounts use the password supplied at `set` time and rely on Windows already allowing that account to log on as a service.
- `AppStdout` and `AppStderr` still use a smaller Go-specific file-opening surface rather than every legacy NSSM `CreateFile` tuning knob.
- `AppKillProcessTree=1` uses a Windows Job Object so child descendants are terminated when the primary process exits or when stop escalation reaches the terminate phase.

## Out of scope

- GUI install/edit/remove dialogs
