package content

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ErrInvalidConfig reports unparseable TOML or a reserved key with the wrong type or value.
var ErrInvalidConfig = errors.New("invalid configuration")

// Reserved keys describe one node and are never inherited by its children.
const (
	keyTitle = "title"
	keySlug  = "slug"
	keyOrder = "order"
)

// Config is a decoded TOML table: string keys mapping to TOML values, with
// nested tables as Config-compatible map[string]any and arrays as []any.
type Config map[string]any

func parseConfig(data []byte) (Config, error) {
	config := Config{}
	if err := toml.Unmarshal(data, (*map[string]any)(&config)); err != nil {
		var decodeErr *toml.DecodeError
		if errors.As(err, &decodeErr) {
			row, column := decodeErr.Position()
			return nil, fmt.Errorf("%w: line %d, column %d: %s", ErrInvalidConfig, row, column, decodeErr.Error())
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return config, nil
}

// Cascade resolves a child's configuration from its parent's resolved
// configuration. The child inherits every parent key except the node-local
// title, slug and order; keys the child specifies override inherited ones.
// Tables merge key by key at every depth; other values, including arrays,
// replace the inherited value. Neither input is modified or aliased.
func Cascade(parent, child Config) Config {
	resolved := make(Config, len(parent)+len(child))
	for key, value := range parent {
		if key != keyTitle && key != keySlug && key != keyOrder {
			resolved[key] = cloneValue(value)
		}
	}
	mergeInto(resolved, child)
	return resolved
}

func mergeInto(dst, src map[string]any) {
	for key, value := range src {
		srcTable, srcIsTable := value.(map[string]any)
		dstTable, dstIsTable := dst[key].(map[string]any)
		if srcIsTable && dstIsTable {
			mergeInto(dstTable, srcTable)
			continue
		}
		dst[key] = cloneValue(value)
	}
}

func cloneValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		clone := make(map[string]any, len(v))
		for key, item := range v {
			clone[key] = cloneValue(item)
		}
		return clone
	case []any:
		clone := make([]any, len(v))
		for i, item := range v {
			clone[i] = cloneValue(item)
		}
		return clone
	default:
		return v
	}
}

// nodeKeys holds the parsed node-local keys of one configuration level.
type nodeKeys struct {
	title    string
	slug     Slug
	order    int64
	hasTitle bool
	hasSlug  bool
	hasOrder bool
}

func parseNodeKeys(config Config) (nodeKeys, error) {
	var keys nodeKeys
	if raw, ok := config[keyTitle]; ok {
		title, isString := raw.(string)
		if !isString || strings.TrimSpace(title) == "" {
			return nodeKeys{}, fmt.Errorf("%w: %q must be a non-empty string", ErrInvalidConfig, keyTitle)
		}
		keys.title, keys.hasTitle = title, true
	}
	if raw, ok := config[keySlug]; ok {
		text, isString := raw.(string)
		if !isString {
			return nodeKeys{}, fmt.Errorf("%w: %q must be a string", ErrInvalidConfig, keySlug)
		}
		slug, err := ParseSlug(text)
		if err != nil {
			return nodeKeys{}, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
		}
		keys.slug, keys.hasSlug = slug, true
	}
	if raw, ok := config[keyOrder]; ok {
		order, isInteger := raw.(int64)
		if !isInteger {
			return nodeKeys{}, fmt.Errorf("%w: %q must be an integer", ErrInvalidConfig, keyOrder)
		}
		keys.order, keys.hasOrder = order, true
	}
	return keys, nil
}
