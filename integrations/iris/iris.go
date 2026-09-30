// Package openapiiris registers documented routes on Iris parties.
package openapiiris

import (
	"net/http"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/handlerconv"
	core "github.com/thebases/go-openapi/core"
)

// adapter implements core.GroupAdapter for an Iris party.
type adapter struct {
	router iris.Party
}

// NewAdapter exposes router through the core adapter contract.
func NewAdapter(router iris.Party) core.GroupAdapter[iris.Handler] {
	return adapter{router: router}
}

func (a adapter) Register(method, path string, handlers ...iris.Handler) error {
	a.router.Handle(method, path, handlers...)
	return nil
}

func (a adapter) MountDocs(m core.DocsMount) error {
	docs := handlerconv.FromStd(m.DocsWithAlias())
	a.router.Get(m.DocumentPath, handlerconv.FromStd(m.Document))
	a.router.Get(m.DocsPath, docs)
	a.router.Get(m.DocsPath+"/{asset:path}", docs)
	return nil
}

func (a adapter) Group(prefix string) (core.GroupAdapter[iris.Handler], error) {
	return adapter{router: a.router.Party(prefix)}, nil
}

// Handle documents spec and registers it on router.
func Handle(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return core.Handle[iris.Handler](api, adapter{router: router}, spec, handlers...)
}

// GET documents spec as a GET operation and registers it on router; see Handle.
func GET(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return Handle(router, api, spec.WithMethod(http.MethodGet), handlers...)
}

// POST documents spec as a POST operation and registers it on router; see Handle.
func POST(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return Handle(router, api, spec.WithMethod(http.MethodPost), handlers...)
}

// PUT documents spec as a PUT operation and registers it on router; see Handle.
func PUT(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return Handle(router, api, spec.WithMethod(http.MethodPut), handlers...)
}

// PATCH documents spec as a PATCH operation and registers it on router; see Handle.
func PATCH(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return Handle(router, api, spec.WithMethod(http.MethodPatch), handlers...)
}

// DELETE documents spec as a DELETE operation and registers it on router; see Handle.
func DELETE(router iris.Party, api *core.API, spec core.RouteSpec, handlers ...iris.Handler) error {
	return Handle(router, api, spec.WithMethod(http.MethodDelete), handlers...)
}

// MountDocs mounts the docs UI and document on router.
func MountDocs(router iris.Party, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return core.MountDocs[iris.Handler](api, adapter{router: router}, docsPath, documentPath, config)
}

// Root wraps router as the root of a Group tree so nested Group(...) calls
// track their own absolute prefix for OpenAPI doc keys while still handing
// iris's native Party the relative segment it expects.
func Root(router iris.Party, api *core.API) core.Group[iris.Handler] {
	return core.NewRootGroup[iris.Handler](api, adapter{router: router})
}
