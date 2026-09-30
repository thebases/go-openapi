// Package openapifiber registers documented routes on Fiber v3 routers.
package openapifiber

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	core "github.com/thebases/go-openapi/core"
)

// adapter implements core.GroupAdapter for a Fiber router. Handlers are `any`
// because Fiber v3 accepts several handler shapes (fiber.Handler, net/http
// handlers, ...) in one route.
type adapter struct {
	router fiber.Router
}

// NewAdapter exposes router through the core adapter contract, for use with
// core.Handle, core.MountDocs, or core.NewRootGroup.
func NewAdapter(router fiber.Router) core.GroupAdapter[any] {
	return adapter{router: router}
}

func (a adapter) Register(method, path string, handlers ...any) error {
	if len(handlers) == 0 {
		return fmt.Errorf("%w: %s %s", core.ErrNoHandler, method, path)
	}
	a.router.Add([]string{method}, path, handlers[0], handlers[1:]...)
	return nil
}

// MountDocs mounts the document and the docs UI. Fiber v3 adapts net/http
// handlers itself, so the core handlers are passed through unchanged.
func (a adapter) MountDocs(m core.DocsMount) error {
	docs := m.DocsWithAlias()
	a.router.Get(m.DocumentPath, m.Document)
	a.router.Get(m.DocsPath, docs)
	a.router.Get(m.DocsPath+"/*", docs)
	return nil
}

func (a adapter) Group(prefix string) (core.GroupAdapter[any], error) {
	return adapter{router: a.router.Group(prefix)}, nil
}

// Handle documents spec and registers it on router.
func Handle(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return core.Handle[any](api, adapter{router: router}, spec, handlers...)
}

// GET documents spec as a GET operation and registers it on router; see Handle.
func GET(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return Handle(router, api, spec.WithMethod(http.MethodGet), handlers...)
}

// POST documents spec as a POST operation and registers it on router; see Handle.
func POST(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return Handle(router, api, spec.WithMethod(http.MethodPost), handlers...)
}

// PUT documents spec as a PUT operation and registers it on router; see Handle.
func PUT(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return Handle(router, api, spec.WithMethod(http.MethodPut), handlers...)
}

// PATCH documents spec as a PATCH operation and registers it on router; see Handle.
func PATCH(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return Handle(router, api, spec.WithMethod(http.MethodPatch), handlers...)
}

// DELETE documents spec as a DELETE operation and registers it on router; see Handle.
func DELETE(router fiber.Router, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return Handle(router, api, spec.WithMethod(http.MethodDelete), handlers...)
}

// MountDocs mounts the docs UI and document on router.
func MountDocs(router fiber.Router, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return core.MountDocs[any](api, adapter{router: router}, docsPath, documentPath, config)
}

// Root wraps router as the root of a Group tree so nested Group(...) calls
// track their own absolute prefix for OpenAPI doc keys while Fiber's native
// Group receives the relative prefix it expects.
func Root(router fiber.Router, api *core.API) core.Group[any] {
	return core.NewRootGroup[any](api, adapter{router: router})
}
