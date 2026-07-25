package docs

import (
	"fmt"
	"html/template"
	"net/http"
)

type documentSource interface {
	JSON() ([]byte, error)
}

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
	var body string

	switch config.Provider {
	case "", Swagger:
		body = fmt.Sprintf(`<div id="swagger-ui"></div><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:%q,dom_id:'#swagger-ui'})</script>`, config.DocumentURL)
	case Scalar:
		body = fmt.Sprintf(`<script id="api-reference" data-url=%q></script><script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>`, config.DocumentURL)
	case Redoc:
		body = fmt.Sprintf(`<redoc spec-url=%q></redoc><script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>`, config.DocumentURL)
	default:
		return "", fmt.Errorf("openapi docs: unknown provider %q", config.Provider)
	}

	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title></head><body>%s</body></html>`, template.HTMLEscapeString(config.Title), body), nil
}
