package scan

import (
	"cmp"
	"fmt"
	"path"
	"slices"

	"github.com/reingma/Unframed/backend/internal/content"
)

// redirects validates the aliases of every valid node against the published
// nav tree and returns the redirect map of the published ones, ordered by
// source. A rejected alias is reported against the file that declares it and
// dropped; its node stays published. Drafts and directories omitted from the
// nav tree have their aliases checked but publish none.
//
// Every source is unpublished and every target published, and a published
// path's ancestors are published, so no target is at or below a source. A
// chain of redirects therefore consumes at least one segment of the
// requested path per hop and always ends; the reader resolves it per request.
func (s *scanner) redirects(root *content.Node) []content.Redirect {
	live := map[string]bool{}
	var walk func(*content.Node)
	walk = func(node *content.Node) {
		live[node.URLPath] = true
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)

	published := func(node *content.Node) bool { return !node.Draft && live[node.URLPath] }
	claims := map[content.Alias]int{}
	for _, node := range s.aliased {
		if published(node) {
			for _, alias := range node.Aliases {
				claims[alias]++
			}
		}
	}

	var redirects []content.Redirect
	for _, node := range s.aliased {
		isPublished := published(node)
		for _, alias := range node.Aliases {
			var err error
			switch {
			case alias.String() == node.URLPath:
				err = fmt.Errorf("%w: %s is the item's own path", ErrSelfAlias, alias)
			case isPublished && claims[alias] > 1:
				err = fmt.Errorf("%w: %s is claimed by more than one item", ErrDuplicateAlias, alias)
			case live[alias.String()]:
				err = fmt.Errorf("%w: %s is published; remove the stale alias", ErrAliasShadowsPath, alias)
			}
			if err != nil {
				s.report(aliasSource(node), err)
				continue
			}
			if isPublished {
				redirects = append(redirects, content.Redirect{From: alias, To: node.URLPath, Prefix: node.Kind == content.KindDirectory})
			}
		}
	}
	slices.SortFunc(redirects, func(a, b content.Redirect) int { return cmp.Compare(a.From.String(), b.From.String()) })
	return redirects
}

// aliasSource is the file that declares a node's aliases.
func aliasSource(node *content.Node) string {
	if node.Kind == content.KindDirectory {
		return path.Join(node.SourcePath, content.ConfigurationFile)
	}
	return node.SourcePath
}
