# Settings Reference

These settings are used with `nssmr get`, `nssmr set`, and `nssmr reset`.

## Value Conventions

| Kind                 | Accepted input                                                           | Used by                                                                                      | Example                                              |
| -------------------- | ------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| Boolean              | `1`, `0`, `true`, `false`, `yes`, `no`, `on`, `off`                      | `AppRotateFiles`, `AppRotateOnline`, `AppTimestampLog`, `AppNoConsole`, `AppKillProcessTree` | `nssmr set MyService AppRotateOnline 1`              |
| Millisecond duration | Integer milliseconds or Go duration strings like `1500ms`, `2s`, `1m30s` | `AppRestartDelay`, `AppThrottle`, `AppStopMethod*`, `AppRotateDelay`                         | `nssmr set MyService AppThrottle 2s`                 |
| Rotation age         | Integer seconds or Go duration strings                                   | `AppRotateSeconds`                                                                           | `nssmr set MyService AppRotateSeconds 1h`            |
| Multi-value list     | Each CLI argument becomes one stored entry                               | `AppEnvironment`, `AppEnvironmentExtra`, `DependOnService`                                   | `nssmr set MyService DependOnService Tcpip Dnscache` |
| Additional key       | Required selector between `<setting>` and the value                      | `AppExit`, `AppEvents`                                                                       | `nssmr set MyService AppExit Default Restart`        |

## Application And Environment

| Setting               | Value format                    | What it controls                                                                           | Reset/default                                                           | Example                                                                          |
| --------------------- | ------------------------------- | ------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `Application`         | Executable path                 | Managed program to launch                                                                  | Required. Replace it with `set`; a blank application does not validate. | `nssmr set MyService Application "C:\apps\worker.exe"`                           |
| `AppParameters`       | Raw argument string             | Command line appended after `Application`                                                  | Reset clears the stored arguments.                                      | `nssmr set MyService AppParameters --listen :8080 --config "C:\apps\worker.yml"` |
| `AppDirectory`        | Directory path                  | Working directory for the child process. Relative log paths resolve from here.             | Reset falls back to the executable directory.                           | `nssmr set MyService AppDirectory "C:\apps"`                                     |
| `AppEnvironment`      | One or more `KEY=VALUE` entries | Replaces the inherited environment when set                                                | Reset returns to the service process environment.                       | `nssmr set MyService AppEnvironment "ENV=prod" "PORT=8080"`                      |
| `AppEnvironmentExtra` | One or more `KEY=VALUE` entries | Merges on top of the inherited environment or `AppEnvironment`. Keys are case-insensitive. | Reset removes the overlay.                                              | `nssmr set MyService AppEnvironmentExtra "PATH=C:\tools" "LOG_LEVEL=debug"`      |

- `get` prints `AppEnvironment` and `AppEnvironmentExtra` one entry per line.

## Restart And Process Policy

| Setting              | Value format                                                                                                                                                   | What it controls                                                                                                             | Reset/default                                                                           | Example                                                       |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `AppExit`            | Additional key `Default`, `*`, or an exit code plus action `Restart`, `Ignore`, or `Exit`                                                                      | Restart policy by exit code. `Suicide` is accepted as an alias for `Exit`.                                                   | Reset `Default` returns to `Restart`. Reset a specific code removes only that override. | `nssmr set MyService AppExit 2 Ignore`                        |
| `AppRestartDelay`    | Milliseconds or duration                                                                                                                                       | Minimum wait before a restart happens                                                                                        | Default is `0`.                                                                         | `nssmr set MyService AppRestartDelay 5s`                      |
| `AppThrottle`        | Milliseconds or duration                                                                                                                                       | If the process exits sooner than this, restart delay backs off exponentially and the service enters a paused-throttle window | Default is `1500` ms.                                                                   | `nssmr set MyService AppThrottle 2s`                          |
| `AppPriority`        | `REALTIME_PRIORITY_CLASS`, `HIGH_PRIORITY_CLASS`, `ABOVE_NORMAL_PRIORITY_CLASS`, `NORMAL_PRIORITY_CLASS`, `BELOW_NORMAL_PRIORITY_CLASS`, `IDLE_PRIORITY_CLASS` | Win32 priority class for the managed process. Short aliases like `high` also work.                                           | Default is `NORMAL_PRIORITY_CLASS`.                                                     | `nssmr set MyService AppPriority ABOVE_NORMAL_PRIORITY_CLASS` |
| `AppAffinity`        | CPU list such as `0-3,6`                                                                                                                                       | Restricts the managed process to selected CPUs                                                                               | Reset returns to all available CPUs.                                                    | `nssmr set MyService AppAffinity 0-3`                         |
| `AppNoConsole`       | Boolean                                                                                                                                                        | Launches the process without creating a new console window                                                                   | Default is `0`.                                                                         | `nssmr set MyService AppNoConsole 1`                          |
| `AppKillProcessTree` | Boolean                                                                                                                                                        | Uses a Windows Job Object to terminate descendants when the app exits or forced stop is reached                              | Default is `1`.                                                                         | `nssmr set MyService AppKillProcessTree 0`                    |

