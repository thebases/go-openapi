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

// GroupRouter is satisfied by both *echo.Echo and *echo.Group, unlike Router
// which does not expose Group.
type GroupRouter interface {
	Router
	Group(prefix string, m ...echo.MiddlewareFunc) *echo.Group
}

func mountDocs(router Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.GET(documentPath, echo.WrapHandler(documentHandler))
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Keep a docs-scoped alias for the configured document basename so
		// requests under /docs do not fall through to the docs asset handler.
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

var groups = core.GroupRegistrar[GroupRouter, echo.HandlerFunc]{
	RouteRegistrar: core.RouteRegistrar[GroupRouter, echo.HandlerFunc]{
		Register: func(router GroupRouter, method, path string, handlers ...echo.HandlerFunc) error {
			if len(handlers) == 0 {
				return nil
			}
			router.Add(method, path, handlers[0])
			return nil
		},
		MountDocs: func(router GroupRouter, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
			return mountDocs(router, docsPath, documentPath, docsHandler, documentHandler)
		},
	},
	NewGroup: func(router GroupRouter, relativePrefix string) GroupRouter {
		return router.Group(relativePrefix)
	},
}

// Root wraps router as the root of a Group tree so nested Group(...) calls
// track their own absolute prefix for OpenAPI doc keys while still handing
// echo's native Group the relative path it expects.
func Root(router GroupRouter, api *core.API) core.Group[GroupRouter, echo.HandlerFunc] {
	return core.NewGroup(router, api, groups, "")
}

func docsDocumentAliasPath(docsPath, documentPath string) string {
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
