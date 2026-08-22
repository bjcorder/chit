// Package cache is chit's local read cache: a SQLite database (pure-Go
// driver, no cgo) that lets navigation render instantly from the last
// fetch instead of hitting a provider on every keystroke. It's a cache,
// not a source of truth — nothing here is authoritative, and every entry
// can be safely dropped and re-fetched. Refresh is manual (a keybinding in
// the TUI), not a background poller.
package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver

	"github.com/bjcorder/chit/internal/config"
	"github.com/bjcorder/chit/internal/domain"
	"github.com/bjcorder/chit/internal/provider"
)

const schema = `
CREATE TABLE IF NOT EXISTS cache_entries (
	provider   TEXT NOT NULL,
	kind       TEXT NOT NULL,
	key        TEXT NOT NULL,
	data       BLOB NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	PRIMARY KEY (provider, kind, key)
);

CREATE TABLE IF NOT EXISTS favorites (
	provider     TEXT NOT NULL,
	container_id TEXT NOT NULL,
	name         TEXT NOT NULL,
	kind         TEXT NOT NULL,
	parent_id    TEXT NOT NULL,
	added_at     TIMESTAMP NOT NULL,
	PRIMARY KEY (provider, container_id)
);
`

// cacheFormatVersion identifies the shape of the domain.Issue/PullRequest
// (and any other cached type) JSON blobs stored in cache_entries. Bump this
// whenever a cached type's fields change meaning — encoding/json silently
// zero-values missing fields on unmarshal, so an old blob deserializes
// without error but with stale, wrong data. (This actually happened: adding
// domain.Issue.StateBadge left already-cached issues with an empty
// StateBadge and the state badge still merged into Badges, which looked
// exactly like the badge-reorder fix hadn't shipped, until the view was
// manually refreshed.) Open wipes cache_entries — never favorites, which
// has its own stable schema unrelated to domain.Issue/PullRequest — the
// first time it sees a different stored version.
const cacheFormatVersion = 2

type entryKind string

const (
	kindRootContainers  entryKind = "root_containers"
	kindChildContainers entryKind = "child_containers"
	kindIssues          entryKind = "issues"
	kindPullRequests    entryKind = "pull_requests"
	kindIssueDetail     entryKind = "issue_detail"
	kindPRDetail        entryKind = "pr_detail"
)

// Store is chit's cache database handle.
type Store struct {
	db *sql.DB
}

// DefaultPath returns the XDG-compliant cache DB location,
// ~/.local/share/chit/chit.db on Linux.
func DefaultPath() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "chit.db"), nil
}

// Open opens (creating if needed) the cache database at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("cache: creating %s: %w", filepath.Dir(path), err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("cache: opening %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cache: creating schema: %w", err)
	}
	if err := invalidateOnVersionMismatch(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cache: checking schema version: %w", err)
	}
	return &Store{db: db}, nil
}

// invalidateOnVersionMismatch drops every cached entry, but not favorites,
// the first time it sees a cacheFormatVersion different from what's
// recorded in this database — see the doc comment on cacheFormatVersion.
func invalidateOnVersionMismatch(db *sql.DB) error {
	var stored int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&stored); err != nil {
		return fmt.Errorf("reading user_version: %w", err)
	}
	if stored == cacheFormatVersion {
		return nil
	}
	if _, err := db.Exec(`DELETE FROM cache_entries`); err != nil {
		return fmt.Errorf("clearing stale cache entries: %w", err)
	}
	// #nosec G201 -- cacheFormatVersion is a compile-time constant, not user input; PRAGMA statements don't support bind parameters in SQLite
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, cacheFormatVersion)); err != nil {
		return fmt.Errorf("writing user_version: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func get[T any](ctx context.Context, s *Store, providerName string, kind entryKind, key string) (T, bool, error) {
	var zero T
	row := s.db.QueryRowContext(ctx, `SELECT data FROM cache_entries WHERE provider = ? AND kind = ? AND key = ?`, providerName, string(kind), key)

	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("cache: reading %s/%s/%s: %w", providerName, kind, key, err)
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, false, fmt.Errorf("cache: decoding %s/%s/%s: %w", providerName, kind, key, err)
	}
	return v, true, nil
}

func set[T any](ctx context.Context, s *Store, providerName string, kind entryKind, key string, value T) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: encoding %s/%s/%s: %w", providerName, kind, key, err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cache_entries (provider, kind, key, data, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (provider, kind, key) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at
	`, providerName, string(kind), key, data, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("cache: writing %s/%s/%s: %w", providerName, kind, key, err)
	}
	return nil
}

func (s *Store) RootContainers(ctx context.Context, providerName string) ([]provider.Container, bool, error) {
	return get[[]provider.Container](ctx, s, providerName, kindRootContainers, "")
}

func (s *Store) SetRootContainers(ctx context.Context, providerName string, containers []provider.Container) error {
	return set(ctx, s, providerName, kindRootContainers, "", containers)
}

func (s *Store) ChildContainers(ctx context.Context, providerName, parentID string) ([]provider.Container, bool, error) {
	return get[[]provider.Container](ctx, s, providerName, kindChildContainers, parentID)
}

func (s *Store) SetChildContainers(ctx context.Context, providerName, parentID string, containers []provider.Container) error {
	return set(ctx, s, providerName, kindChildContainers, parentID, containers)
}

func (s *Store) Issues(ctx context.Context, providerName, containerID string) ([]domain.Issue, bool, error) {
	return get[[]domain.Issue](ctx, s, providerName, kindIssues, containerID)
}

func (s *Store) SetIssues(ctx context.Context, providerName, containerID string, issues []domain.Issue) error {
	return set(ctx, s, providerName, kindIssues, containerID, issues)
}

func (s *Store) IssueDetail(ctx context.Context, providerName, issueID string) (domain.Issue, bool, error) {
	return get[domain.Issue](ctx, s, providerName, kindIssueDetail, issueID)
}

func (s *Store) SetIssueDetail(ctx context.Context, providerName, issueID string, issue domain.Issue) error {
	return set(ctx, s, providerName, kindIssueDetail, issueID, issue)
}

func (s *Store) PullRequests(ctx context.Context, providerName, containerID string) ([]domain.PullRequest, bool, error) {
	return get[[]domain.PullRequest](ctx, s, providerName, kindPullRequests, containerID)
}

func (s *Store) SetPullRequests(ctx context.Context, providerName, containerID string, prs []domain.PullRequest) error {
	return set(ctx, s, providerName, kindPullRequests, containerID, prs)
}

func (s *Store) PRDetail(ctx context.Context, providerName, prID string) (domain.PullRequest, bool, error) {
	return get[domain.PullRequest](ctx, s, providerName, kindPRDetail, prID)
}

func (s *Store) SetPRDetail(ctx context.Context, providerName, prID string, pr domain.PullRequest) error {
	return set(ctx, s, providerName, kindPRDetail, prID, pr)
}
