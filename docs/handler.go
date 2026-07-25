package docs

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

type documentSource interface {
	JSON() ([]byte, error)
}

// uiFS keeps the bundled docs UIs inside the binary so /docs can serve a complete
// page without relying on extra static-file routes or remote CDNs.
//
//go:embed ui/swagger/* ui/base/*
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
	if config.DocumentURL == "" {
		config.DocumentURL = "/openapi.json"
	}

	html, err := render(config)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}), nil
}

func render(config Config) (string, error) {
	uiDir := resolveUIDir(config.Provider)

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

	page := docsPageData{
		Title:            template.HTMLEscapeString(config.Title),
		SwaggerCSS:       template.CSS(swaggerCSS),
		CustomCSS:        template.CSS(customCSS),
		SwaggerBundle:    template.JS(swaggerBundle),
		StandalonePreset: template.JS(standalonePreset),
		Initializer:      template.JS(initializer),
	}

	var html strings.Builder
	if err := docsPageTemplate.Execute(&html, page); err != nil {
		return "", err
	}

	return html.String(), nil
}

type docsPageData struct {
	Title            string
	SwaggerCSS       template.CSS
	CustomCSS        template.CSS
	SwaggerBundle    template.JS
	StandalonePreset template.JS
	Initializer      template.JS
}

var docsPageTemplate = template.Must(template.New("docs-page").Parse(`<!doctype html>
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

func resolveUIDir(provider Provider) string {
	switch provider {
	case Base, Scalar:
		return "ui/base"
	case Swagger, "":
		return "ui/swagger"
	default:
		return "ui/swagger"
	}
}

func readUIAsset(uiDir, name string) (string, error) {
	raw, err := fs.ReadFile(uiFS, uiDir+"/"+name)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
