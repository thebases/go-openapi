package openapiiris

import (
	"net/http"
	"path"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/handlerconv"
	openapi "github.com/thebases/go-openapi/openapi"
)

func mountDocs(router iris.Party, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.Get(documentPath, handlerconv.FromStd(documentHandler))
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Keep a docs-scoped alias for the default document URL so requests under
		// /docs do not fall through to the docs asset handler and return 404.
		router.Get(aliasPath, handlerconv.FromStd(documentHandler))
	}
	router.Get(docsPath, handlerconv.FromStd(docsHandler))
	router.Get(docsPath+"/{asset:path}", handlerconv.FromStd(docsHandler))
	return nil
}

var routes = openapi.RouteRegistrar[iris.Party, iris.Handler]{
	Register: func(router iris.Party, method, path string, handlers ...iris.Handler) {
		router.Handle(method, path, handlers...)
	},
	MountDocs: mountDocs,
}

var docs = openapi.DocsRegistrar[iris.Party]{
	Mount: mountDocs,
}

func Handle(router iris.Party, api *openapi.API, spec openapi.RouteSpec, handlers ...iris.Handler) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router iris.Party, api *openapi.API, spec openapi.RouteSpec, handlers ...iris.Handler) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router iris.Party, api *openapi.API, spec openapi.RouteSpec, handlers ...iris.Handler) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router iris.Party, api *openapi.API, spec openapi.RouteSpec, handlers ...iris.Handler) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router iris.Party, api *openapi.API, spec openapi.RouteSpec, handlers ...iris.Handler) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router iris.Party, api *openapi.API, spec openapi.RouteSpec, handlers ...iris.Handler) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router iris.Party, api *openapi.API, docsPath, documentPath string, config openapi.DocsConfig) error {
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
