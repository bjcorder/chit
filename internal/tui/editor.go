package tui

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// editorClosedMsg reports that a $EDITOR session chit opened for composing
// a comment or reply has exited.
type editorClosedMsg struct {
	tmpPath         string
	issueID         string
	parentCommentID string // empty for a new top-level comment
	err             error
}

// editorCommand returns the command to run for composing text: $VISUAL,
// then $EDITOR, then "vi" as a last resort — the same fallback chain most
// terminal tools (git included) use.
func editorCommand() string {
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vi"
}

// openEditorCmd creates a temp file and returns a Cmd that suspends the TUI
// to edit it, resolving to editorClosedMsg once the editor exits.
func openEditorCmd(issueID, parentCommentID string) tea.Cmd {
	tmp, err := os.CreateTemp("", "chit-comment-*.md")
	if err != nil {
		return func() tea.Msg { return editorClosedMsg{issueID: issueID, parentCommentID: parentCommentID, err: err} }
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()

	cmd := exec.Command(editorCommand(), tmpPath) // #nosec G204 -- editorCommand() is $VISUAL/$EDITOR/"vi", the user's own configured editor, run against a chit-created temp file
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorClosedMsg{tmpPath: tmpPath, issueID: issueID, parentCommentID: parentCommentID, err: err}
	})
}

// readAndCleanupTempFile reads back what the user wrote in $EDITOR and
// removes the temp file regardless of whether the read succeeded.
func readAndCleanupTempFile(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is a chit-created temp file from openEditorCmd, never user-controlled input
	_ = os.Remove(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
