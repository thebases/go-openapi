package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	openapidocs "github.com/thebases/go-openapi/ui"
)

// API accumulates an OpenAPI document from route registrations. It is safe for
// concurrent use: registrations may run while the document is being served.
type API struct {
	mu sync.RWMutex

	doc          Document
	version      SpecVersion
	descriptions fs.FS
	docsProvider DocsProvider
	docsCSS      string
	docsEnabled  bool
	docsMounted  map[any]bool

	// rev is bumped on every successful mutation of doc; built caches the
	// rendered document for one rev so serving never re-walks or re-marshals
	// an unchanged document.
	rev   uint64
	built *snapshot
}

// snapshot is an immutable rendering of the document at one revision.
type snapshot struct {
	rev  uint64
	doc  Document
	json []byte
}

// Option configures an API created by New.
type Option func(*API)

// DocsConfig configures a docs UI mount; see ui.Config.
type DocsConfig = openapidocs.Config

// DocsProvider selects the docs UI theme; see ui.Provider.
type DocsProvider = openapidocs.Provider

// Docs UI providers, re-exported from the ui package.
const (
	DocsSwagger DocsProvider = openapidocs.Swagger
	DocsBase    DocsProvider = openapidocs.Base
	DocsScalar  DocsProvider = openapidocs.Scalar
)

// New creates an API with a minimal valid document (title "API", version
// "0.0.0", OpenAPI DefaultVersion) and applies options in order.
func New(options ...Option) *API {
	api := &API{
		doc: Document{
			OpenAPI: DefaultVersion.String(),
			Info: Info{
				Title:   "API",
				Version: "0.0.0",
			},
			Paths: map[string]*PathItem{},
			Components: &Components{
				Schemas:  map[string]*SchemaOrReference{},
				Examples: map[string]*ExampleOrReference{},
			},
		},
		version:      DefaultVersion,
		docsProvider: DocsSwagger,
		docsMounted:  map[any]bool{},
	}

	for _, option := range options {
		option(api)
	}

	return api
}

// WithTitle sets info.title.
func WithTitle(title string) Option {
	return func(api *API) {
		api.doc.Info.Title = title
	}
}

// WithDescription sets info.description. A relative single-line "*.md" value
// is replaced by that file's content read from the description FS.
func WithDescription(description string) Option {
	return func(api *API) {
		api.doc.Info.Description = description
	}
}

// WithVersion sets info.version (the API's own version, not the OpenAPI one).
func WithVersion(version string) Option {
	return func(api *API) {
		api.doc.Info.Version = version
	}
}

// WithServer appends a server entry.
func WithServer(url, description string) Option {
	return func(api *API) {
		api.doc.Servers = append(api.doc.Servers, Server{URL: url, Description: description})
	}
}

// WithDocStyle selects the docs UI theme and enables automatic docs mounting
// on the first route registration.
func WithDocStyle(provider DocsProvider) Option {
	return func(api *API) {
		api.docsProvider = provider
		api.docsEnabled = true
	}
}

// WithCustomCSS sets extra CSS injected into the docs UI after the theme's own
// styles, for every docs mount of this API. It does not enable docs mounting on
// its own; combine it with WithDocStyle or a manual MountDocs. A non-empty
// DocsConfig.CustomCSS on a specific mount replaces it for that mount.
func WithCustomCSS(css string) Option {
	return func(api *API) {
		api.docsCSS = css
	}
}

// WithDescriptionFS sets the filesystem that "*.md" description values are
// read from. Paths must be fs.ValidPath-style (relative, no ".." segments), so
// descriptions can never read outside fsys. Passing an embed.FS keeps
// descriptions inside the binary, independent of the process working
// directory. Without this option, descriptions are read from os.DirFS(".").
func WithDescriptionFS(fsys fs.FS) Option {
	return func(api *API) {
		api.descriptions = fsys
	}
}

