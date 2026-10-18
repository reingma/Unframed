package content

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

// NodeKind distinguishes directories from articles.
type NodeKind uint8

const (
	KindDirectory NodeKind = iota + 1
	KindArticle
)

// Node is one valid directory or article of the content tree.
type Node struct {
	Kind NodeKind
	// SourcePath is the file or directory path in the Source; "." for the root.
	SourcePath string
	// Name is the source name without extensions; zero for the root.
	Name  Name
	Title string
	// Slug is zero for the root and for landing pages.
	Slug Slug
	// URLPath is the slugified tree path, such as "/mathematics/on-martingales".
	URLPath string
	Order   int64
	Ordered bool
	// Draft marks an "X.draft.md" article. Drafts are never in a nav tree.
	Draft bool
	// Landing marks an "index.md" article, its directory's landing content.
	Landing bool
	// Config is the resolved configuration after the root → directory → article cascade.
	Config Config
	// Index is a directory's published landing article, if any.
	Index *Node
	// Children are a directory's published nav entries, ordered by order
	// then title. Directories without published content are omitted.
	Children []*Node
}

// ValidationError is a per-item error. The item, and for a directory its
// whole subtree, is excluded from the manifest; other items are unaffected.
type ValidationError struct {
	Path string
	Err  error
}

func (e ValidationError) Error() string { return e.Path + ": " + e.Err.Error() }
func (e ValidationError) Unwrap() error { return e.Err }

// Manifest is the validated content tree, fully derived from its Source.
type Manifest struct {
	// Root is the public nav tree.
	Root *Node
	// Articles indexes every valid article, drafts included, by SourcePath.
	Articles []*Node
	// Errors lists per-item validation errors, ordered by Path.
	Errors []ValidationError
}

// Scan reads a content tree into a manifest and logs one summary record to
// logger, which must not be nil. Invalid items are excluded and
// reported in Manifest.Errors. Scan fails only when the tree cannot be listed,
// the context ends, or the root configuration is missing or invalid.
func Scan(ctx context.Context, source Source, logger *slog.Logger) (*Manifest, error) {
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
	slices.SortFunc(s.articles, func(a, b *Node) int { return cmp.Compare(a.SourcePath, b.SourcePath) })
	slices.SortStableFunc(s.errors, func(a, b ValidationError) int { return cmp.Compare(a.Path, b.Path) })
	drafts := 0
	for _, article := range s.articles {
		if article.Draft {
			drafts++
		}
	}
	logger.InfoContext(ctx, "content tree scanned",
		"files", len(files), "articles", len(s.articles), "drafts", drafts,
		"validation_errors", len(s.errors), "duration", time.Since(started))
	return &Manifest{Root: root, Articles: s.articles, Errors: s.errors}, nil
}

type treeDir struct {
	path      string
	base      string
	hasConfig bool
	dirs      map[string]*treeDir
	articles  []string
}

func newTreeDir(dirPath string) *treeDir {
	return &treeDir{path: dirPath, base: path.Base(dirPath), dirs: map[string]*treeDir{}}
}

// buildTree groups content files by directory. Files other than
// ".configuration" and Markdown articles are assets and are not indexed.
func buildTree(files []File) *treeDir {
	root := newTreeDir(".")
	for _, file := range files {
		if !isContentPath(file.Path) {
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
		case base == configurationFile:
			dir.hasConfig = true
		case strings.HasSuffix(base, articleExtension):
			dir.articles = append(dir.articles, file.Path)
		}
	}
	return root
}

type scanner struct {
	ctx      context.Context
	source   Source
	articles []*Node
	errors   []ValidationError
}

func (s *scanner) report(itemPath string, err error) {
	s.errors = append(s.errors, ValidationError{Path: itemPath, Err: err})
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

func (s *scanner) root(tree *treeDir) (*Node, error) {
	if !tree.hasConfig {
		return nil, fmt.Errorf("%w: missing %s", ErrInvalidRootConfig, configurationFile)
	}
	data, itemErr, fatal := s.read(configurationFile)
	if fatal != nil {
		return nil, fatal
	}
	if itemErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRootConfig, itemErr)
	}
	own, err := parseConfig(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRootConfig, err)
	}
	keys, err := parseNodeKeys(own)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRootConfig, err)
	}
	if keys.hasSlug || keys.hasOrder {
		return nil, fmt.Errorf("%w: the root cannot set %q or %q", ErrInvalidRootConfig, keySlug, keyOrder)
	}
	return &Node{
		Kind:       KindDirectory,
		SourcePath: ".",
		Title:      keys.title,
		URLPath:    "/",
		Config:     Cascade(nil, own),
	}, nil
}

