package docs

import (
	"embed"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"slices"
	"strings"
)

type documentSource interface {
	JSON() ([]byte, error)
}

// uiFS keeps the bundled docs UI assets inside the binary so /docs can serve a
// complete page without relying on extra static-file routes. Swagger/Base stay
// fully embedded; Scalar uses checked-in templates plus CDN-hosted runtime
// assets.
//
//go:embed theme/swagger/* theme/base/* theme/scalar/*
var uiFS embed.FS

func DocumentHandler(source documentSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		raw, err := source.JSON()
		if err != nil {
			http.Error(w, "failed to render OpenAPI document", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(raw)
	})
}

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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assetName, isAssetRequest := resolveUIRequest(r, config.DocsPath)
		if isAssetRequest {
			if serveUIAsset(w, assetName, resolveUIDir(config.Provider)) {
				return
			}
			http.NotFound(w, r)
			return
		}
		if assetName != "" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}), nil
}

func render(config Config) (string, error) {
	uiDir := resolveUIDir(config.Provider)
	if config.Provider == Base {
		return renderBase(config, uiDir)
	}
	if config.Provider == Scalar {
		return renderScalar(config, uiDir)
	}

	swaggerCSS, err := readUIAsset(uiDir, "swagger-ui.css")
	if err != nil {
		return "", err
	}
	customCSS, err := readUIAsset(uiDir, "index.css")
	if err != nil {
		return "", err
	}
	swaggerBundle, err := readUIAsset(uiDir, "swagger-ui-bundle.js")
	if err != nil {
		return "", err
	}
	standalonePreset, err := readUIAsset(uiDir, "swagger-ui-standalone-preset.js")
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

	page := swaggerPageData{
		Title:            template.HTMLEscapeString(config.Title),
		SwaggerCSS:       template.CSS(swaggerCSS),
		CustomCSS:        template.CSS(customCSS),
		SwaggerBundle:    template.JS(swaggerBundle),
		StandalonePreset: template.JS(standalonePreset),
		Initializer:      template.JS(initializer),
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
		DefaultLogo:      "img/logo.svg",
		DefaultFavicon16: "img/favicon-16x16.png",
		DefaultFavicon32: "img/favicon-32x32.png",
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
		cdnBaseURL = "https://cdn.jsdelivr.net/npm/@scalar/api-reference"
	}

	page := scalarPageData{
		Title:         template.HTMLEscapeString(config.Title),
		CustomCSS:     template.CSS(customCSS),
		ScriptURL:     template.HTMLEscapeString(cdnBaseURL),
		DocumentURL:   scalarDocumentURL(config.DocsPath, config.DocumentURL),
		InitializerJS: template.JS(initializer),
	}

	var html strings.Builder
	if err := scalarPageTemplate.Execute(&html, page); err != nil {
		return "", err
	}

	return html.String(), nil
}

type swaggerPageData struct {
	Title            string
	SwaggerCSS       template.CSS
	CustomCSS        template.CSS
	SwaggerBundle    template.JS
	StandalonePreset template.JS
	Initializer      template.JS
}

type basePageData struct {
	Title            string
	DocumentURL      string
	DefaultLogo      string
	DefaultFavicon16 string
	DefaultFavicon32 string
}

type scalarPageData struct {
	Title         string
	CustomCSS     template.CSS
	ScriptURL     string
	DocumentURL   string
	InitializerJS template.JS
}

// swaggerPageTemplate remains the inline HTML shell for Swagger-based providers.
// Scalar intentionally uses its own template and does not share this page shell.
var swaggerPageTemplate = template.Must(template.New("swagger-page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.Title}}</title>
  <style>{{.SwaggerCSS}}</style>
  <style>{{.CustomCSS}}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script>{{.SwaggerBundle}}</script>
  <script>{{.StandalonePreset}}</script>
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

func resolveUIDir(provider Provider) string {
	switch provider {
	case Scalar:
		return "theme/scalar"
	case Base:
		return "theme/base"
	case Swagger, "":
		return "theme/swagger"
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

func serveUIAsset(w http.ResponseWriter, assetName, uiDir string) bool {
	cleanedName, ok := sanitizeUIAssetPath(assetName)
	if !ok {
		return false
	}

	raw, err := fs.ReadFile(uiFS, uiDir+"/"+cleanedName)
	if err != nil {
		return false
	}

	contentType := uiAssetContentType(cleanedName)
	if contentType == "" {
		contentType = http.DetectContentType(raw)
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(raw)
	return true
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