// AddOperation records operation under method and an OpenAPI-style path
// ("/merchants/{id}"). The operation is deep-copied, so later changes by the
// caller do not affect the document.
func (api *API) AddOperation(method, path string, operation Operation) error {
	if !strings.HasPrefix(path, "/") {
		return ErrInvalidPath
	}
	if len(operation.Responses) == 0 {
		return ErrMissingResponses
	}

	method = strings.ToUpper(method)

	api.mu.Lock()
	defer api.mu.Unlock()

	item := api.doc.Paths[path]
	if item == nil {
		item = &PathItem{}
	}

	// Resolve the method slot before storing a new PathItem, so an unsupported
	// method never leaves an empty path entry behind.
	target, err := item.operationSlot(method)
	if err != nil {
		return err
	}
	if *target != nil {
		return fmt.Errorf("%w: %s %s", ErrDuplicateRoute, method, path)
	}

	stored := cloneValue(operation)
	*target = &stored
	if api.doc.Paths == nil {
		api.doc.Paths = map[string]*PathItem{}
	}
	api.doc.Paths[path] = item
	api.rev++
	return nil
}

// removeOperation undoes a successful AddOperation, dropping the path entry
// once it holds no operation. It is used to roll back when native route
// registration fails after the operation was recorded.
func (api *API) removeOperation(method, path string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	item := api.doc.Paths[path]
	if item == nil {
		return
	}
	target, err := item.operationSlot(strings.ToUpper(method))
	if err != nil || *target == nil {
		return
	}
	*target = nil
	if item.isEmpty() {
		delete(api.doc.Paths, path)
	}
	api.rev++
}

// rollbackOperation is deferred by route registration: unless *registered was
// set, the operation recorded for method/path is removed again. It also runs
// while a native router panics, so the document never keeps a route that was
// not actually mounted.
func (api *API) rollbackOperation(registered *bool, method, path string) {
	if !*registered {
		api.removeOperation(method, path)
	}
}

// RegisterSchema adds a reusable schema under components.schemas.
func (api *API) RegisterSchema(name string, schema *SchemaOrReference) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.commit(putComponent(&api.components().Schemas, "schema", name, schema))
}

// RegisterExample adds a reusable example under components.examples.
func (api *API) RegisterExample(name string, example *ExampleOrReference) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.commit(putComponent(&api.components().Examples, "example", name, example))
}

// RegisterSecurityScheme adds a scheme under components.securitySchemes.
func (api *API) RegisterSecurityScheme(name string, scheme *SecuritySchemeOrReference) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.commit(putComponent(&api.components().SecuritySchemes, "security scheme", name, scheme))
}

// components returns doc.Components, creating it on first use. Callers must
// hold api.mu for writing.
func (api *API) components() *Components {
	if api.doc.Components == nil {
		api.doc.Components = &Components{}
	}
	return api.doc.Components
}

// commit bumps the revision after a successful mutation so the next read
// rebuilds the cached snapshot. Callers must hold api.mu for writing.
func (api *API) commit(err error) error {
	if err == nil {
		api.rev++
	}
	return err
}

// putComponent validates name and value and stores value in *m, refusing to
// overwrite an existing entry.
func putComponent[T any](m *map[string]*T, kind, name string, value *T) error {
	if strings.TrimSpace(name) == "" || value == nil {
		return fmt.Errorf("openapi: %s name and value are required", kind)
	}
	if !componentNamePattern.MatchString(name) {
		return fmt.Errorf("%w: %s %q", ErrInvalidComponentName, kind, name)
	}
	if *m == nil {
		*m = map[string]*T{}
	}
	if _, exists := (*m)[name]; exists {
		return fmt.Errorf("%w: %s %q already registered", ErrDuplicateComponent, kind, name)
	}
	(*m)[name] = value
	return nil
}

// Document returns a deep copy of the document with Markdown descriptions
// resolved. If resolution fails, it returns an unresolved copy; use Snapshot
// to observe the error.
func (api *API) Document() Document {
	doc, err := api.Snapshot()
	if err != nil {
		api.mu.RLock()
		defer api.mu.RUnlock()
		return cloneValue(api.doc)
	}
	return doc
}

// Snapshot returns a deep copy of the resolved document, or the error that
// prevented rendering it (for example a missing Markdown description).
// Calling it once at startup turns such errors into fail-fast boot errors
// instead of 500s on the document endpoint.
func (api *API) Snapshot() (Document, error) {
	snap, err := api.snapshot()
	if err != nil {
		return Document{}, err
	}
	return cloneValue(snap.doc), nil
}

