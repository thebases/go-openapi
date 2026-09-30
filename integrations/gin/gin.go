// Package openapigin registers documented routes on Gin routers.
package openapigin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	core "github.com/thebases/go-openapi/core"
)

// GroupRouter is satisfied by both *gin.Engine and *gin.RouterGroup, unlike
// gin.IRoutes which does not expose Group.
type GroupRouter interface {
	gin.IRoutes
	Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup
}

// adapter implements core.GroupAdapter for a Gin router.
type adapter struct {
	router gin.IRoutes
}

// NewAdapter exposes router through the core adapter contract.
func NewAdapter(router GroupRouter) core.GroupAdapter[gin.HandlerFunc] {
	return adapter{router: router}
}

func (a adapter) Register(method, path string, handlers ...gin.HandlerFunc) error {
	a.router.Handle(method, path, handlers...)
	return nil
}

// MountDocs mounts the document and the docs UI. Gin rejects a static route
// beside a wildcard under the same prefix, so the docs-scoped document alias
// is answered by the docs handler itself (DocsWithAlias).
func (a adapter) MountDocs(m core.DocsMount) error {
	docs := gin.WrapH(m.DocsWithAlias())
	a.router.GET(m.DocumentPath, gin.WrapH(m.Document))
	a.router.GET(m.DocsPath, docs)
	a.router.GET(m.DocsPath+"/*asset", docs)
	return nil
}

func (a adapter) Group(prefix string) (core.GroupAdapter[gin.HandlerFunc], error) {
	router, ok := a.router.(GroupRouter)
	if !ok {
		return nil, errors.New("openapi gin: router does not support Group")
	}
	return adapter{router: router.Group(prefix)}, nil
}

// Handle documents spec and registers it on router.
func Handle(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return core.Handle[gin.HandlerFunc](api, adapter{router: router}, spec, handlers...)
}

// GET documents spec as a GET operation and registers it on router; see Handle.
func GET(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodGet), handlers...)
}

// POST documents spec as a POST operation and registers it on router; see Handle.
func POST(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPost), handlers...)
}

// PUT documents spec as a PUT operation and registers it on router; see Handle.
func PUT(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPut), handlers...)
}

// PATCH documents spec as a PATCH operation and registers it on router; see Handle.
func PATCH(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodPatch), handlers...)
}

// DELETE documents spec as a DELETE operation and registers it on router; see Handle.
func DELETE(router gin.IRoutes, api *core.API, spec core.RouteSpec, handlers ...gin.HandlerFunc) error {
	return Handle(router, api, spec.WithMethod(http.MethodDelete), handlers...)
}

// MountDocs mounts the docs UI and document on router.
func MountDocs(router gin.IRoutes, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return core.MountDocs[gin.HandlerFunc](api, adapter{router: router}, docsPath, documentPath, config)
}

// Root wraps router as the root of a Group tree so nested Group(...) calls
// track their own absolute prefix for OpenAPI doc keys while still handing
// gin's native Group the relative path it expects.
func Root(router GroupRouter, api *core.API) core.Group[gin.HandlerFunc] {
	return core.NewRootGroup[gin.HandlerFunc](api, adapter{router: router})
}
