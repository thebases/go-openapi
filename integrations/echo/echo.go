// Package openapiecho registers documented routes on Echo routers.
package openapiecho

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	core "github.com/thebases/go-openapi/core"
)

// Router is the subset of *echo.Echo and *echo.Group used for registration.
type Router interface {
	Add(method, path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// GroupRouter is satisfied by both *echo.Echo and *echo.Group, unlike Router
// which does not expose Group.
type GroupRouter interface {
	Router
	Group(prefix string, m ...echo.MiddlewareFunc) *echo.Group
}

// adapter implements core.GroupAdapter for an Echo router.
type adapter struct {
	router Router
}

// NewAdapter exposes router through the core adapter contract.
func NewAdapter(router GroupRouter) core.GroupAdapter[echo.HandlerFunc] {
	return adapter{router: router}
}

// Register mounts exactly one handler: Echo middleware has its own type, so
// extra handlers are rejected instead of silently dropped.
func (a adapter) Register(method, path string, handlers ...echo.HandlerFunc) error {
	if len(handlers) != 1 {
		return fmt.Errorf("openapi echo: %s %s takes exactly one handler, got %d", method, path, len(handlers))
	}
	a.router.Add(method, path, handlers[0])
	return nil
}

func (a adapter) MountDocs(m core.DocsMount) error {
	docs := echo.WrapHandler(m.DocsWithAlias())
	a.router.GET(m.DocumentPath, echo.WrapHandler(m.Document))
	a.router.GET(m.DocsPath, docs)
	a.router.GET(m.DocsPath+"/*", docs)
	return nil
}

func (a adapter) Group(prefix string) (core.GroupAdapter[echo.HandlerFunc], error) {
	router, ok := a.router.(GroupRouter)
	if !ok {
		return nil, errors.New("openapi echo: router does not support Group")
	}
	return adapter{router: router.Group(prefix)}, nil
}

// Handle documents spec and registers it on router.
func Handle(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return core.Handle[echo.HandlerFunc](api, adapter{router: router}, spec, handlers...)
}

// GET documents spec as a GET operation and registers it on router; see Handle.
func GET(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodGet), handlers...)
}

// POST documents spec as a POST operation and registers it on router; see Handle.
func POST(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPost), handlers...)
}

// PUT documents spec as a PUT operation and registers it on router; see Handle.
func PUT(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPut), handlers...)
}

// PATCH documents spec as a PATCH operation and registers it on router; see Handle.
func PATCH(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPatch), handlers...)
}

// DELETE documents spec as a DELETE operation and registers it on router; see Handle.
func DELETE(router Router, api *core.API, spec core.RouteSpec, handlers ...echo.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodDelete), handlers...)
}

// MountDocs mounts the docs UI and document on router.
func MountDocs(router Router, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return core.MountDocs[echo.HandlerFunc](api, adapter{router: router}, docsPath, documentPath, config)
}

// Root wraps router as the root of a Group tree so nested Group(...) calls
// track their own absolute prefix for OpenAPI doc keys while still handing
// echo's native Group the relative path it expects.
func Root(router GroupRouter, api *core.API) core.Group[echo.HandlerFunc] {
	return core.NewRootGroup[echo.HandlerFunc](api, adapter{router: router})
}
