package openapigin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapi "github.com/thebases/go-openapi/openapi"
)

func mountDocs(router gin.IRoutes, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.GET(documentPath, gin.WrapH(documentHandler))
	router.GET(docsPath, gin.WrapH(docsHandler))
	router.GET(docsPath+"/*asset", gin.WrapH(docsHandler))
	return nil
}

var routes = openapi.RouteRegistrar[gin.IRoutes, gin.HandlerFunc]{
	Register: func(router gin.IRoutes, method, path string, handlers ...gin.HandlerFunc) {
		router.Handle(method, path, handlers...)
	},
	MountDocs: mountDocs,
}

var docs = openapi.DocsRegistrar[gin.IRoutes]{
	Mount: mountDocs,
}

func Handle(router gin.IRoutes, api *openapi.API, spec openapi.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router gin.IRoutes, api *openapi.API, spec openapi.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router gin.IRoutes, api *openapi.API, spec openapi.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router gin.IRoutes, api *openapi.API, spec openapi.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router gin.IRoutes, api *openapi.API, spec openapi.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router gin.IRoutes, api *openapi.API, spec openapi.RouteSpec, handlers ...gin.HandlerFunc) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router gin.IRoutes, api *openapi.API, docsPath, documentPath string, config openapi.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
}
