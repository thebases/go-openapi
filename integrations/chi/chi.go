package openapichi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	openapi "github.com/thebases/go-openapi/openapi"
)

func mountDocs(router chi.Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	router.Handle(documentPath, documentHandler)
	router.Handle(docsPath, docsHandler)
	router.Handle(docsPath+"/*", docsHandler)
	return nil
}

var routes = openapi.RouteRegistrar[chi.Router, http.HandlerFunc]{
	Register: func(router chi.Router, method, path string, handlers ...http.HandlerFunc) {
		if len(handlers) == 0 {
			return
		}
		router.Method(method, path, handlers[0])
	},
	MountDocs: mountDocs,
}

var docs = openapi.DocsRegistrar[chi.Router]{
	Mount: mountDocs,
}

func Handle(router chi.Router, api *openapi.API, method, path string, operation openapi.Operation, handler http.Handler) error {
	if err := openapi.Chi.Handle(router, api, method, path, operation, handler); err != nil {
		return err
	}
	return nil
}

func GET(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return routes.GET(router, api, path, operation, handler)
}

func POST(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return routes.POST(router, api, path, operation, handler)
}

func PUT(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return routes.PUT(router, api, path, operation, handler)
}

func PATCH(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return routes.PATCH(router, api, path, operation, handler)
}

func DELETE(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return routes.DELETE(router, api, path, operation, handler)
}

func MountDocs(router chi.Router, api *openapi.API, docsPath, documentPath string, config openapi.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
}
