package openapigin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapidocs "github.com/thebases/go-openapi/docs"
	openapi "github.com/thebases/go-openapi/openapi"
)

func Handle(router gin.IRoutes, api *openapi.API, method, path string, operation openapi.Operation, handlers ...gin.HandlerFunc) error {
	if err := api.AddOperation(method, openapi.CanonicalPath(path), operation); err != nil {
		return err
	}

	router.Handle(method, path, handlers...)
	return nil
}

func GET(router gin.IRoutes, api *openapi.API, path string, operation openapi.Operation, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, http.MethodGet, path, operation, handlers...)
}

func POST(router gin.IRoutes, api *openapi.API, path string, operation openapi.Operation, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, http.MethodPost, path, operation, handlers...)
}

func PUT(router gin.IRoutes, api *openapi.API, path string, operation openapi.Operation, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, http.MethodPut, path, operation, handlers...)
}

func PATCH(router gin.IRoutes, api *openapi.API, path string, operation openapi.Operation, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, http.MethodPatch, path, operation, handlers...)
}

func DELETE(router gin.IRoutes, api *openapi.API, path string, operation openapi.Operation, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, http.MethodDelete, path, operation, handlers...)
}

func MountDocs(router gin.IRoutes, api *openapi.API, docsPath, documentPath string, config openapidocs.Config) error {
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

	router.GET(documentPath, func(c *gin.Context) {
		raw, err := api.JSON()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
	})
	router.GET(docsPath, gin.WrapH(docsHandler))
	return nil
}
