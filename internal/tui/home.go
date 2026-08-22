package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
	"github.com/bjcorder/chit/internal/cache"
	"github.com/bjcorder/chit/internal/provider"
)

// homeScreen is the app's root screen: a Favorites section (if any) plus
// every enabled provider's root containers, grouped by provider — a single
// flat list rather than a provider-picker step first (see design decision:
// grouped single home).
type homeScreen struct {
	ctx context.Context
	a   *app.App
	st  styles

	providerNames []string

	favorites       []cache.Favorite
	favoritesLoaded bool

	rootByProvider   map[string][]provider.Container
	loadedByProvider map[string]bool
	errByProvider    map[string]error

	rows   []row
	cursor int
	status string
}

func newHomeScreen(ctx context.Context, a *app.App, st styles) *homeScreen {
	names := map[string]bool{}
	for n := range a.IssueTrackers {
		names[n] = true
	}
	for n := range a.CodeHosts {
		names[n] = true
	}

	return &homeScreen{
		ctx:              ctx,
		a:                a,
		st:               st,
		providerNames:    sortedProviderNames(names),
		rootByProvider:   map[string][]provider.Container{},
		loadedByProvider: map[string]bool{},
		errByProvider:    map[string]error{},
	}
}

func (s *homeScreen) Title() string { return "chit" }

func (s *homeScreen) Init() tea.Cmd {
	return s.load(false)
}

func (s *homeScreen) load(forceRefresh bool) tea.Cmd {
	cmds := []tea.Cmd{loadFavorites(s.ctx, s.a)}
	for _, name := range s.providerNames {
		cmds = append(cmds, loadRootContainers(s.ctx, s.a, name, forceRefresh))
	}
	return tea.Batch(cmds...)
}

func (s *homeScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case favoritesMsg:
		s.favoritesLoaded = true
		if msg.err == nil {
			s.favorites = msg.favorites
		}
		s.rebuildRows()
		return s, nil

	case rootContainersMsg:
		s.loadedByProvider[msg.providerName] = true
		if msg.err != nil {
			s.errByProvider[msg.providerName] = msg.err
		} else {
			delete(s.errByProvider, msg.providerName)
			s.rootByProvider[msg.providerName] = msg.containers
		}
		s.status = ""
		s.rebuildRows()
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
			return s, s.load(true)
		case "f":
			return s, s.toggleFavoriteAtCursor()
		case "enter", "l":
			return s, s.selectAtCursor()
		}
	}
	return s, nil
}

func (s *homeScreen) rebuildRows() {
	var rows []row

	if len(s.favorites) > 0 {
		rows = append(rows, row{header: "Favorites"})
		for i, f := range s.favorites {
			rows = append(rows, row{
				label:       fmt.Sprintf("[%s] %s", f.Provider, f.Container.Name),
				favorite:    true,
				section:     "favorites",
				sourceIndex: i,
			})
		}
	}

	for _, name := range s.providerNames {
		header := strings.ToUpper(name[:1]) + name[1:]
		if !s.loadedByProvider[name] {
			header += " (loading…)"
		} else if s.errByProvider[name] != nil {
			header += " (error: " + s.errByProvider[name].Error() + ")"
		}
		rows = append(rows, row{header: header})

		for i, c := range s.rootByProvider[name] {
			rows = append(rows, row{
				label:       c.Name,
				favorite:    s.isFavorite(name, c.ID),
				section:     name,
				sourceIndex: i,
			})
		}
	}

	s.rows = rows
	if s.cursor >= len(rows) {
		s.cursor = initialCursor(rows)
	}
}

func (s *homeScreen) isFavorite(providerName, containerID string) bool {
	for _, f := range s.favorites {
		if f.Provider == providerName && f.Container.ID == containerID {
			return true
		}
	}
	return false
}

func (s *homeScreen) containerAtCursor() (providerName string, c provider.Container, ok bool) {
	if s.cursor < 0 || s.cursor >= len(s.rows) {
		return "", provider.Container{}, false
	}
	r := s.rows[s.cursor]
	if r.header != "" || r.section == "" {
		return "", provider.Container{}, false
	}

	if r.section == "favorites" {
		f := s.favorites[r.sourceIndex]
		return f.Provider, f.Container, true
	}
	return r.section, s.rootByProvider[r.section][r.sourceIndex], true
}

func (s *homeScreen) toggleFavoriteAtCursor() tea.Cmd {
	name, c, ok := s.containerAtCursor()
	if !ok {
		return nil
	}
	return toggleFavorite(s.ctx, s.a, name, c, s.isFavorite(name, c.ID))
}

func (s *homeScreen) selectAtCursor() tea.Cmd {
	name, c, ok := s.containerAtCursor()
	if !ok {
		return nil
	}
	if c.Kind == provider.KindChild {
		return pushScreen(newItemsScreen(s.ctx, s.a, s.st, name, c))
	}
	return pushScreen(newChildrenScreen(s.ctx, s.a, s.st, name, c))
}

func (s *homeScreen) View(width, height int) string {
	rowsHeight := height - 1 // reserve a line for the status message
	body := renderRowsScrolled(s.rows, s.cursor, rowsHeight, s.st)
	if s.status != "" {
		body += "\n" + s.st.dim.Render(s.status)
	}
	return body
}
