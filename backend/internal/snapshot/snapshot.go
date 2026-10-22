// Package snapshot holds the immutable output of one successful build,
// encoded once for serving, and the store that swaps snapshots atomically.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/reingma/Unframed/backend/internal/build"
	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/document"
)

// Document is one pre-encoded JSON document and the SHA-256 hex digest of
// its bytes. JSON must not be modified.
type Document struct {
	JSON []byte
	Hash string
}

func newDocument(data []byte) Document {
	sum := sha256.Sum256(data)
	return Document{JSON: data, Hash: hex.EncodeToString(sum[:])}
}

// Diagnostic is a validation error of a build. URLPath is the URL the item
// would have from its source name, ignoring slug overrides.
type Diagnostic struct {
	Path    string `json:"path"`
	URLPath string `json:"urlPath"`
	Message string `json:"message"`
}

// Snapshot is the immutable output of one successful build. Every value it
// returns belongs to that build.
type Snapshot struct {
	buildID     string
	builtAt     time.Time
	manifest    Document
	envelopes   map[build.Key]Document
	diagnostics []Diagnostic
}

// New encodes the manifest and every envelope of result. Each encoded
// envelope is decoded back through document.DecodeEnvelope; one that fails,
// such as an unknown schemaVersion, is excluded and reported as a diagnostic
// instead of being served, and the manifest drops its links to excluded
// published envelopes. New fails only when the manifest does not encode to a
// valid document.
func New(result *build.Result, builtAt time.Time) (*Snapshot, error) {
	s := &Snapshot{
		buildID:   result.BuildID,
		builtAt:   builtAt,
		envelopes: make(map[build.Key]Document, len(result.Envelopes)),
	}
	sources := map[build.Key]string{}
	if result.Content != nil {
		for _, article := range result.Content.Articles {
			sources[build.Key{URLPath: article.URLPath, Draft: article.Draft}] = article.SourcePath
		}
		dirs := urlPaths(result.Content.Root)
		for _, validationErr := range result.Content.Errors {
			s.diagnostics = append(s.diagnostics, newDiagnostic(dirs, validationErr.Path, validationErr.Err.Error()))
		}
	}
	excluded := map[string]bool{}
	for key, envelope := range result.Envelopes {
		data, err := json.Marshal(envelope)
		if err == nil {
			_, err = document.DecodeEnvelope(data)
		}
		if err != nil {
			sourcePath := sources[key]
			if sourcePath == "" {
				sourcePath = key.URLPath
			}
			s.diagnostics = append(s.diagnostics, Diagnostic{Path: sourcePath, URLPath: key.URLPath, Message: err.Error()})
			if !key.Draft {
				excluded[key.URLPath] = true
			}
			continue
		}
		s.envelopes[key] = newDocument(data)
	}

	manifest := result.Manifest
	manifest.Nav = withoutExcluded(manifest.Nav, excluded)
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode manifest: %w", err)
	}
	if _, err := document.DecodeManifest(data); err != nil {
		return nil, fmt.Errorf("encode manifest: %w", err)
	}
	s.manifest = newDocument(data)
	return s, nil
}

// withoutExcluded returns a copy of nav that no longer links to the excluded
// published URLs: excluded articles are dropped and a directory whose index
// is excluded loses HasIndex.
func withoutExcluded(nav document.NavNode, excluded map[string]bool) document.NavNode {
	if nav.Kind != document.NavDirectory {
		return nav
	}
	if excluded[nav.URLPath] {
		nav.HasIndex = false
	}
	children := make([]document.NavNode, 0, len(nav.Children))
	for _, child := range nav.Children {
		if child.Kind == document.NavArticle && excluded[child.URLPath] {
			continue
		}
		children = append(children, withoutExcluded(child, excluded))
	}
	nav.Children = children
	return nav
}

// newDiagnostic derives an item's URL from its source path: hidden
// configuration files name their directory, extensions are dropped, an
// index names its directory, and names are lowercased.
// urlPaths maps the source path of every valid directory to its URL path.
func urlPaths(root *content.Node) map[string]string {
	paths := map[string]string{}
	var walk func(*content.Node)
	walk = func(n *content.Node) {
		if n == nil || n.Kind != content.KindDirectory {
			return
		}
		paths[n.SourcePath] = n.URLPath
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return paths
}

// newDiagnostic derives the URL an invalid item would have had: its nearest
// valid ancestor directory's URL, which honors slug overrides, followed by
// the lowercased remaining source names. The item's own slug override is not
// applied because its configuration may be what failed.
func newDiagnostic(dirs map[string]string, sourcePath, message string) Diagnostic {
	p := strings.TrimSuffix(sourcePath, content.ConfigurationFile)
	p = strings.TrimSuffix(p, "/")
	p = strings.TrimSuffix(strings.TrimSuffix(p, ".draft.md"), ".md")
	if path.Base(p) == "index" {
		p = path.Dir(p)
	}
	if p == "" {
		p = "."
	}
	ancestor, rest := p, ""
	for {
		if base, ok := dirs[ancestor]; ok {
			urlPath := strings.TrimSuffix(base, "/") + "/" + strings.ToLower(rest)
			if urlPath != "/" {
				urlPath = strings.TrimSuffix(urlPath, "/")
			}
			return Diagnostic{Path: sourcePath, URLPath: urlPath, Message: message}
		}
		if ancestor == "." {
			break
		}
		rest = path.Join(path.Base(ancestor), rest)
		ancestor = path.Dir(ancestor)
	}
	return Diagnostic{Path: sourcePath, URLPath: "/" + strings.ToLower(strings.TrimPrefix(p, ".")), Message: message}
}

func (s *Snapshot) BuildID() string    { return s.buildID }
func (s *Snapshot) BuiltAt() time.Time { return s.builtAt }

// Manifest returns the public manifest.
func (s *Snapshot) Manifest() Document { return s.manifest }

// Envelope returns the envelope at urlPath, the draft one when draft is set.
func (s *Snapshot) Envelope(urlPath string, draft bool) (Document, bool) {
	d, ok := s.envelopes[build.Key{URLPath: urlPath, Draft: draft}]
	return d, ok
}

// Envelopes is the number of servable envelopes, drafts included.
func (s *Snapshot) Envelopes() int { return len(s.envelopes) }

// Diagnostics lists the build's validation errors. The slice must not be modified.
func (s *Snapshot) Diagnostics() []Diagnostic { return slices.Clone(s.diagnostics) }

// DiagnosticsFor lists the validation errors of the item at urlPath and of
// its ancestor directories, whose errors exclude their whole subtree.
func (s *Snapshot) DiagnosticsFor(urlPath string) []Diagnostic {
	var matched []Diagnostic
	for _, d := range s.diagnostics {
		if d.URLPath == urlPath || strings.HasPrefix(urlPath, d.URLPath+"/") {
			matched = append(matched, d)
		}
	}
	return matched
}

// Store holds the current snapshot. Loads and swaps are atomic, so a reader
// sees one whole snapshot. Its zero value holds none.
type Store struct {
	current atomic.Pointer[Snapshot]
}

// Current returns the serving snapshot, or nil before the first build.
func (s *Store) Current() *Snapshot { return s.current.Load() }

// Swap makes next the serving snapshot.
func (s *Store) Swap(next *Snapshot) { s.current.Store(next) }
