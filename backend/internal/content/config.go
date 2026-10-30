package content

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
)

var (
	// ErrInvalidConfig reports configuration that is not valid TOML or breaks
	// a field rule. Errors wrapping it are *ConfigError values.
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrInvalidColor reports a color outside the "#RRGGBB" form.
	ErrInvalidColor = errors.New("invalid color")
)

// Field limits of the configuration schema.
const (
	MaxTitleLength       = 120
	MaxDescriptionLength = 300
	MaxOrder             = 9999
	MaxAliases           = 32
)

// Level is where a configuration appears in the content tree. It decides
// which fields are required or allowed.
type Level uint8

const (
	// LevelSite is the root ".configuration".
	LevelSite Level = iota + 1
	// LevelDirectory is a directory ".configuration".
	LevelDirectory
	// LevelArticle is an article's frontmatter.
	LevelArticle
	// LevelLanding is an "index.md" landing page's frontmatter.
	LevelLanding
)

// Config is one validated configuration level. Title, Description, Slug,
// Order and Aliases describe their node only; Theme cascades root → directory → article.
// Unknown keys are ignored.
type Config struct {
	// Title is empty when unset.
	Title string
	// Description is empty when unset.
	Description string
	// Slug is zero when unset; never set at LevelSite or LevelLanding.
	Slug Slug
	// Order is in [0, MaxOrder] when Ordered; never set at LevelSite or LevelLanding.
	Order   int
	Ordered bool
	// Aliases are distinct former URL paths of the node; never set at
	// LevelSite or LevelLanding.
	Aliases []Alias
	// Theme is complete at LevelSite and holds overrides at other levels.
	Theme Theme
}

// Theme is the color motif of a node.
type Theme struct {
	Accent     Color
	Background Color
	Text       Color
}

// Override returns t with every color set in o replacing t's.
func (t Theme) Override(o Theme) Theme {
	if !o.Accent.IsZero() {
		t.Accent = o.Accent
	}
	if !o.Background.IsZero() {
		t.Background = o.Background
	}
	if !o.Text.IsZero() {
		t.Text = o.Text
	}
	return t
}

// Complete reports whether every color is set.
func (t Theme) Complete() bool {
	return !t.Accent.IsZero() && !t.Background.IsZero() && !t.Text.IsZero()
}

// Color is an sRGB color normalized to lowercase "#rrggbb". Its zero value is unset.
type Color struct{ hex string }

// ParseColor accepts "#RRGGBB" with hexadecimal digits in either case.
func ParseColor(raw string) (Color, error) {
	if len(raw) != 7 || raw[0] != '#' {
		return Color{}, fmt.Errorf("%w %q: use #RRGGBB", ErrInvalidColor, raw)
	}
	for i := 1; i < len(raw); i++ {
		if !isHexDigit(raw[i]) {
			return Color{}, fmt.Errorf("%w %q: use #RRGGBB", ErrInvalidColor, raw)
		}
	}
	return Color{hex: strings.ToLower(raw)}, nil
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func (c Color) String() string { return c.hex }
func (c Color) IsZero() bool   { return c.hex == "" }

// Rule names the validation rule a FieldError broke.
type Rule string

const (
	RuleSyntax     Rule = "syntax"
	RuleRequired   Rule = "required"
	RuleNotAllowed Rule = "not-allowed"
	RuleType       Rule = "type"
	RuleText       Rule = "text"
	RuleSlug       Rule = "slug"
	RuleRange      Rule = "range"
	RuleColor      Rule = "color"
	RuleAlias      Rule = "alias"
)

// FieldError is one broken rule in a configuration file.
type FieldError struct {
	// Key is the dotted key path, such as "theme.accent"; empty for a syntax
	// error not tied to a key.
	Key string
	// Line and Column are 1-based; zero when the key is absent or the TOML
	// decoder reports no position, as for a duplicate key.
	Line, Column int
	Rule         Rule
	// Value is the offending value in TOML-like notation; empty when absent.
	Value   string
	Message string
}

func (e FieldError) Error() string {
	var b strings.Builder
	if e.Line > 0 {
		fmt.Fprintf(&b, "%d:%d: ", e.Line, e.Column)
	}
	if e.Key != "" {
		b.WriteString(e.Key + ": ")
	}
	b.WriteString(e.Message)
	if e.Value != "" {
		b.WriteString(" (found " + e.Value + ")")
	}
	return b.String()
}

// ConfigError lists every broken rule of one configuration file, in schema order.
type ConfigError struct {
	Fields []FieldError
}

func (e *ConfigError) Error() string {
	messages := make([]string, len(e.Fields))
	for i, field := range e.Fields {
		messages[i] = field.Error()
	}
	return ErrInvalidConfig.Error() + ": " + strings.Join(messages, "; ")
}

func (e *ConfigError) Unwrap() error { return ErrInvalidConfig }

// ParseConfig decodes and validates one configuration level. Empty data is
// valid except at LevelSite, which requires title and a complete theme.
func ParseConfig(level Level, data []byte) (Config, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, &ConfigError{Fields: []FieldError{syntaxError(err)}}
	}
	v := validator{data: data, raw: raw}
	nodeLocal := level == LevelDirectory || level == LevelArticle
	config := Config{
		Title:       v.text("title", MaxTitleLength, level == LevelSite),
		Description: v.text("description", MaxDescriptionLength, false),
		Slug:        v.slug(nodeLocal),
	}
	config.Order, config.Ordered = v.order(nodeLocal)
	config.Aliases = v.aliases(nodeLocal)
	config.Theme = v.theme(level == LevelSite)
	if len(v.errs) > 0 {
		return Config{}, &ConfigError{Fields: v.errs}
	}
	return config, nil
}

