package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/provider"
)

func TestChildrenScreenLoadsAndSelectsIntoItems(t *testing.T) {
	a := newTestApp(t)
	a.IssueTrackers["gh"] = &fakeTracker{
		childContainers: map[string][]provider.Container{
			"bjcorder": {{ID: "bjcorder/chit", Name: "chit", ParentID: "bjcorder"}},
		},
	}

	s := newChildrenScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "bjcorder", Name: "bjcorder"})
	msg := runCmd(t, loadChildContainers(context.Background(), a, "gh", "bjcorder", false))
	updated, _ := s.Update(msg)
	s = updated.(*childrenScreen)

	if !s.loaded || len(s.containers) != 1 || s.containers[0].Name != "chit" {
		t.Fatalf("containers not loaded correctly: %+v", s.containers)
	}
	if !strings.Contains(s.View(80, 24), "chit") {
		t.Errorf("View() missing container name: %q", s.View(80, 24))
	}

	s.cursor = initialCursor(s.rows)
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pushMsg := runCmd(t, cmd).(pushScreenMsg)
	items, ok := pushMsg.screen.(*itemsScreen)
	if !ok || items.container.ID != "bjcorder/chit" {
		t.Fatalf("pushed screen = %+v, want itemsScreen for bjcorder/chit", pushMsg.screen)
	}
}

func TestChildrenScreenIgnoresStaleResponses(t *testing.T) {
	a := newTestApp(t)
	s := newChildrenScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "bjcorder"})

	updated, _ := s.Update(childContainersMsg{providerName: "gh", parentID: "some-other-parent", containers: []provider.Container{{ID: "x"}}})
	s = updated.(*childrenScreen)

	if s.loaded {
		t.Error("a response for a different parentID should be ignored")
	}
}

func TestChildrenScreenEmptyState(t *testing.T) {
	a := newTestApp(t)
	s := newChildrenScreen(context.Background(), a, newStyles("notty"), "gh", provider.Container{ID: "bjcorder"})

	updated, _ := s.Update(childContainersMsg{providerName: "gh", parentID: "bjcorder", containers: nil})
	s = updated.(*childrenScreen)

	if !strings.Contains(s.View(80, 24), "no repos") {
		t.Errorf("expected an empty-state message: %q", s.View(80, 24))
	}
}
