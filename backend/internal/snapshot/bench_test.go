package snapshot_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/reingma/Unframed/backend/internal/build"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
	"github.com/reingma/Unframed/backend/internal/publish"
	"github.com/reingma/Unframed/backend/internal/snapshot"
)

// BenchmarkSnapshot measures the build-side cost of encoding, hashing and
// validating the documents of build's 1,000-article BenchmarkBuild tree.
func BenchmarkSnapshot(b *testing.B) {
	body := "# Heading\n\nBody text with *emphasis*, `code` and a [link](/a).\n\n" +
		"| a | b |\n| - | - |\n| 1 | 2 |\n\n- [x] one\n- [ ] two\n\n```go\nfmt.Println(1)\n```\n"
	files := contenttest.MemSource{".configuration": tree[".configuration"]}
	for d := range 20 {
		dir := fmt.Sprintf("Section-%02d", d)
		files[dir+"/.configuration"] = fmt.Sprintf("order = %d\n", d)
		for a := range 50 {
			files[fmt.Sprintf("%s/Article-%02d.md", dir, a)] = "+++\ntitle = \"An article\"\n+++\n" + body
		}
	}
	result, err := build.Build(context.Background(), files, publish.NewLocal(), slog.New(slog.DiscardHandler))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		s, err := snapshot.New(result, time.Now())
		if err != nil || s.Envelopes() != 1000 {
			b.Fatal(err)
		}
	}
}
