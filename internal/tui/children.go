package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/provider"
)

// childrenScreen lists the containers directly under a root container: a
// GitHub org/user's repos, or (largely a no-op for Linear, which has only
// one root) a Linear workspace's teams.
type childrenScreen struct {
	ctx          context.Context
	a            *app.App
	st           styles
	providerName string
	parent       provider.Container

	containers []provider.Container
	favorites  map[string]bool
	loaded     bool
	err        error

	rows   []row
	cursor int
	status string
}

func newChildrenScreen(ctx context.Context, a *app.App, st styles, providerName string, parent provider.Container) *childrenScreen {
	return &childrenScreen{ctx: ctx, a: a, st: st, providerName: providerName, parent: parent, favorites: map[string]bool{}}
}

func (s *childrenScreen) Title() string { return s.parent.Name }

func (s *childrenScreen) Init() tea.Cmd {
	return tea.Batch(
		loadChildContainers(s.ctx, s.a, s.providerName, s.parent.ID, false),
		loadFavorites(s.ctx, s.a),
	)
}

func (s *childrenScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case childContainersMsg:
		if msg.providerName != s.providerName || msg.parentID != s.parent.ID {
			return s, nil
		}
		s.loaded = true
		s.err = msg.err
		if msg.err == nil {
			s.containers = msg.containers
		}
		s.rebuildRows()
		return s, nil

	case favoritesMsg:
		if msg.err == nil {
			s.favorites = map[string]bool{}
			for _, f := range msg.favorites {
				if f.Provider == s.providerName {
					s.favorites[f.Container.ID] = true
				}
			}
			s.rebuildRows()
		}
		return s, nil

	case favoriteToggledMsg:
		if msg.err != nil {
			s.status = "favorite: " + msg.err.Error()
			return s, nil
		}
		return s, loadFavorites(s.ctx, s.a)

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			s.cursor = moveCursor(s.rows, s.cursor, 1)
		case "k", "up":
			s.cursor = moveCursor(s.rows, s.cursor, -1)
		case "r":
			s.status = "refreshing…"
			return s, loadChildContainers(s.ctx, s.a, s.providerName, s.parent.ID, true)
		case "f":
			if c, ok := s.containerAtCursor(); ok {
				return s, toggleFavorite(s.ctx, s.a, s.providerName, c, s.favorites[c.ID])
			}
		case "enter", "l":
			if c, ok := s.containerAtCursor(); ok {
				return s, pushScreen(newItemsScreen(s.ctx, s.a, s.st, s.providerName, c))
			}
		}
	}
	return s, nil
}

func (s *childrenScreen) rebuildRows() {
	rows := make([]row, 0, len(s.containers))
	for i, c := range s.containers {
		rows = append(rows, row{label: c.Name, favorite: s.favorites[c.ID], sourceIndex: i})
	}
	s.rows = rows
	if s.cursor >= len(rows) {
		s.cursor = initialCursor(rows)
	}
}

func (s *childrenScreen) containerAtCursor() (provider.Container, bool) {
	if s.cursor < 0 || s.cursor >= len(s.rows) {
		return provider.Container{}, false
	}
	return s.containers[s.rows[s.cursor].sourceIndex], true
}

func (s *childrenScreen) View(width, height int) string {
	if !s.loaded {
		return s.st.dim.Render("loading…")
	}
	if s.err != nil {
		return s.st.errorText.Render("error: " + s.err.Error())
	}
	if len(s.containers) == 0 {
		return s.st.dim.Render("(no repos)")
	}
	rowsHeight := height - 1 // reserve a line for the status message
	body := renderRowsScrolled(s.rows, s.cursor, rowsHeight, s.st)
	if s.status != "" {
		body += "\n" + s.st.dim.Render(s.status)
	}
	return body
}
