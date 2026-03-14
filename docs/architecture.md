# Architecture

The Go port is split into a small number of focused internal packages:

- `internal/config`: portable service model, supported setting definitions, and dump rendering
- `internal/scm`: Windows-only SCM and registry integration
- `internal/runtime`: child-process launch, stdio logging/rotation, and restart policy
- `internal/svcwrap`: the SCM-facing service host that supervises the managed process and runs AppEvents hooks
- `internal/support`: portable helpers for command-line quoting and environment merging

## Runtime model

1. The installed Windows service launches `nssmr service <name>`.
2. `internal/svcwrap` loads the service definition from SCM plus the `Parameters` registry key.
3. `internal/runtime` starts the managed executable with the requested environment, working directory, stdio redirection, and optional timestamp/rotation logging.
4. `internal/svcwrap` runs `AppEvents` hooks around start, stop, exit, power, and rotate transitions.
5. When the child exits, the supervisor applies `AppExit`, `AppRestartDelay`, and `AppThrottle` to decide whether and when to restart it.

## Design choices in this first milestone

- The port keeps the familiar registry contract where it adds value for migration and operability.
- The CLI is Windows-first but pure-Go core packages are kept portable so most tests can run on Linux and macOS CI.
- Process-tree teardown uses a Job Object rather than the older window/thread walking logic. That is simpler, more reliable, and a good fit for a fresh Windows-only implementation.
- The legacy GUI is intentionally out of scope for now; parity work is concentrated in the CLI, SCM, and runtime layers.
