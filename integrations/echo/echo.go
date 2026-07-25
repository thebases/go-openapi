package openapiecho

import (
	"net/http"

	"github.com/labstack/echo/v4"
	openapi "github.com/thebases/go-openapi/openapi"
)

type Router interface {
	Add(method, path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

func mountDocs(router Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.GET(documentPath, echo.WrapHandler(documentHandler))
	router.GET(docsPath, echo.WrapHandler(docsHandler))
	router.GET(docsPath+"/*", echo.WrapHandler(docsHandler))
	return nil
}

var routes = openapi.RouteRegistrar[Router, echo.HandlerFunc]{
	Register: func(router Router, method, path string, handlers ...echo.HandlerFunc) {
		if len(handlers) == 0 {
			return
		}
		router.Add(method, path, handlers[0])
	},
	MountDocs: mountDocs,
}

var docs = openapi.DocsRegistrar[Router]{
	Mount: mountDocs,
}

func Handle(router Router, api *openapi.API, spec openapi.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router Router, api *openapi.API, spec openapi.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router Router, api *openapi.API, spec openapi.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router Router, api *openapi.API, spec openapi.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router Router, api *openapi.API, spec openapi.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router Router, api *openapi.API, spec openapi.RouteSpec, handlers ...echo.HandlerFunc) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router Router, api *openapi.API, docsPath, documentPath string, config openapi.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
}
