package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"

	openapidocs "github.com/thebases/go-openapi/docs"
)

var (
	ErrInvalidPath       = errors.New("openapi: path must start with /")
	ErrUnsupportedMethod = errors.New("openapi: unsupported HTTP method")
	ErrDuplicateRoute    = errors.New("openapi: operation already registered")
	ErrMissingResponses  = errors.New("openapi: operation must define at least one response")

	nativeParamPattern = regexp.MustCompile(`(^|/)([:*])([A-Za-z_][A-Za-z0-9_]*)`)
)

type API struct {
	mu sync.RWMutex

	doc          Document
	docsProvider DocsProvider
	docsEnabled  bool
	docsMounted  map[string]bool
}

type Option func(*API)

type DocsConfig = openapidocs.Config

type DocsProvider = openapidocs.Provider

const (
	DocsSwagger DocsProvider = openapidocs.Swagger
	DocsBase    DocsProvider = openapidocs.Base
	DocsScalar  DocsProvider = openapidocs.Scalar
)

var (
	Docs  docsNamespace
	Gin   ginNamespace
	Fiber fiberNamespace
	Chi   chiNamespace
)

type docsNamespace struct{}
type ginNamespace struct{}
type fiberNamespace struct{}
type chiNamespace struct{}

func New(options ...Option) *API {
	api := &API{
		doc: Document{
			OpenAPI: "3.0.3",
			Info: Info{
				Title:   "API",
				Version: "0.0.0",
			},
			Paths: map[string]*PathItem{},
			Components: &Components{
				Schemas: map[string]*SchemaOrReference{},
			},
		},
		docsProvider: DocsSwagger,
		docsMounted:  map[string]bool{},
	}

	for _, option := range options {
		option(api)
	}

	return api
}

func WithTitle(title string) Option {
	return func(api *API) {
		api.doc.Info.Title = title
	}
}

func WithDescription(description string) Option {
	return func(api *API) {
		api.doc.Info.Description = description
	}
}

func WithVersion(version string) Option {
	return func(api *API) {
		api.doc.Info.Version = version
	}
}

func WithServer(url, description string) Option {
	return func(api *API) {
		api.doc.Servers = append(api.doc.Servers, Server{URL: url, Description: description})
	}
}

func WithDocStyle(provider DocsProvider) Option {
	return func(api *API) {
		api.docsProvider = provider
		api.docsEnabled = true
	}
}

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
		api.doc.Paths[path] = item
	}

	target, err := operationSlot(item, method)
	if err != nil {
		return err
	}
	if *target != nil {
		return fmt.Errorf("%w: %s %s", ErrDuplicateRoute, method, path)
	}

	copy := operation
	*target = &copy
	return nil
}

func (api *API) RegisterSchema(name string, schema *SchemaOrReference) error {
	if strings.TrimSpace(name) == "" || schema == nil {
		return errors.New("openapi: schema name and value are required")
	}

	api.mu.Lock()
	defer api.mu.Unlock()

	if api.doc.Components == nil {
		api.doc.Components = &Components{Schemas: map[string]*SchemaOrReference{}}
	}
	if api.doc.Components.Schemas == nil {
		api.doc.Components.Schemas = map[string]*SchemaOrReference{}
	}
	if _, exists := api.doc.Components.Schemas[name]; exists {
		return fmt.Errorf("openapi: schema %q already registered", name)
	}

	api.doc.Components.Schemas[name] = schema
	return nil
}

func (api *API) Document() Document {
	api.mu.RLock()
	defer api.mu.RUnlock()

	raw, _ := json.Marshal(api.doc)
	var result Document
	_ = json.Unmarshal(raw, &result)
	return result
}

func (api *API) JSON() ([]byte, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return json.MarshalIndent(api.doc, "", "  ")
}

func (api *API) docsTitle() string {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return api.doc.Info.Title
}

func (api *API) shouldAutoMountDocs() bool {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return api.docsEnabled
}

