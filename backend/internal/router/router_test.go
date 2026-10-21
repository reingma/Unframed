package router_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/reingma/Unframed/backend/internal/api"
	"github.com/reingma/Unframed/backend/internal/build"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
	"github.com/reingma/Unframed/backend/internal/document"
	"github.com/reingma/Unframed/backend/internal/publish"
	"github.com/reingma/Unframed/backend/internal/rebuild"
	"github.com/reingma/Unframed/backend/internal/router"
	"github.com/reingma/Unframed/backend/internal/snapshot"
)

const (
	previewToken = "preview-secret-0123"
	adminToken   = "admin-secret-4567"
	rootConfig   = "title = \"Unframed\"\n[theme]\naccent = \"#336699\"\nbackground = \"#FFFFFF\"\ntext = \"#111111\"\n"
)

var site = contenttest.MemSource{
	".configuration":              rootConfig,
	"index.md":                    "Home.\n",
	"Essays/Hello-World.md":       "Published hello.\n",
	"Essays/Hello-World.draft.md": "Draft hello.\n",
	"Essays/Secret.draft.md":      "Draft only.\n",
	"Essays/Broken.md":            "+++\n[theme]\naccent = \"red\"\n+++\nNever rendered.\n",
	"Notes/index.md":              "Notes landing.\n",
	"Notes/First.md":              "First note.\n",
}

// source is a content source whose files can change between builds.
type source struct {
	mu    sync.Mutex
	files contenttest.MemSource
}

func (s *source) set(files contenttest.MemSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = files
}

func (s *source) build(ctx context.Context) (*snapshot.Snapshot, error) {
	s.mu.Lock()
	files := s.files
	s.mu.Unlock()
	return rebuild.FromSource(files, publish.NewLocal(), slog.New(slog.DiscardHandler))(ctx)
}