func syntaxError(err error) FieldError {
	var decodeErr *toml.DecodeError
	if !errors.As(err, &decodeErr) {
		return FieldError{Rule: RuleSyntax, Message: strings.TrimPrefix(err.Error(), "toml: ")}
	}
	line, column := decodeErr.Position()
	return FieldError{
		Key:     strings.Join(decodeErr.Key(), "."),
		Line:    line,
		Column:  column,
		Rule:    RuleSyntax,
		Message: strings.TrimPrefix(decodeErr.Error(), "toml: "),
	}
}

// validator checks typed fields of a decoded TOML document, collecting errors.
type validator struct {
	data      []byte
	raw       map[string]any
	positions map[string]unstable.Position
	errs      []FieldError
}

func (v *validator) fail(key string, rule Rule, value any, format string, args ...any) {
	field := FieldError{Key: key, Rule: rule, Message: fmt.Sprintf(format, args...)}
	if value != nil {
		field.Value = describe(value)
		pos := v.position(key)
		field.Line, field.Column = pos.Line, pos.Column
	}
	v.errs = append(v.errs, field)
}

// lookup returns the value at a dotted key path within table.
func lookup(table map[string]any, key string) (any, bool) {
	var current any = table
	for part := range strings.SplitSeq(key, ".") {
		nested, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = nested[part]; !ok {
			return nil, false
		}
	}
	return current, true
}

func (v *validator) present(key string, allowed bool) (any, bool) {
	value, ok := lookup(v.raw, key)
	if ok && !allowed {
		v.fail(key, RuleNotAllowed, value, "not allowed here")
		return nil, false
	}
	return value, ok
}

func (v *validator) text(key string, maxLength int, required bool) string {
	value, ok := v.present(key, true)
	if !ok {
		if required {
			v.fail(key, RuleRequired, nil, "is required")
		}
		return ""
	}
	text, isString := value.(string)
	if !isString {
		v.fail(key, RuleType, value, "must be a string")
		return ""
	}
	text = strings.TrimSpace(text)
	switch {
	case text == "":
		v.fail(key, RuleText, value, "must not be empty")
	case utf8.RuneCountInString(text) > maxLength:
		v.fail(key, RuleText, value, "must be at most %d characters", maxLength)
	case strings.IndexFunc(text, unicode.IsControl) >= 0:
		v.fail(key, RuleText, value, "must be a single line without control characters")
	default:
		return text
	}
	return ""
}

func (v *validator) slug(allowed bool) Slug {
	const key = "slug"
	value, ok := v.present(key, allowed)
	if !ok {
		return Slug{}
	}
	text, isString := value.(string)
	if !isString {
		v.fail(key, RuleType, value, "must be a string")
		return Slug{}
	}
	slug, err := ParseSlug(text)
	if err != nil {
		v.fail(key, RuleSlug, value, "must be lowercase letters, digits and single hyphens")
		return Slug{}
	}
	return slug
}

func (v *validator) order(allowed bool) (int, bool) {
	const key = "order"
	value, ok := v.present(key, allowed)
	if !ok {
		return 0, false
	}
	order, isInteger := value.(int64)
	switch {
	case !isInteger:
		v.fail(key, RuleType, value, "must be an integer")
	case order < 0 || order > MaxOrder:
		v.fail(key, RuleRange, value, "must be between 0 and %d", MaxOrder)
	default:
		return int(order), true
	}
	return 0, false
}

