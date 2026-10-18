package content

import (
	"errors"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// Independent oracles for the documented grammars.
var (
	nameGrammar = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)
	slugGrammar = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

func TestParseName(t *testing.T) {
	for raw, want := range map[string][2]string{
		"On-Martingales":    {"On Martingales", "on-martingales"},
		"Probability-Notes": {"Probability Notes", "probability-notes"},
		"index":             {"index", "index"},
		"2026-Plan":         {"2026 Plan", "2026-plan"},
		"A":                 {"A", "a"},
	} {
		name, err := ParseName(raw)
		if err != nil {
			t.Fatalf("ParseName(%q) = %v", raw, err)
		}
		if name.String() != raw || name.Title() != want[0] || name.Slug().String() != want[1] {
			t.Errorf("ParseName(%q) = %q, title %q, slug %q", raw, name, name.Title(), name.Slug())
		}
	}
	for _, raw := range []string{"", "-", "On-", "-On", "On--Martingales", "On Martingales", "On_Martingales", "On.Martingales", "Ünicode", "a/b", ".hidden"} {
		if name, err := ParseName(raw); !errors.Is(err, ErrInvalidName) || name != (Name{}) {
			t.Errorf("ParseName(%q) = %q, %v; want ErrInvalidName", raw, name, err)
		}
	}
}

func TestParseSlug(t *testing.T) {
	if slug, err := ParseSlug("on-martingales"); err != nil || slug.String() != "on-martingales" {
		t.Fatalf("ParseSlug = %q, %v", slug, err)
	}
	for _, raw := range []string{"", "On-Martingales", "on--martingales", "on martingales", "-on"} {
		if slug, err := ParseSlug(raw); !errors.Is(err, ErrInvalidSlug) || slug != (Slug{}) {
			t.Errorf("ParseSlug(%q) = %q, %v; want ErrInvalidSlug", raw, slug, err)
		}
	}
}

// FuzzParseName checks the parser against the grammar oracle and the
// derivation invariants for every accepted name.
func FuzzParseName(f *testing.F) {
	for _, seed := range []string{"On-Martingales", "a", "a-b-c", "", "-", "a--b", "a-", "A_B", "é", "a\x00b", "On Martingales"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		name, err := ParseName(raw)
		if accepted := err == nil; accepted != nameGrammar.MatchString(raw) {
			t.Fatalf("ParseName(%q) accepted=%v, grammar disagrees", raw, accepted)
		}
		if err != nil {
			if name != (Name{}) {
				t.Fatalf("rejected name %q escaped as %q", raw, name)
			}
			return
		}
		checkDerivations(t, name)
	})
}

func TestDerivationProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(34, 33))
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	for range 2000 {
		segments := make([]string, 1+rng.IntN(5))
		for i := range segments {
			segment := make([]byte, 1+rng.IntN(8))
			for j := range segment {
				segment[j] = alphabet[rng.IntN(len(alphabet))]
			}
			segments[i] = string(segment)
		}
		raw := strings.Join(segments, "-")
		name, err := ParseName(raw)
		if err != nil {
			t.Fatalf("generated valid name %q rejected: %v", raw, err)
		}
		checkDerivations(t, name)
	}
}

func checkDerivations(t *testing.T, name Name) {
	t.Helper()
	raw, title, slug := name.String(), name.Title(), name.Slug().String()
	if strings.ReplaceAll(title, " ", "-") != raw {
		t.Fatalf("title %q does not round-trip to %q", title, raw)
	}
	if title != strings.TrimSpace(title) || strings.Contains(title, "  ") || strings.Contains(title, "-") {
		t.Fatalf("title %q has stray spacing or hyphens", title)
	}
	if !slugGrammar.MatchString(slug) || !strings.EqualFold(slug, raw) {
		t.Fatalf("slug %q is not the lowercased name %q", slug, raw)
	}
	if parsed, err := ParseSlug(slug); err != nil || parsed != name.Slug() {
		t.Fatalf("derived slug %q does not parse: %v", slug, err)
	}
}
