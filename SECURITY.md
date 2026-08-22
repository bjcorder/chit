# Security Policy

chit stores API credentials via your OS's native secret store (the Secret
Service API on Linux, Keychain on macOS, Credential Manager on Windows) and
delegates all GitHub authentication to the `gh` CLI rather than handling
GitHub tokens itself. It never writes credentials to plaintext config files
or logs.

## Reporting a Vulnerability

Please report suspected vulnerabilities privately via
[GitHub Security Advisories](https://github.com/bjcorder/chit/security/advisories/new)
for this repository rather than opening a public issue. You should receive
an initial response within a few days.