## Stop Behavior

| Setting                | Value format             | What it controls                                                                        | Reset/default         | Example                                       |
| ---------------------- | ------------------------ | --------------------------------------------------------------------------------------- | --------------------- | --------------------------------------------- |
| `AppStopMethodSkip`    | Numeric bitmask          | Disables stop phases from the default console -> window -> thread -> terminate sequence | Default is `0`.       | `nssmr set MyService AppStopMethodSkip 8`     |
| `AppStopMethodConsole` | Milliseconds or duration | Wait after sending `CTRL_C_EVENT`                                                       | Default is `1500` ms. | `nssmr set MyService AppStopMethodConsole 3s` |
| `AppStopMethodWindow`  | Milliseconds or duration | Wait after posting window close messages                                                | Default is `1500` ms. | `nssmr set MyService AppStopMethodWindow 3s`  |
| `AppStopMethodThreads` | Milliseconds or duration | Wait after posting `WM_QUIT` to process threads                                         | Default is `1500` ms. | `nssmr set MyService AppStopMethodThreads 3s` |

### `AppStopMethodSkip` Bits

| Bit value | Skips this phase                 |
| --------- | -------------------------------- |
| `1`       | Console control (`CTRL_C_EVENT`) |
| `2`       | Window close messages            |
| `4`       | Thread `WM_QUIT`                 |
| `8`       | Forced terminate                 |

- If you skip bit `8`, the service can stop while the managed process keeps running if the graceful phases do not end it first.

## Hooks

| Setting     | Value format                              | What it controls                                                                                                                                | Reset/default                    | Example                                                               |
| ----------- | ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- | --------------------------------------------------------------------- |
| `AppEvents` | Additional hook key plus a command string | Runs `cmd.exe /d /s /c <command>` around supported lifecycle events. Hooks inherit the service environment and receive `NSSM_*` hook variables. | Reset removes the selected hook. | `nssmr set MyService AppEvents Start/Pre "C:\hooks\before-start.cmd"` |

### Supported Hook Keys

| Hook key       | When it runs                                                                      | Example                                                                  |
| -------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `Start/Pre`    | Before launching the managed process. Exit code `99` aborts the start or restart. | `nssmr set MyService AppEvents Start/Pre "C:\hooks\before-start.cmd"`    |
| `Start/Post`   | After the service reaches the running state                                       | `nssmr set MyService AppEvents Start/Post "C:\hooks\after-start.cmd"`    |
| `Stop/Pre`     | Before the stop sequence begins                                                   | `nssmr set MyService AppEvents Stop/Pre "C:\hooks\before-stop.cmd"`      |
| `Exit/Post`    | After the child process exits                                                     | `nssmr set MyService AppEvents Exit/Post "C:\hooks\after-exit.cmd"`      |
| `Power/Change` | When Windows reports a power status change                                        | `nssmr set MyService AppEvents Power/Change "C:\hooks\power-change.cmd"` |
| `Power/Resume` | When Windows resumes from sleep                                                   | `nssmr set MyService AppEvents Power/Resume "C:\hooks\resume.cmd"`       |
| `Rotate/Pre`   | Before an online log rotation request is handled                                  | `nssmr set MyService AppEvents Rotate/Pre "C:\hooks\before-rotate.cmd"`  |
| `Rotate/Post`  | After an online log rotation request is handled                                   | `nssmr set MyService AppEvents Rotate/Post "C:\hooks\after-rotate.cmd"`  |

