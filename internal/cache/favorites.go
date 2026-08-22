package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/bjcorder/chit/internal/provider"
)

// Favorite is a starred container (org/repo, workspace/team) for quick-jump
// navigation. Only containers are favoritable in v1 — not individual
// issues/PRs.
type Favorite struct {
	Provider  string
	Container provider.Container
}

func (s *Store) Favorites(ctx context.Context) ([]Favorite, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT provider, container_id, name, kind, parent_id FROM favorites
		ORDER BY provider, name
	`)
	if err != nil {
		return nil, fmt.Errorf("cache: listing favorites: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var favorites []Favorite
	for rows.Next() {
		var f Favorite
		var kind string
		if err := rows.Scan(&f.Provider, &f.Container.ID, &f.Container.Name, &kind, &f.Container.ParentID); err != nil {
			return nil, fmt.Errorf("cache: reading favorite row: %w", err)
		}
		f.Container.Kind = provider.ContainerKind(kind)
		favorites = append(favorites, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cache: listing favorites: %w", err)
	}
	return favorites, nil
}

func (s *Store) AddFavorite(ctx context.Context, providerName string, c provider.Container) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO favorites (provider, container_id, name, kind, parent_id, added_at) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (provider, container_id) DO UPDATE SET name = excluded.name, kind = excluded.kind, parent_id = excluded.parent_id
	`, providerName, c.ID, c.Name, string(c.Kind), c.ParentID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("cache: adding favorite %s/%s: %w", providerName, c.ID, err)
	}
	return nil
}

func (s *Store) RemoveFavorite(ctx context.Context, providerName, containerID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM favorites WHERE provider = ? AND container_id = ?`, providerName, containerID)
	if err != nil {
		return fmt.Errorf("cache: removing favorite %s/%s: %w", providerName, containerID, err)
	}
	return nil
}

func (s *Store) IsFavorite(ctx context.Context, providerName, containerID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM favorites WHERE provider = ? AND container_id = ?`, providerName, containerID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("cache: checking favorite %s/%s: %w", providerName, containerID, err)
	}
	return n > 0, nil
}
