package perf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Large content tree shape: the 1,000-article tree that BenchmarkBuild
// measures, so the served workloads and the build benchmark share one size.
const (
	LargeTreeSections = 20
	LargeTreeArticles = 50
)

const largeTreeBody = "# Heading\n\nBody text with *emphasis*, `code` and a [link](/a).\n\n" +
	"| a | b |\n| - | - |\n| 1 | 2 |\n\n- [x] one\n- [ ] two\n\n```go\nfmt.Println(1)\n```\n"

// LargeTree returns the files of the large content tree keyed by
// slash-separated content path. Article URL paths are
// section-NN/article-NN.
func LargeTree() map[string]string {
	files := map[string]string{
		".configuration": "title = \"Unframed\"\n[theme]\naccent = \"#336699\"\nbackground = \"#FFFFFF\"\ntext = \"#111111\"\n",
	}
	for d := range LargeTreeSections {
		dir := fmt.Sprintf("Section-%02d", d)
		files[dir+"/.configuration"] = fmt.Sprintf("order = %d\n[theme]\naccent = \"#228b22\"\nfont = \"serif\"\n", d)
		for a := range LargeTreeArticles {
			files[fmt.Sprintf("%s/Article-%02d.md", dir, a)] =
				"+++\ntitle = \"An article\"\ndescription = \"A summary\"\n[math]\nengine = \"katex\"\n+++\n" + largeTreeBody
		}
	}
	return files
}

// WriteLargeTree writes LargeTree under dir. It refuses an existing non-empty
// dir so that it never mixes with, or overwrites, other content.
func WriteLargeTree(dir string) error {
	entries, err := os.ReadDir(dir)
	switch {
	case err == nil && len(entries) > 0:
		return fmt.Errorf("content directory %s is not empty", dir)
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return err
	}
	for name, data := range LargeTree() {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(data), 0o644); err != nil {
			return err
		}
	}
	return nil
}
