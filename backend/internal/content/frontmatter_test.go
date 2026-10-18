package content

import (
	"bytes"
	"errors"
	"testing"
)

func TestSplitFrontmatter(t *testing.T) {
	cases := []struct {
		name, input, front, body string
	}{
		{"preamble", "+++\ntitle = \"A\"\n+++\n# Body\n", "title = \"A\"\n", "# Body\n"},
		{"crlf", "+++\r\ntitle = \"A\"\r\n+++\r\nBody", "title = \"A\"\r\n", "Body"},
		{"byte order mark", "\xef\xbb\xbf+++\na = 1\n+++\nBody", "a = 1\n", "Body"},
		{"empty preamble", "+++\n+++\nBody", "", "Body"},
		{"closing at end of file", "+++\na = 1\n+++", "a = 1\n", ""},
		{"no preamble", "# Title\n+++\n", "", "# Title\n+++\n"},
		{"empty document", "", "", ""},
		{"delimiter must be exact", "+++ \na = 1\n+++\n", "", "+++ \na = 1\n+++\n"},
		{"body keeps later delimiters", "+++\na = 1\n+++\ntext\n+++\n", "a = 1\n", "text\n+++\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			front, body, err := SplitFrontmatter([]byte(c.input))
			if err != nil || string(front) != c.front || string(body) != c.body {
				t.Fatalf("SplitFrontmatter = %q, %q, %v; want %q, %q", front, body, err, c.front, c.body)
			}
		})
	}
	for _, input := range []string{"+++", "+++\n", "+++\na = 1\n", "+++\r\na = 1\r\n++++\r\n"} {
		if _, _, err := SplitFrontmatter([]byte(input)); !errors.Is(err, ErrInvalidFrontmatter) {
			t.Errorf("SplitFrontmatter(%q) = %v; want ErrInvalidFrontmatter", input, err)
		}
	}
}

// FuzzSplitFrontmatter composes documents from a preamble and a body and
// checks that splitting recovers both; it also checks arbitrary documents
// never lose or invent body bytes.
func FuzzSplitFrontmatter(f *testing.F) {
	f.Add([]byte("title = \"A\"\n"), []byte("# Body\n+++\n"))
	f.Add([]byte(""), []byte(""))
	f.Add([]byte("a = 1\r\n"), []byte("\r\n"))
	f.Fuzz(func(t *testing.T, front, body []byte) {
		if !containsDelimiterLine(front) && (len(front) == 0 || front[len(front)-1] == '\n') {
			document := append(append(append([]byte("+++\n"), front...), "+++\n"...), body...)
			gotFront, gotBody, err := SplitFrontmatter(document)
			if err != nil || !bytes.Equal(gotFront, front) || !bytes.Equal(gotBody, body) {
				t.Fatalf("split(%q) = %q, %q, %v", document, gotFront, gotBody, err)
			}
		}
		raw := append(append([]byte(nil), front...), body...)
		_, gotBody, err := SplitFrontmatter(raw)
		if err == nil && !bytes.HasSuffix(raw, gotBody) {
			t.Fatalf("body %q is not a suffix of %q", gotBody, raw)
		}
	})
}

func containsDelimiterLine(data []byte) bool {
	for line := range bytes.Lines(data) {
		if string(bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))) == frontmatterDelimiter {
			return true
		}
	}
	return false
}
