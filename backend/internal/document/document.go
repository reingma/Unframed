// Package document defines the versioned JSON documents Go serves to the
// frontend: one envelope per article and the public manifest. The
// document contract in docs/specs/document-contract.md is their canonical
// description.
package document

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/reingma/Unframed/backend/internal/content"
)

// SchemaVersion is the version emitted on every document. Decoding any
// other version fails with ErrUnsupportedSchemaVersion.
const SchemaVersion = 1

var (
	// ErrUnsupportedSchemaVersion reports a missing or unknown schemaVersion.
	ErrUnsupportedSchemaVersion = errors.New("unsupported schema version")
	// ErrInvalidDocument reports a document that is not valid JSON or breaks
	// a rule of its schema version.
	ErrInvalidDocument = errors.New("invalid document")
)

// BodyKind names the representation of an envelope body.
type BodyKind string

// BodyHTML is a body of rendered HTML.
const BodyHTML BodyKind = "html"

// Envelope is one article as served: its identity, resolved configuration
// and rendered body.
type Envelope struct {
	SchemaVersion  int            `json:"schemaVersion"`
	Metadata       Metadata       `json:"metadata"`
	ResolvedConfig ResolvedConfig `json:"resolvedConfig"`
	Body           Body           `json:"body"`
}

// Metadata identifies and describes an article. It is never inherited.
type Metadata struct {
	URLPath     string `json:"urlPath"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Draft       bool   `json:"draft"`
	Landing     bool   `json:"landing"`
}

// ResolvedConfig is an article's configuration after the root → directory →
// article cascade.
type ResolvedConfig struct {
	Theme Theme `json:"theme"`
}

// Theme holds lowercase "#rrggbb" colors.
type Theme struct {
	Accent     string `json:"accent"`
	Background string `json:"background"`
	Text       string `json:"text"`
}

// Body is an article's rendered content.
type Body struct {
	Kind    BodyKind `json:"kind"`
	Content string   `json:"content"`
}

// NewEnvelope returns the envelope of article with its rendered HTML.
func NewEnvelope(article *content.Node, html string) Envelope {
	return Envelope{
		SchemaVersion: SchemaVersion,
		Metadata: Metadata{
			URLPath:     article.URLPath,
			Title:       article.Title,
			Description: article.Description,
			Draft:       article.Draft,
			Landing:     article.Landing,
		},
		ResolvedConfig: ResolvedConfig{Theme: newTheme(article.Theme)},
		Body:           Body{Kind: BodyHTML, Content: html},
	}
}

func newTheme(theme content.Theme) Theme {
	return Theme{Accent: theme.Accent.String(), Background: theme.Background.String(), Text: theme.Text.String()}
}

// NavKind distinguishes manifest directories from articles.
type NavKind string

const (
	NavDirectory NavKind = "directory"
	NavArticle   NavKind = "article"
)

// Manifest is the public site document: the site settings and the published
// navigation tree. It never contains drafts or validation errors.
type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	Site          Site    `json:"site"`
	Nav           NavNode `json:"nav"`
}

// Site is the root configuration.
type Site struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Theme       Theme  `json:"theme"`
}

// NavNode is one published directory or article. HasIndex marks a directory
// with a landing article at its URL; only directories have children.
type NavNode struct {
	Kind        NavKind   `json:"kind"`
	URLPath     string    `json:"urlPath"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	HasIndex    bool      `json:"hasIndex"`
	Children    []NavNode `json:"children"`
}

// NewManifest returns the public manifest of a scanned nav tree, which
// already excludes drafts and invalid items.
func NewManifest(root *content.Node) Manifest {
	return Manifest{
		SchemaVersion: SchemaVersion,
		Site:          Site{Title: root.Title, Description: root.Description, Theme: newTheme(root.Theme)},
		Nav:           newNavNode(root),
	}
}

func newNavNode(node *content.Node) NavNode {
	nav := NavNode{URLPath: node.URLPath, Title: node.Title, Description: node.Description, Children: []NavNode{}}
	if node.Kind == content.KindArticle {
		nav.Kind = NavArticle
		return nav
	}
	nav.Kind = NavDirectory
	nav.HasIndex = node.Index != nil
	for _, child := range node.Children {
		nav.Children = append(nav.Children, newNavNode(child))
	}
	return nav
}

// DecodeEnvelope parses an envelope, rejecting unknown versions and body
// kinds and breaks of the version-1 rules. Unknown members are ignored.
func DecodeEnvelope(data []byte) (Envelope, error) {
	var envelope Envelope
	if err := decode(data, &envelope); err != nil {
		return Envelope{}, err
	}
	var errs []error
	if envelope.Body.Kind != BodyHTML {
		errs = append(errs, fmt.Errorf("body.kind: unknown kind %q", envelope.Body.Kind))
	}
	errs = append(errs, checkURLPath("metadata.urlPath", envelope.Metadata.URLPath))
	errs = append(errs, checkTitle("metadata.title", envelope.Metadata.Title))
	errs = append(errs, checkTheme("resolvedConfig.theme", envelope.ResolvedConfig.Theme))
	if err := invalid(errs); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

// DecodeManifest parses a manifest, rejecting unknown versions and breaks of
// the version-1 rules. Unknown members are ignored.
func DecodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := decode(data, &manifest); err != nil {
		return Manifest{}, err
	}
	errs := []error{
		checkTitle("site.title", manifest.Site.Title),
		checkTheme("site.theme", manifest.Site.Theme),
	}
	if manifest.Nav.Kind != NavDirectory || manifest.Nav.URLPath != "/" {
		errs = append(errs, errors.New(`nav: must be the root directory at "/"`))
	}
	errs = append(errs, checkNav("nav", manifest.Nav))
	if err := invalid(errs); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// decode checks the schema version before decoding v, so a document of
// another version fails as unsupported rather than as malformed.
func decode(data []byte, v any) error {
	var probe struct {
		SchemaVersion *int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	if probe.SchemaVersion == nil {
		return fmt.Errorf("%w: schemaVersion is missing", ErrUnsupportedSchemaVersion)
	}
	if *probe.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: %d (supported: %d)", ErrUnsupportedSchemaVersion, *probe.SchemaVersion, SchemaVersion)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	return nil
}

func invalid(errs []error) error {
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	return nil
}

func checkURLPath(key, urlPath string) error {
	if !strings.HasPrefix(urlPath, "/") {
		return fmt.Errorf("%s: must start with /", key)
	}
	return nil
}

func checkTitle(key, title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("%s: must not be empty", key)
	}
	return nil
}

func checkTheme(key string, theme Theme) error {
	var errs []error
	for name, value := range map[string]string{"accent": theme.Accent, "background": theme.Background, "text": theme.Text} {
		if _, err := content.ParseColor(value); err != nil {
			errs = append(errs, fmt.Errorf("%s.%s: %w", key, name, err))
		}
	}
	return errors.Join(errs...)
}

func checkNav(key string, node NavNode) error {
	errs := []error{checkURLPath(key+".urlPath", node.URLPath), checkTitle(key+".title", node.Title)}
	switch node.Kind {
	case NavDirectory:
	case NavArticle:
		if node.HasIndex || len(node.Children) > 0 {
			errs = append(errs, fmt.Errorf("%s: an article has no index or children", key))
		}
	default:
		errs = append(errs, fmt.Errorf("%s.kind: unknown kind %q", key, node.Kind))
	}
	for i, child := range node.Children {
		errs = append(errs, checkNav(fmt.Sprintf("%s.children[%d]", key, i), child))
	}
	return errors.Join(errs...)
}
