package openapichi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	core "github.com/thebases/go-openapi/core"
)

func TestMountDocsServesDefaultDocumentAndDocsAlias(t *testing.T) {
	router := chi.NewRouter()
	api := core.New(
		core.WithTitle("Chi Test API"),
		core.WithVersion("1.0.0"),
		core.WithDocStyle(core.DocsSwagger),
	)

	if err := MountDocs(router, api, "/docs", "/openapi.json", core.DocsConfig{}); err != nil {
		t.Fatalf("mount docs: %v", err)
	}

	testCases := []struct {
		path        string
		statusCode  int
		contentType string
	}{
		{path: "/openapi.json", statusCode: http.StatusOK, contentType: "application/json"},
		{path: "/docs/openapi.json", statusCode: http.StatusOK, contentType: "application/json"},
		{path: "/docs/", statusCode: http.StatusOK, contentType: "text/html"},
		{path: "/docs/logo.svg", statusCode: http.StatusOK, contentType: "image/svg+xml"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			router.ServeHTTP(recorder, request)

			if recorder.Code != tc.statusCode {
				t.Fatalf("unexpected status for %s: got %d want %d", tc.path, recorder.Code, tc.statusCode)
			}
			if got := recorder.Header().Get("Content-Type"); got == "" || got[:len(tc.contentType)] != tc.contentType {
				t.Fatalf("unexpected content type for %s: got %q want prefix %q", tc.path, got, tc.contentType)
			}
		})
	}
}

func TestMountDocsRedirectsBareDocsPath(t *testing.T) {
	router := chi.NewRouter()
	api := core.New(
		core.WithTitle("Chi Test API"),
		core.WithVersion("1.0.0"),
		core.WithDocStyle(core.DocsSwagger),
	)

	if err := MountDocs(router, api, "/docs", "/openapi.json", core.DocsConfig{}); err != nil {
		t.Fatalf("mount docs: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMovedPermanently {
		t.Fatalf("unexpected redirect status: got %d want %d", recorder.Code, http.StatusMovedPermanently)
	}
	if got := recorder.Header().Get("Location"); got != "/docs/" {
		t.Fatalf("unexpected redirect location: got %q want %q", got, "/docs/")
	}
}

func TestHandleRegistersRouteAndReturnsHandlerResponse(t *testing.T) {
	router := chi.NewRouter()
	api := core.New(
		core.WithTitle("Chi Test API"),
		core.WithVersion("1.0.0"),
		core.WithDocStyle(core.DocsSwagger),
	)

	err := GET(router, api, core.Route("/merchants/{id}", core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}), func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chi.URLParam(r, "id")))
	})
	if err != nil {
		t.Fatalf("register chi route: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/merchants/123", nil)
	router.ServeHTTP(recorder, request)

	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected route status: got %d want %d", recorder.Code, http.StatusOK)
	}
	if string(body) != "123" {
		t.Fatalf("unexpected route body: got %q want %q", string(body), "123")
	}
}
