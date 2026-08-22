package tui

import "testing"

func TestOpenURLRejectsNonHTTPScheme(t *testing.T) {
	for _, url := range []string{"javascript:alert(1)", "file:///etc/passwd", "-rf", ""} {
		if err := openURL(url); err == nil {
			t.Errorf("openURL(%q) = nil error, want rejection of a non-http(s) target", url)
		}
	}
}

func TestOpenURLAcceptsHTTPS(t *testing.T) {
	// "true" is a real, fast-exiting binary — this exercises the actual
	// exec.Command path without launching a real browser.
	t.Setenv("BROWSER", "true")
	if err := openURL("https://example.com"); err != nil {
		t.Errorf("openURL(valid https url) = %v, want nil", err)
	}
}
