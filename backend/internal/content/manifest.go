package content

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
	// Assets lists the non-Markdown files of valid directories, ordered by
	// Path. They are served at their tree path and are not part of the nav tree.
	Assets []File
	// Errors lists per-item validation errors, ordered by Path.
	Errors []ValidationError
}