func (api *API) markDocsMounted(key string) bool {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.docsMounted[key] {
		return false
	}
	api.docsMounted[key] = true
	return true
}

func (api *API) unmarkDocsMounted(key string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	delete(api.docsMounted, key)
}

func operationSlot(item *PathItem, method string) (**Operation, error) {
	switch method {
	case "GET":
		return &item.Get, nil
	case "PUT":
		return &item.Put, nil
	case "POST":
		return &item.Post, nil
	case "DELETE":
		return &item.Delete, nil
	case "OPTIONS":
		return &item.Options, nil
	case "HEAD":
		return &item.Head, nil
	case "PATCH":
		return &item.Patch, nil
	case "TRACE":
		return &item.Trace, nil
	default:
		return nil, ErrUnsupportedMethod
	}
}

func CanonicalPath(path string) string {
	return nativeParamPattern.ReplaceAllStringFunc(path, func(match string) string {
		prefix := ""
		token := match
		if strings.HasPrefix(match, "/") {
			prefix = "/"
			token = strings.TrimPrefix(match, "/")
		}
		return prefix + "{" + token[1:] + "}"
	})
}

func StringSchema() *SchemaOrReference {
	return InlineSchema(&Schema{Type: "string"})
}

func IntegerSchema(format string) *SchemaOrReference {
	return InlineSchema(&Schema{Type: "integer", Format: format})
}

func ArraySchema(items *SchemaOrReference) *SchemaOrReference {
	return InlineSchema(&Schema{Type: "array", Items: items})
}

func RefSchema(name string) *SchemaOrReference {
	return SchemaRef(name)
}

func PathParameter(name string, schema *SchemaOrReference) ParameterOrReference {
	return ParameterOrReference{Value: &Parameter{Name: name, In: "path", Required: true, Schema: schema}}
}

func QueryParameter(name string, schema *SchemaOrReference) ParameterOrReference {
	return ParameterOrReference{Value: &Parameter{Name: name, In: "query", Schema: schema}}
}

func JSONResponse(description string, schema *SchemaOrReference) ResponseOrReference {
	return ResponseOrReference{Value: &Response{Description: description, Content: map[string]MediaType{"application/json": {Schema: schema}}}}
}

func (docsNamespace) Handler(config DocsConfig) (http.Handler, error) {
	return openapidocs.DocsHandler(config)
}

func (docsNamespace) DocumentHandler(api *API) http.Handler {
	return openapidocs.DocumentHandler(api)
}

func (ginNamespace) Handle(router any, api *API, method, path string, operation Operation, handlers ...any) error {
	if err := registerOperation(api, method, path, operation); err != nil {
		return err
	}
	if err := mountDocsIfConfigured(router, api, mountGinDocs); err != nil {
		return err
	}
	return callVariadicMethod(router, "Handle", []any{method, path}, handlers)
}

func (ginNamespace) GET(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Gin.Handle(router, api, http.MethodGet, path, operation, handlers...)
}
func (ginNamespace) POST(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Gin.Handle(router, api, http.MethodPost, path, operation, handlers...)
}
func (ginNamespace) PUT(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Gin.Handle(router, api, http.MethodPut, path, operation, handlers...)
}
func (ginNamespace) PATCH(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Gin.Handle(router, api, http.MethodPatch, path, operation, handlers...)
}
func (ginNamespace) DELETE(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Gin.Handle(router, api, http.MethodDelete, path, operation, handlers...)
}

func (ginNamespace) MountDocs(router any, api *API, docsPath, documentPath string, config DocsConfig) error {
	docsPath, documentPath, docsHandler, documentHandler, err := prepareDocsMount(api, docsPath, documentPath, config)
	if err != nil {
		return err
	}
	return mountGinDocs(router, docsPath, documentPath, docsHandler, documentHandler)
}

