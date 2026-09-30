// Package openapichi registers documented routes on chi routers.
package openapichi

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	core "github.com/thebases/go-openapi/core"
)

// adapter implements core.GroupAdapter for a chi router.
type adapter struct {
	router chi.Router
}

// NewAdapter exposes router through the core adapter contract.
func NewAdapter(router chi.Router) core.GroupAdapter[http.HandlerFunc] {
	return adapter{router: router}
}

// Register mounts exactly one handler: chi composes middleware with Use/With,
// so extra handlers are rejected instead of silently dropped.
func (a adapter) Register(method, path string, handlers ...http.HandlerFunc) error {
	if len(handlers) != 1 {
		return fmt.Errorf("openapi chi: %s %s takes exactly one handler, got %d", method, path, len(handlers))
	}
	a.router.Method(method, path, handlers[0])
	return nil
}

func (a adapter) MountDocs(m core.DocsMount) error {
	docs := m.DocsWithAlias()
	a.router.Handle(m.DocumentPath, m.Document)
	a.router.Handle(m.DocsPath, docs)
	a.router.Handle(m.DocsPath+"/*", docs)
	return nil
}

// Group mounts a fresh sub-router at prefix. Chi has no queryable "current
// prefix" on a router, so core.Group tracks the absolute prefix itself.
func (a adapter) Group(prefix string) (core.GroupAdapter[http.HandlerFunc], error) {
	sub := chi.NewRouter()
	a.router.Mount(prefix, sub)
	return adapter{router: sub}, nil
}

// Handle documents spec and registers handler on router.
func Handle(router chi.Router, api *core.API, spec core.RouteSpec, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("%w: %s %s", core.ErrNoHandler, spec.Method, spec.Path)
	}
	return core.Handle[http.HandlerFunc](api, adapter{router: router}, spec, handler.ServeHTTP)
}

// GET documents spec as a GET operation and registers it on router; see Handle.
func GET(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodGet), handler)
}

// POST documents spec as a POST operation and registers it on router; see Handle.
func POST(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPost), handler)
}

// PUT documents spec as a PUT operation and registers it on router; see Handle.
func PUT(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPut), handler)
}

// PATCH documents spec as a PATCH operation and registers it on router; see Handle.
func PATCH(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPatch), handler)
}

// DELETE documents spec as a DELETE operation and registers it on router; see Handle.
func DELETE(router chi.Router, api *core.API, spec core.RouteSpec, handler http.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodDelete), handler)
}

// MountDocs mounts the docs UI and document on router.
func MountDocs(router chi.Router, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return core.MountDocs[http.HandlerFunc](api, adapter{router: router}, docsPath, documentPath, config)
}

// Root wraps router as the root of a Group tree so nested Group(...) calls
// track their own absolute prefix for OpenAPI doc keys while mounting a fresh
// chi.Router at each relative prefix for native routing.
func Root(router chi.Router, api *core.API) core.Group[http.HandlerFunc] {
	return core.NewRootGroup[http.HandlerFunc](api, adapter{router: router})
}
