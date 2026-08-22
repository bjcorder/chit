package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeScreen is a minimal screen double for testing Model's navigation
// stack behavior in isolation from any real screen's data loading.
type fakeScreen struct {
	title      string
	updates    int
	lastMsg    tea.Msg
	pushOnKey  string // if a KeyMsg matching this arrives, push pushTarget
	pushTarget screen
	initCmd    tea.Cmd
}

func (f *fakeScreen) Init() tea.Cmd { return f.initCmd }

func (f *fakeScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	f.updates++
	f.lastMsg = msg
	if key, ok := msg.(tea.KeyMsg); ok && f.pushOnKey != "" && key.String() == f.pushOnKey {
		return f, pushScreen(f.pushTarget)
	}
	return f, nil
}

func (f *fakeScreen) View(width, height int) string { return "view:" + f.title }

func (f *fakeScreen) Title() string { return f.title }

func newTestModel(top screen) Model {
	return Model{styles: newStyles(), stack: []screen{top}, width: 80, height: 24}
}

func TestPushScreenMsgAppendsToStack(t *testing.T) {
	child := &fakeScreen{title: "child", initCmd: func() tea.Msg { return nil }}
	m := newTestModel(&fakeScreen{title: "root"})

	updated, cmd := m.Update(pushScreenMsg{screen: child})
	m = updated.(Model)

	if len(m.stack) != 2 || m.top() != child {
		t.Fatalf("stack = %+v, want [root, child]", m.stack)
	}
	if cmd == nil {
		t.Error("expected the pushed screen's Init() cmd to be returned")
	}
}

func TestPopScreenMsgRemovesTopOfStack(t *testing.T) {
	m := newTestModel(&fakeScreen{title: "root"})
	m.stack = append(m.stack, &fakeScreen{title: "child"})

	updated, _ := m.Update(popScreenMsg{})
	m = updated.(Model)

	if len(m.stack) != 1 || m.top().Title() != "root" {
		t.Fatalf("stack = %+v, want just [root]", m.stack)
	}
}

func TestPopScreenMsgOnRootIsNoOp(t *testing.T) {
	m := newTestModel(&fakeScreen{title: "root"})

	updated, _ := m.Update(popScreenMsg{})
	m = updated.(Model)

	if len(m.stack) != 1 {
		t.Fatalf("stack = %+v, want unchanged single-element stack", m.stack)
	}
}

func TestEscKeyPopsStack(t *testing.T) {
	m := newTestModel(&fakeScreen{title: "root"})
	m.stack = append(m.stack, &fakeScreen{title: "child"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if len(m.stack) != 1 {
		t.Fatalf("stack = %+v, want popped back to just [root]", m.stack)
	}
}

func TestQOnRootQuits(t *testing.T) {
	m := newTestModel(&fakeScreen{title: "root"})

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(Model)

	if !m.quitting {
		t.Error("expected quitting=true when 'q' pressed on root screen")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestQOnChildScreenPopsInsteadOfQuitting(t *testing.T) {
	m := newTestModel(&fakeScreen{title: "root"})
	m.stack = append(m.stack, &fakeScreen{title: "child"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(Model)

	if m.quitting {
		t.Error("'q' on a child screen should pop, not quit")
	}
	if len(m.stack) != 1 {
		t.Fatalf("stack = %+v, want popped to [root]", m.stack)
	}
}

func TestHelpKeyTogglesHelpAndSwallowsNextKey(t *testing.T) {
	top := &fakeScreen{title: "root"}
	m := newTestModel(top)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)
	if !m.showHelp {
		t.Fatal("expected showHelp=true after '?'")
	}
	if !strings.Contains(m.View(), "keybindings") {
		t.Errorf("help view missing expected content: %q", m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)
	if m.showHelp {
		t.Error("expected showHelp=false after any key closes it")
	}
	if top.updates != 0 {
		t.Error("the key that closed help should not also reach the underlying screen")
	}
}

func TestUnhandledMessagesForwardToTopScreen(t *testing.T) {
	top := &fakeScreen{title: "root"}
	m := newTestModel(top)

	type customMsg struct{}
	m.Update(customMsg{})

	if top.updates != 1 {
		t.Errorf("top screen Update called %d times, want 1", top.updates)
	}
}

func TestBreadcrumbJoinsStackTitles(t *testing.T) {
	got := breadcrumb([]screen{&fakeScreen{title: "chit"}, &fakeScreen{title: "cli"}, &fakeScreen{title: "#123"}})
	want := "chit › cli › #123"
	if got != want {
		t.Errorf("breadcrumb = %q, want %q", got, want)
	}
}