func (fiberNamespace) Handle(router any, api *API, method, path string, operation Operation, handlers ...any) error {
	if err := registerOperation(api, method, path, operation); err != nil {
		return err
	}
	if err := mountDocsIfConfigured(router, api, mountFiberDocs); err != nil {
		return err
	}
	return callVariadicMethod(router, "Add", []any{[]string{method}, path}, handlers)
}
func (fiberNamespace) GET(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Fiber.Handle(router, api, http.MethodGet, path, operation, handlers...)
}
func (fiberNamespace) POST(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Fiber.Handle(router, api, http.MethodPost, path, operation, handlers...)
}
func (fiberNamespace) PUT(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Fiber.Handle(router, api, http.MethodPut, path, operation, handlers...)
}
func (fiberNamespace) PATCH(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Fiber.Handle(router, api, http.MethodPatch, path, operation, handlers...)
}
func (fiberNamespace) DELETE(router any, api *API, path string, operation Operation, handlers ...any) error {
	return Fiber.Handle(router, api, http.MethodDelete, path, operation, handlers...)
}

func (fiberNamespace) MountDocs(router any, api *API, docsPath, documentPath string, config DocsConfig) error {
	docsPath, documentPath, docsHandler, documentHandler, err := prepareDocsMount(api, docsPath, documentPath, config)
	if err != nil {
		return err
	}
	return mountFiberDocs(router, docsPath, documentPath, docsHandler, documentHandler)
}

func (chiNamespace) Handle(router any, api *API, method, path string, operation Operation, handler http.Handler) error {
	if err := registerOperation(api, method, path, operation); err != nil {
		return err
	}
	if err := mountDocsIfConfigured(router, api, mountChiDocs); err != nil {
		return err
	}
	return callMethodExact(router, "Method", method, path, handler)
}
func (chiNamespace) GET(router any, api *API, path string, operation Operation, handler http.HandlerFunc) error {
	return Chi.Handle(router, api, http.MethodGet, path, operation, handler)
}
func (chiNamespace) POST(router any, api *API, path string, operation Operation, handler http.HandlerFunc) error {
	return Chi.Handle(router, api, http.MethodPost, path, operation, handler)
}
func (chiNamespace) PUT(router any, api *API, path string, operation Operation, handler http.HandlerFunc) error {
	return Chi.Handle(router, api, http.MethodPut, path, operation, handler)
}
func (chiNamespace) PATCH(router any, api *API, path string, operation Operation, handler http.HandlerFunc) error {
	return Chi.Handle(router, api, http.MethodPatch, path, operation, handler)
}
func (chiNamespace) DELETE(router any, api *API, path string, operation Operation, handler http.HandlerFunc) error {
	return Chi.Handle(router, api, http.MethodDelete, path, operation, handler)
}

func (chiNamespace) MountDocs(router any, api *API, docsPath, documentPath string, config DocsConfig) error {
	docsPath, documentPath, docsHandler, documentHandler, err := prepareDocsMount(api, docsPath, documentPath, config)
	if err != nil {
		return err
	}
	return mountChiDocs(router, docsPath, documentPath, docsHandler, documentHandler)
}

func mountGinDocs(router any, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	documentRouteHandler, err := makeGinHTTPHandler(router, documentHandler)
	if err != nil {
		return err
	}
	docsRouteHandler, err := makeGinHTTPHandler(router, docsHandler)
	if err != nil {
		return err
	}
	if err := callVariadicMethod(router, "GET", []any{documentPath}, []any{documentRouteHandler.Interface()}); err != nil {
		return err
	}
	if err := callVariadicMethod(router, "GET", []any{docsPath}, []any{docsRouteHandler.Interface()}); err != nil {
		return err
	}
	return callVariadicMethod(router, "GET", []any{docsPath + "/*asset"}, []any{docsRouteHandler.Interface()})
}

