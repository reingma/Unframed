package document

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/content"
)

var update = flag.Bool("update", false, "rewrite golden documents")

func color(t testing.TB, raw string) content.Color {
	t.Helper()
	c, err := content.ParseColor(raw)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func theme(t testing.TB) content.Theme {
	return content.Theme{Accent: color(t, "#336699"), Background: color(t, "#FFFFFF"), Text: color(t, "#111111")}
}

func article(t testing.TB) *content.Node {
	return &content.Node{
		Kind: content.KindArticle, SourcePath: "Mathematics/On-Martingales.md", URLPath: "/mathematics/on-martingales",
		Title: "On Martingales", Description: "Fair games", Theme: theme(t), Body: []byte("# On Martingales\n"),
	}
}

func tree(t testing.TB) *content.Node {
	landing := &content.Node{Kind: content.KindArticle, URLPath: "/mathematics", Title: "Mathematics", Landing: true, Theme: theme(t)}
	maths := &content.Node{Kind: content.KindDirectory, URLPath: "/mathematics", Title: "Mathematics", Description: "Notes", Index: landing, Children: []*content.Node{article(t)}}
	return &content.Node{Kind: content.KindDirectory, URLPath: "/", Title: "Unframed", Description: "A blog", Theme: theme(t), Children: []*content.Node{maths}}
}

func golden(t *testing.T, name string, v any) []byte {
	t.Helper()
	got, err := json.Marshal(v, jsontext.WithIndent("  "))
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s mismatch\n got:\n%s\nwant:\n%s", name, got, want)
	}
	return want
}

func TestEnvelopeGoldenAndRoundTrip(t *testing.T) {
	envelope := NewEnvelope(article(t), "<h1>On Martingales</h1>\n")
	data := golden(t, "envelope.json", envelope)
	decoded, err := DecodeEnvelope(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, envelope) {
		t.Fatalf("round trip\n got %+v\nwant %+v", decoded, envelope)
	}
}

func TestManifestGoldenAndRoundTrip(t *testing.T) {
	manifest := NewManifest(tree(t))
	data := golden(t, "manifest.json", manifest)
	decoded, err := DecodeManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, manifest) {
		t.Fatalf("round trip\n got %+v\nwant %+v", decoded, manifest)
	}
}

// mutate decodes a golden document, applies change and re-encodes it.
func mutate(t *testing.T, name string, change func(map[string]any)) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	change(doc)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDecodeRejectsUnsupportedVersions(t *testing.T) {
	versions := map[string]func(map[string]any){
		"missing": func(d map[string]any) { delete(d, "schemaVersion") },
		"null":    func(d map[string]any) { d["schemaVersion"] = nil },
		"zero":    func(d map[string]any) { d["schemaVersion"] = 0 },
		"future":  func(d map[string]any) { d["schemaVersion"] = 2 },
	}
	decoders := map[string]func([]byte) error{
		"envelope.json": func(b []byte) error { _, err := DecodeEnvelope(b); return err },
		"manifest.json": func(b []byte) error { _, err := DecodeManifest(b); return err },
	}
	for file, decode := range decoders {
		for name, change := range versions {
			t.Run(file+"/"+name, func(t *testing.T) {
				if err := decode(mutate(t, file, change)); !errors.Is(err, ErrUnsupportedSchemaVersion) {
					t.Fatalf("error = %v, want ErrUnsupportedSchemaVersion", err)
				}
			})
		}
		t.Run(file+"/wrong type", func(t *testing.T) {
			err := decode(mutate(t, file, func(d map[string]any) { d["schemaVersion"] = "1" }))
			if !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("error = %v, want ErrInvalidDocument", err)
			}
		})
	}
}

func TestDecodeRejectsInvalidDocuments(t *testing.T) {
	envelopeCases := map[string]func(map[string]any){
		"unknown body kind": func(d map[string]any) { d["body"].(map[string]any)["kind"] = "blocks" },
		"relative url":      func(d map[string]any) { d["metadata"].(map[string]any)["urlPath"] = "mathematics" },
		"empty title":       func(d map[string]any) { d["metadata"].(map[string]any)["title"] = " " },
		"bad color": func(d map[string]any) {
			d["resolvedConfig"].(map[string]any)["theme"].(map[string]any)["text"] = "black"
		},
	}
	for name, change := range envelopeCases {
		t.Run("envelope/"+name, func(t *testing.T) {
			if _, err := DecodeEnvelope(mutate(t, "envelope.json", change)); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("error = %v, want ErrInvalidDocument", err)
			}
		})
	}
	child := func(d map[string]any) map[string]any {
		return d["nav"].(map[string]any)["children"].([]any)[0].(map[string]any)
	}
	manifestCases := map[string]func(map[string]any){
		"non-root nav":       func(d map[string]any) { d["nav"].(map[string]any)["urlPath"] = "/x" },
		"unknown nav kind":   func(d map[string]any) { child(d)["kind"] = "page" },
		"article with index": func(d map[string]any) { child(d)["children"].([]any)[0].(map[string]any)["hasIndex"] = true },
		"empty nested title": func(d map[string]any) { child(d)["title"] = "" },
		"missing site theme": func(d map[string]any) { delete(d["site"].(map[string]any), "theme") },
	}
	for name, change := range manifestCases {
		t.Run("manifest/"+name, func(t *testing.T) {
			if _, err := DecodeManifest(mutate(t, "manifest.json", change)); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("error = %v, want ErrInvalidDocument", err)
			}
		})
	}
	for name, data := range map[string]string{"not json": "{", "duplicate member": `{"schemaVersion":1,"schemaVersion":1}`} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeEnvelope([]byte(data)); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("error = %v, want ErrInvalidDocument", err)
			}
		})
	}
}

func TestDecodeIgnoresUnknownMembers(t *testing.T) {
	data := mutate(t, "envelope.json", func(d map[string]any) { d["addedLater"] = true })
	if _, err := DecodeEnvelope(data); err != nil {
		t.Fatalf("an additive member was rejected: %v", err)
	}
}

func TestManifestOmitsInternals(t *testing.T) {
	data, err := json.Marshal(NewManifest(tree(t)))
	if err != nil {
		t.Fatal(err)
	}
	for _, internal := range []string{"Mathematics/On-Martingales.md", "sourcePath", "draft", "order"} {
		if strings.Contains(string(data), internal) {
			t.Errorf("manifest exposes %q: %s", internal, data)
		}
	}
}

// FuzzDecodeEnvelope checks that Decode never panics and that every accepted
// envelope survives a re-encode unchanged.
func FuzzDecodeEnvelope(f *testing.F) {
	if data, err := os.ReadFile(filepath.Join("testdata", "envelope.json")); err == nil {
		f.Add(data)
	}
	f.Add([]byte(`{"schemaVersion":2}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		envelope, err := DecodeEnvelope(data)
		if err != nil {
			return
		}
		again, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeEnvelope(again)
		if err != nil || !reflect.DeepEqual(decoded, envelope) {
			t.Fatalf("re-encoded envelope changed: %v\n%+v\n%+v", err, decoded, envelope)
		}
	})
}
