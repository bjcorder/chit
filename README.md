# chit

A TUI for browsing and acting on issues and pull requests across pluggable
providers — GitHub (Issues + PRs) and Linear (Issues) to start.

## Status

Early scaffold. The provider architecture, config loader, and domain model
are in place; the GitHub/Linear providers and the TUI itself are not built
yet.

## Design

- **Providers** implement one or both of two capability interfaces —
  `IssueTracker` and `CodeHost` (see `internal/provider`) — and register
  themselves at compile time via `init()`. Which registered providers
  actually run is decided at runtime by `~/.config/chit/config.toml`.
- **GitHub** integration shells out to the `gh` CLI, which already owns
  GitHub auth via your OS keyring — chit never handles a GitHub token
  directly.
- **Linear** integration talks to Linear's GraphQL API directly using a
  personal API key, stored via your OS's native secret store.
- **Domain model** (`internal/domain`) normalizes both providers' data into
  a small set of types — `Issue`, `PullRequest`, `Comment`, and a flat
  `Badge{Label, Color}` list for state/priority/labels/CI status — so the
  UI layer never special-cases a provider.

## Requirements

- Go 1.24+
- [`gh`](https://cli.github.com/), authenticated (`gh auth status`), for the
  GitHub provider

## Development

```sh
go build ./...
go test ./...
go vet ./...
```

## License

[MIT](LICENSE)