func mountFiberDocs(router any, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	documentRouteHandler, err := makeFiberHTTPHandler(router, documentHandler)
	if err != nil {
		return err
	}
	docsRouteHandler, err := makeFiberHTTPHandler(router, docsHandler)
	if err != nil {
		return err
	}
	if err := callVariadicMethod(router, "Get", []any{documentPath}, []any{documentRouteHandler.Interface()}); err != nil {
		return err
	}
	if err := callVariadicMethod(router, "Get", []any{docsPath}, []any{docsRouteHandler.Interface()}); err != nil {
		return err
	}
	return callVariadicMethod(router, "Get", []any{docsPath + "/*"}, []any{docsRouteHandler.Interface()})
}

func mountChiDocs(router any, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	if err := callMethodExact(router, "Handle", documentPath, documentHandler); err != nil {
		return err
	}
	if err := callMethodExact(router, "Handle", docsPath, docsHandler); err != nil {
		return err
	}
	return callMethodExact(router, "Handle", docsPath+"/*", docsHandler)
}
func callVariadicMethod(target any, name string, fixedArgs []any, variadicArgs []any) error {
	method := reflect.ValueOf(target).MethodByName(name)
	if !method.IsValid() {
		return fmt.Errorf("openapi facade: %T does not expose %s", target, name)
	}
	methodType := method.Type()
	if !methodType.IsVariadic() {
		return fmt.Errorf("openapi facade: %T.%s is not variadic", target, name)
	}
	if len(fixedArgs)+1 != methodType.NumIn() {
		return fmt.Errorf("openapi facade: %T.%s signature mismatch", target, name)
	}
	values := make([]reflect.Value, 0, len(fixedArgs)+len(variadicArgs))
	for index, arg := range fixedArgs {
		value, err := assignValue(arg, methodType.In(index))
		if err != nil {
			return err
		}
		values = append(values, value)
	}
	variadicType := methodType.In(methodType.NumIn() - 1).Elem()
	for _, arg := range variadicArgs {
		value, err := assignValue(arg, variadicType)
		if err != nil {
			return err
		}
		values = append(values, value)
	}
	method.Call(values)
	return nil
}

func callMethodExact(target any, name string, args ...any) error {
	method := reflect.ValueOf(target).MethodByName(name)
	if !method.IsValid() {
		return fmt.Errorf("openapi facade: %T does not expose %s", target, name)
	}
	methodType := method.Type()
	if methodType.NumIn() != len(args) {
		return fmt.Errorf("openapi facade: %T.%s signature mismatch", target, name)
	}
	values := make([]reflect.Value, 0, len(args))
	for index, arg := range args {
		value, err := assignValue(arg, methodType.In(index))
		if err != nil {
			return err
		}
		values = append(values, value)
	}
	method.Call(values)
	return nil
}

func assignValue(value any, target reflect.Type) (reflect.Value, error) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return reflect.Zero(target), nil
	}
	if reflected.Type().AssignableTo(target) {
		return reflected, nil
	}
	if reflected.Type().ConvertibleTo(target) {
		return reflected.Convert(target), nil
	}
	return reflect.Value{}, fmt.Errorf("openapi facade: cannot use %T as %s", value, target)
}

func makeGinHandler(router any, fn func(ctx reflect.Value)) (reflect.Value, error) {
	method := reflect.ValueOf(router).MethodByName("GET")
	if !method.IsValid() {
		return reflect.Value{}, fmt.Errorf("openapi facade: %T does not expose GET", router)
	}
	handlerType := method.Type().In(1).Elem()
	return reflect.MakeFunc(handlerType, func(args []reflect.Value) []reflect.Value {
		fn(args[0])
		return nil
	}), nil
}

func makeGinHTTPHandler(router any, handler http.Handler) (reflect.Value, error) {
	// Gin can reuse the native response writer and request directly, so docs and
	// document handlers stay aligned with the net/http behavior.
	return makeGinHandler(router, func(ctx reflect.Value) {
		writer := contextField(ctx, "Writer")
		request := contextField(ctx, "Request")
		if !writer.IsValid() || !request.IsValid() || request.IsNil() {
			callMethodIfPresent(ctx, "AbortWithStatus", http.StatusInternalServerError)
			return
		}
		handler.ServeHTTP(writer.Interface().(http.ResponseWriter), request.Interface().(*http.Request))
	})
}

