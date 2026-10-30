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
		{"/Preview/" + token + "/articles/a", "/Preview/<redacted>/articles/a"},
		{"/PREVIEW/" + token, "/PREVIEW/<redacted>"},
		{"/articles/pReViEw/" + token, "/articles/pReViEw/<redacted>"},
		{"/preview/" + token + "/articles/preview/" + token, "/preview/<redacted>"},
		{"/preview/" + token + "/articles/a/Preview/" + token, "/preview/<redacted>"},
		{"/preview/" + token + "/articles/preview", "/preview/<redacted>/articles/preview"},
		{"/articles/essays/a", "/articles/essays/a"},
	} {
		got := redactPath(tc.path)
		if got != tc.want || strings.Contains(got, token) {
			t.Errorf("redactPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func FuzzRedactPath(f *testing.F) {
	for _, seed := range []struct{ prefix, segment, suffix string }{
		{"", "preview", "/articles/a"},
		{"", "Preview", "/diagnostics"},
		{"", "PREVIEW", ""},
		{"/articles", "pReViEw", "/articles/a"},
		{"/preview/x/articles", "preview", ""},
		{"/", "preview", "/articles/é"},
	} {
		f.Add(seed.prefix, seed.segment, seed.suffix)
	}
	const token = "S3CRET-TOKEN"
	f.Fuzz(func(t *testing.T, prefix, segment, suffix string) {
		if !strings.Contains(strings.ToLower(prefix+suffix), "/preview/") && prefix+suffix != "" {
			if got := redactPath(prefix + suffix); got != prefix+suffix {
				t.Fatalf("redactPath(%q) = %q, want the path unchanged", prefix+suffix, got)
			}
		}
		if !strings.EqualFold(segment, "preview") || strings.Contains(prefix+suffix, token) {
			return
		}
		p := prefix + "/" + segment + "/" + token + suffix
		if got := redactPath(p); strings.Contains(got, token) {
			t.Fatalf("redactPath(%q) = %q, which logs the token", p, got)
		}
	})
}
