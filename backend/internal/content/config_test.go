package content

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"
)

func TestParseConfig(t *testing.T) {
	config, err := ParseConfig([]byte("title = \"Notes\"\norder = 2\n[theme]\naccent = \"#336699\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{"title": "Notes", "order": int64(2), "theme": map[string]any{"accent": "#336699"}}
	if !reflect.DeepEqual(config, want) {
		t.Fatalf("parseConfig = %#v, want %#v", config, want)
	}
	if config, err := ParseConfig(nil); err != nil || len(config) != 0 {
		t.Fatalf("empty configuration = %#v, %v", config, err)
	}
	for _, raw := range []string{"title = ", "title = \"a\"\ntitle = \"b\"", "[theme\n", "= 1"} {
		if _, err := ParseConfig([]byte(raw)); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("ParseConfig(%q) = %v; want ErrInvalidConfig", raw, err)
		}
	}
}

func TestParseNodeKeys(t *testing.T) {
	keys, err := ParseNodeKeys(Config{"title": "Custom", "slug": "custom", "order": int64(-1)})
	if err != nil || keys.Title != "Custom" || keys.Slug.String() != "custom" || keys.Order != -1 || !keys.HasTitle || !keys.HasSlug || !keys.HasOrder {
		t.Fatalf("parseNodeKeys = %+v, %v", keys, err)
	}
	if keys, err := ParseNodeKeys(Config{"other": 1}); err != nil || keys.HasTitle || keys.HasSlug || keys.HasOrder {
		t.Fatalf("absent keys = %+v, %v", keys, err)
	}
	for _, config := range []Config{
		{"title": ""}, {"title": "  "}, {"title": int64(1)},
		{"slug": "Has-Upper"}, {"slug": "two words"}, {"slug": int64(1)},
		{"order": 1.5}, {"order": "1"},
	} {
		if _, err := ParseNodeKeys(config); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("ParseNodeKeys(%v) = %v; want ErrInvalidConfig", config, err)
		}
	}
}

func TestCascadeNodeLocalKeys(t *testing.T) {
	parent := Config{"title": "Parent", "slug": "parent", "order": int64(1), "theme": "dark"}
	got := Cascade(parent, Config{"order": int64(5)})
	want := Config{"order": int64(5), "theme": "dark"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Cascade = %#v, want %#v", got, want)
	}
}

func TestCascadeDoesNotAlias(t *testing.T) {
	parent := Config{"theme": map[string]any{"accent": "red"}, "tags": []any{"a"}}
	child := Config{"theme": map[string]any{"font": "serif"}, "list": []any{map[string]any{"k": "v"}}}
	parentCopy, childCopy := cloneValue(map[string]any(parent)), cloneValue(map[string]any(child))
	resolved := Cascade(parent, child)
	resolved["theme"].(map[string]any)["accent"] = "mutated"
	resolved["tags"].([]any)[0] = "mutated"
	resolved["list"].([]any)[0].(map[string]any)["k"] = "mutated"
	if !reflect.DeepEqual(map[string]any(parent), parentCopy) || !reflect.DeepEqual(map[string]any(child), childCopy) {
		t.Fatal("Cascade aliased or modified its inputs")
	}
}

// TestCascadeProperties checks, over generated configurations, that every
// value the child specifies wins and every other inheritable parent value
// survives unchanged.
func TestCascadeProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(34, 1))
	for i := range 1000 {
		parent, child := randomConfig(rng, 3), randomConfig(rng, 3)
		resolved := Cascade(parent, child)
		for _, leaf := range leaves(child) {
			leafPath, value := leaf.path, leaf.value
			if table, isTable := value.(map[string]any); isTable && len(table) == 0 {
				if got, ok := lookup(resolved, leafPath); !ok || reflect.TypeOf(got) != reflect.TypeOf(table) {
					t.Fatalf("case %d: empty child table at %v resolved to %#v", i, leafPath, got)
				}
				continue
			}
			if got, ok := lookup(resolved, leafPath); !ok || !reflect.DeepEqual(got, value) {
				t.Fatalf("case %d: child value at %v = %#v, resolved has %#v", i, leafPath, value, got)
			}
		}
		for _, leaf := range leaves(parent) {
			leafPath, value := leaf.path, leaf.value
			if leafPath[0] == KeyTitle || leafPath[0] == KeySlug || leafPath[0] == KeyOrder {
				if _, specified := child[leafPath[0]]; !specified {
					if _, inherited := resolved[leafPath[0]]; inherited {
						t.Fatalf("case %d: node-local key %q was inherited", i, leafPath[0])
					}
				}
				continue
			}
			if shadowed(child, leafPath) {
				continue
			}
			if got, ok := lookup(resolved, leafPath); !ok || !reflect.DeepEqual(got, value) {
				t.Fatalf("case %d: inherited value at %v = %#v, resolved has %#v", i, leafPath, value, got)
			}
		}
		for _, leaf := range leaves(resolved) {
			leafPath := leaf.path
			_, inChild := lookup(child, leafPath)
			_, inParent := lookup(parent, leafPath)
			if !inChild && !inParent {
				t.Fatalf("case %d: resolved invented %v", i, leafPath)
			}
		}
	}
}

var cascadeKeys = []string{"title", "slug", "order", "theme", "accent", "math", "font"}

func randomConfig(rng *rand.Rand, depth int) Config {
	config := Config{}
	for range rng.IntN(5) {
		key := cascadeKeys[rng.IntN(len(cascadeKeys))]
		switch kind := rng.IntN(4); {
		case kind == 0 && depth > 0:
			config[key] = map[string]any(randomConfig(rng, depth-1))
		case kind == 1:
			config[key] = []any{int64(rng.IntN(3)), "x"}
		case kind == 2:
			config[key] = int64(rng.IntN(10))
		default:
			config[key] = fmt.Sprintf("v%d", rng.IntN(10))
		}
	}
	return config
}

type leaf struct {
	path  []string
	value any
}

// leaves lists every non-table value, and every empty table, with its key path.
func leaves(table map[string]any) []leaf {
	var out []leaf
	var walk func(map[string]any, []string)
	walk = func(table map[string]any, prefix []string) {
		for key, value := range table {
			leafPath := append(append([]string(nil), prefix...), key)
			if nested, ok := value.(map[string]any); ok && len(nested) > 0 {
				walk(nested, leafPath)
				continue
			}
			out = append(out, leaf{path: leafPath, value: value})
		}
	}
	walk(table, nil)
	return out
}

func lookup(table map[string]any, leafPath []string) (any, bool) {
	var current any = table
	for _, key := range leafPath {
		nested, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = nested[key]; !ok {
			return nil, false
		}
	}
	return current, true
}

// shadowed reports whether the child specifies the path, or a prefix of it
// with a non-table value that replaces the parent's table.
func shadowed(child map[string]any, leafPath []string) bool {
	var current any = child
	for _, key := range leafPath {
		nested, ok := current.(map[string]any)
		if !ok {
			return true
		}
		if current, ok = nested[key]; !ok {
			return false
		}
	}
	return true
}
