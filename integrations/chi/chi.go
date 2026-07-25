package openapichi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	openapidocs "github.com/thebases/go-openapi/docs"
	openapi "github.com/thebases/go-openapi/openapi"
)

func Handle(router chi.Router, api *openapi.API, method, path string, operation openapi.Operation, handler http.Handler) error {
	if err := api.AddOperation(method, openapi.CanonicalPath(path), operation); err != nil {
		return err
	}

	router.Method(method, path, handler)
	return nil
}

func GET(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return Handle(router, api, http.MethodGet, path, operation, handler)
}

func POST(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return Handle(router, api, http.MethodPost, path, operation, handler)
}

func PUT(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return Handle(router, api, http.MethodPut, path, operation, handler)
}

func PATCH(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return Handle(router, api, http.MethodPatch, path, operation, handler)
}

func DELETE(router chi.Router, api *openapi.API, path string, operation openapi.Operation, handler http.HandlerFunc) error {
	return Handle(router, api, http.MethodDelete, path, operation, handler)
}

func MountDocs(router chi.Router, api *openapi.API, docsPath, documentPath string, config openapidocs.Config) error {
	if docsPath == "" {
		docsPath = "/docs"
	}
	if documentPath == "" {
		documentPath = "/openapi.json"
	}

	config.DocumentURL = documentPath

	docsHandler, err := openapidocs.DocsHandler(config)
	if err != nil {
		return err
	}

	router.Handle(documentPath, openapidocs.DocumentHandler(api))
	router.Handle(docsPath, docsHandler)
	return nil
}
