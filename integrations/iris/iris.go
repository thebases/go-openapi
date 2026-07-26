package openapiiris

import (
	"net/http"
	"path"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/handlerconv"
	core "github.com/thebases/go-openapi/core"
)

func mountDocs(router iris.Party, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.Get(documentPath, handlerconv.FromStd(documentHandler))
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Keep a docs-scoped alias for the configured document basename so
		// requests under /docs do not fall through to the docs asset handler.
		router.Get(aliasPath, handlerconv.FromStd(documentHandler))
	}
	router.Get(docsPath, handlerconv.FromStd(docsHandler))
	router.Get(docsPath+"/{asset:path}", handlerconv.FromStd(docsHandler))
	return nil
}

var routes = core.RouteRegistrar[iris.Party, iris.Handler]{
	Register: func(router iris.Party, method, path string, handlers ...iris.Handler) error {
		router.Handle(method, path, handlers...)
		return nil
	},
	MountDocs: mountDocs,
}

var docs = core.DocsRegistrar[iris.Party]{
	Mount: mountDocs,
}

func Handle(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router iris.Party, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
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
