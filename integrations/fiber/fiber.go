package openapifiber

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
	core "github.com/thebases/go-openapi/core"
)

func mountDocs(router fiber.Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	// Replay standard net/http handlers into Fiber so framework integrations do
	// not need to duplicate docs generation behavior.
	serveHTTP := func(c fiber.Ctx, handler http.Handler) error {
		request := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: c.Path()}}
		recorder := &fiberHTTPWriter{header: http.Header{}}
		handler.ServeHTTP(recorder, request)
		for key, values := range recorder.header {
			if len(values) == 0 {
				continue
			}
			if strings.EqualFold(key, "Set-Cookie") {
				for _, value := range values {
					c.Append(key, value)
				}
				continue
			}
			c.Set(key, values[0])
		}
		return c.Status(recorder.statusOrOK()).Send(recorder.body)
	}

	router.Get(documentPath, func(c fiber.Ctx) error {
		return serveHTTP(c, documentHandler)
	})
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Keep a docs-scoped alias for the default document URL so requests under
		// /docs do not fall through to the docs asset handler and return 404.
		router.Get(aliasPath, func(c fiber.Ctx) error {
			return serveHTTP(c, documentHandler)
		})
	}
	router.Get(docsPath, func(c fiber.Ctx) error {
		return serveHTTP(c, docsHandler)
	})
	router.Get(docsPath+"/*", func(c fiber.Ctx) error {
		return serveHTTP(c, docsHandler)
	})
	return nil
}

var routes = core.RouteRegistrar[fiber.Router, fiber.Handler]{
	Register: func(router fiber.Router, method, path string, handlers ...fiber.Handler) {
		if len(handlers) == 0 {
			return
		}

		// Fiber v3 requires one fixed handler argument before the variadic tail.
		tail := make([]any, 0, len(handlers)-1)
		for _, handler := range handlers[1:] {
			tail = append(tail, handler)
		}

		router.Add([]string{method}, path, handlers[0], tail...)
	},
	MountDocs: mountDocs,
}

var docs = core.DocsRegistrar[fiber.Router]{
	Mount: mountDocs,
}

func Handle(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...fiber.Handler) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...fiber.Handler) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...fiber.Handler) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...fiber.Handler) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...fiber.Handler) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...fiber.Handler) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router fiber.Router, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
}

type fiberHTTPWriter struct {
	header http.Header
	body   []byte
	status int
}

func (w *fiberHTTPWriter) Header() http.Header {
	return w.header
}

func (w *fiberHTTPWriter) WriteHeader(status int) {
	w.status = status
}

func (w *fiberHTTPWriter) Write(p []byte) (int, error) {
	w.body = append(w.body, p...)
	return len(p), nil
}

func (w *fiberHTTPWriter) statusOrOK() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func docsDocumentAliasPath(docsPath, documentPath string) string {
	if documentPath != "/openapi.json" {
		return ""
	}
	trimmedDocsPath := strings.TrimRight(docsPath, "/")
	if trimmedDocsPath == "" || trimmedDocsPath == "/" {
		return ""
	}
	aliasPath := trimmedDocsPath + "/" + path.Base(documentPath)
	if aliasPath == documentPath {
		return ""
	}
	return aliasPath
}
