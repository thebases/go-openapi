package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsHandlerFallsBackToSwaggerUI(t *testing.T) {
	handler, err := DocsHandler(Config{
		Provider:    Provider("unknown-ui"),
		Title:       "Merchant API",
		DocumentURL: "/openapi.json",
	})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/docs", nil))

	body := recorder.Body.String()
	if !strings.Contains(body, `url: "/openapi.json"`) {
		t.Fatalf("expected rewritten document URL, got %q", body)
	}
	if !strings.Contains(body, `<div id="swagger-ui"></div>`) {
		t.Fatalf("expected embedded swagger container, got %q", body)
	}
}

func TestDocsHandlerSupportsBaseUI(t *testing.T) {
	handler, err := DocsHandler(Config{
		Provider:    Base,
		Title:       "Merchant API",
		DocumentURL: "/merchant/openapi.json",
	})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/docs", nil))

	body := recorder.Body.String()
	if !strings.Contains(body, `Base API`) || !strings.Contains(body, `Loading API documentation`) {
		t.Fatalf("expected base UI shell content, got %q", body)
	}
	if !strings.Contains(body, `<title>Merchant API</title>`) {
		t.Fatalf("expected configured title, got %q", body)
	}
	if !strings.Contains(body, `src="/docs/js/openapi.js"`) || !strings.Contains(body, `src="/docs/js/snippet.js"`) || !strings.Contains(body, `src="/docs/js/app.js"`) {
		t.Fatalf("expected base UI shell assets, got %q", body)
	}
	if !strings.Contains(body, `window.__DOCS_DOCUMENT_URL__ = `) || !strings.Contains(body, `merchant\/openapi.json`) {
		t.Fatalf("expected base UI document URL bootstrap, got %q", body)
	}
	if !strings.Contains(body, `href="https://cdn.jsdelivr.net/npm/basecoat-css@1.0.2/dist/basecoat-maia.cdn.min.css"`) {
		t.Fatalf("expected base UI CDN stylesheet link, got %q", body)
	}
	if !strings.Contains(body, `onerror="this.onerror=null;this.href='\/docs/css/basecoat-maia.cdn.min.css';"`) {
		t.Fatalf("expected base UI local stylesheet fallback, got %q", body)
	}
	if !strings.Contains(body, `href="/docs/css/app.css"`) {
		t.Fatalf("expected base UI app stylesheet link, got %q", body)
	}
	if strings.Index(body, `basecoat-maia.cdn.min.css`) > strings.Index(body, `/docs/css/app.css`) {
		t.Fatalf("expected app.css link to appear after Basecoat, got %q", body)
	}
	if strings.Contains(body, `type="module"`) {
		t.Fatalf("expected base UI app shell to use a plain script tag, got %q", body)
	}
	if strings.Contains(body, `appendAppCss`) {
		t.Fatalf("expected base UI to stop relying on runtime stylesheet append order, got %q", body)
	}
}
func TestDocsHandlerSupportsScalarProvider(t *testing.T) {
	handler, err := DocsHandler(Config{
		Provider:    Scalar,
		Title:       "Scalar API",
		DocsPath:    "/docs",
		DocumentURL: "/scalar/openapi.json",
	})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/docs", nil))

	body := recorder.Body.String()
	if !strings.Contains(body, `window.__DOCS_DOCUMENT_URL__ = `) || !strings.Contains(body, `scalar\/openapi.json`) {
		t.Fatalf("expected scalar UI to receive docs-relative document URL, got %q", body)
	}
	if !strings.Contains(body, `<title>Scalar API</title>`) {
		t.Fatalf("expected configured title, got %q", body)
	}
	if !strings.Contains(body, `Scalar.createApiReference('#app'`) {
		t.Fatalf("expected scalar UI initializer, got %q", body)
	}
	if !strings.Contains(body, `https://cdn.jsdelivr.net/npm/@scalar/api-reference`) {
		t.Fatalf("expected scalar CDN script, got %q", body)
	}
}

func TestDocsHandlerServesEmbeddedImageAssets(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Swagger})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/logo.svg", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected OK for embedded asset, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Header().Get("Content-Type"), "image/svg+xml") {
		t.Fatalf("expected svg content type, got %q", recorder.Header().Get("Content-Type"))
	}
	if !strings.Contains(recorder.Body.String(), "<svg") {
		t.Fatalf("expected svg body, got %q", recorder.Body.String())
	}
}

func TestDocsHandlerServesNestedBaseAssets(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Base})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/js/app.js", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected OK for base asset, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/javascript; charset=utf-8" {
		t.Fatalf("expected stable ES module content type, got %q", got)
	}
	if !strings.Contains(recorder.Body.String(), "const SPEC_URL =") || !strings.Contains(recorder.Body.String(), "loadOpenApiDoc") {
		t.Fatalf("expected base app bundle script, got %q", recorder.Body.String())
	}
}

func TestDocsHandlerRejectsNestedAssetTraversal(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Swagger})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/../logo.svg", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for traversal path, got %d", recorder.Code)
	}
}


