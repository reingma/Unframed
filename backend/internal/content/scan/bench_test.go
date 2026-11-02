package scan

import (
	"context"
	"fmt"
	"testing"

	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
)

// BenchmarkScan scans a generated tree of 20 directories with 50 articles
// each, through the filesystem and in-memory sources, and in memory with
// every node at the alias cap.
func BenchmarkScan(b *testing.B) {
	files := map[string]string{".configuration": validRoot}
	for d := range 20 {
		dir := fmt.Sprintf("Section-%02d", d)
		files[dir+"/.configuration"] = fmt.Sprintf("order = %d\n[theme]\naccent = \"#228b22\"\nfont = \"serif\"\n", d)
		for a := range 50 {
			files[fmt.Sprintf("%s/Article-%02d.md", dir, a)] = "+++\ntitle = \"An article\"\ndescription = \"A summary\"\n[math]\nengine = \"katex\"\n+++\n# Heading\n\nBody text.\n"
		}
	}
	sources := map[string]content.Source{"fs": contenttest.NewFS(b, files), "memory": contenttest.MemSource(files)}
	sources["aliases"] = contenttest.MemSource(contenttest.WithMaxAliases(files))
	for name, source := range sources {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				manifest, err := Scan(context.Background(), source, quietLogger())
				if err != nil || len(manifest.Articles) != 1000 || name == "aliases" && len(manifest.Redirects) != 1020*content.MaxAliases {
					b.Fatal(err)
				}
			}
		})
	}
}
