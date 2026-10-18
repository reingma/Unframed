package content

import (
	"errors"
	"math/rand/v2"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

const validSite = "title = \"Unframed\"\ndescription = \"Notes\"\n[theme]\naccent = \"#3366AA\"\nbackground = \"#ffffff\"\ntext = \"#111111\"\n"

func TestParseConfigValid(t *testing.T) {
	site, err := ParseConfig(LevelSite, []byte(validSite))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Title: "Unframed", Description: "Notes", Theme: Theme{
		Accent: Color{"#3366aa"}, Background: Color{"#ffffff"}, Text: Color{"#111111"},
	}}
	if site != want {
		t.Fatalf("site = %+v, want %+v", site, want)
	}

	article, err := ParseConfig(LevelArticle, []byte("title = \"  Padded  \"\nslug = \"custom\"\norder = 0\ntheme = { text = \"#000000\" }\n"))
	if err != nil {
		t.Fatal(err)
	}
	if article.Title != "Padded" || article.Slug.String() != "custom" || !article.Ordered || article.Order != 0 || article.Theme != (Theme{Text: Color{"#000000"}}) {
		t.Fatalf("article = %+v", article)
	}

	for _, level := range []Level{LevelDirectory, LevelArticle, LevelLanding} {
		if config, err := ParseConfig(level, nil); err != nil || config != (Config{}) {
			t.Errorf("empty level %d = %+v, %v", level, config, err)
		}
	}
}

func TestParseConfigIgnoresUnknownKeys(t *testing.T) {
	data := "summary = 1\n[math]\nengine = \"katex\"\n[theme]\nfont = \"serif\"\n[[links]]\ntitle = 3\n"
	if config, err := ParseConfig(LevelDirectory, []byte(data)); err != nil || config != (Config{}) {
		t.Fatalf("unknown keys = %+v, %v", config, err)
	}
}

type wantField struct {
	key          string
	rule         Rule
	line, column int
}

func TestParseConfigFieldErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		level Level
		data  string
		want  []wantField
	}{
		"site requires title and theme": {LevelSite, "", []wantField{
			{"title", RuleRequired, 0, 0}, {"theme.accent", RuleRequired, 0, 0},
			{"theme.background", RuleRequired, 0, 0}, {"theme.text", RuleRequired, 0, 0},
		}},
		"site rejects slug and order": {LevelSite, validSite + "[extra]\n", nil},
		"wrong title type":            {LevelDirectory, "title = 3\n", []wantField{{"title", RuleType, 1, 1}}},
		"blank title":                 {LevelArticle, "\n  title = \"   \"\n", []wantField{{"title", RuleText, 2, 3}}},
		"multi-line title":            {LevelArticle, "title = \"a\\nb\"\n", []wantField{{"title", RuleText, 1, 1}}},
		"long title":                  {LevelArticle, "title = \"" + strings.Repeat("é", MaxTitleLength+1) + "\"\n", []wantField{{"title", RuleText, 1, 1}}},
		"long description":            {LevelArticle, "description = \"" + strings.Repeat("a", MaxDescriptionLength+1) + "\"\n", []wantField{{"description", RuleText, 1, 1}}},
		"slug form":                   {LevelDirectory, "slug = \"Not-Lower\"\n", []wantField{{"slug", RuleSlug, 1, 1}}},
		"order range":                 {LevelArticle, "order = 10000\n", []wantField{{"order", RuleRange, 1, 1}}},
		"negative order":              {LevelArticle, "order = -1\n", []wantField{{"order", RuleRange, 1, 1}}},
		"fractional order":            {LevelArticle, "order = 1.5\n", []wantField{{"order", RuleType, 1, 1}}},
		"landing slug and order":      {LevelLanding, "order = 1\nslug = \"x\"\n", []wantField{{"slug", RuleNotAllowed, 2, 1}, {"order", RuleNotAllowed, 1, 1}}},
		"theme not a table":           {LevelDirectory, "theme = \"dark\"\n", []wantField{{"theme", RuleType, 1, 1}}},
		"color form":                  {LevelDirectory, "[theme]\naccent = \"blue\"\n", []wantField{{"theme.accent", RuleColor, 2, 1}}},
		"inline table color":          {LevelArticle, "theme = { accent = 1 }\n", []wantField{{"theme.accent", RuleType, 1, 11}}},
		"dotted key color":            {LevelArticle, "theme.text = \"#12345\"\n", []wantField{{"theme.text", RuleColor, 1, 1}}},
		"every error is reported":     {LevelDirectory, "order = \"1\"\ntitle = \"\"\n[theme]\ntext = \"#zzzzzz\"\n", []wantField{{"title", RuleText, 2, 1}, {"order", RuleType, 1, 1}, {"theme.text", RuleColor, 4, 1}}},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.want == nil {
				tc.data = "slug = \"home\"\norder = 1\n" + tc.data
				tc.want = []wantField{{"slug", RuleNotAllowed, 1, 1}, {"order", RuleNotAllowed, 2, 1}}
			}
			_, err := ParseConfig(tc.level, []byte(tc.data))
			var configErr *ConfigError
			if !errors.Is(err, ErrInvalidConfig) || !errors.As(err, &configErr) {
				t.Fatalf("ParseConfig = %v; want a *ConfigError", err)
			}
			var got []wantField
			for _, field := range configErr.Fields {
				got = append(got, wantField{field.Key, field.Rule, field.Line, field.Column})
				if field.Message == "" || (field.Rule != RuleRequired) == (field.Value == "") {
					t.Errorf("field %+v: message or found value missing", field)
				}
			}
			if !equalFields(got, tc.want) {
				t.Fatalf("fields = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func equalFields(a, b []wantField) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseConfigSyntaxError(t *testing.T) {
	for raw, positioned := range map[string]bool{"title = ": true, "[theme\n": true, "= 1": true, "title = \"a\"\ntitle = \"b\"": false} {
		_, err := ParseConfig(LevelArticle, []byte(raw))
		var configErr *ConfigError
		if !errors.As(err, &configErr) || len(configErr.Fields) != 1 {
			t.Fatalf("ParseConfig(%q) = %v; want one syntax error", raw, err)
		}
		field := configErr.Fields[0]
		if field.Rule != RuleSyntax || strings.HasPrefix(field.Message, "toml:") || positioned && (field.Line < 1 || field.Column < 1) {
			t.Errorf("ParseConfig(%q) field = %+v", raw, field)
		}
	}
}

func TestParseColor(t *testing.T) {
	for raw, want := range map[string]string{"#000000": "#000000", "#AbCdEf": "#abcdef", "#123abc": "#123abc"} {
		if color, err := ParseColor(raw); err != nil || color.String() != want {
			t.Errorf("ParseColor(%q) = %q, %v; want %q", raw, color, err, want)
		}
	}
	for _, raw := range []string{"", "#", "000000", "#00000", "#0000000", "#gggggg", "blue", "#00 000"} {
		if _, err := ParseColor(raw); !errors.Is(err, ErrInvalidColor) {
			t.Errorf("ParseColor(%q) = %v; want ErrInvalidColor", raw, err)
		}
	}
}

// TestThemeOverrideProperties checks over generated themes that set colors
// win, unset colors inherit, and completeness is preserved.
func TestThemeOverrideProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(34, 2))
	colors := []Color{{}, {"#000000"}, {"#ffffff"}, {"#336699"}}
	random := func() Theme {
		return Theme{colors[rng.IntN(len(colors))], colors[rng.IntN(len(colors))], colors[rng.IntN(len(colors))]}
	}
	pick := func(override, inherited Color) Color {
		if override.IsZero() {
			return inherited
		}
		return override
	}
	for i := range 1000 {
		parent, child := random(), random()
		got := parent.Override(child)
		want := Theme{pick(child.Accent, parent.Accent), pick(child.Background, parent.Background), pick(child.Text, parent.Text)}
		if got != want {
			t.Fatalf("case %d: %+v.Override(%+v) = %+v, want %+v", i, parent, child, got, want)
		}
		if parent.Complete() && !got.Complete() {
			t.Fatalf("case %d: override lost completeness", i)
		}
		if got.Override(child) != got {
			t.Fatalf("case %d: override is not idempotent", i)
		}
	}
}

