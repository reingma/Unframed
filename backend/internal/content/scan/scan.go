// Package scan reads a content tree from a content.Source into a validated
// content.Manifest.
package scan

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/reingma/Unframed/backend/internal/content"
)

var (
	// ErrInvalidRootConfig is the fatal error for a missing or invalid root ".configuration".
	ErrInvalidRootConfig = errors.New("invalid root configuration")
	// ErrSlugCollision reports siblings that resolve to the same URL.
	ErrSlugCollision = errors.New("slug collision")
	// ErrUnreadable reports a content file the source could not read.
	ErrUnreadable = errors.New("unreadable content file")
)

const (
	articleExtension = ".md"
	draftExtension   = ".draft.md"
	landingSlug      = "index"
)

// Scan reads a content tree into a manifest and logs one summary record to
// logger, which must not be nil. Invalid items are excluded and
// reported in content.Manifest.Errors. Scan fails only when the tree cannot be listed,
// the context ends, or the root configuration is missing or invalid.
func Scan(ctx context.Context, source content.Source, logger *slog.Logger) (*content.Manifest, error) {
	started := time.Now()
	files, err := source.Files(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan content tree: %w", err)
	}
	tree := buildTree(files)
	s := &scanner{ctx: ctx, source: source}
	root, err := s.root(tree)
	if err != nil {
		return nil, err
	}
	if err := s.fill(root, tree); err != nil {
		return nil, err
	}
	slices.SortFunc(s.articles, func(a, b *content.Node) int { return cmp.Compare(a.SourcePath, b.SourcePath) })
	slices.SortFunc(s.assets, func(a, b content.File) int { return cmp.Compare(a.Path, b.Path) })
	slices.SortStableFunc(s.errors, func(a, b content.ValidationError) int { return cmp.Compare(a.Path, b.Path) })
	drafts := 0
	for _, article := range s.articles {
		if article.Draft {
			drafts++
		}
	}
	logger.InfoContext(ctx, "content tree scanned",
		"files", len(files), "articles", len(s.articles), "drafts", drafts, "assets", len(s.assets),
		"validation_errors", len(s.errors), "duration", time.Since(started))
	return &content.Manifest{Root: root, Articles: s.articles, Assets: s.assets, Errors: s.errors}, nil
}

type treeDir struct {
	path      string
	base      string
	hasConfig bool
	dirs      map[string]*treeDir
	articles  []string
	assets    []content.File
}

func newTreeDir(dirPath string) *treeDir {
	return &treeDir{path: dirPath, base: path.Base(dirPath), dirs: map[string]*treeDir{}}
}

// buildTree groups content files by directory. Files other than
// ".configuration" and Markdown articles are assets.
func buildTree(files []content.File) *treeDir {
	root := newTreeDir(".")
	for _, file := range files {
		if !content.IsContentPath(file.Path) {
			continue
		}
		dirPath, base := path.Split(file.Path)
		dir := root
		if dirPath != "" {
			for segment := range strings.SplitSeq(strings.TrimSuffix(dirPath, "/"), "/") {
				child, ok := dir.dirs[segment]
				if !ok {
					child = newTreeDir(path.Join(dir.path, segment))
					dir.dirs[segment] = child
				}
				dir = child
			}
		}
		switch {
		case base == content.ConfigurationFile:
			dir.hasConfig = true
		case strings.HasSuffix(base, articleExtension):
			dir.articles = append(dir.articles, file.Path)
		default:
			dir.assets = append(dir.assets, file)
		}
	}
	return root
}

type scanner struct {
	ctx      context.Context
	source   content.Source
	articles []*content.Node
	assets   []content.File
	errors   []content.ValidationError
}

func (s *scanner) report(itemPath string, err error) {
	s.errors = append(s.errors, content.ValidationError{Path: itemPath, Err: err})
}

// read returns the file contents, or a per-item error. A non-nil fatal error
// means the context ended.
func (s *scanner) read(name string) (data []byte, itemErr, fatal error) {
	data, err := s.source.ReadFile(s.ctx, name)
	if err != nil {
		if ctxErr := s.ctx.Err(); ctxErr != nil {
			return nil, nil, ctxErr
		}
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err), nil
	}
	return data, nil, nil
}

func (s *scanner) root(tree *treeDir) (*content.Node, error) {
	if !tree.hasConfig {
		return nil, fmt.Errorf("%w: missing %s", ErrInvalidRootConfig, content.ConfigurationFile)
	}
	data, itemErr, fatal := s.read(content.ConfigurationFile)
	if fatal != nil {
		return nil, fatal
	}
	if itemErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRootConfig, itemErr)
	}
	own, err := content.ParseConfig(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRootConfig, err)
	}
	keys, err := content.ParseNodeKeys(own)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRootConfig, err)
	}
	if keys.HasSlug || keys.HasOrder {
		return nil, fmt.Errorf("%w: the root cannot set %q or %q", ErrInvalidRootConfig, content.KeySlug, content.KeyOrder)
	}
	return &content.Node{
		Kind:       content.KindDirectory,
		SourcePath: ".",
		Title:      keys.Title,
		URLPath:    "/",
		Config:     content.Cascade(nil, own),
	}, nil
}

type candidate struct {
	node *content.Node
	dir  *treeDir
}

