package docs

import (
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
}

func TestDocsHandlerSupportsScalarProvider(t *testing.T) {
	handler, err := DocsHandler(Config{
		Provider:    Scalar,
		Title:       "Scalar API",
		DocumentURL: "/scalar/openapi.json",
	})
	if err != nil {
		t.Fatalf("create docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/docs", nil))

	body := recorder.Body.String()
	if !strings.Contains(body, `url: "/scalar/openapi.json"`) {
		t.Fatalf("expected scalar UI to receive document URL, got %q", body)
	}
	if !strings.Contains(body, "<title>Scalar API</title>") {
		t.Fatalf("expected configured title, got %q", body)
	}
}
