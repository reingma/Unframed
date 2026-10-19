package render

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite expected HTML fixtures")

// TestRenderFixtures renders each testdata/*.md fixture and compares it with
// the expected testdata/*.html.
func TestRenderFixtures(t *testing.T) {
	sources, err := filepath.Glob("testdata/*.md")
	if err != nil || len(sources) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	for _, source := range sources {
		name := strings.TrimSuffix(filepath.Base(source), ".md")
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Render(body)
			if err != nil {
				t.Fatal(err)
			}
			expected := strings.TrimSuffix(source, ".md") + ".html"
			if *update {
				if err := os.WriteFile(expected, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(expected)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("HTML mismatch\n got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestRenderOmitsUnsafeContent protects the safe-by-default rule
// independently of fixture contents.
func TestRenderOmitsUnsafeContent(t *testing.T) {
	got, err := Render([]byte("<script>alert(1)</script>\n\n[x](javascript:alert(1)) <img src=x onerror=alert(1)>\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, unsafe := range []string{"<script", "javascript:", "<img", "onerror"} {
		if strings.Contains(got, unsafe) {
			t.Errorf("rendered HTML contains %q:\n%s", unsafe, got)
		}
	}
}

func FuzzRender(f *testing.F) {
	for _, seed := range []string{"", "# a", "<div>x</div>", "| a |\n| - |\n| b |", "[x](javascript:y)", "```\n<b>\n```"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		got, err := Render(body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(got), "<script") {
			t.Fatalf("raw script survived rendering: %q", got)
		}
	})
}
