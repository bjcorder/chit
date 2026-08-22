# chit

A TUI for browsing and acting on issues and pull requests across pluggable
providers — GitHub (Issues + PRs) and Linear (Issues) to start.

## Status

Working v1. Browse GitHub orgs/repos and Linear workspaces/teams, view
issues and PRs (badges, glamour-rendered markdown, CI checks, commit
lists), comment and reply, approve/merge/mark-ready PRs, and open links
via an inline Vimium-style hint overlay — all vim-keybound, cache-first,
with manual refresh.

## Quick start

```sh
go build -o chit ./cmd/chit
./chit providers list
./chit providers enable github          # gh handles auth itself
./chit providers set-key linear         # prompts for a personal API key
./chit providers enable linear
./chit
```

Press `?` in the TUI for the full keybinding reference.

## Design

- **Providers** implement one or both of two capability interfaces —
  `IssueTracker` and `CodeHost` (see `internal/provider`) — and register
  themselves at compile time via `init()`. Which registered providers
  actually run is decided at runtime by `~/.config/chit/config.toml`.
- **GitHub** integration shells out to the `gh` CLI, which already owns
  GitHub auth via your OS keyring — chit never handles a GitHub token
  directly.
- **Linear** integration talks to Linear's GraphQL API directly using a
  personal API key, stored via your OS's native secret store
  (`internal/secret`, Secret Service/Keychain/Credential Manager, falling
  back to an encrypted file vault).
- **Domain model** (`internal/domain`) normalizes both providers' data into
  a small set of types — `Issue`, `PullRequest`, `Comment`, and a flat
  `Badge{Label, Color}` list for state/priority/labels/CI status — so the
  UI layer never special-cases a provider.
- **Cache** (`internal/cache`, SQLite) makes navigation instant on repeat
  visits; every screen is cache-first with `r` to force a refresh past it.
- **TUI** (`internal/tui`, Bubble Tea) is a stack of screens — home
  (favorites + every enabled provider's root containers) → children →
  items → detail — coordinated through push/pop messages so no screen
  needs to know about the navigation stack itself.

## Requirements

- Go 1.25+
- [`gh`](https://cli.github.com/), authenticated (`gh auth status`), for the
  GitHub provider
- `$EDITOR` (or `$VISUAL`) set, for composing comments and replies

## Development

```sh
go build ./...
go test ./...
go vet ./...
```

## License

[MIT](LICENSE)
