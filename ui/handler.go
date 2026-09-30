package ui

import (
	"embed"
	"fmt"
	"hash/fnv"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// DocumentSource renders the OpenAPI document served by DocumentHandler.
// *core.API implements it.
type DocumentSource interface {
	JSON() ([]byte, error)
}

// defaultScalarURL pins the Scalar runtime to one reviewed release, so a new
// upstream publish can never change what runs on the docs page.
const defaultScalarURL = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.72.2"

// uiFS keeps the bundled docs UI assets inside the binary so /docs can serve a
// complete page without relying on extra static-file routes. Swagger/Base stay
// fully embedded; Scalar uses checked-in templates plus CDN-hosted runtime
// assets.
//
//go:embed theme/swagger theme/base theme/scalar
var uiFS embed.FS

// DocumentHandler serves source's JSON with an ETag, answering a matching
// If-None-Match with 304 so polling clients do not re-download the document.
func DocumentHandler(source DocumentSource) http.Handler {
	return documentHandler{source: source}
}

type documentHandler struct {
	source DocumentSource
}

func (h documentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, err := h.source.JSON()
	if err != nil {
		http.Error(w, "failed to render OpenAPI document", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	writeCacheable(w, r, raw, contentETag(raw), "no-cache")
}

// DocsHandler renders the docs page for config once and returns a handler
// serving it plus the provider's embedded assets.
func DocsHandler(config Config) (http.Handler, error) {
	if config.Title == "" {
		config.Title = "API documentation"
	}
	if config.DocsPath == "" {
		config.DocsPath = "/docs"
	}
	if config.DocumentURL == "" {
		config.DocumentURL = "/openapi.json"
	}

	html, err := render(config)
	if err != nil {
		return nil, err
	}
	page := []byte(html)
	return docsHandler{config: config, uiDir: resolveUIDir(config.Provider), page: page, etag: contentETag(page)}, nil
}

type docsHandler struct {
	config Config
	uiDir  string
	page   []byte
	etag   string
}

func (h docsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assetName, isAssetRequest := resolveUIRequest(r, h.config.DocsPath)
	if isAssetRequest {
		if !serveUIAsset(w, r, assetName, h.uiDir) {
			http.NotFound(w, r)
		}
		return
	}
	if assetName != "" {
		http.NotFound(w, r)
		return
	}

	if h.config.ContentSecurityPolicy != "" {
		w.Header().Set("Content-Security-Policy", h.config.ContentSecurityPolicy)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeCacheable(w, r, h.page, h.etag, "no-cache")
}

// writeCacheable writes body with the given validator, or 304 when the
// client already holds it.
func writeCacheable(w http.ResponseWriter, r *http.Request, body []byte, etag, cacheControl string) {
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", cacheControl)
	if r != nil && etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(body)
}

// etagMatches implements the If-None-Match list comparison (weak compare).
func etagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == etag || candidate == "*" {
			return true
		}
	}
	return false
}

// contentETag is a strong validator derived from the bytes; FNV is enough
// because it only has to detect change, not resist forgery.
func contentETag(body []byte) string {
	hash := fnv.New64a()
	_, _ = hash.Write(body)
	return fmt.Sprintf(`"%x"`, hash.Sum64())
}

func render(config Config) (string, error) {
	uiDir := resolveUIDir(config.Provider)
	switch config.Provider {
	case Base:
		return renderBase(config, uiDir)
	case Scalar:
		return renderScalar(config, uiDir)
	default:
		return renderSwagger(config, uiDir)
	}
}

// renderSwagger links the large Swagger bundles as versioned, immutable
// assets instead of inlining ~2 MB into every HTML response.
func renderSwagger(config Config, uiDir string) (string, error) {
	customCSS, err := readUIAsset(uiDir, "index.css")
	if err != nil {
		return "", err
	}
	initializer, err := readUIAsset(uiDir, "swagger-initializer.js")
	if err != nil {
		return "", err
	}

	// The bundled initializer ships with a placeholder petstore URL, so rewrite it
	// per request to keep the selected UI pointed at the caller's document route.
	initializer = strings.Replace(initializer, "https://petstore.swagger.io/v2/swagger.json", config.DocumentURL, 1)

	base := docsAssetBasePath(config.DocsPath)
	page := swaggerPageData{
		Title:         config.Title,
		SwaggerCSSURL: versionedAssetURL(base, uiDir, "swagger-ui.css"),
		CustomCSS:     template.CSS(customCSS),
		BundleURL:     versionedAssetURL(base, uiDir, "swagger-ui-bundle.js"),
		PresetURL:     versionedAssetURL(base, uiDir, "swagger-ui-standalone-preset.js"),
		Initializer:   template.JS(initializer),
		UserCSS:       customCSSBlock(config.CustomCSS),
	}

	var html strings.Builder
	if err := swaggerPageTemplate.Execute(&html, page); err != nil {
		return "", err
	}

	return html.String(), nil
}

func renderBase(config Config, uiDir string) (string, error) {
	indexTemplate, err := readUIAsset(uiDir, "index.html")
	if err != nil {
		return "", err
	}

	pageTemplate, err := template.New("base-page").Parse(indexTemplate)
	if err != nil {
		return "", err
	}

	page := basePageData{
		Title:            config.Title,
		DocumentURL:      config.DocumentURL,
		AssetBasePath:    docsAssetBasePath(config.DocsPath),
		DefaultLogo:      "img/logo.svg",
		DefaultFavicon16: "img/favicon-16x16.png",
		DefaultFavicon32: "img/favicon-32x32.png",
		CustomCSS:        customCSSBlock(config.CustomCSS),
	}

	var html strings.Builder
	if err := pageTemplate.Execute(&html, page); err != nil {
		return "", err
	}

	return html.String(), nil
}

func renderScalar(config Config, uiDir string) (string, error) {
	customCSS, err := readUIAsset(uiDir, "index.css")
	if err != nil {
		return "", err
	}
	initializer, err := readUIAsset(uiDir, "scalar-initializer.js")
	if err != nil {
		return "", err
	}

	cdnBaseURL := strings.TrimRight(config.CDNBaseURL, "/")
	if cdnBaseURL == "" {
		cdnBaseURL = defaultScalarURL
	}

	// html/template escapes Title and ScriptURL for their contexts itself;
	// pre-escaping them rendered "A & B" as "A &amp;amp; B".
	page := scalarPageData{
		Title:         config.Title,
		CustomCSS:     template.CSS(customCSS),
		ScriptURL:     cdnBaseURL,
		DocumentURL:   scalarDocumentURL(config.DocsPath, config.DocumentURL),
		InitializerJS: template.JS(initializer),
		UserCSS:       customCSSBlock(config.CustomCSS),
	}

	var html strings.Builder
	if err := scalarPageTemplate.Execute(&html, page); err != nil {
		return "", err
	}

	return html.String(), nil
}

type swaggerPageData struct {
	Title         string
	SwaggerCSSURL string
	CustomCSS     template.CSS
	BundleURL     string
	PresetURL     string
	Initializer   template.JS
	UserCSS       template.CSS
}

type basePageData struct {
	Title            string
	DocumentURL      string
	AssetBasePath    string
	DefaultLogo      string
	DefaultFavicon16 string
	DefaultFavicon32 string
	CustomCSS        template.CSS
}

type scalarPageData struct {
	Title         string
	CustomCSS     template.CSS
	ScriptURL     string
	DocumentURL   string
	InitializerJS template.JS
	UserCSS       template.CSS
}

// swaggerPageTemplate remains the inline HTML shell for Swagger-based providers.
// Scalar intentionally uses its own template and does not share this page shell.
var swaggerPageTemplate = template.Must(template.New("swagger-page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="{{.SwaggerCSSURL}}">
  <style>{{.CustomCSS}}</style>
  {{if .UserCSS}}<style id="docs-custom-css">{{.UserCSS}}</style>{{end}}
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="{{.BundleURL}}"></script>
  <script src="{{.PresetURL}}"></script>
  <script>{{.Initializer}}</script>
</body>
</html>`))

var scalarPageTemplate = template.Must(template.New("scalar-page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.Title}}</title>
  <style>{{.CustomCSS}}</style>
  {{if .UserCSS}}<style id="docs-custom-css">{{.UserCSS}}</style>{{end}}
</head>
<body>
  <div id="app"></div>
  <script src="{{.ScriptURL}}"></script>
  <script>
    window.__DOCS_DOCUMENT_URL__ = "{{.DocumentURL}}";
    {{.InitializerJS}}
  </script>
</body>
</html>`))

// styleCloseTag matches a closing </style> tag in any letter case.
var styleCloseTag = regexp.MustCompile(`(?i)</(style)`)

// customCSSBlock turns developer-supplied CSS into template-safe content for a
// <style> element. template.CSS is inserted verbatim, so a literal "</style"
// would end the element early and let the rest be parsed as HTML; escaping it
// as "<\/style" keeps it inert (inside a CSS string it still reads as "</style").
func customCSSBlock(css string) template.CSS {
	css = strings.TrimSpace(css)
	if css == "" {
		return ""
	}
	return template.CSS(styleCloseTag.ReplaceAllString(css, `<\/$1`))
}

func resolveUIDir(provider Provider) string {
	switch provider {
	case Scalar:
		return "theme/scalar"
	case Base:
		return "theme/base"
	default:
		return "theme/swagger"
	}
}

func readUIAsset(uiDir, name string) (string, error) {
	raw, err := fs.ReadFile(uiFS, uiDir+"/"+name)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// uiAsset is one embedded file with its precomputed response metadata.
type uiAsset struct {
	body        []byte
	etag        string
	contentType string
}

// uiAssets caches embedded files by path: fs.ReadFile copies the file and the
// ETag needs a full hash, neither of which should happen per request.
var uiAssets sync.Map

func loadUIAsset(uiDir, name string) (*uiAsset, bool) {
	key := uiDir + "/" + name
	if cached, ok := uiAssets.Load(key); ok {
		return cached.(*uiAsset), true
	}
	raw, err := fs.ReadFile(uiFS, key)
	if err != nil {
		return nil, false
	}
	contentType := uiAssetContentType(name)
	if contentType == "" {
		contentType = http.DetectContentType(raw)
	}
	asset, _ := uiAssets.LoadOrStore(key, &uiAsset{body: raw, etag: contentETag(raw), contentType: contentType})
	return asset.(*uiAsset), true
}

// versionedAssetURL appends the asset's content hash, so the URL changes
// whenever the embedded file does and can be cached as immutable.
func versionedAssetURL(base, uiDir, name string) string {
	asset, ok := loadUIAsset(uiDir, name)
	if !ok {
		return base + "/" + name
	}
	return base + "/" + name + "?v=" + strings.Trim(asset.etag, `"`)
}

// serveUIAsset writes an embedded asset. A request carrying the content hash
// (?v=) is cached for a year; others revalidate through the ETag.
func serveUIAsset(w http.ResponseWriter, r *http.Request, assetName, uiDir string) bool {
	cleanedName, ok := sanitizeUIAssetPath(assetName)
	if !ok {
		return false
	}
	asset, ok := loadUIAsset(uiDir, cleanedName)
	if !ok {
		return false
	}

	cacheControl := "no-cache"
	if r != nil && r.URL != nil && r.URL.Query().Get("v") == strings.Trim(asset.etag, `"`) {
		cacheControl = "public, max-age=31536000, immutable"
	}
	w.Header().Set("Content-Type", asset.contentType)
	writeCacheable(w, r, asset.body, asset.etag, cacheControl)
	return true
}

func scalarDocumentURL(docsPath, documentURL string) string {
	if documentURL == "" {
		return documentURL
	}
	if strings.HasPrefix(documentURL, "http://") || strings.HasPrefix(documentURL, "https://") || strings.HasPrefix(documentURL, "//") {
		return documentURL
	}
	if !strings.HasPrefix(documentURL, "/") {
		return documentURL
	}

	baseSegments := splitURLPath(strings.Trim(path.Clean(docsPath), "/"))
	targetSegments := splitURLPath(strings.Trim(path.Clean(documentURL), "/"))
	if len(baseSegments) == 0 {
		return "/" + strings.Join(targetSegments, "/")
	}

	common := 0
	for common < len(baseSegments) && common < len(targetSegments) && baseSegments[common] == targetSegments[common] {
		common++
	}

	relative := make([]string, 0, (len(baseSegments)-common)+(len(targetSegments)-common))
	for i := common; i < len(baseSegments); i++ {
		relative = append(relative, "..")
	}
	relative = append(relative, targetSegments[common:]...)
	if len(relative) == 0 {
		return "."
	}
	return strings.Join(relative, "/")
}

func docsAssetBasePath(docsPath string) string {
	cleaned := path.Clean("/" + docsPath)
	if cleaned == "." || cleaned == "/" {
		return ""
	}
	return cleaned
}

func splitURLPath(value string) []string {
	if value == "" || value == "." {
		return nil
	}
	parts := strings.Split(value, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

func resolveUIRequest(r *http.Request, docsPath string) (string, bool) {
	if r == nil || r.URL == nil {
		return "", false
	}
	if strings.Contains(r.URL.Path, "..") {
		return "..", true
	}

	cleanedPath := path.Clean("/" + r.URL.Path)
	docsRoot := path.Clean("/" + docsPath)
	if docsRoot == "." {
		docsRoot = "/"
	}

	switch {
	case cleanedPath == "/" || cleanedPath == docsRoot:
		return "", false
	case strings.HasPrefix(cleanedPath, docsRoot+"/"):
		cleanedPath = strings.TrimPrefix(cleanedPath, docsRoot)
	}

	assetName := strings.TrimPrefix(cleanedPath, "/")
	if assetName == "" {
		return "", false
	}
	return assetName, true
}

func uiAssetContentType(assetName string) string {
	// Embedded docs assets must use a deterministic JS MIME type because the Go
	// extension registry can resolve .js differently per host OS, which breaks
	// Base theme ES module loading when browsers receive text/plain.
	switch path.Ext(assetName) {
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	default:
		return mime.TypeByExtension(path.Ext(assetName))
	}
}

func sanitizeUIAssetPath(assetName string) (string, bool) {
	cleaned := strings.TrimPrefix(path.Clean("/"+assetName), "/")
	if cleaned == "" || cleaned == "." {
		return "", false
	}
	if strings.HasPrefix(cleaned, "..") {
		return "", false
	}

	parts := strings.Split(cleaned, "/")
	if slices.Contains(parts, "..") {
		return "", false
	}
	return cleaned, true
}
