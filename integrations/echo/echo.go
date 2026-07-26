package openapiecho

import (
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	core "github.com/thebases/go-openapi/core"
)

type Router interface {
	Add(method, path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

func mountDocs(router Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.GET(documentPath, echo.WrapHandler(documentHandler))
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Keep a docs-scoped alias for the default document URL so requests under
		// /docs do not fall through to the docs asset handler and return 404.
		router.GET(aliasPath, echo.WrapHandler(documentHandler))
	}
	router.GET(docsPath, echo.WrapHandler(docsHandler))
	router.GET(docsPath+"/*", echo.WrapHandler(docsHandler))
	return nil
}

var routes = core.RouteRegistrar[Router, echo.HandlerFunc]{
	Register: func(router Router, method, path string, handlers ...echo.HandlerFunc) error {
		if len(handlers) == 0 {
			return nil
		}
		router.Add(method, path, handlers[0])
		return nil
	},
	MountDocs: mountDocs,
}

var docs = core.DocsRegistrar[Router]{
	Mount: mountDocs,
}

func Handle(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router Router, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
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
