package openapigin

import (
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	core "github.com/thebases/go-openapi/core"
)

func mountDocs(router gin.IRoutes, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.GET(documentPath, gin.WrapH(documentHandler))
	mountedDocsHandler := docsHandler
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Gin rejects sibling wildcard and static routes under the same prefix, so
		// serve the docs-scoped document alias from the docs wildcard handler.
		mountedDocsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL != nil && r.URL.Path == aliasPath {
				documentHandler.ServeHTTP(w, r)
				return
			}
			docsHandler.ServeHTTP(w, r)
		})
	}
	router.GET(docsPath, gin.WrapH(mountedDocsHandler))
	router.GET(docsPath+"/*asset", gin.WrapH(mountedDocsHandler))
	return nil
}

var routes = core.RouteRegistrar[gin.IRoutes, gin.HandlerFunc]{
	Register: func(router gin.IRoutes, method, path string, handlers ...gin.HandlerFunc) error {
		router.Handle(method, path, handlers...)
		return nil
	},
	MountDocs: mountDocs,
}

var docs = core.DocsRegistrar[gin.IRoutes]{
	Mount: mountDocs,
}

func Handle(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router gin.IRoutes, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
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
