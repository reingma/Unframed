package content

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkScan scans a generated tree of 20 directories with 50 articles
// each, through the filesystem and in-memory sources.
func BenchmarkScan(b *testing.B) {
	files := map[string]string{".configuration": "title = \"Site\"\n[theme]\naccent = \"blue\"\n"}
	for d := range 20 {
		dir := fmt.Sprintf("Section-%02d", d)
		files[dir+"/.configuration"] = fmt.Sprintf("order = %d\n[theme]\nfont = \"serif\"\n", d)
		for a := range 50 {
			files[fmt.Sprintf("%s/Article-%02d.md", dir, a)] = "+++\ntitle = \"An article\"\n[math]\nengine = \"katex\"\n+++\n# Heading\n\nBody text.\n"
		}
	}
	for name, source := range map[string]Source{"fs": newFSSource(b, files), "memory": memSource(files)} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				manifest, err := Scan(context.Background(), source, quietLogger())
				if err != nil || len(manifest.Articles) != 1000 {
					b.Fatal(err)
				}
			}
		})
	}
}
