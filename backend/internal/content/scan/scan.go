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
	// ErrSelfAlias reports an alias equal to its own node's URL path.
	ErrSelfAlias = errors.New("self alias")
	// ErrDuplicateAlias reports an alias claimed by more than one node.
	ErrDuplicateAlias = errors.New("duplicate alias")
	// ErrAliasShadowsPath reports an alias equal to a published URL path.
	ErrAliasShadowsPath = errors.New("alias shadows a published path")
	// ErrRedirectLoop reports aliases that redirect back to themselves.
	ErrRedirectLoop = errors.New("redirect loop")
)

const (
	articleExtension = ".md"
	draftExtension   = ".draft.md"
	landingSlug      = "index"
)

// Scan reads a content tree into a manifest and logs one summary record to
// logger, which must not be nil. Invalid items are excluded and reported in
// content.Manifest.Errors. Scan fails only when the tree cannot be listed, the
// context ends, or the root configuration is missing or invalid.
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
	redirects := s.redirects(root)
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
		"redirects", len(redirects), "validation_errors", len(s.errors), "duration", time.Since(started))
	return &content.Manifest{Root: root, Articles: s.articles, Assets: s.assets, Redirects: redirects, Errors: s.errors}, nil
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
	// aliased lists every valid node with aliases, published or not.
	aliased []*content.Node
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

// readConfig returns a directory's ".configuration", or nil when it has none.
func (s *scanner) readConfig(tree *treeDir) (data []byte, itemErr, fatal error) {
	if !tree.hasConfig {
		return nil, nil, nil
	}
	return s.read(path.Join(tree.path, content.ConfigurationFile))
}

// root validates the site configuration. Its required fields make a missing
// or empty root ".configuration" invalid, which is fatal because every node
// inherits the site theme.
func (s *scanner) root(tree *treeDir) (*content.Node, error) {
	data, itemErr, fatal := s.readConfig(tree)
	if fatal != nil {
		return nil, fatal
	}
	var config content.Config
	if itemErr == nil {
		config, itemErr = content.ParseConfig(content.LevelSite, data)
	}
	if itemErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRootConfig, content.ValidationError{Path: content.ConfigurationFile, Err: itemErr})
	}
	return &content.Node{
		Kind:        content.KindDirectory,
		SourcePath:  ".",
		Title:       config.Title,
		Description: config.Description,
		URLPath:     "/",
		Theme:       config.Theme,
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
		if len(node.Aliases) > 0 {
			s.aliased = append(s.aliased, node)
		}
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
	data, itemErr, fatal := s.readConfig(tree)
	if fatal != nil {
		return nil, fatal
	}
	var config content.Config
	if itemErr == nil {
		config, itemErr = content.ParseConfig(content.LevelDirectory, data)
	}
	if itemErr != nil {
		s.report(path.Join(tree.path, content.ConfigurationFile), itemErr)
		return nil, nil
	}
	node := &content.Node{Kind: content.KindDirectory, SourcePath: tree.path, Name: name}
	applyConfig(node, parent, name, config)
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
	landing := name.Slug().String() == landingSlug
	level := content.LevelArticle
	if landing {
		level = content.LevelLanding
	}
	var config content.Config
	var body []byte
	if itemErr == nil {
		var frontmatter []byte
		frontmatter, body, itemErr = content.SplitFrontmatter(data)
		if itemErr == nil {
			config, itemErr = content.ParseConfig(level, frontmatter)
		}
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
		Body:       body,
	}
	if landing {
		node.Title = parent.Title
		if config.Title != "" {
			node.Title = config.Title
		}
		node.Description = config.Description
		node.URLPath = parent.URLPath
		node.Theme = parent.Theme.Override(config.Theme)
		return node, nil
	}
	applyConfig(node, parent, name, config)
	return node, nil
}

// applyConfig derives a non-landing node's display fields from its name and
// own configuration, and resolves its theme from its parent's.
func applyConfig(node, parent *content.Node, name content.Name, config content.Config) {
	node.Title, node.Slug = name.Title(), name.Slug()
	if config.Title != "" {
		node.Title = config.Title
	}
	if config.Slug != (content.Slug{}) {
		node.Slug = config.Slug
	}
	node.Description = config.Description
	node.Order, node.Ordered = config.Order, config.Ordered
	node.Aliases = config.Aliases
	node.Theme = parent.Theme.Override(config.Theme)
	node.URLPath = joinURL(parent.URLPath, node.Slug)
}

func joinURL(parent string, slug content.Slug) string {
	if parent == "/" {
		return "/" + slug.String()
	}
	return parent + "/" + slug.String()
}
