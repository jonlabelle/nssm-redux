# Compatibility

`nssmr` intentionally starts with the highest-value NSSM features first: CLI management, SCM integration, registry-backed settings, and child-process supervision.

## Supported commands

- `install`
- `remove`
- `start`
- `stop`
- `restart`
- `status`
- `statuscode`
- `list`
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
- `AppStdin`
- `AppStdout`
- `AppStderr`
- `AppNoConsole`
- `AppKillProcessTree`
- `DisplayName`
- `Description`
- `Start`
- `DependOnService`

## Behavior notes

- Managed-service settings live under `HKLM\SYSTEM\CurrentControlSet\Services\<name>\Parameters`.
- `AppExit` accepts `Restart`, `Ignore`, and `Exit`. `Suicide` is accepted as an alias for `Exit` when importing legacy values.
- `dump` currently emits `install` plus follow-up `set` commands. It does not try to re-tokenize `AppParameters` back into the original argv shape.
- `AppEnvironment` replaces the base environment. `AppEnvironmentExtra` is layered on top of the chosen base.
- `AppStdout` and `AppStderr` currently append to files rather than exposing the original NSSM `CreateFile` tuning surface.
- `AppKillProcessTree=1` uses a Windows Job Object so child descendants are terminated with the primary process.

## Not yet implemented

- GUI install/edit/remove dialogs
- Hook events under `AppEvents`
- Output rotation and timestamp logging
- `AppPriority`
- `AppAffinity`
- Service account/password flows
- Fine-grained stop-method controls (`AppStopMethod*`)
- Pause/continue and process-tree inspection commands
