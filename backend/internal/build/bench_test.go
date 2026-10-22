package build_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/reingma/Unframed/backend/internal/build"
	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
	"github.com/reingma/Unframed/backend/internal/publish"
	"github.com/reingma/Unframed/backend/internal/snapshot"
	"github.com/reingma/Unframed/backend/internal/subscriber/logs"
)

// BenchmarkBuild measures one complete content build of the 1,000-article
// tree of scan's BenchmarkScan with a longer article body: build.Build
// followed by snapshot.New, which is the work every rebuild performs before
// the result is served.
func BenchmarkBuild(b *testing.B) {
	body := "# Heading\n\nBody text with *emphasis*, `code` and a [link](/a).\n\n" +
		"| a | b |\n| - | - |\n| 1 | 2 |\n\n- [x] one\n- [ ] two\n\n```go\nfmt.Println(1)\n```\n"
	files := map[string]string{".configuration": "title = \"Unframed\"\n[theme]\naccent = \"#336699\"\nbackground = \"#FFFFFF\"\ntext = \"#111111\"\n"}
	for d := range 20 {
		dir := fmt.Sprintf("Section-%02d", d)
		files[dir+"/.configuration"] = fmt.Sprintf("order = %d\n[theme]\naccent = \"#228b22\"\nfont = \"serif\"\n", d)
		for a := range 50 {
			files[fmt.Sprintf("%s/Article-%02d.md", dir, a)] =
				"+++\ntitle = \"An article\"\ndescription = \"A summary\"\n[math]\nengine = \"katex\"\n+++\n" + body
		}
	}
	logger := slog.New(slog.DiscardHandler)
	publisher := publish.NewLocal(logs.Subscriber{Logger: logger})
	for name, source := range map[string]content.Source{
		"fs":     contenttest.NewFS(b, files),
		"memory": contenttest.MemSource(files),
	} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				result, err := build.Build(context.Background(), source, publisher, logger)
				if err != nil {
					b.Fatal(err)
				}
				s, err := snapshot.New(result, time.Now())
				if err != nil || s.Envelopes() != 1000 {
					b.Fatal(err)
				}
			}
		})
	}
}
