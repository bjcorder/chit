package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/provider"
)

func TestHomeScreenAggregatesMultipleProviders(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["linear"] = &fakeTracker{rootContainers: []provider.Container{{ID: "ws1", Name: "Acme", Kind: provider.KindRoot}}}
	a.IssueTrackers["github"] = &fakeTracker{rootContainers: []provider.Container{{ID: "bjcorder", Name: "bjcorder", Kind: provider.KindRoot}}}
	a.CodeHosts["github"] = &fakeHost{}

	s := newHomeScreen(context.Background(), a, newStyles("notty"))
	batchMsg := runCmd(t, s.Init())
	msgs := flattenBatch(t, batchMsg)
	for _, msg := range msgs {
		updated, _ := s.Update(msg)
		s = updated.(*homeScreen)
	}

	view := s.View(80, 24)
	if !strings.Contains(view, "Acme") || !strings.Contains(view, "bjcorder") {
		t.Errorf("home view missing containers from both providers: %q", view)
	}
	if !strings.Contains(view, "Github") || !strings.Contains(view, "Linear") {
		t.Errorf("home view missing provider section headers: %q", view)
	}
}

func TestHomeScreenShowsLoadingUntilProviderResponds(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["github"] = &fakeTracker{rootContainers: nil}

	s := newHomeScreen(context.Background(), a, newStyles("notty"))
	s.rebuildRows() // before any load completes
	view := s.View(80, 24)
	if !strings.Contains(view, "loading") {
		t.Errorf("expected a loading indicator before data arrives: %q", view)
	}
}

func TestHomeScreenShowsProviderError(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["github"] = &fakeTracker{rootErr: errFake}

	s := newHomeScreen(context.Background(), a, newStyles("notty"))
	updated, _ := s.Update(rootContainersMsg{providerName: "github", err: errFake})
	s = updated.(*homeScreen)

	if !strings.Contains(s.View(80, 24), "error") {
		t.Errorf("expected error to be shown in provider header: %q", s.View(80, 24))
	}
}

func TestHomeScreenFavoritesSectionOnlyAppearsWhenNonEmpty(t *testing.T) {
	a := newTestApp(t)
	s := newHomeScreen(context.Background(), a, newStyles("notty"))

	updated, _ := s.Update(favoritesMsg{favorites: nil})
	s = updated.(*homeScreen)
	if strings.Contains(s.View(80, 24), "Favorites") {
		t.Error("Favorites header should not appear with zero favorites")
	}
}

func TestHomeScreenEnterOnRootContainerPushesChildren(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["github"] = &fakeTracker{}

	s := newHomeScreen(context.Background(), a, newStyles("notty"))
	updated, _ := s.Update(rootContainersMsg{providerName: "github", containers: []provider.Container{{ID: "bjcorder", Name: "bjcorder", Kind: provider.KindRoot}}})
	s = updated.(*homeScreen)
	s.cursor = initialCursor(s.rows)

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pushMsg := runCmd(t, cmd).(pushScreenMsg)
	if _, ok := pushMsg.screen.(*childrenScreen); !ok {
		t.Fatalf("pushed screen is %T, want *childrenScreen", pushMsg.screen)
	}
}

func TestHomeScreenFavoriteToggleSendsCorrectContainer(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["github"] = &fakeTracker{}

	s := newHomeScreen(context.Background(), a, newStyles("notty"))
	updated, _ := s.Update(rootContainersMsg{providerName: "github", containers: []provider.Container{{ID: "bjcorder", Name: "bjcorder"}}})
	s = updated.(*homeScreen)
	s.cursor = initialCursor(s.rows)

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	msg := runCmd(t, cmd).(favoriteToggledMsg)
	if msg.providerName != "github" || msg.containerID != "bjcorder" || !msg.nowFavorite {
		t.Errorf("favoriteToggledMsg = %+v, want github/bjcorder/true", msg)
	}
}

func TestHomeScreenClearsStatusWhenContainersArrive(t *testing.T) {
	a := newTestApp(t)
	s := newHomeScreen(context.Background(), a, newStyles("notty"))
	s.status = "refreshing…"

	updated, _ := s.Update(rootContainersMsg{providerName: "github", containers: nil})
	s = updated.(*homeScreen)

	if s.status != "" {
		t.Errorf("status = %q, want cleared once containers arrive", s.status)
	}
}

// flattenBatch executes a tea.BatchMsg (the []tea.Cmd bubbletea's
// tea.Batch produces) and collects each sub-command's resulting message.
func flattenBatch(t *testing.T, msg tea.Msg) []tea.Msg {
	t.Helper()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	var out []tea.Msg
	for _, cmd := range batch {
		out = append(out, cmd())
	}
	return out
}
