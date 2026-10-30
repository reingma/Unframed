package scan

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/reingma/Unframed/backend/internal/content"
)

// redirects validates the aliases of every valid node against the published
// nav tree and returns the redirect map of the published ones. A rejected
// alias is reported against the file that declares it and dropped; its node
// stays published. Drafts and directories omitted from the nav tree have
// their aliases checked but publish none.
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
	claims := map[string]int{}
	for _, node := range s.aliased {
		if published(node) {
			for _, alias := range node.Aliases {
				claims[alias.String()]++
			}
		}
	}
	// next maps each uniquely claimed alias to its node's URL path, the hop
	// a redirect would take, so following it detects loops.
	next := map[string]string{}
	for _, node := range s.aliased {
		if published(node) {
			for _, alias := range node.Aliases {
				if claims[alias.String()] == 1 && alias.String() != node.URLPath {
					next[alias.String()] = node.URLPath
				}
			}
		}
	}

	var redirects []content.Redirect
	var sources []*content.Node
	for _, node := range s.aliased {
		isPublished := published(node)
		for _, alias := range node.Aliases {
			from := alias.String()
			var err error
			switch {
			case from == node.URLPath:
				err = fmt.Errorf("%w: %s is the item's own path", ErrSelfAlias, from)
			case isPublished && claims[from] > 1:
				err = fmt.Errorf("%w: %s is claimed by more than one item", ErrDuplicateAlias, from)
			case isPublished && loops(next, from):
				err = fmt.Errorf("%w: %s redirects back to itself", ErrRedirectLoop, from)
			case live[from]:
				err = fmt.Errorf("%w: %s is published; remove the stale alias", ErrAliasShadowsPath, from)
			}
			if err != nil {
				s.report(aliasSource(node), err)
				continue
			}
			if isPublished {
				redirects = append(redirects, content.Redirect{From: from, To: node.URLPath, Prefix: node.Kind == content.KindDirectory})
				sources = append(sources, node)
			}
		}
	}
	reentrant := reentrantPrefixes(redirects)
	kept := redirects[:0]
	for i, r := range redirects {
		if reentrant[i] {
			s.report(aliasSource(sources[i]), fmt.Errorf("%w: %s redirects into a path it redirects again", ErrRedirectLoop, r.From))
			continue
		}
		kept = append(kept, r)
	}
	return resolveChains(kept)
}

// reentrantPrefixes marks prefix redirects that can apply again to their own
// result. Prefix redirect p leads to q when q.From is below p.To, since a
// suffix of p can then match q; a cycle of such steps, including a directory
// alias below its own path, would chain forever.
func reentrantPrefixes(redirects []content.Redirect) []bool {
	leads := func(p, q content.Redirect) bool {
		return p.Prefix && q.Prefix && strings.HasPrefix(q.From, p.To+"/")
	}
	reentrant := make([]bool, len(redirects))
	for start := range redirects {
		seen := make([]bool, len(redirects))
		stack := []int{start}
		for len(stack) > 0 && !reentrant[start] {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for next := range redirects {
				if !leads(redirects[current], redirects[next]) {
					continue
				}
				if next == start {
					reentrant[start] = true
					break
				}
				if !seen[next] {
					seen[next] = true
					stack = append(stack, next)
				}
			}
		}
	}
	return reentrant
}

// loops reports whether following redirects from alias returns to it.
func loops(next map[string]string, alias string) bool {
	current := alias
	for range len(next) {
		target, ok := next[current]
		if !ok {
			return false
		}
		if target == alias {
			return true
		}
		current = target
	}
	return false
}

// aliasSource is the file that declares a node's aliases.
func aliasSource(node *content.Node) string {
	if node.Kind == content.KindDirectory {
		return path.Join(node.SourcePath, content.ConfigurationFile)
	}
	return node.SourcePath
}

// resolveChains adds a redirect for every path a prefix redirect sends to
// another alias, so each redirect reaches its final target in one hop. A
// prefix redirect from A to T sends A+s to T+s; when T+s is under another
// alias, A+s gets that alias's target. Each derivation follows one step
// that reentrantPrefixes checks, which has no cycles, so the expansion
// terminates. Declared redirects are never replaced.
func resolveChains(redirects []content.Redirect) []content.Redirect {
	byFrom := map[string]bool{}
	for _, r := range redirects {
		byFrom[r.From] = true
	}
	for pending := redirects; len(pending) > 0; {
		var added []content.Redirect
		for _, outer := range redirects {
			if !outer.Prefix {
				continue
			}
			for _, inner := range pending {
				suffix, ok := strings.CutPrefix(inner.From, outer.To+"/")
				if !ok {
					continue
				}
				derived := content.Redirect{From: outer.From + "/" + suffix, To: inner.To, Prefix: inner.Prefix}
				if !byFrom[derived.From] {
					byFrom[derived.From] = true
					added = append(added, derived)
				}
			}
		}
		redirects = append(redirects, added...)
		pending = added
	}
	slices.SortFunc(redirects, func(a, b content.Redirect) int { return cmp.Compare(a.From, b.From) })
	return redirects
}