type candidate struct {
	node *Node
	dir  *treeDir
}

func (s *scanner) fill(dir *Node, tree *treeDir) error {
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
		case node.Kind == KindDirectory:
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

func compareNav(a, b *Node) int {
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
func (s *scanner) prepareDirectory(parent *Node, tree *treeDir) (*Node, error) {
	name, err := ParseName(tree.base)
	if err != nil {
		s.report(tree.path, err)
		return nil, nil
	}
	own := Config{}
	if tree.hasConfig {
		configPath := path.Join(tree.path, configurationFile)
		data, itemErr, fatal := s.read(configPath)
		if fatal != nil {
			return nil, fatal
		}
		if itemErr == nil {
			own, itemErr = parseConfig(data)
		}
		if itemErr != nil {
			s.report(configPath, itemErr)
			return nil, nil
		}
	}
	keys, err := parseNodeKeys(own)
	if err != nil {
		s.report(path.Join(tree.path, configurationFile), err)
		return nil, nil
	}
	node := &Node{Kind: KindDirectory, SourcePath: tree.path, Name: name, Config: Cascade(parent.Config, own)}
	applyNodeKeys(node, name, keys)
	node.URLPath = joinURL(parent.URLPath, node.Slug)
	return node, nil
}

// prepareArticle validates an article's name and frontmatter. It returns nil
// after reporting a per-item error.
func (s *scanner) prepareArticle(parent *Node, articlePath string) (*Node, error) {
	base := path.Base(articlePath)
	stem, draft := strings.CutSuffix(base, draftExtension)
	if !draft {
		stem = strings.TrimSuffix(base, articleExtension)
	}
	name, err := ParseName(stem)
	if err != nil {
		s.report(articlePath, err)
		return nil, nil
	}
	data, itemErr, fatal := s.read(articlePath)
	if fatal != nil {
		return nil, fatal
	}
	var own Config
	if itemErr == nil {
		var frontmatter []byte
		frontmatter, _, itemErr = SplitFrontmatter(data)
		if itemErr == nil {
			own, itemErr = parseConfig(frontmatter)
		}
	}
	var keys nodeKeys
	if itemErr == nil {
		keys, itemErr = parseNodeKeys(own)
	}
	landing := name.Slug().String() == landingSlug
	if itemErr == nil && landing && (keys.hasSlug || keys.hasOrder) {
		itemErr = fmt.Errorf("%w: a landing page cannot set %q or %q", ErrInvalidConfig, keySlug, keyOrder)
	}
	if itemErr != nil {
		s.report(articlePath, itemErr)
		return nil, nil
	}
	node := &Node{
		Kind:       KindArticle,
		SourcePath: articlePath,
		Name:       name,
		Draft:      draft,
		Landing:    landing,
		Config:     Cascade(parent.Config, own),
	}
	if landing {
		node.Title = parent.Title
		if keys.hasTitle {
			node.Title = keys.title
		}
		node.URLPath = parent.URLPath
		return node, nil
	}
	applyNodeKeys(node, name, keys)
	node.URLPath = joinURL(parent.URLPath, node.Slug)
	return node, nil
}

func applyNodeKeys(node *Node, name Name, keys nodeKeys) {
	node.Title, node.Slug = name.Title(), name.Slug()
	if keys.hasTitle {
		node.Title = keys.title
	}
	if keys.hasSlug {
		node.Slug = keys.slug
	}
	node.Order, node.Ordered = keys.order, keys.hasOrder
}

func joinURL(parent string, slug Slug) string {
	if parent == "/" {
		return "/" + slug.String()
	}
	return parent + "/" + slug.String()
}
