package core

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"

	openapidocs "github.com/thebases/go-openapi/ui"
)

// Adapter is the contract a framework integration implements to mount routes
// and the docs UI on one native router. H is the framework's handler type.
type Adapter[H any] interface {
	// Register mounts handlers on the native router. It must return an error
	// (never silently drop handlers) when the framework cannot accept them.
	Register(method, path string, handlers ...H) error
	// MountDocs exposes the docs UI and the OpenAPI document.
	MountDocs(mount DocsMount) error
}

// GroupAdapter is an Adapter that can also create native sub-routers.
type GroupAdapter[H any] interface {
	Adapter[H]
	// Group returns an adapter for a native sub-router mounted at prefix,
	// relative to this adapter's router.
	Group(prefix string) (GroupAdapter[H], error)
}

// DocsKeyer is optionally implemented by an Adapter whose value is not a
// stable identity for its native router (for example an adapter that wraps
// function values). The returned key decides "docs already mounted here".
type DocsKeyer interface {
	DocsKey() any
}

// DocsMount carries everything an adapter needs to expose the docs UI.
type DocsMount struct {
	DocsPath     string
	DocumentPath string
	// AliasPath is DocsPath + "/" + basename(DocumentPath), or "" when the
	// document already lives under DocsPath. The docs UI requests the
	// document relative to itself, so this path must serve Document.
	AliasPath string
	Docs      http.Handler
	Document  http.Handler
}

// DocsWithAlias returns a handler for DocsPath and its wildcard that also
// answers AliasPath with the document. Adapters register it on both docs
// routes instead of adding a static alias route, which some routers (Gin)
// reject beside a wildcard.
func (m DocsMount) DocsWithAlias() http.Handler {
	if m.AliasPath == "" {
		return m.Docs
	}
	return aliasHandler{alias: m.AliasPath, document: m.Document, docs: m.Docs}
}

// RouteSpec keeps the runtime route identity and the OpenAPI operation together
// so integrations can pass one route contract through registration helpers.
type RouteSpec struct {
	Method    string
	Path      string
	Operation Operation
}

// Route pairs a native route path with its operation.
func Route(path string, operation Operation) RouteSpec {
	return RouteSpec{
		Path:      path,
		Operation: operation,
	}
}

// WithMethod returns a copy of spec with Method set.
func (spec RouteSpec) WithMethod(method string) RouteSpec {
	spec.Method = method
	return spec
}

// Handle documents spec.Operation and mounts handlers through adapter as one
// unit: when native registration fails (or panics) the operation is removed
// again, so the document only lists routes that exist. spec.Path is both the
// native path and, after CanonicalPath, the document path.
func Handle[H any](api *API, adapter Adapter[H], spec RouteSpec, handlers ...H) error {
	return register(api, adapter, adapter, spec.Method, spec.Path, spec.Path, spec.Operation, handlers)
}

// MountDocs mounts the docs UI (docsPath, default "/docs") and the document
// (documentPath, default "/openapi.json") through adapter. Empty config
// fields inherit the API's docs style, title, and custom CSS.
func MountDocs[H any](api *API, adapter Adapter[H], docsPath, documentPath string, config DocsConfig) error {
	mount, err := prepareDocsMount(api, docsPath, documentPath, config)
	if err != nil {
		return err
	}
	return adapter.MountDocs(mount)
}

// register is the single registration path shared by Handle and Group.
// docsRoot receives the auto-mounted docs (the root router of a Group tree),
// target receives the route itself.
func register[H any](api *API, docsRoot, target Adapter[H], method, docPath, nativePath string, operation Operation, handlers []H) error {
	method = strings.ToUpper(method)
	if len(handlers) == 0 {
		return fmt.Errorf("%w: %s %s", ErrNoHandler, method, docPath)
	}

	// Docs mount first: it records nothing in the document, so a failure here
	// needs no rollback.
	if err := mountDocsIfConfigured(api, docsRoot); err != nil {
		return err
	}

	canonical := CanonicalPath(docPath)
	if err := api.AddOperation(method, canonical, operation); err != nil {
		return err
	}

	registered := false
	defer api.rollbackOperation(&registered, method, canonical)
	if err := target.Register(method, nativePath, handlers...); err != nil {
		return err
	}
	registered = true
	return nil
}