// JSON renders the document for the configured OpenAPI version. The result is
// cached per document revision, so repeated calls between registrations cost
// one copy of the bytes.
func (api *API) JSON() ([]byte, error) {
	snap, err := api.snapshot()
	if err != nil {
		return nil, err
	}
	return bytes.Clone(snap.json), nil
}

// snapshot returns the cached rendering for the current revision, rebuilding
// it at most once per revision. The fast path only takes the read lock.
func (api *API) snapshot() (*snapshot, error) {
	api.mu.RLock()
	built := api.built
	current := built != nil && built.rev == api.rev
	api.mu.RUnlock()
	if current {
		return built, nil
	}

	api.mu.Lock()
	defer api.mu.Unlock()
	// Another goroutine may have rebuilt while we waited for the write lock.
	if api.built != nil && api.built.rev == api.rev {
		return api.built, nil
	}

	snap, err := api.render()
	if err != nil {
		// Errors are not cached, so fixing a description file on disk
		// recovers without a new registration.
		return nil, err
	}
	api.built = snap
	return snap, nil
}

// render builds a snapshot from a deep copy of the document, so description
// resolution never mutates the registered model. Callers must hold api.mu.
func (api *API) render() (*snapshot, error) {
	if !api.version.valid() {
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedVersion, int(api.version))
	}

	doc := cloneValue(api.doc)
	resolver := descriptionResolver{fsys: api.descriptions}
	if err := resolver.resolveDocument(&doc); err != nil {
		return nil, err
	}

	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}

	// The Go model is JSON Schema 2020-12 shaped (OAS 3.1/3.2 semantics); 3.0.4
	// output needs its Schema Objects downgraded to the incompatible 3.0 subset.
	if api.version == Version30 {
		if raw, err = downgradeToVersion30(raw); err != nil {
			return nil, err
		}
	}

	return &snapshot{rev: api.rev, doc: doc, json: raw}, nil
}

func (api *API) docsTitle() string {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return api.doc.Info.Title
}

func (api *API) docsCustomCSS() string {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return api.docsCSS
}

func (api *API) shouldAutoMountDocs() bool {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return api.docsEnabled
}

func (api *API) markDocsMounted(key any) bool {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.docsMounted[key] {
		return false
	}
	api.docsMounted[key] = true
	return true
}

func (api *API) unmarkDocsMounted(key any) {
	api.mu.Lock()
	defer api.mu.Unlock()
	delete(api.docsMounted, key)
}

// CanonicalPath converts any supported framework route syntax (":id",
// "*path", "{id:int}", Fiber "<constraint>"/"?" suffixes, unnamed "*"/"+"
// wildcards) into OpenAPI "{name}" templating.
func CanonicalPath(path string) string {
	return FiberPathToOpenAPI(IrisPathToOpenAPI(path))
}

// StringSchema returns an inline {type: string} schema.
func StringSchema() *SchemaOrReference {
	return InlineSchema(&Schema{Type: "string"})
}

// IntegerSchema returns an inline integer schema with the given format.
func IntegerSchema(format string) *SchemaOrReference {
	return InlineSchema(&Schema{Type: "integer", Format: format})
}

// ArraySchema returns an inline array schema of items.
func ArraySchema(items *SchemaOrReference) *SchemaOrReference {
	return InlineSchema(&Schema{Type: "array", Items: items})
}

// RefSchema is an alias of SchemaRef kept for compatibility.
func RefSchema(name string) *SchemaOrReference {
	return SchemaRef(name)
}

// PathParameter returns a required path parameter.
func PathParameter(name string, schema *SchemaOrReference) ParameterOrReference {
	return ParameterOrReference{Value: &Parameter{Name: name, In: "path", Required: true, Schema: schema}}
}

// QueryParameter returns an optional query parameter.
func QueryParameter(name string, schema *SchemaOrReference) ParameterOrReference {
	return ParameterOrReference{Value: &Parameter{Name: name, In: "query", Schema: schema}}
}

// JSONResponse returns an application/json response.
func JSONResponse(description string, schema *SchemaOrReference) ResponseOrReference {
	return ResponseOrReference{Value: &Response{Description: description, Content: map[string]MediaType{"application/json": {Schema: schema}}}}
}
