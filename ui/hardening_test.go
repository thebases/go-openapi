package ui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

type staticSource []byte

func (s staticSource) JSON() ([]byte, error) { return s, nil }

func serve(t *testing.T, h http.Handler, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	for key, values := range header {
		request.Header[key] = values
	}
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	return recorder
}

func TestTitleIsEscapedExactlyOnce(t *testing.T) {
	for _, provider := range []Provider{Swagger, Scalar, Base} {
		handler, err := DocsHandler(Config{Provider: provider, Title: "A & B"})
		if err != nil {
			t.Fatal(err)
		}
		body := serve(t, handler, "/docs", nil).Body.String()
		if !strings.Contains(body, "<title>A &amp; B</title>") {
			t.Errorf("%s: title not escaped once: %q", provider, regexp.MustCompile(`<title>.*</title>`).FindString(body))
		}
	}
}

func TestSwaggerLinksVersionedImmutableAssets(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Swagger, DocsPath: "/docs"})
	if err != nil {
		t.Fatal(err)
	}
	page := serve(t, handler, "/docs", nil)
	if page.Body.Len() > 16<<10 {
		t.Fatalf("swagger page should link its bundles, got %d bytes", page.Body.Len())
	}
	src := regexp.MustCompile(`src="(/docs/swagger-ui-bundle\.js\?v=[0-9a-f]+)"`).FindStringSubmatch(page.Body.String())
	if src == nil {
		t.Fatalf("missing versioned bundle link in %q", page.Body.String())
	}

	asset := serve(t, handler, src[1], nil)
	if asset.Code != http.StatusOK || !strings.Contains(asset.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("versioned asset: %d %q", asset.Code, asset.Header().Get("Cache-Control"))
	}
	revalidated := serve(t, handler, "/docs/swagger-ui-bundle.js", http.Header{"If-None-Match": {asset.Header().Get("ETag")}})
	if revalidated.Code != http.StatusNotModified || revalidated.Body.Len() != 0 {
		t.Fatalf("expected 304 for matching ETag, got %d", revalidated.Code)
	}
}

func TestDocumentHandlerSupportsConditionalRequests(t *testing.T) {
	handler := DocumentHandler(staticSource(`{"openapi":"3.2.0"}`))
	first := serve(t, handler, "/openapi.json", nil)
	etag := first.Header().Get("ETag")
	if first.Code != http.StatusOK || etag == "" {
		t.Fatalf("first response: %d etag=%q", first.Code, etag)
	}
	if second := serve(t, handler, "/openapi.json", http.Header{"If-None-Match": {etag}}); second.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", second.Code)
	}
}

func TestContentSecurityPolicyHeader(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Base, ContentSecurityPolicy: "default-src 'self'"})
	if err != nil {
		t.Fatal(err)
	}
	if got := serve(t, handler, "/docs", nil).Header().Get("Content-Security-Policy"); got != "default-src 'self'" {
		t.Fatalf("unexpected CSP %q", got)
	}
}

func TestScalarRuntimeIsPinned(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Scalar})
	if err != nil {
		t.Fatal(err)
	}
	if body := serve(t, handler, "/docs", nil).Body.String(); !strings.Contains(body, `src="`+defaultScalarURL+`"`) {
		t.Fatalf("scalar script not pinned: %q", body)
	}
}

func TestAssetTraversalIsRejected(t *testing.T) {
	handler, err := DocsHandler(Config{Provider: Base, DocsPath: "/docs"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/docs/../handler.go", "/docs/%2e%2e/handler.go", "/docs/css/../../config.go"} {
		if code := serve(t, handler, path, nil).Code; code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", path, code)
		}
	}
}

func FuzzSanitizeUIAssetPathNeverEscapes(f *testing.F) {
	for _, seed := range []string{"css/app.css", "../x", "a/../../b", "..", "./a"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		cleaned, ok := sanitizeUIAssetPath(name)
		if ok && (strings.HasPrefix(cleaned, "/") || strings.Contains("/"+cleaned+"/", "/../")) {
			t.Fatalf("sanitize(%q) = %q escapes the theme directory", name, cleaned)
		}
	})
}
