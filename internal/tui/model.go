// Package tui is chit's Bubble Tea application: a stack of screens
// (home -> children -> items -> detail) driven by vim-style keybindings,
// backed by internal/app's cache-first data loading.
package tui

import (
	"context"
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bjcorder/chit/internal/app"
)

// screen is one level of chit's navigation stack.
type screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (screen, tea.Cmd)
	View(width, height int) string
	Title() string
}

// pushScreenMsg and popScreenMsg let a screen ask the top-level Model to
// change the navigation stack without the screen needing to know about the
// stack itself.
type pushScreenMsg struct{ screen screen }
type popScreenMsg struct{}

func pushScreen(s screen) tea.Cmd {
	return func() tea.Msg { return pushScreenMsg{screen: s} }
}

// Model is chit's root Bubble Tea model.
type Model struct {
	ctx    context.Context
	app    *app.App
	styles styles

	stack  []screen
	width  int
	height int

	showHelp bool
	quitting bool
}

// New builds the root Model, with the home screen as the base of the
// navigation stack. It must be called before tea.NewProgram(...).Run() —
// see resolveGlamourStyle.
func New(ctx context.Context, a *app.App) Model {
	st := newStyles(resolveGlamourStyle())
	return Model{
		ctx:    ctx,
		app:    a,
		styles: st,
		stack:  []screen{newHomeScreen(ctx, a, st)},
	}
}

func (m Model) Init() tea.Cmd {
	return m.top().Init()
}

func (m Model) top() screen {
	return m.stack[len(m.stack)-1]
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case pushScreenMsg:
		m.stack = append(m.stack, msg.screen)
		return m, msg.screen.Init()

	case popScreenMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			return m, m.top().Init()
		}
		return m, nil

	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				return m, m.top().Init()
			}
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.showHelp = true
			return m, nil
		case "esc", "backspace", "h":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				return m, m.top().Init()
			}
			return m, nil
		case "F":
			m.stack = []screen{newHomeScreenFocusedOnFavorites(m.ctx, m.app, m.styles)}
			return m, m.top().Init()
		}
	}

	top, cmd := m.top().Update(msg)
	m.stack[len(m.stack)-1] = top
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.showHelp {
		return renderHelp(m.styles)
	}

	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	header := m.styles.title.Render(breadcrumb(m.stack))
	body := m.top().View(m.width, contentHeight)
	footer := m.styles.help.Render("j/k move  enter select  h/esc back  f favorite  r refresh  ? help  q quit")

	return header + "\n" + body + "\n" + footer
}

func breadcrumb(stack []screen) string {
	titles := make([]string, len(stack))
	for i, s := range stack {
		titles[i] = s.Title()
	}
	out := titles[0]
	for _, t := range titles[1:] {
		out += " › " + t
	}
	return out
}

func renderHelp(st styles) string {
	lines := []string{
		"chit — keybindings",
		"",
		"  j / down       move down",
		"  k / up         move up",
		"  enter          open selected item",
		"  h / esc / bksp back",
		"  f              toggle favorite (on a container row)",
		"  F              jump to favorites, from anywhere",
		"  p              toggle issues/PRs (on an items screen, GitHub only)",
		"  x              toggle closed issues/merged PRs (hidden by default)",
		"  r              refresh from provider (bypass cache)",
		"  ?              toggle this help",
		"  q / ctrl+c     back, or quit from the home screen",
		"",
		"on an issue/PR detail screen:",
		"  c              compose a new comment ($EDITOR)",
		"  v              pick a comment to reply to ($EDITOR)",
		"  f              show link hints — type a label to open it",
		"",
		"on a PR detail screen:",
		"  a              approve (then y to confirm, n to cancel)",
		"  m              propose/cycle an allowed merge method, y to confirm",
		"  d              mark a draft PR ready for review",
		"",
		"press any key to close",
	}
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return st.help.Render(out)
}

// sortedProviderNames is a small shared helper for screens that need a
// stable iteration order over app.App's provider maps.
func sortedProviderNames(names map[string]bool) []string {
	out := make([]string, 0, len(names))
	for n := range names {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
