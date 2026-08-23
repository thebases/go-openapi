package openapichi

import (
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	core "github.com/thebases/go-openapi/core"
)

func mountDocs(router chi.Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.Handle(documentPath, documentHandler)
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Keep a docs-scoped alias for the configured document basename so
		// requests under /docs do not fall through to the docs asset handler.
		router.Handle(aliasPath, documentHandler)
	}
	router.Handle(docsPath, docsHandler)
	router.Handle(docsPath+"/*", docsHandler)
	return nil
}

var routes = core.RouteRegistrar[chi.Router, http.HandlerFunc]{
	Register: func(router chi.Router, method, path string, handlers ...http.HandlerFunc) error {
		if len(handlers) == 0 {
			return nil
		}
		router.Method(method, path, handlers[0])
		return nil
	},
	MountDocs: mountDocs,
}

var docs = core.DocsRegistrar[chi.Router]{
	Mount: mountDocs,
}

func Handle(router chi.Router, api *core.API, spec core.RouteSpec, handler http.Handler) error {
	if err := core.Chi.Handle(router, api, spec, handler); err != nil {
		return err
	}
	return nil
}

func GET(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return routes.GET(router, api, spec, handler)
}

func POST(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return routes.POST(router, api, spec, handler)
}

func PUT(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return routes.PUT(router, api, spec, handler)
}

func PATCH(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return routes.PATCH(router, api, spec, handler)
}

func DELETE(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return routes.DELETE(router, api, spec, handler)
}

func MountDocs(router chi.Router, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
}

var groups = core.GroupRegistrar[chi.Router, http.HandlerFunc]{
	RouteRegistrar: routes,
	NewGroup: func(router chi.Router, relativePrefix string) chi.Router {
		// Chi has no queryable "current prefix" on a router, unlike Fiber/Gin/Echo.
		// It resolves nesting by mounting a fresh sub-router at a pattern instead,
		// so build that sub-router here and let core.Group track the absolute
		// prefix itself for OpenAPI doc keys.
		sub := chi.NewRouter()
		router.Mount(relativePrefix, sub)
		return sub
	},
}

// Root wraps router as the root of a Group tree so nested Group(...) calls
// track their own absolute prefix for OpenAPI doc keys while mounting a fresh
// chi.Router at each relative prefix for native routing.
func Root(router chi.Router, api *core.API) core.Group[chi.Router, http.HandlerFunc] {
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