func makeFiberHandler(router any, fn func(ctx reflect.Value) error) (reflect.Value, error) {
	method := reflect.ValueOf(router).MethodByName("Get")
	if !method.IsValid() {
		return reflect.Value{}, fmt.Errorf("openapi facade: %T does not expose Get", router)
	}
	handlerType := method.Type().In(1).Elem()
	return reflect.MakeFunc(handlerType, func(args []reflect.Value) []reflect.Value {
		err := fn(args[0])
		if handlerType.NumOut() == 0 {
			return nil
		}
		if err == nil {
			return []reflect.Value{reflect.Zero(handlerType.Out(0))}
		}
		return []reflect.Value{reflect.ValueOf(err)}
	}), nil
}

func makeFiberHTTPHandler(router any, handler http.Handler) (reflect.Value, error) {
	// Fiber needs a response recorder bridge because the docs package emits
	// standard net/http handlers while Fiber expects its own handler contract.
	return makeFiberHandler(router, func(ctx reflect.Value) error {
		recorder := &memoryResponseWriter{header: http.Header{}}
		handler.ServeHTTP(recorder, &http.Request{})
		for key, values := range recorder.header {
			for _, value := range values {
				if !callMethodIfPresent(ctx, "Append", key, value) {
					callMethodIfPresent(ctx, "Set", key, value)
				}
			}
		}
		statusResult, ok := callMethodValue(ctx, "Status", recorder.statusOrOK())
		if !ok {
			return callErrorMethod(ctx, "Send", recorder.body)
		}
		return callErrorMethod(statusResult, "Send", recorder.body)
	})
}

func callMethod(target reflect.Value, name string, args ...any) []reflect.Value {
	method := target.MethodByName(name)
	values := make([]reflect.Value, 0, len(args))
	for index, arg := range args {
		value, _ := assignValue(arg, method.Type().In(index))
		values = append(values, value)
	}
	return method.Call(values)
}

func callMethodIfPresent(target reflect.Value, name string, args ...any) bool {
	method := target.MethodByName(name)
	if !method.IsValid() {
		return false
	}
	values := make([]reflect.Value, 0, len(args))
	for index, arg := range args {
		value, err := assignValue(arg, method.Type().In(index))
		if err != nil {
			return false
		}
		values = append(values, value)
	}
	method.Call(values)
	return true
}

func callMethodValue(target reflect.Value, name string, args ...any) (reflect.Value, bool) {
	method := target.MethodByName(name)
	if !method.IsValid() {
		return reflect.Value{}, false
	}
	values := make([]reflect.Value, 0, len(args))
	for index, arg := range args {
		value, err := assignValue(arg, method.Type().In(index))
		if err != nil {
			return reflect.Value{}, false
		}
		values = append(values, value)
	}
	result := method.Call(values)
	if len(result) == 0 {
		return reflect.Value{}, false
	}
	return result[0], true
}

func callErrorMethod(target reflect.Value, name string, args ...any) error {
	result, ok := callMethodValue(target, name, args...)
	if !ok || result.IsNil() {
		return nil
	}
	if err, ok := result.Interface().(error); ok {
		return err
	}
	return nil
}

func contextField(target reflect.Value, name string) reflect.Value {
	if target.Kind() == reflect.Pointer && !target.IsNil() {
		if field := target.Elem().FieldByName(name); field.IsValid() {
			return field
		}
	}
	return reflect.Value{}
}

type memoryResponseWriter struct {
	header http.Header
	body   []byte
	status int
}

func (w *memoryResponseWriter) Header() http.Header    { return w.header }
func (w *memoryResponseWriter) WriteHeader(status int) { w.status = status }
func (w *memoryResponseWriter) Write(p []byte) (int, error) {
	w.body = append(w.body, p...)
	return len(p), nil
}
func (w *memoryResponseWriter) statusOrOK() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}
