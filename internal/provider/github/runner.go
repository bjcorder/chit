package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes a `gh` invocation and returns its stdout. It exists so
// tests can substitute a fake instead of shelling out to the real `gh`
// binary — see github_test.go.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// execRunner shells out to the real `gh` binary. Arguments are passed as a
// argv slice via exec.CommandContext, never through a shell, so there is no
// shell-injection risk regardless of what a comment body or other argument
// contains.
type execRunner struct{}

func (execRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...) // #nosec G204 -- args is a fixed argv slice, never shell-interpreted; see type comment
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		// Some subcommands (`gh pr checks` in particular) use the exit code
		// to signal status, not command failure, and still print valid JSON
		// to stdout — so callers that care get stdout back alongside the
		// error rather than losing it.
		return stdout.Bytes(), fmt.Errorf("gh %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}