## Logging And I/O

| Setting              | Value format              | What it controls                                                                                                                      | Reset/default                                 | Example                                               |
| -------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- | ----------------------------------------------------- |
| `AppStdin`           | File path                 | File opened for standard input. Relative paths resolve from `AppDirectory`.                                                           | Reset unsets stdin redirection.               | `nssmr set MyService AppStdin "data\input.txt"`       |
| `AppStdout`          | File path                 | Redirect target for standard output. Parent directories are created automatically.                                                    | Reset unsets stdout redirection.              | `nssmr set MyService AppStdout "logs\worker.out.log"` |
| `AppStderr`          | File path                 | Redirect target for standard error. If it matches `AppStdout`, both streams share one file.                                           | Reset unsets stderr redirection.              | `nssmr set MyService AppStderr "logs\worker.err.log"` |
| `AppRotateFiles`     | Boolean                   | Enables startup log rotation. If no age or size threshold is set, any existing file is rotated on start.                              | Default is `0`.                               | `nssmr set MyService AppRotateFiles 1`                |
| `AppRotateOnline`    | Boolean                   | Enables runtime rotation for `rotate` requests and size-triggered rollover while the service is running. Requires `AppRotateFiles=1`. | Default is `0`.                               | `nssmr set MyService AppRotateOnline 1`               |
| `AppRotateSeconds`   | Seconds or duration       | Minimum file age before startup rotation occurs                                                                                       | Default is `0`, which disables the age check. | `nssmr set MyService AppRotateSeconds 3600`           |
| `AppRotateBytes`     | Unsigned 32-bit low half  | Low 32 bits of the size threshold used for startup rotation and online rollover                                                       | Default is `0`.                               | `nssmr set MyService AppRotateBytes 1073741824`       |
| `AppRotateBytesHigh` | Unsigned 32-bit high half | High 32 bits of the size threshold. Use it for thresholds above 4 GiB.                                                                | Default is `0`.                               | `nssmr set MyService AppRotateBytesHigh 1`            |
| `AppRotateDelay`     | Milliseconds or duration  | Sleep after renaming a log before reopening it                                                                                        | Default is `0`.                               | `nssmr set MyService AppRotateDelay 500ms`            |
| `AppTimestampLog`    | Boolean                   | Prefixes each log line with `YYYY-MM-DD HH:MM:SS.mmm: `                                                                               | Default is `0`.                               | `nssmr set MyService AppTimestampLog 1`               |

- `AppRotateSeconds` is checked when an existing log file is opened at service start. Runtime rotation while the service is already running is size-based.
- Total size threshold is `AppRotateBytesHigh << 32 | AppRotateBytes`.

## Service Metadata And Identity

| Setting           | Value format                                                                                   | What it controls                                                                                     | Reset/default                      | Example                                                      |
| ----------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------- | ------------------------------------------------------------ |
| `DisplayName`     | String                                                                                         | SCM display name shown in Services                                                                   | Reset returns to the service name. | `nssmr set MyService DisplayName "My Worker Service"`        |
| `Description`     | String                                                                                         | SCM service description                                                                              | Reset clears the description.      | `nssmr set MyService Description "Background queue worker"`  |
| `Start`           | `SERVICE_AUTO_START`, `SERVICE_DELAYED_AUTO_START`, `SERVICE_DEMAND_START`, `SERVICE_DISABLED` | Service startup type. Short aliases like `automatic`, `delayed`, `manual`, and `disabled` also work. | Default is `SERVICE_AUTO_START`.   | `nssmr set MyService Start SERVICE_DELAYED_AUTO_START`       |
| `DependOnService` | One or more service names                                                                      | SCM dependencies that must start first                                                               | Reset clears the dependency list.  | `nssmr set MyService DependOnService Tcpip Dnscache`         |
| `ObjectName`      | Service account, plus a password for custom accounts                                           | SCM logon identity. Built-in accounts and `NT Service\<service>` need no password.                   | Reset returns to `LocalSystem`.    | `nssmr set MyService ObjectName "NT AUTHORITY\LocalService"` |

- For a custom account, pass the password as the final argument: `nssmr set MyService ObjectName "ACME\svc-worker" "p@ssw0rd"`.