// Group registers routes under an accumulated absolute prefix. The prefix is
// tracked here instead of being read back from framework-internal state, which
// Chi and Iris do not expose. Docs are auto-mounted on the root router only.
type Group[H any] struct {
	api     *API
	root    Adapter[H]
	adapter GroupAdapter[H]
	prefix  string
	// err is a deferred Group(...) failure, reported by the next Handle so
	// Group calls stay chainable.
	err error
}

// NewRootGroup starts a Group tree at adapter's router.
func NewRootGroup[H any](api *API, adapter GroupAdapter[H]) Group[H] {
	return Group[H]{api: api, root: adapter, adapter: adapter}
}

// Group returns a child group mounted at relativePrefix. A native error is
// reported by the child's Handle (and Err) instead of panicking.
func (g Group[H]) Group(relativePrefix string) Group[H] {
	child := g
	child.prefix = joinPath(g.prefix, relativePrefix)
	if g.err != nil {
		return child
	}
	adapter, err := g.adapter.Group(relativePrefix)
	if err != nil {
		child.err = fmt.Errorf("openapi: group %q: %w", child.prefix, err)
		return child
	}
	child.adapter = adapter
	return child
}

// Prefix returns the group's absolute path prefix.
func (g Group[H]) Prefix() string { return g.prefix }

// Err returns the error from creating this group, if any.
func (g Group[H]) Err() error { return g.err }

// Handle registers a route at relativePath under the group prefix.
func (g Group[H]) Handle(method, relativePath string, op Operation, handlers ...H) error {
	if g.err != nil {
		return g.err
	}
	return register(g.api, g.root, g.adapter, method, joinPath(g.prefix, relativePath), relativePath, op, handlers)
}

// GET registers a GET route at relativePath under the group prefix.
func (g Group[H]) GET(relativePath string, op Operation, handlers ...H) error {
	return g.Handle(http.MethodGet, relativePath, op, handlers...)
}

// POST registers a POST route at relativePath under the group prefix.
func (g Group[H]) POST(relativePath string, op Operation, handlers ...H) error {
	return g.Handle(http.MethodPost, relativePath, op, handlers...)
}

// PUT registers a PUT route at relativePath under the group prefix.
func (g Group[H]) PUT(relativePath string, op Operation, handlers ...H) error {
	return g.Handle(http.MethodPut, relativePath, op, handlers...)
}

// PATCH registers a PATCH route at relativePath under the group prefix.
func (g Group[H]) PATCH(relativePath string, op Operation, handlers ...H) error {
	return g.Handle(http.MethodPatch, relativePath, op, handlers...)
}

// DELETE registers a DELETE route at relativePath under the group prefix.
func (g Group[H]) DELETE(relativePath string, op Operation, handlers ...H) error {
	return g.Handle(http.MethodDelete, relativePath, op, handlers...)
}

// joinPath resolves relativePath against prefix into a single absolute path,
// deduping the slash at the join point. relativePath "/" collapses onto
// prefix itself (a group's own root route), matching how Fiber/Gin/Echo/Iris
// treat an empty sub-path under a mounted group.
func joinPath(prefix, relativePath string) string {
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	trimmedPrefix := strings.TrimRight(prefix, "/")
	if trimmedPrefix == "" {
		return relativePath
	}
	// Group("merchants") is accepted by every framework, so the doc key must
	// still be absolute.
	if !strings.HasPrefix(trimmedPrefix, "/") {
		trimmedPrefix = "/" + trimmedPrefix
	}
	if relativePath == "/" {
		return trimmedPrefix
	}
	return trimmedPrefix + relativePath
}

