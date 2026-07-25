package openapifiber

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestMountDocsPreservesSingleCSSContentType(t *testing.T) {
	app := fiber.New()
	cssHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write([]byte("body{color:red}"))
	})
	jsonHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{}`))
	})

	if err := mountDocs(app, "/docs", "/openapi.json", cssHandler, jsonHandler); err != nil {
		t.Fatalf("mount docs: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/css/app.css", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request docs asset: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Values("Content-Type"); len(got) != 1 || got[0] != "text/css; charset=utf-8" {
		t.Fatalf("expected one css content type, got %v", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if strings.TrimSpace(string(body)) != "body{color:red}" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestMountDocsServesDocumentAliasUnderDocsPath(t *testing.T) {
	app := fiber.New()
	docsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	jsonHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"openapi":"3.0.3"}`))
	})

	if err := mountDocs(app, "/docs", "/openapi.json", docsHandler, jsonHandler); err != nil {
		t.Fatalf("mount docs: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request docs document alias: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected OK, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if strings.TrimSpace(string(body)) != `{"openapi":"3.0.3"}` {
		t.Fatalf("unexpected body: %q", string(body))
	}
}