func (v *validator) aliases(allowed bool) []Alias {
	const key = "aliases"
	value, ok := v.present(key, allowed)
	if !ok {
		return nil
	}
	list, isArray := value.([]any)
	if !isArray {
		v.fail(key, RuleType, value, "must be an array of strings")
		return nil
	}
	if len(list) > MaxAliases {
		v.fail(key, RuleRange, value, "must have at most %d entries", MaxAliases)
		return nil
	}
	aliases := make([]Alias, 0, len(list))
	seen := map[Alias]bool{}
	valid := true
	for _, item := range list {
		text, isString := item.(string)
		if !isString {
			v.fail(key, RuleType, item, "must be an array of strings")
			valid = false
			continue
		}
		alias, err := ParseAlias(text)
		switch {
		case err != nil:
			v.fail(key, RuleAlias, item, "must be an absolute path of lowercase slugs, such as /essays/old-title")
			valid = false
		case seen[alias]:
			v.fail(key, RuleAlias, item, "must not repeat")
			valid = false
		default:
			seen[alias] = true
			aliases = append(aliases, alias)
		}
	}
	if !valid {
		return nil
	}
	return aliases
}

func (v *validator) theme(required bool) Theme {
	if value, ok := v.raw["theme"]; ok {
		if _, isTable := value.(map[string]any); !isTable {
			v.fail("theme", RuleType, value, "must be a table")
			return Theme{}
		}
	}
	return Theme{
		Accent:     v.color("theme.accent", required),
		Background: v.color("theme.background", required),
		Text:       v.color("theme.text", required),
	}
}

func (v *validator) color(key string, required bool) Color {
	value, ok := lookup(v.raw, key)
	if !ok {
		if required {
			v.fail(key, RuleRequired, nil, "is required")
		}
		return Color{}
	}
	text, isString := value.(string)
	if !isString {
		v.fail(key, RuleType, value, "must be a string")
		return Color{}
	}
	color, err := ParseColor(text)
	if err != nil {
		v.fail(key, RuleColor, value, "must be a #RRGGBB color")
		return Color{}
	}
	return color
}

// position returns where key is written. The index is built only once a
// rule fails, so valid configuration is parsed once.
func (v *validator) position(key string) unstable.Position {
	if v.positions == nil {
		v.positions = keyPositions(v.data)
	}
	return v.positions[key]
}

// keyPositions maps each dotted key path of a valid TOML document to the
// position of its first key segment.
func keyPositions(data []byte) map[string]unstable.Position {
	positions := map[string]unstable.Position{}
	parser := unstable.Parser{}
	parser.Reset(data)
	var prefix []string
	var record func(node *unstable.Node, prefix []string)
	record = func(node *unstable.Node, prefix []string) {
		path, first := keyPath(node, prefix)
		if _, seen := positions[strings.Join(path, ".")]; !seen && first != nil {
			positions[strings.Join(path, ".")] = parser.Shape(first.Raw).Start
		}
		if value := node.Value(); value.Kind == unstable.InlineTable {
			for children := value.Children(); children.Next(); {
				record(children.Node(), path)
			}
		}
	}
	for parser.NextExpression() {
		node := parser.Expression()
		switch node.Kind {
		case unstable.Table:
			path, first := keyPath(node, nil)
			prefix = path
			if _, seen := positions[strings.Join(path, ".")]; !seen && first != nil {
				positions[strings.Join(path, ".")] = parser.Shape(first.Raw).Start
			}
		case unstable.ArrayTable:
			prefix = []string{"[]"}
		case unstable.KeyValue:
			record(node, prefix)
		}
	}
	return positions
}

func keyPath(node *unstable.Node, prefix []string) ([]string, *unstable.Node) {
	path := append([]string(nil), prefix...)
	var first *unstable.Node
	for parts := node.Key(); parts.Next(); {
		part := parts.Node()
		if first == nil {
			first = part
		}
		path = append(path, string(part.Data))
	}
	return path, first
}

// describe renders a decoded TOML value for an error message, truncated so
// long or multi-line values stay readable.
func describe(value any) string {
	var text string
	switch v := value.(type) {
	case string:
		text = fmt.Sprintf("%q", v)
	case map[string]any:
		return "a table"
	case []any:
		return "an array"
	default:
		text = fmt.Sprint(v)
	}
	const limit = 40
	if utf8.RuneCountInString(text) > limit {
		text = string([]rune(text)[:limit]) + "…"
	}
	return text
}