// FuzzParseConfig checks that every accepted configuration satisfies the
// schema and every rejection is a described *ConfigError.
func FuzzParseConfig(f *testing.F) {
	for _, seed := range []string{
		validSite, "", "title = 3", "order = 9999\nslug = \"a-b\"", "theme = { accent = \"#AABBCC\" }",
		"title = \"a\\u0007\"", "[theme]\ntext = \"#12345g\"", "[[theme]]\naccent = \"#000000\"",
	} {
		for level := LevelSite; level <= LevelLanding; level++ {
			f.Add(uint8(level), seed)
		}
	}
	f.Fuzz(func(t *testing.T, rawLevel uint8, data string) {
		level := Level(rawLevel%4 + 1)
		config, err := ParseConfig(level, []byte(data))
		if err != nil {
			var configErr *ConfigError
			if !errors.Is(err, ErrInvalidConfig) || !errors.As(err, &configErr) || len(configErr.Fields) == 0 {
				t.Fatalf("rejection %v is not a described *ConfigError", err)
			}
			for _, field := range configErr.Fields {
				if field.Rule == "" || field.Message == "" || (field.Line > 0) != (field.Column > 0) {
					t.Fatalf("incomplete field error %+v", field)
				}
			}
			return
		}
		for _, text := range []struct {
			value string
			max   int
		}{{config.Title, MaxTitleLength}, {config.Description, MaxDescriptionLength}} {
			if text.value != strings.TrimSpace(text.value) || utf8.RuneCountInString(text.value) > text.max || strings.IndexFunc(text.value, unicode.IsControl) >= 0 {
				t.Fatalf("accepted text %q", text.value)
			}
		}
		if config.Order < 0 || config.Order > MaxOrder || (!config.Ordered && config.Order != 0) {
			t.Fatalf("accepted order %d", config.Order)
		}
		if config.Slug != (Slug{}) {
			if _, err := ParseSlug(config.Slug.String()); err != nil {
				t.Fatalf("accepted slug %q", config.Slug)
			}
		}
		for _, color := range []Color{config.Theme.Accent, config.Theme.Background, config.Theme.Text} {
			if _, err := ParseColor(color.String()); !color.IsZero() && (err != nil || color.String() != strings.ToLower(color.String())) {
				t.Fatalf("accepted color %q", color)
			}
		}
		nodeLocal := level == LevelDirectory || level == LevelArticle
		if !nodeLocal && (config.Ordered || config.Slug != (Slug{})) {
			t.Fatalf("level %d accepted slug or order: %+v", level, config)
		}
		if level == LevelSite && (config.Title == "" || !config.Theme.Complete()) {
			t.Fatalf("site accepted without required fields: %+v", config)
		}
	})
}
