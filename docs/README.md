# Documentation

These pages document the current `nssmr` CLI surface in terms of what you type, what gets stored, and which features already match classic NSSM.

| Page                                           | What it covers                                                                      |
| ---------------------------------------------- | ----------------------------------------------------------------------------------- |
| [command-reference.md](command-reference.md)   | Every CLI command, its syntax, and a copy-paste example                             |
| [settings-reference.md](settings-reference.md) | Supported `get`/`set`/`reset` settings with accepted values, defaults, and examples |
| [compatibility.md](compatibility.md)           | Current command and setting coverage compared with classic NSSM                     |
| [architecture.md](architecture.md)             | Project layout and the service/runtime design used in the Go port                   |

## CLI Parameter Conventions

| Placeholder      | Meaning                                               | Notes                                                                                              | Example                                        |
| ---------------- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `<service>`      | Windows service name managed by `nssmr`               | Required by almost every command                                                                   | `MyWorker`                                     |
| `<application>`  | Path to the managed executable                        | `install` stores this as `Application`                                                             | `"C:\apps\worker.exe"`                         |
| `[arguments...]` | Initial application arguments passed during `install` | Stored as one `AppParameters` string                                                               | `--listen :8080 --config "C:\apps\worker.yml"` |
| `<setting>`      | One supported configuration key                       | Most names match classic NSSM                                                                      | `AppStdout`                                    |
| `[additional]`   | Extra selector required by a few settings             | `AppExit` needs `Default`, `*`, or an exit code. `AppEvents` needs a hook key such as `Start/Pre`. | `Default`                                      |
| `[value...]`     | One or more values written by `set`                   | Multi-value settings keep each argument as its own entry                                           | `"ENV=prod" "PORT=8080"`                       |

## Common Patterns

| Goal                      | Command                                                       |
| ------------------------- | ------------------------------------------------------------- |
| Install a service         | `nssmr install MyService "C:\apps\worker.exe" --listen :8080` |
| Read one setting          | `nssmr get MyService AppStdout`                               |
| Read a keyed setting      | `nssmr get MyService AppExit Default`                         |
| Update a setting          | `nssmr set MyService AppPriority ABOVE_NORMAL_PRIORITY_CLASS` |
| Reset a setting           | `nssmr reset MyService AppPriority`                           |
| Export the current config | `nssmr dump MyService`                                        |

- `get` prints multi-value settings one entry per line.
- `set` replaces the stored value for that setting; it does not append unless the setting itself is multi-value and you pass all desired entries in the same command.
- `install` is intentionally minimal. Most advanced behavior is configured afterward with `set`.
