package openapifiber

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	openapi "github.com/thebases/go-openapi/core"
	openapidocs "github.com/thebases/go-openapi/docs"
)

func Handle(router fiber.Router, api *openapi.API, method, path string, operation openapi.Operation, handlers ...fiber.Handler) error {
	if err := api.AddOperation(method, openapi.CanonicalPath(path), operation); err != nil {
		return err
	}

	router.Add([]string{method}, path, handlers...)
	return nil
}

func GET(router fiber.Router, api *openapi.API, path string, operation openapi.Operation, handlers ...fiber.Handler) error {
	return Handle(router, api, http.MethodGet, path, operation, handlers...)
}

func POST(router fiber.Router, api *openapi.API, path string, operation openapi.Operation, handlers ...fiber.Handler) error {
	return Handle(router, api, http.MethodPost, path, operation, handlers...)
}

func PUT(router fiber.Router, api *openapi.API, path string, operation openapi.Operation, handlers ...fiber.Handler) error {
	return Handle(router, api, http.MethodPut, path, operation, handlers...)
}

func PATCH(router fiber.Router, api *openapi.API, path string, operation openapi.Operation, handlers ...fiber.Handler) error {
	return Handle(router, api, http.MethodPatch, path, operation, handlers...)
}

func DELETE(router fiber.Router, api *openapi.API, path string, operation openapi.Operation, handlers ...fiber.Handler) error {
	return Handle(router, api, http.MethodDelete, path, operation, handlers...)
}

func MountDocs(router fiber.Router, api *openapi.API, docsPath, documentPath string, config openapidocs.Config) error {
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

	router.Get(documentPath, func(c fiber.Ctx) error {
		raw, err := api.JSON()
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		return c.Send(raw)
	})

	router.Get(docsPath, func(c fiber.Ctx) error {
		recorder := &fiberHTTPWriter{header: http.Header{}}
		docsHandler.ServeHTTP(recorder, &http.Request{})
		for key, values := range recorder.header {
			for _, value := range values {
				c.Append(key, value)
			}
		}
		return c.Status(recorder.statusOrOK()).Send(recorder.body)
	})

	return nil
}

type fiberHTTPWriter struct {
	header http.Header
	body   []byte
	status int
}

func (w *fiberHTTPWriter) Header() http.Header {
	return w.header
}

func (w *fiberHTTPWriter) WriteHeader(status int) {
	w.status = status
}

func (w *fiberHTTPWriter) Write(p []byte) (int, error) {
	w.body = append(w.body, p...)
	return len(p), nil
}

func (w *fiberHTTPWriter) statusOrOK() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}