type server struct {
	handler http.Handler
	store   *snapshot.Store
	source  *source
	logs    *syncBuffer
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newServer(t *testing.T, preview, admin string) *server {
	t.Helper()
	logs := &syncBuffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	src := &source{files: site}
	store := &snapshot.Store{}
	manager := rebuild.New(store, src.build, logger)
	if err := manager.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { manager.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	handlers := api.New(store, manager, api.NewToken(preview), api.NewToken(admin), logger)
	return &server{handler: api.Observe(router.New(handlers), logger), store: store, source: src, logs: logs}
}

func (s *server) do(t *testing.T, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for k, values := range header {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

func (s *server) get(t *testing.T, target string) *httptest.ResponseRecorder {
	return s.do(t, http.MethodGet, target, nil)
}

func envelope(t *testing.T, rec *httptest.ResponseRecorder) document.Envelope {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	env, err := document.DecodeEnvelope(rec.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return env
}

const genericNotFound = `{"error":"not found"}`

func expectGenericNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNotFound || rec.Body.String() != genericNotFound {
		t.Fatalf("got %d %s, want the generic 404", rec.Code, rec.Body)
	}
}

func TestHealthcheck(t *testing.T) {
	s := newServer(t, "", "")
	if rec := s.get(t, "/healthcheck"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := s.do(t, http.MethodPost, "/healthcheck", nil); rec.Code != http.StatusMethodNotAllowed || !strings.Contains(rec.Header().Get("Allow"), http.MethodGet) {
		t.Fatalf("POST /healthcheck = %d, Allow %q", rec.Code, rec.Header().Get("Allow"))
	}
	for _, p := range []string{"/healthz", "/missing"} {
		expectGenericNotFound(t, s.get(t, p))
	}
}

// Next.js structures the site from the manifest and fetches each article at
// /articles followed by its nav urlPath.
func TestManifestURLPathsResolveToArticles(t *testing.T) {
	s := newServer(t, "", "")
	rec := s.get(t, "/manifest")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	manifest, err := document.DecodeManifest(rec.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	var visit func(document.NavNode)
	fetched := 0
	visit = func(n document.NavNode) {
		if n.Kind == document.NavArticle || n.HasIndex {
			env := envelope(t, s.get(t, "/articles"+strings.TrimSuffix(n.URLPath, "/")+suffix(n.URLPath)))
			if env.Metadata.URLPath != n.URLPath || env.Metadata.Draft {
				t.Fatalf("nav %s served %+v", n.URLPath, env.Metadata)
			}
			fetched++
		}
		for _, child := range n.Children {
			visit(child)
		}
	}
	visit(manifest.Nav)
	if fetched != 4 {
		t.Fatalf("fetched %d articles from the nav, want root, hello-world, notes and first", fetched)
	}
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "broken") {
		t.Fatalf("manifest lists a draft or invalid item: %s", rec.Body)
	}
}

// suffix keeps the root's "/" so the root landing page is /articles/.
func suffix(urlPath string) string {
	if urlPath == "/" {
		return "/"
	}
	return ""
}

func TestPublicArticles(t *testing.T) {
	s := newServer(t, previewToken, "")
	if env := envelope(t, s.get(t, "/articles/essays/hello-world")); !strings.Contains(env.Body.Content, "Published hello") {
		t.Fatalf("public article served %q", env.Body.Content)
	}
	if env := envelope(t, s.get(t, "/articles/")); !env.Metadata.Landing {
		t.Fatalf("root is not the landing page: %+v", env.Metadata)
	}
	for _, p := range []string{
		"/articles/essays/secret",  // draft only
		"/articles/essays/broken",  // invalid item
		"/articles/essays/missing", // unknown
		"/articles",
		"/articles/essays/hello-world/",
		"/articles/essays//hello-world",
		"/articles/Essays/Hello-World",
		"/articles/essays/hello-world.draft",
		"/articles/essays/../essays/hello-world",
		"/articles/essays/%2e%2e/essays/hello-world",
		"/articles/essays%2Fhello-world",
		"/articles/essays%2fhello-world",
		"/articles/essays%5Chello-world",
		"/articles/essays/hello%20world",
		"/preview/wrong/articles/essays/secret",
		"/preview/wrong/diagnostics",
	} {
		t.Run(p, func(t *testing.T) { expectGenericNotFound(t, s.get(t, p)) })
	}
}

func TestPreview(t *testing.T) {
	s := newServer(t, previewToken, adminToken)
	base := "/preview/" + previewToken
	if env := envelope(t, s.get(t, base+"/articles/essays/hello-world")); !env.Metadata.Draft || !strings.Contains(env.Body.Content, "Draft hello") {
		t.Fatalf("preview served %+v", env)
	}
	if env := envelope(t, s.get(t, base+"/articles/essays/secret")); !env.Metadata.Draft {
		t.Fatalf("preview of a draft-only article = %+v", env.Metadata)
	}
	if env := envelope(t, s.get(t, base+"/articles/notes/first")); env.Metadata.Draft {
		t.Fatal("preview without a draft did not fall back to the published article")
	}

	rec := s.get(t, base+"/articles/essays/broken")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("preview of an invalid item = %d", rec.Code)
	}
	var miss struct {
		URLPath     string
		Diagnostics []snapshot.Diagnostic
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &miss, json.MatchCaseInsensitiveNames(true)); err != nil {
		t.Fatal(err)
	}
	if len(miss.Diagnostics) != 1 || miss.Diagnostics[0].Path != "Essays/Broken.md" || miss.Diagnostics[0].Message == "" {
		t.Fatalf("preview miss does not fail loud: %s", rec.Body)
	}
	expectGenericNotFound(t, s.get(t, base+"/articles/essays/../secret"))
	expectGenericNotFound(t, s.get(t, "/preview/"+previewToken+"x/articles/essays/secret"))
}

func TestPreviewDisabledWithoutToken(t *testing.T) {
	s := newServer(t, "", "")
	expectGenericNotFound(t, s.get(t, "/preview//articles/essays/secret"))
	expectGenericNotFound(t, s.get(t, "/preview/x/diagnostics"))
}

func TestUnknownSchemaVersionIsNotServed(t *testing.T) {
	s := newServer(t, previewToken, "")
	result, err := build.Build(context.Background(), site, publish.NewLocal(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	key := build.Key{URLPath: "/notes/first"}
	env := result.Envelopes[key]
	env.SchemaVersion = 99
	result.Envelopes[key] = env
	next, err := snapshot.New(result, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	s.store.Swap(next)

	expectGenericNotFound(t, s.get(t, "/articles/notes/first"))
	rec := s.get(t, "/preview/"+previewToken+"/articles/notes/first")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Notes/First.md") || !strings.Contains(rec.Body.String(), "schema version") {
		t.Fatalf("preview of an unknown schemaVersion = %d %s", rec.Code, rec.Body)
	}
}

func TestDiagnostics(t *testing.T) {
	s := newServer(t, previewToken, adminToken)
	rec := s.get(t, "/preview/"+previewToken+"/diagnostics")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Serving struct {
			BuildID          string
			Envelopes        int
			ValidationErrors int
		}
		Diagnostics []snapshot.Diagnostic
		LastBuild   struct {
			Trigger string
			Outcome string
			BuildID string
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got, json.MatchCaseInsensitiveNames(true)); err != nil {
		t.Fatal(err)
	}
	if got.Serving.BuildID != s.store.Current().BuildID() || got.Serving.ValidationErrors != 1 || len(got.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %s", rec.Body)
	}
	if got.LastBuild.Trigger != "startup" || got.LastBuild.Outcome != "succeeded" || got.LastBuild.BuildID != got.Serving.BuildID {
		t.Fatalf("last build = %+v", got.LastBuild)
	}

	s.source.set(contenttest.MemSource{".configuration": "broken = ["})
	bearer := http.Header{"Authorization": {"Bearer " + adminToken}}
	if rec := s.do(t, http.MethodPost, "/admin/reload", bearer); rec.Code != http.StatusInternalServerError {
		t.Fatalf("failed reload = %d %s", rec.Code, rec.Body)
	}
	rec = s.get(t, "/preview/"+previewToken+"/diagnostics")
	if !strings.Contains(rec.Body.String(), `"outcome":"failed"`) || !strings.Contains(rec.Body.String(), `"buildId":"`+s.store.Current().BuildID()) {
		t.Fatalf("diagnostics after a failed rebuild = %s", rec.Body)
	}
	envelope(t, s.get(t, "/articles/notes/first")) // last good snapshot still serves
}

func TestReload(t *testing.T) {
	s := newServer(t, previewToken, adminToken)
	before := s.store.Current().BuildID()
	for name, header := range map[string]http.Header{
		"missing":       nil,
		"wrong":         {"Authorization": {"Bearer nope"}},
		"preview token": {"Authorization": {"Bearer " + previewToken}},
		"not bearer":    {"Authorization": {"Basic " + adminToken}},
		"raw token":     {"Authorization": {adminToken}},
	} {
		t.Run(name, func(t *testing.T) {
			if rec := s.do(t, http.MethodPost, "/admin/reload", header); rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d", rec.Code)
			}
		})
	}
	if rec := s.do(t, http.MethodPost, "/admin/reload?token="+adminToken, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token in URL accepted: %d", rec.Code)
	}
	if s.store.Current().BuildID() != before {
		t.Fatal("an unauthorized reload rebuilt")
	}
	if rec := s.get(t, "/admin/reload"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET reload = %d", rec.Code)
	}

	s.source.set(contenttest.MemSource{".configuration": rootConfig, "Only.md": "Only.\n"})
	rec := s.do(t, http.MethodPost, "/admin/reload", http.Header{"Authorization": {"Bearer " + adminToken}})
	if rec.Code != http.StatusOK {
		t.Fatalf("reload = %d %s", rec.Code, rec.Body)
	}
	var got struct {
		BuildID          string
		Envelopes        int
		ValidationErrors int
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got, json.MatchCaseInsensitiveNames(true)); err != nil {
		t.Fatal(err)
	}
	if got.BuildID == before || got.BuildID != s.store.Current().BuildID() || got.Envelopes != 1 || got.ValidationErrors != 0 {
		t.Fatalf("reload = %+v, before %s", got, before)
	}
	envelope(t, s.get(t, "/articles/only"))
}

func TestReloadDisabledWithoutAdminToken(t *testing.T) {
	s := newServer(t, previewToken, "")
	for _, header := range []http.Header{nil, {"Authorization": {"Bearer "}}, {"Authorization": {"Bearer " + previewToken}}} {
		expectGenericNotFound(t, s.do(t, http.MethodPost, "/admin/reload", header))
	}
}

func TestRequestCorrelationAndLogs(t *testing.T) {
	s := newServer(t, previewToken, adminToken)
	rec := s.do(t, http.MethodGet, "/preview/"+previewToken+"/articles/essays/secret", http.Header{api.RequestIDHeader: {"next-req.42_a"}})
	if got := rec.Header().Get(api.RequestIDHeader); got != "next-req.42_a" {
		t.Fatalf("request ID = %q, want the incoming one", got)
	}
	s.do(t, http.MethodPost, "/admin/reload", http.Header{"Authorization": {"Bearer " + adminToken}})
	for _, bad := range []string{"", "has space", strings.Repeat("a", 129), "semi;colon"} {
		rec := s.do(t, http.MethodGet, "/manifest", http.Header{api.RequestIDHeader: {bad}})
		if got := rec.Header().Get(api.RequestIDHeader); got == bad || !regexpID(got) {
			t.Fatalf("invalid request ID %q produced %q", bad, got)
		}
	}

	logs := s.logs.String()
	if strings.Contains(logs, previewToken) || strings.Contains(logs, adminToken) {
		t.Fatalf("logs contain a token:\n%s", logs)
	}
	buildID := s.store.Current().BuildID()
	var found bool
	for line := range strings.SplitSeq(strings.TrimSpace(logs), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		if entry["request_id"] == "next-req.42_a" {
			found = entry["path"] == "/preview/<redacted>/articles/essays/secret" && entry["status"] == float64(200) && entry["build_id"] != ""
		}
		if entry["msg"] == "content rebuild succeeded" && entry["trigger"] == "reload" && entry["build_id"] != buildID {
			t.Fatalf("reload log names build %v, serving %s", entry["build_id"], buildID)
		}
	}
	if !found {
		t.Fatalf("no correlated, redacted request log:\n%s", logs)
	}
}

func regexpID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	return strings.Trim(id, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-") == ""
}
