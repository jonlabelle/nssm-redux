# Command Reference

## Create And Control Services

| Command    | Syntax                                                 | What it does                                                                                                       | Example                                                       |
| ---------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------- |
| `install`  | `nssmr install <service> <application> [arguments...]` | Creates a managed service. Everything after `<application>` becomes the initial `AppParameters` string.            | `nssmr install MyService "C:\apps\worker.exe" --listen :8080` |
| `remove`   | `nssmr remove <service>`                               | Deletes the service and its `Parameters` registry tree.                                                            | `nssmr remove MyService`                                      |
| `start`    | `nssmr start <service>`                                | Starts the service if it is not already running.                                                                   | `nssmr start MyService`                                       |
| `stop`     | `nssmr stop <service>`                                 | Stops the service and waits for it to reach `SERVICE_STOPPED`.                                                     | `nssmr stop MyService`                                        |
| `restart`  | `nssmr restart <service>`                              | Stops, then starts the service again.                                                                              | `nssmr restart MyService`                                     |
| `pause`    | `nssmr pause <service>`                                | SCM parity command. Useful only during throttle backoff; it does not suspend the child process.                    | `nssmr pause MyService`                                       |
| `continue` | `nssmr continue <service>`                             | Cancels the remaining throttle-backoff wait and retries immediately when the service is paused in restart backoff. | `nssmr continue MyService`                                    |
| `rotate`   | `nssmr rotate <service>`                               | Requests online log rotation for a running service. Requires `AppRotateFiles=1` and `AppRotateOnline=1`.           | `nssmr rotate MyService`                                      |

## Inspect Services

| Command      | Syntax                                    | What it does                                                                                       | Example                                    |
| ------------ | ----------------------------------------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| `status`     | `nssmr status <service>`                  | Prints the symbolic SCM state name such as `SERVICE_RUNNING`.                                      | `nssmr status MyService`                   |
| `statuscode` | `nssmr statuscode <service>`              | Prints the numeric Windows service state code, not the application exit code.                      | `nssmr statuscode MyService`               |
| `list`       | `nssmr list`                              | Lists services installed by `nssmr`.                                                               | `nssmr list`                               |
| `processes`  | `nssmr processes <service> [service...]`  | Prints the current process tree for one or more managed services.                                  | `nssmr processes ApiService WorkerService` |
| `dump`       | `nssmr dump <service> [new-service-name]` | Emits `install` plus follow-up `set` and `reset` commands that recreate the current configuration. | `nssmr dump MyService CloneService`        |

## Read And Change Settings

| Command | Syntax                                                  | What it does                                                                | Example                                      |
| ------- | ------------------------------------------------------- | --------------------------------------------------------------------------- | -------------------------------------------- |
| `get`   | `nssmr get <service> <setting> [additional]`            | Prints the current value. Multi-value settings print one line per entry.    | `nssmr get MyService AppEnvironmentExtra`    |
| `set`   | `nssmr set <service> <setting> [additional] [value...]` | Replaces the current value for a setting.                                   | `nssmr set MyService AppDirectory "C:\apps"` |
| `reset` | `nssmr reset <service> <setting> [additional]`          | Removes the stored value or restores the built-in default for that setting. | `nssmr reset MyService AppThrottle`          |

## Utility And Internal Commands

| Command   | Syntax                    | What it does                                                                                          | Example                   |
| --------- | ------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------- |
| `version` | `nssmr version`           | Prints the CLI version string.                                                                        | `nssmr version`           |
| `service` | `nssmr service <service>` | Internal SCM entrypoint used by the installed service. It is not intended for normal interactive use. | `nssmr service MyService` |

## Parameter Notes

| Topic                    | Details                                                                                                                    |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| `install` arguments      | Everything after `<application>` becomes the initial `AppParameters` string.                                               |
| Additional keys          | `AppExit` and `AppEvents` require `[additional]`. Examples: `Default`, `42`, `Start/Pre`.                                  |
| Quoting                  | Quote Windows paths with spaces and any argument that your shell would otherwise split.                                    |
| `status` vs `statuscode` | `status` returns a symbolic state name. `statuscode` returns the raw Windows service state number.                         |
| `dump` output            | `dump` recreates the current stored configuration. It does not reconstruct the original shell quoting for `AppParameters`. |
