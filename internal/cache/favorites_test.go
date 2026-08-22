package cache

import (
	"context"
	"testing"

	"github.com/bjcorder/chit/internal/provider"
)

func TestFavoritesEmptyInitially(t *testing.T) {
	s := openTestStore(t)
	got, err := s.Favorites(context.Background())
	if err != nil {
		t.Fatalf("Favorites: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %+v, want empty", got)
	}
}

func TestAddAndListFavorite(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	c := provider.Container{ID: "cli/cli", Kind: provider.KindChild, Name: "cli", ParentID: "cli"}
	if err := s.AddFavorite(ctx, "github", c); err != nil {
		t.Fatalf("AddFavorite: %v", err)
	}

	got, err := s.Favorites(ctx)
	if err != nil {
		t.Fatalf("Favorites: %v", err)
	}
	if len(got) != 1 || got[0].Provider != "github" || got[0].Container.ID != "cli/cli" || got[0].Container.ParentID != "cli" {
		t.Fatalf("got %+v", got)
	}

	isFav, err := s.IsFavorite(ctx, "github", "cli/cli")
	if err != nil || !isFav {
		t.Errorf("IsFavorite = %v, %v, want true, nil", isFav, err)
	}
}

func TestAddFavoriteTwiceIsIdempotent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	c := provider.Container{ID: "cli/cli", Name: "cli"}

	if err := s.AddFavorite(ctx, "github", c); err != nil {
		t.Fatalf("AddFavorite (1st): %v", err)
	}
	if err := s.AddFavorite(ctx, "github", c); err != nil {
		t.Fatalf("AddFavorite (2nd): %v", err)
	}

	got, err := s.Favorites(ctx)
	if err != nil {
		t.Fatalf("Favorites: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d favorites, want exactly 1 after adding the same container twice", len(got))
	}
}

func TestRemoveFavorite(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	c := provider.Container{ID: "cli/cli", Name: "cli"}

	_ = s.AddFavorite(ctx, "github", c)
	if err := s.RemoveFavorite(ctx, "github", "cli/cli"); err != nil {
		t.Fatalf("RemoveFavorite: %v", err)
	}

	isFav, err := s.IsFavorite(ctx, "github", "cli/cli")
	if err != nil || isFav {
		t.Errorf("IsFavorite after removal = %v, %v, want false, nil", isFav, err)
	}
}

func TestRemoveFavoriteNotPresentIsNotAnError(t *testing.T) {
	s := openTestStore(t)
	if err := s.RemoveFavorite(context.Background(), "github", "does/not-exist"); err != nil {
		t.Errorf("RemoveFavorite of missing favorite returned error: %v", err)
	}
}

func TestFavoritesAcrossProvidersDoNotCollide(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.AddFavorite(ctx, "github", provider.Container{ID: "same-id", Name: "gh"})
	_ = s.AddFavorite(ctx, "linear", provider.Container{ID: "same-id", Name: "lin"})

	got, err := s.Favorites(ctx)
	if err != nil {
		t.Fatalf("Favorites: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d favorites, want 2 (same container ID under different providers)", len(got))
	}
}
