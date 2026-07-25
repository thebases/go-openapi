package openapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	openapidocs "github.com/thebases/go-openapi/ui"
)

// RouteRegistrar keeps OpenAPI registration and native route mounting in one place
// so framework integrations only provide the framework-specific bind step.
type RouteRegistrar[Router any, Handler any] struct {
	Register  func(router Router, method, path string, handlers ...Handler)
	MountDocs func(router Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error
}

// RouteSpec keeps the runtime route identity and the OpenAPI operation together
// so integrations can pass one route contract through registration helpers.
type RouteSpec struct {
	Method    string
	Path      string
	Operation Operation
}

func Route(path string, operation Operation) RouteSpec {
	return RouteSpec{
		Path:      path,
		Operation: operation,
	}
}

func (spec RouteSpec) WithMethod(method string) RouteSpec {
	spec.Method = method
	return spec
}

func (registrar RouteRegistrar[Router, Handler]) Handle(router Router, api *API, spec RouteSpec, handlers ...Handler) error {
	if registrar.Register == nil {
		return errors.New("openapi: route registrar is not configured")
	}
	if err := registerOperation(api, spec.Method, spec.Path, spec.Operation); err != nil {
		return err
	}
	if err := mountDocsIfConfigured(router, api, registrar.MountDocs); err != nil {
		return err
	}

	registrar.Register(router, spec.Method, spec.Path, handlers...)
	return nil
}

func (registrar RouteRegistrar[Router, Handler]) GET(router Router, api *API, spec RouteSpec, handlers ...Handler) error {
	return registrar.Handle(router, api, spec.WithMethod(http.MethodGet), handlers...)
}

func (registrar RouteRegistrar[Router, Handler]) POST(router Router, api *API, spec RouteSpec, handlers ...Handler) error {
	return registrar.Handle(router, api, spec.WithMethod(http.MethodPost), handlers...)
}

func (registrar RouteRegistrar[Router, Handler]) PUT(router Router, api *API, spec RouteSpec, handlers ...Handler) error {
	return registrar.Handle(router, api, spec.WithMethod(http.MethodPut), handlers...)
}

func (registrar RouteRegistrar[Router, Handler]) PATCH(router Router, api *API, spec RouteSpec, handlers ...Handler) error {
	return registrar.Handle(router, api, spec.WithMethod(http.MethodPatch), handlers...)
}

func (registrar RouteRegistrar[Router, Handler]) DELETE(router Router, api *API, spec RouteSpec, handlers ...Handler) error {
	return registrar.Handle(router, api, spec.WithMethod(http.MethodDelete), handlers...)
}

// DocsRegistrar centralizes docs handler preparation so future framework adapters
// only need to translate net/http handlers into their native handler shape.
type DocsRegistrar[Router any] struct {
	Mount func(router Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error
}

func (registrar DocsRegistrar[Router]) MountDocs(router Router, api *API, docsPath, documentPath string, config DocsConfig) error {
	if registrar.Mount == nil {
		return errors.New("openapi: docs registrar is not configured")
	}

	resolvedDocsPath, resolvedDocumentPath, docsHandler, documentHandler, err := prepareDocsMount(api, docsPath, documentPath, config)
	if err != nil {
		return err
	}

	return registrar.Mount(router, resolvedDocsPath, resolvedDocumentPath, docsHandler, documentHandler)
}

func registerOperation(api *API, method, path string, operation Operation) error {
	return api.AddOperation(method, CanonicalPath(path), operation)
}

func prepareDocsMount(api *API, docsPath, documentPath string, config DocsConfig) (string, string, http.Handler, http.Handler, error) {
	if docsPath == "" {
		docsPath = "/docs"
	}
	if documentPath == "" {
		documentPath = "/openapi.json"
	}

	// API-level docs style acts as the mount default so callers can set it once
	// during openapi.New(...) and still override it per mount when needed.
	if config.Provider == "" {
		config.Provider = api.docsProvider
	}
	if config.Title == "" {
		config.Title = api.docsTitle()
	}
	config.DocsPath = docsPath
	config.DocumentURL = documentPath

	docsHandler, err := openapidocs.DocsHandler(config)
	if err != nil {
		return "", "", nil, nil, err
	}

	return docsPath, documentPath, mountableDocsHandler(docsPath, docsHandler), openapidocs.DocumentHandler(api), nil
}

func mountDocsIfConfigured[Router any](router Router, api *API, mount func(router Router, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error) error {
	if mount == nil || !api.shouldAutoMountDocs() {
		return nil
	}

	key := routerMountKey(router)
	if !api.markDocsMounted(key) {
		return nil
	}

	docsPath, documentPath, docsHandler, documentHandler, err := prepareDocsMount(api, "", "", DocsConfig{})
	if err != nil {
		api.unmarkDocsMounted(key)
		return err
	}
	if err := mount(router, docsPath, documentPath, docsHandler, documentHandler); err != nil {
		api.unmarkDocsMounted(key)
		return err
	}
	return nil
}

func routerMountKey(router any) string {
	value := reflect.ValueOf(router)
	if !value.IsValid() {
		return "<nil>"
	}

	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return fmt.Sprintf("%T:%x", router, value.Pointer())
	default:
		if value.CanAddr() {
			return fmt.Sprintf("%T:%x", router, value.Addr().Pointer())
		}
		return fmt.Sprintf("%T:%v", router, router)
	}
}

func mountableDocsHandler(docsPath string, next http.Handler) http.Handler {
	trimmed := strings.TrimRight(docsPath, "/")
	if trimmed == "" {
		trimmed = "/"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL == nil {
			r.URL = &url.URL{Path: "/"}
		}
		if r.URL.Path == trimmed {
			http.Redirect(w, r, trimmed+"/", http.StatusMovedPermanently)
			return
		}

		request := r.Clone(r.Context())
		if request.URL == nil {
			request.URL = &url.URL{Path: "/"}
		}

		// Trim the mount prefix before handing the request to the docs package so
		// it can serve "/" plus its embedded asset names consistently.
		switch {
		case request.URL.Path == trimmed+"/":
			request.URL.Path = "/"
		case strings.HasPrefix(request.URL.Path, trimmed+"/"):
			request.URL.Path = strings.TrimPrefix(request.URL.Path, trimmed)
		}

		next.ServeHTTP(w, request)
	})
}
