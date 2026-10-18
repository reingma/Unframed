package build

import (
	"context"
	"fmt"
	"testing"

	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
	"github.com/reingma/Unframed/backend/internal/event"
)

// BenchmarkBuild builds the 1,000-article tree of scan's BenchmarkScan with
// a longer article body, so the difference is rendering and publishing.
func BenchmarkBuild(b *testing.B) {
	body := "# Heading\n\nBody text with *emphasis*, `code` and a [link](/a).\n\n" +
		"| a | b |\n| - | - |\n| 1 | 2 |\n\n- [x] one\n- [ ] two\n\n```go\nfmt.Println(1)\n```\n"
	files := map[string]string{".configuration": validRoot}
	for d := range 20 {
		dir := fmt.Sprintf("Section-%02d", d)
		files[dir+"/.configuration"] = fmt.Sprintf("order = %d\n[theme]\naccent = \"#228b22\"\nfont = \"serif\"\n", d)
		for a := range 50 {
			files[fmt.Sprintf("%s/Article-%02d.md", dir, a)] =
				"+++\ntitle = \"An article\"\ndescription = \"A summary\"\n[math]\nengine = \"katex\"\n+++\n" + body
		}
	}
	publisher := event.LogPublisher{Logger: quiet()}
	for name, source := range map[string]content.Source{
		"fs":     contenttest.NewFS(b, files),
		"memory": contenttest.MemSource(files),
	} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				result, err := Build(context.Background(), source, publisher, quiet())
				if err != nil || len(result.Envelopes) != 1000 {
					b.Fatal(err)
				}
			}
		})
	}
}
