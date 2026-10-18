package content

import (
	"bytes"
	"errors"
)

// ErrInvalidFrontmatter reports an opening +++ delimiter without a closing one.
var ErrInvalidFrontmatter = errors.New("invalid frontmatter")

const frontmatterDelimiter = "+++"

var byteOrderMark = []byte("\xef\xbb\xbf")

// SplitFrontmatter separates an article's TOML preamble from its Markdown body.
// A preamble starts when the first line, after an optional UTF-8 byte order
// mark, is exactly "+++" and ends at the next line that is exactly "+++".
// Lines may end in LF or CRLF. Without an opening delimiter there is no
// preamble and the body is the whole document. The returned slices alias data.
func SplitFrontmatter(data []byte) (frontmatter, body []byte, err error) {
	data = bytes.TrimPrefix(data, byteOrderMark)
	first, rest, hasNewline := cutLine(data)
	if string(first) != frontmatterDelimiter || !hasNewline {
		if string(first) == frontmatterDelimiter {
			return nil, nil, ErrInvalidFrontmatter
		}
		return nil, data, nil
	}
	for offset := 0; offset <= len(rest); {
		line, remainder, more := cutLine(rest[offset:])
		if string(line) == frontmatterDelimiter {
			return rest[:offset], remainder, nil
		}
		if !more {
			break
		}
		offset = len(rest) - len(remainder)
	}
	return nil, nil, ErrInvalidFrontmatter
}

// cutLine returns the first line without its LF or CRLF terminator.
func cutLine(data []byte) (line, rest []byte, found bool) {
	line, rest, found = bytes.Cut(data, []byte("\n"))
	return bytes.TrimSuffix(line, []byte("\r")), rest, found
}
