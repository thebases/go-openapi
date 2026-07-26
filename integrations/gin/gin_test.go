package openapigin

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMountDocsServesDocumentAliasWithoutGinWildcardConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	err := mountDocs(
		router,
		"/docs",
		"/openapi.json",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("docs-ui"))
		}),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"openapi":"3.0.3"}`))
		}),
	)
	if err != nil {
		t.Fatalf("mount docs: %v", err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected alias status: %d", response.Code)
	}
	if response.Body.String() != `{"openapi":"3.0.3"}` {
		t.Fatalf("unexpected alias body: %q", response.Body.String())
	}
}

func TestMountDocsServesDocsAssetsAfterAliasWrapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	err := mountDocs(
		router,
		"/docs",
		"/merchant/spec.json",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("docs-ui"))
		}),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"openapi":"3.0.3"}`))
		}),
	)
	if err != nil {
		t.Fatalf("mount docs: %v", err)
	}

	aliasResponse := httptest.NewRecorder()
	router.ServeHTTP(aliasResponse, httptest.NewRequest(http.MethodGet, "/docs/spec.json", nil))
	if aliasResponse.Code != http.StatusOK {
		t.Fatalf("unexpected renamed alias status: %d", aliasResponse.Code)
	}
	if aliasResponse.Body.String() != `{"openapi":"3.0.3"}` {
		t.Fatalf("unexpected renamed alias body: %q", aliasResponse.Body.String())
	}

	docsResponse := httptest.NewRecorder()
	router.ServeHTTP(docsResponse, httptest.NewRequest(http.MethodGet, "/docs/logo.svg", nil))
	if docsResponse.Code != http.StatusOK {
		t.Fatalf("unexpected docs asset status: %d", docsResponse.Code)
	}
	body, err := io.ReadAll(docsResponse.Body)
	if err != nil {
		t.Fatalf("read docs asset body: %v", err)
	}
	if string(body) != "docs-ui" {
		t.Fatalf("unexpected docs asset body: %q", string(body))
	}
}
