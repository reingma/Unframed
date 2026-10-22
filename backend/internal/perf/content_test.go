package perf

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/reingma/Unframed/backend/internal/build"
	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/publish"
)

func TestWriteLargeTreeBuildsEveryArticle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "content")
	if err := WriteLargeTree(dir); err != nil {
		t.Fatal(err)
	}
	source, err := content.OpenFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	result, err := build.Build(context.Background(), source, publish.NewLocal(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if want := LargeTreeSections * LargeTreeArticles; len(result.Envelopes) != want || len(result.Content.Errors) != 0 {
		t.Fatalf("built %d envelopes with %d validation errors, want %d and none",
			len(result.Envelopes), len(result.Content.Errors), want)
	}
}

func TestWriteLargeTreeRefusesNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep.md")
	if err := os.WriteFile(keep, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteLargeTree(dir); err == nil {
		t.Fatal("WriteLargeTree wrote into a non-empty directory")
	}
	if data, _ := os.ReadFile(keep); string(data) != "mine" {
		t.Fatal("existing file changed")
	}
}
