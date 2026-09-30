package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubRoute is an Adapter test double that records registrations and docs
// mounts.
type stubRoute struct {
	method     string
	path       string
	calls      int
	mountCalls int
}

func (r *stubRoute) Register(method, path string, handlers ...string) error {
	r.method = method
	r.path = path
	r.calls++
	return nil
}

func (r *stubRoute) MountDocs(DocsMount) error {
	r.mountCalls++
	return nil
}

func stubOperation() Operation {
	return Operation{Responses: map[string]ResponseOrReference{"200": JSONResponse("ok", StringSchema())}}
}

func TestHandleRegistersOperationAndRoute(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubRoute{}

	if err := Handle[string](api, router, Route("/merchants/:id", stubOperation()).WithMethod(http.MethodGet), "handler"); err != nil {
		t.Fatalf("register route: %v", err)
	}
	if router.calls != 1 {
		t.Fatalf("expected 1 route registration, got %d", router.calls)
	}
	if router.method != http.MethodGet {
		t.Fatalf("unexpected method: %s", router.method)
	}
	// The native router keeps its own syntax; only the document is canonical.
	if router.path != "/merchants/:id" {
		t.Fatalf("unexpected path: %s", router.path)
	}
	if api.Document().Paths["/merchants/{id}"] == nil {
		t.Fatal("expected canonical path in document")
	}
}

func TestHandleAutoMountsDocsOncePerRouter(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"), WithDocStyle(DocsSwagger))
	router := &stubRoute{}

	if err := Handle[string](api, router, Route("/merchants/:id", stubOperation()).WithMethod(http.MethodGet), "handler"); err != nil {
		t.Fatalf("first register route: %v", err)
	}
	if err := Handle[string](api, router, Route("/merchants", stubOperation()).WithMethod(http.MethodPost), "handler"); err != nil {
		t.Fatalf("second register route: %v", err)
	}
	if router.mountCalls != 1 {
		t.Fatalf("expected docs to mount once, got %d", router.mountCalls)
	}
}

func TestHandleSkipsAutoMountWithoutDocStyle(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubRoute{}

	if err := Handle[string](api, router, Route("/merchants/:id", stubOperation()).WithMethod(http.MethodGet), "handler"); err != nil {
		t.Fatalf("register route: %v", err)
	}
	if router.mountCalls != 0 {
		t.Fatalf("expected docs to stay disabled, got %d mounts", router.mountCalls)
	}
}

func TestPrepareDocsMountDefaults(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	mount, err := prepareDocsMount(api, "", "", DocsConfig{Provider: DocsSwagger, Title: "Merchant API"})
	if err != nil {
		t.Fatalf("prepare docs mount: %v", err)
	}
	if mount.DocsPath != "/docs" {
		t.Fatalf("unexpected docs path: %s", mount.DocsPath)
	}
	if mount.DocumentPath != "/openapi.json" {
		t.Fatalf("unexpected document path: %s", mount.DocumentPath)
	}
	if mount.Docs == nil || mount.Document == nil {
		t.Fatal("expected docs handlers")
	}
}

func TestPrepareDocsMountUsesAPIStyleByDefault(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"), WithDocStyle(DocsBase))
	mount, err := prepareDocsMount(api, "/docs", "/openapi.json", DocsConfig{})
	docsHandler := mount.Docs
	if err != nil {
		t.Fatalf("prepare docs mount: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	docsHandler.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, `window.__DOCS_DOCUMENT_URL__ = `) || !strings.Contains(body, `openapi.json`) {
		t.Fatalf("expected inherited docs style to keep configured document URL, got %q", body)
	}
	if !strings.Contains(body, "<title>Merchant API</title>") {
		t.Fatalf("expected docs title, got %q", body)
	}
}

func TestPrepareDocsMountAllowsPerMountProviderOverride(t *testing.T) {
	api := New(WithDocStyle(DocsBase))
	mount, err := prepareDocsMount(api, "/docs", "/openapi.json", DocsConfig{Provider: DocsScalar, Title: "Override API"})
	docsHandler := mount.Docs
	if err != nil {
		t.Fatalf("prepare docs mount: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	docsHandler.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "<title>Override API</title>") {
		t.Fatalf("expected override docs title, got %q", body)
	}
	if !strings.Contains(body, `window.__DOCS_DOCUMENT_URL__ = `) || !strings.Contains(body, `openapi.json`) {
		t.Fatalf("expected scalar docs to use docs-relative document URL, got %q", body)
	}
}

func TestPrepareDocsMountCustomCSSInheritanceAndOverride(t *testing.T) {
	api := New(WithDocStyle(DocsBase), WithCustomCSS(".api-level{}"))
	cases := []struct {
		name   string
		config DocsConfig
		want   string
	}{
		{name: "inherits API css", config: DocsConfig{}, want: ".api-level{}"},
		{name: "per-mount override", config: DocsConfig{CustomCSS: ".mount-level{}"}, want: ".mount-level{}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mount, err := prepareDocsMount(api, "/docs", "/openapi.json", tc.config)
			docsHandler := mount.Docs
			if err != nil {
				t.Fatalf("prepare docs mount: %v", err)
			}

			recorder := httptest.NewRecorder()
			docsHandler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs/", nil))
			body := recorder.Body.String()
			if !strings.Contains(body, `<style id="docs-custom-css">`+tc.want+`</style>`) {
				t.Fatalf("expected custom css %q, got %q", tc.want, body)
			}
		})
	}
}

func TestPrepareDocsMountServesDocsAssetsUnderMountPath(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"), WithDocStyle(DocsSwagger))
	mount, err := prepareDocsMount(api, "/docs", "/openapi.json", DocsConfig{})
	docsHandler := mount.Docs
	if err != nil {
		t.Fatalf("prepare docs mount: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs/logo.svg", nil)
	docsHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected OK for mounted docs asset, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Header().Get("Content-Type"), "image/svg+xml") {
		t.Fatalf("expected svg content type, got %q", recorder.Header().Get("Content-Type"))
	}
	if !strings.Contains(recorder.Body.String(), "<svg") {
		t.Fatalf("expected svg body, got %q", recorder.Body.String())
	}
}
