package api

import (
	"strings"
	"testing"
)

func TestRedactPathNeverLogsAToken(t *testing.T) {
	const token = "s3cret-token"
	for _, tc := range []struct{ path, want string }{
		{"/preview/" + token + "/articles/essays/a", "/preview/<redacted>/articles/essays/a"},
		{"/preview/" + token + "/articles/", "/preview/<redacted>/articles/"},
		{"/preview/" + token + "/diagnostics", "/preview/<redacted>/diagnostics"},
		{"/preview/" + token, "/preview/<redacted>"},
		{"/preview//" + token + "/articles/a", "/preview/<redacted>"},
		{"/preview/x/" + token + "/articles/a", "/preview/<redacted>"},
		{"/preview/" + token + "/diagnostics/" + token, "/preview/<redacted>"},
		{"//preview/" + token + "/articles/a", "/" + "/preview/<redacted>"},
		{"/articles/preview/" + token, "/articles/preview/<redacted>"},
		{"/articles/essays/a", "/articles/essays/a"},
	} {
		got := redactPath(tc.path)
		if got != tc.want || strings.Contains(got, token) {
			t.Errorf("redactPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
