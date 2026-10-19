// Package render converts article Markdown to HTML.
package render

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// markdown renders CommonMark with the GitHub Flavored Markdown extensions:
// tables, strikethrough, autolinks and task lists. goldmark's safe default
// replaces raw HTML with a comment and drops links with dangerous schemes
// such as javascript:. A goldmark.Markdown is safe for concurrent use.
var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

// Render returns the HTML of a Markdown body.
func Render(body []byte) (string, error) {
	var out bytes.Buffer
	out.Grow(len(body) + len(body)/2)
	if err := markdown.Convert(body, &out); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return out.String(), nil
}
