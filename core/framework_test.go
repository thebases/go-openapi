package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubRoute struct {
	method string
	path   string
	calls  int
}

func TestRouteRegistrarRegistersOperationAndRoute(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubRoute{}
	registrar := RouteRegistrar[*stubRoute, string]{
		Register: func(router *stubRoute, method, path string, handlers ...string) {
			router.method = method
			router.path = path
			router.calls++
		},
	}

	err := registrar.GET(router, api, Route("/merchants/:id", Operation{
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("ok", StringSchema()),
		},
	}))
	if err != nil {
		t.Fatalf("register route: %v", err)
	}
	if router.calls != 1 {
		t.Fatalf("expected 1 route registration, got %d", router.calls)
	}
	if router.method != http.MethodGet {
		t.Fatalf("unexpected method: %s", router.method)
	}
	if router.path != "/merchants/:id" {
		t.Fatalf("unexpected path: %s", router.path)
	}
	if api.Document().Paths["/merchants/{id}"] == nil {
		t.Fatal("expected canonical path in document")
	}
}

func TestRouteRegistrarAutoMountsDocsOnce(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"), WithDocStyle(DocsSwagger))
	router := &stubRoute{}
	mountCalls := 0
	registrar := RouteRegistrar[*stubRoute, string]{
		Register: func(router *stubRoute, method, path string, handlers ...string) {
			router.calls++
		},
		MountDocs: func(router *stubRoute, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
			mountCalls++
			return nil
		},
	}

	operation := Operation{
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("ok", StringSchema()),
		},
	}
	if err := registrar.GET(router, api, Route("/merchants/:id", operation)); err != nil {
		t.Fatalf("first register route: %v", err)
	}
	if err := registrar.POST(router, api, Route("/merchants", operation)); err != nil {
		t.Fatalf("second register route: %v", err)
	}
	if mountCalls != 1 {
		t.Fatalf("expected docs to mount once, got %d", mountCalls)
	}
}

func TestRouteRegistrarSkipsAutoMountWithoutDocStyle(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubRoute{}
	mountCalls := 0
	registrar := RouteRegistrar[*stubRoute, string]{
		Register: func(router *stubRoute, method, path string, handlers ...string) {
			router.calls++
		},
		MountDocs: func(router *stubRoute, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
			mountCalls++
			return nil
		},
	}

	err := registrar.GET(router, api, Route("/merchants/:id", Operation{
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("ok", StringSchema()),
		},
	}))
	if err != nil {
		t.Fatalf("register route: %v", err)
	}
	if mountCalls != 0 {
		t.Fatalf("expected docs to stay disabled, got %d mounts", mountCalls)
	}
}

func TestPrepareDocsMountDefaults(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	docsPath, documentPath, docsHandler, documentHandler, err := prepareDocsMount(api, "", "", DocsConfig{Provider: DocsSwagger, Title: "Merchant API"})
	if err != nil {
		t.Fatalf("prepare docs mount: %v", err)
	}
	if docsPath != "/docs" {
		t.Fatalf("unexpected docs path: %s", docsPath)
	}
	if documentPath != "/openapi.json" {
		t.Fatalf("unexpected document path: %s", documentPath)
	}
	if docsHandler == nil || documentHandler == nil {
		t.Fatal("expected docs handlers")
	}
}

func TestPrepareDocsMountUsesAPIStyleByDefault(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"), WithDocStyle(DocsBase))
	_, _, docsHandler, _, err := prepareDocsMount(api, "/docs", "/openapi.json", DocsConfig{})
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
	_, _, docsHandler, _, err := prepareDocsMount(api, "/docs", "/openapi.json", DocsConfig{Provider: DocsScalar, Title: "Override API"})
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

func TestPrepareDocsMountServesDocsAssetsUnderMountPath(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"), WithDocStyle(DocsSwagger))
	_, _, docsHandler, _, err := prepareDocsMount(api, "/docs", "/openapi.json", DocsConfig{})
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
