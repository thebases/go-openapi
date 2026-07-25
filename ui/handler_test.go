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
	if !strings.Contains(body, "<div id=\"swagger-ui\"></div>") {
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
	if !strings.Contains(body, `url: "/merchant/openapi.json"`) {
		t.Fatalf("expected base UI to receive document URL, got %q", body)
	}
	if !strings.Contains(body, "<title>Merchant API</title>") {
		t.Fatalf("expected configured title, got %q", body)
	}
	if !strings.Contains(body, "base-docs-shell") || !strings.Contains(body, "#operation-") {
		t.Fatalf("expected base UI layout script, got %q", body)
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
	if !strings.Contains(body, `window.__DOCS_DOCUMENT_URL__ = `) || !strings.Contains(body, `openapi.json`) {
		t.Fatalf("expected scalar UI to receive docs-relative document URL, got %q", body)
	}
	if !strings.Contains(body, "<title>Scalar API</title>") {
		t.Fatalf("expected configured title, got %q", body)
	}
	if !strings.Contains(body, "Scalar.createApiReference('#app'") {
		t.Fatalf("expected scalar UI initializer, got %q", body)
	}
	if !strings.Contains(body, "https://cdn.jsdelivr.net/npm/@scalar/api-reference") {
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

func TestDocsHandlerRejectsNestedAssetTraversal(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Swagger})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/theme/swagger/logo.svg", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nested asset path, got %d", recorder.Code)
	}
}