// prepareDocsMount resolves docs defaults against the API settings and builds
// the handlers every adapter mounts.
func prepareDocsMount(api *API, docsPath, documentPath string, config DocsConfig) (DocsMount, error) {
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
	if config.CustomCSS == "" {
		config.CustomCSS = api.docsCustomCSS()
	}
	config.DocsPath = docsPath
	config.DocumentURL = documentPath

	docsHandler, err := openapidocs.DocsHandler(config)
	if err != nil {
		return DocsMount{}, err
	}

	return DocsMount{
		DocsPath:     docsPath,
		DocumentPath: documentPath,
		AliasPath:    docsAliasPath(docsPath, documentPath),
		Docs:         newPrefixStripHandler(docsPath, docsHandler),
		Document:     openapidocs.DocumentHandler(api),
	}, nil
}

// docsAliasPath returns the docs-scoped path of the document basename, or ""
// when that is the document path itself or the docs live at the root.
func docsAliasPath(docsPath, documentPath string) string {
	trimmedDocsPath := strings.TrimRight(docsPath, "/")
	if trimmedDocsPath == "" {
		return ""
	}
	aliasPath := trimmedDocsPath + "/" + path.Base(documentPath)
	if aliasPath == documentPath {
		return ""
	}
	return aliasPath
}

// mountDocsIfConfigured auto-mounts docs once per native router when the API
// was created with WithDocStyle.
func mountDocsIfConfigured[H any](api *API, adapter Adapter[H]) error {
	if !api.shouldAutoMountDocs() {
		return nil
	}

	key := docsKey(adapter)
	if !api.markDocsMounted(key) {
		return nil
	}

	mount, err := prepareDocsMount(api, "", "", DocsConfig{})
	if err == nil {
		err = adapter.MountDocs(mount)
	}
	if err != nil {
		api.unmarkDocsMounted(key)
	}
	return err
}

// docsKey identifies the native router behind adapter for the once-per-router
// docs mount. Comparable adapters (a struct holding the router pointer) are
// their own key.
func docsKey(adapter any) any {
	if keyed, ok := adapter.(DocsKeyer); ok {
		adapter = keyed.DocsKey()
	}
	if value := reflect.ValueOf(adapter); value.IsValid() && value.Comparable() {
		return adapter
	}
	return routerMountKey(adapter)
}

// routerMountKey builds a string identity for a non-comparable router value.
func routerMountKey(router any) string {
	value := reflect.ValueOf(router)
	if !value.IsValid() {
		return "<nil>"
	}

	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return fmt.Sprintf("%T:%x", router, value.Pointer())
	default:
		return fmt.Sprintf("%T:%v", router, router)
	}
}

// aliasHandler serves the document at alias and everything else from docs.
type aliasHandler struct {
	alias    string
	document http.Handler
	docs     http.Handler
}

func (h aliasHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL != nil && r.URL.Path == h.alias {
		h.document.ServeHTTP(w, r)
		return
	}
	h.docs.ServeHTTP(w, r)
}

// prefixStripHandler redirects the bare mount path to its trailing-slash form
// and strips the mount prefix before calling next, so the ui package can
// serve "/" plus embedded asset names consistently at any mount path.
type prefixStripHandler struct {
	prefix string
	next   http.Handler
}

func newPrefixStripHandler(docsPath string, next http.Handler) prefixStripHandler {
	trimmed := strings.TrimRight(docsPath, "/")
	if trimmed == "" {
		trimmed = "/"
	}
	return prefixStripHandler{prefix: trimmed, next: next}
}

func (h prefixStripHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL == nil {
		r.URL = &url.URL{Path: "/"}
	}
	if r.URL.Path == h.prefix && h.prefix != "/" {
		http.Redirect(w, r, h.prefix+"/", http.StatusMovedPermanently)
		return
	}

	request := r.Clone(r.Context())
	switch {
	case request.URL.Path == h.prefix+"/":
		request.URL.Path = "/"
	case strings.HasPrefix(request.URL.Path, h.prefix+"/"):
		request.URL.Path = strings.TrimPrefix(request.URL.Path, h.prefix)
	}

	h.next.ServeHTTP(w, request)
}