func (s *scanner) fill(dir *content.Node, tree *treeDir) error {
	s.assets = append(s.assets, tree.assets...)
	var candidates []candidate
	for _, segment := range slices.Sorted(maps.Keys(tree.dirs)) {
		node, err := s.prepareDirectory(dir, tree.dirs[segment])
		if err != nil {
			return err
		}
		if node != nil {
			candidates = append(candidates, candidate{node: node, dir: tree.dirs[segment]})
		}
	}
	slices.Sort(tree.articles)
	for _, articlePath := range tree.articles {
		node, err := s.prepareArticle(dir, articlePath)
		if err != nil {
			return err
		}
		if node != nil {
			candidates = append(candidates, candidate{node: node})
		}
	}

	for _, c := range s.withoutCollisions(candidates) {
		node := c.node
		switch {
		case node.Kind == content.KindDirectory:
			if err := s.fill(node, c.dir); err != nil {
				return err
			}
			if node.Index != nil || len(node.Children) > 0 {
				dir.Children = append(dir.Children, node)
			}
		case node.Draft:
			s.articles = append(s.articles, node)
		case node.Landing:
			s.articles = append(s.articles, node)
			dir.Index = node
		default:
			s.articles = append(s.articles, node)
			dir.Children = append(dir.Children, node)
		}
	}
	slices.SortFunc(dir.Children, compareNav)
	return nil
}

// withoutCollisions reports and drops every sibling sharing a URL with another
// sibling of the same publication state.
func (s *scanner) withoutCollisions(candidates []candidate) []candidate {
	type key struct {
		draft bool
		url   string
	}
	counts := map[key]int{}
	for _, c := range candidates {
		counts[key{c.node.Draft, c.node.URLPath}]++
	}
	kept := candidates[:0]
	for _, c := range candidates {
		if counts[key{c.node.Draft, c.node.URLPath}] > 1 {
			s.report(c.node.SourcePath, fmt.Errorf("%w: %s is used by more than one sibling", ErrSlugCollision, c.node.URLPath))
			continue
		}
		kept = append(kept, c)
	}
	return kept
}

func compareNav(a, b *content.Node) int {
	if a.Ordered != b.Ordered {
		if a.Ordered {
			return -1
		}
		return 1
	}
	if a.Ordered && a.Order != b.Order {
		return cmp.Compare(a.Order, b.Order)
	}
	if c := cmp.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)); c != 0 {
		return c
	}
	return cmp.Compare(a.Slug.String(), b.Slug.String())
}

// prepareDirectory validates a directory's name and configuration. It returns
// nil after reporting a per-item error, which excludes the whole subtree.
func (s *scanner) prepareDirectory(parent *content.Node, tree *treeDir) (*content.Node, error) {
	name, err := content.ParseName(tree.base)
	if err != nil {
		s.report(tree.path, err)
		return nil, nil
	}
	own := content.Config{}
	if tree.hasConfig {
		configPath := path.Join(tree.path, content.ConfigurationFile)
		data, itemErr, fatal := s.read(configPath)
		if fatal != nil {
			return nil, fatal
		}
		if itemErr == nil {
			own, itemErr = content.ParseConfig(data)
		}
		if itemErr != nil {
			s.report(configPath, itemErr)
			return nil, nil
		}
	}
	keys, err := content.ParseNodeKeys(own)
	if err != nil {
		s.report(path.Join(tree.path, content.ConfigurationFile), err)
		return nil, nil
	}
	node := &content.Node{Kind: content.KindDirectory, SourcePath: tree.path, Name: name, Config: content.Cascade(parent.Config, own)}
	applyNodeKeys(node, name, keys)
	node.URLPath = joinURL(parent.URLPath, node.Slug)
	return node, nil
}

// prepareArticle validates an article's name and frontmatter. It returns nil
// after reporting a per-item error.
func (s *scanner) prepareArticle(parent *content.Node, articlePath string) (*content.Node, error) {
	base := path.Base(articlePath)
	stem, draft := strings.CutSuffix(base, draftExtension)
	if !draft {
		stem = strings.TrimSuffix(base, articleExtension)
	}
	name, err := content.ParseName(stem)
	if err != nil {
		s.report(articlePath, err)
		return nil, nil
	}
	data, itemErr, fatal := s.read(articlePath)
	if fatal != nil {
		return nil, fatal
	}
	var own content.Config
	if itemErr == nil {
		var frontmatter []byte
		frontmatter, _, itemErr = content.SplitFrontmatter(data)
		if itemErr == nil {
			own, itemErr = content.ParseConfig(frontmatter)
		}
	}
	var keys content.NodeKeys
	if itemErr == nil {
		keys, itemErr = content.ParseNodeKeys(own)
	}
	landing := name.Slug().String() == landingSlug
	if itemErr == nil && landing && (keys.HasSlug || keys.HasOrder) {
		itemErr = fmt.Errorf("%w: a landing page cannot set %q or %q", content.ErrInvalidConfig, content.KeySlug, content.KeyOrder)
	}
	if itemErr != nil {
		s.report(articlePath, itemErr)
		return nil, nil
	}
	node := &content.Node{
		Kind:       content.KindArticle,
		SourcePath: articlePath,
		Name:       name,
		Draft:      draft,
		Landing:    landing,
		Config:     content.Cascade(parent.Config, own),
	}
	if landing {
		node.Title = parent.Title
		if keys.HasTitle {
			node.Title = keys.Title
		}
		node.URLPath = parent.URLPath
		return node, nil
	}
	applyNodeKeys(node, name, keys)
	node.URLPath = joinURL(parent.URLPath, node.Slug)
	return node, nil
}

func applyNodeKeys(node *content.Node, name content.Name, keys content.NodeKeys) {
	node.Title, node.Slug = name.Title(), name.Slug()
	if keys.HasTitle {
		node.Title = keys.Title
	}
	if keys.HasSlug {
		node.Slug = keys.Slug
	}
	node.Order, node.Ordered = keys.Order, keys.HasOrder
}

func joinURL(parent string, slug content.Slug) string {
	if parent == "/" {
		return "/" + slug.String()
	}
	return parent + "/" + slug.String()
}
