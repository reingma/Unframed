package content

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidName reports a directory or article name outside the slug-form grammar.
	ErrInvalidName = errors.New("invalid name")
	// ErrInvalidSlug reports a URL slug outside the lowercase slug-form grammar.
	ErrInvalidSlug = errors.New("invalid slug")
	// ErrInvalidAlias reports an alias that is not an internal URL path.
	ErrInvalidAlias = errors.New("invalid alias")
)

// Name is a source name in slug form: ASCII letter/digit segments joined by
// single hyphens, such as "On-Martingales". Its zero value is the unnamed root.
type Name struct{ raw string }

// ParseName accepts names matching ^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$ unchanged.
func ParseName(raw string) (Name, error) {
	if !isSlugForm(raw, true) {
		return Name{}, fmt.Errorf("%w %q: use letters, digits and single hyphens", ErrInvalidName, raw)
	}
	return Name{raw: raw}, nil
}

func (n Name) String() string { return n.raw }

// Title is the de-slugified display title: each hyphen becomes a space.
func (n Name) Title() string { return strings.ReplaceAll(n.raw, "-", " ") }

// Slug is the derived URL slug: the lowercased name.
func (n Name) Slug() Slug { return Slug{s: strings.ToLower(n.raw)} }

// Slug is one URL path segment: lowercase ASCII letter/digit segments joined
// by single hyphens. Its zero value means no slug (the root or a landing page).
type Slug struct{ s string }

// ParseSlug accepts slugs matching ^[a-z0-9]+(-[a-z0-9]+)*$ unchanged.
func ParseSlug(raw string) (Slug, error) {
	if !isSlugForm(raw, false) {
		return Slug{}, fmt.Errorf("%w %q: use lowercase letters, digits and single hyphens", ErrInvalidSlug, raw)
	}
	return Slug{s: raw}, nil
}

func (s Slug) String() string { return s.s }

// Alias is a former internal URL path of a node: "/" followed by slugs
// joined by "/", such as "/essays/old-title". It is never "/" and never has
// a scheme, host, query or trailing slash. Its zero value is no alias.
type Alias struct{ path string }

// ParseAlias accepts paths matching ^(/[a-z0-9]+(-[a-z0-9]+)*)+$ unchanged.
func ParseAlias(raw string) (Alias, error) {
	rest, ok := strings.CutPrefix(raw, "/")
	if !ok {
		return Alias{}, fmt.Errorf("%w %q: use an absolute path of lowercase slugs, such as /essays/old-title", ErrInvalidAlias, raw)
	}
	for segment := range strings.SplitSeq(rest, "/") {
		if !isSlugForm(segment, false) {
			return Alias{}, fmt.Errorf("%w %q: use an absolute path of lowercase slugs, such as /essays/old-title", ErrInvalidAlias, raw)
		}
	}
	return Alias{path: raw}, nil
}

func (a Alias) String() string { return a.path }

func isSlugForm(raw string, allowUpper bool) bool {
	if raw == "" {
		return false
	}
	previousHyphen := true
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c == '-':
			if previousHyphen {
				return false
			}
			previousHyphen = true
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', allowUpper && c >= 'A' && c <= 'Z':
			previousHyphen = false
		default:
			return false
		}
	}
	return !previousHyphen
}
