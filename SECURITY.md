# Security Policy

## Supported Versions

`nssmr` is still pre-release. As of March 17, 2026, this repository does not have any tagged releases yet, so security fixes are developed on `main`.

| Version                                          | Supported          | Notes                                                                  |
| ------------------------------------------------ | ------------------ | ---------------------------------------------------------------------- |
| `main`                                           | :white_check_mark: | Security fixes land here first                                         |
| Latest tagged release                            | :warning:          | Once releases begin, support is expected to follow the newest tag only |
| Older tags, snapshots, and fork-specific changes | :x:                | Upgrade to the latest supported code before requesting a fix           |

Until tagged releases exist, please reproduce issues against the latest `main` branch before reporting them.

## Scope

This project is a Windows-focused service manager that installs and supervises arbitrary executables as Windows services. Reports are especially helpful when they involve:

- privilege escalation or unintended execution under a more privileged service account
- unexpected command execution, argument parsing, quoting, or hook handling issues
- unsafe handling of registry-backed service configuration
- log file, path, or file-permission issues that could expose or overwrite data unexpectedly
- release artifact integrity problems, including checksum mismatches in published GitHub releases
- dependency or supply-chain issues affecting the shipped binaries, GitHub Actions workflows, or Go module dependencies

Behavior that is only the result of an intentionally configured wrapped executable, service account, or `AppEvents` hook is usually not considered a vulnerability by itself unless `nssmr` crosses an unexpected trust or privilege boundary.

## Reporting a Vulnerability

Please do **not** open a public GitHub issue, discussion, or pull request for security problems.

Use one of these private reporting paths:

1. If GitHub private vulnerability reporting is enabled for this repository, use the **Security** tab and choose **Report a vulnerability**.
2. If that option is not available, use a non-public contact method listed on the repository owner profile at <https://github.com/jonlabelle> and mention `nssm-redux` in the message.

Please include as much of the following as you can:

- the affected commit, branch, or release, plus the output of `nssmr version` if available
- the Windows version and architecture involved
- whether the issue requires a specific service account, registry setting, hook, or log configuration
- clear reproduction steps, proof of concept, and expected vs actual behavior
- the security impact you believe the issue has
- whether the issue affects default behavior or only a non-default configuration

If your report involves a published binary, include the exact release asset name and any checksum mismatch against `SHA256SUMS.txt`.

## What To Expect

- The maintainer will try to acknowledge new reports within 3 business days.
- The maintainer will try to provide a status update at least once per week while the report is being triaged or fixed.
- If the report is accepted, the fix will land on `main` first and may be included in the next tagged release once releases begin.
- If the report is declined, the response should explain why, for example if the behavior is expected, requires an already-compromised system, or depends on insecure local configuration outside `nssmr` itself.

Please test only on systems you own or are authorized to assess, avoid disrupting real services, and avoid accessing or modifying data that is not yours.
