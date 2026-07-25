package openapi

import (
	"fmt"
	"net/http"
	"reflect"

	core "github.com/thebases/go-openapi/core"
	openapidocs "github.com/thebases/go-openapi/docs"
)

type API struct {
	core *core.API
}

type Option = core.Option

type DocsConfig = openapidocs.Config

type DocsProvider = openapidocs.Provider

const (
	DocsSwagger DocsProvider = openapidocs.Swagger
	DocsScalar  DocsProvider = openapidocs.Scalar
	DocsRedoc   DocsProvider = openapidocs.Redoc
)

var (
	ErrInvalidPath       = core.ErrInvalidPath
	ErrUnsupportedMethod = core.ErrUnsupportedMethod
	ErrDuplicateRoute    = core.ErrDuplicateRoute
	ErrMissingResponses  = core.ErrMissingResponses

	WithTitle       = core.WithTitle
	WithDescription = core.WithDescription
	WithVersion     = core.WithVersion
	WithServer      = core.WithServer
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
	return &API{core: core.New(options...)}
}

func (api *API) AddOperation(method, path string, operation Operation) error {
	return api.core.AddOperation(method, path, toCoreOperation(operation))
}

func (api *API) RegisterSchema(name string, schema *SchemaOrReference) error {
	return api.core.RegisterSchema(name, toCoreSchemaRef(schema))
}

func (api *API) Document() Document {
	return fromCoreDocument(api.core.Document())
}

func (api *API) JSON() ([]byte, error) {
	return api.core.JSON()
}

func (docsNamespace) Handler(config DocsConfig) (http.Handler, error) {
	return openapidocs.DocsHandler(config)
}

func (docsNamespace) DocumentHandler(api *API) http.Handler {
	return openapidocs.DocumentHandler(api.core)
}

func (ginNamespace) Handle(router any, api *API, method, path string, operation Operation, handlers ...any) error {
	if err := api.AddOperation(method, CanonicalPath(path), operation); err != nil {
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

	documentHandler, err := makeGinHandler(router, func(ctx reflect.Value) {
		raw, jsonErr := api.JSON()
		if jsonErr != nil {
			callMethodIfPresent(ctx, "AbortWithError", http.StatusInternalServerError, jsonErr)
			return
		}
		callMethod(ctx, "Data", http.StatusOK, "application/json; charset=utf-8", raw)
	})
	if err != nil {
		return err
	}

	docsRouteHandler, err := makeGinHandler(router, func(ctx reflect.Value) {
		writer := contextField(ctx, "Writer")
		request := contextField(ctx, "Request")
		if !writer.IsValid() || !request.IsValid() || request.IsNil() {
			callMethodIfPresent(ctx, "AbortWithStatus", http.StatusInternalServerError)
			return
		}
		docsHandler.ServeHTTP(writer.Interface().(http.ResponseWriter), request.Interface().(*http.Request))
	})
	if err != nil {
		return err
	}

	if err := callVariadicMethod(router, "GET", []any{documentPath}, []any{documentHandler.Interface()}); err != nil {
		return err
	}
	return callVariadicMethod(router, "GET", []any{docsPath}, []any{docsRouteHandler.Interface()})
}

func (fiberNamespace) Handle(router any, api *API, method, path string, operation Operation, handlers ...any) error {
	if err := api.AddOperation(method, CanonicalPath(path), operation); err != nil {
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

	documentHandler, err := makeFiberHandler(router, func(ctx reflect.Value) error {
		raw, jsonErr := api.JSON()
		if jsonErr != nil {
			return jsonErr
		}
		callMethodIfPresent(ctx, "Set", "Content-Type", "application/json; charset=utf-8")
		return callErrorMethod(ctx, "Send", raw)
	})
	if err != nil {
		return err
	}

	docsRouteHandler, err := makeFiberHandler(router, func(ctx reflect.Value) error {
		recorder := &memoryResponseWriter{header: http.Header{}}
		docsHandler.ServeHTTP(recorder, &http.Request{})
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
	if err != nil {
		return err
	}

	if err := callVariadicMethod(router, "Get", []any{documentPath}, []any{documentHandler.Interface()}); err != nil {
		return err
	}
	return callVariadicMethod(router, "Get", []any{docsPath}, []any{docsRouteHandler.Interface()})
}

func (chiNamespace) Handle(router any, api *API, method, path string, operation Operation, handler http.Handler) error {
	if err := api.AddOperation(method, CanonicalPath(path), operation); err != nil {
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

	if err := callMethodExact(router, "Handle", documentPath, openapidocs.DocumentHandler(api.core)); err != nil {
		return err
	}
	return callMethodExact(router, "Handle", docsPath, docsHandler)
}

func CanonicalPath(path string) string {
	return core.CanonicalPath(path)
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

func toCoreOperation(operation Operation) core.Operation {
	result := core.Operation{
		Tags:        append([]string(nil), operation.Tags...),
		Summary:     operation.Summary,
		Description: operation.Description,
		OperationID: operation.OperationID,
		Deprecated:  operation.Deprecated,
		Responses:   core.Responses{},
	}

	for _, parameter := range operation.Parameters {
		if parameter.Value == nil {
			continue
		}
		result.Parameters = append(result.Parameters, toCoreParameter(parameter.Value))
	}

	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		result.RequestBody = toCoreRequestBody(operation.RequestBody.Value)
	}

	for code, response := range operation.Responses {
		if response.Value == nil {
			continue
		}
		result.Responses[code] = toCoreResponse(response.Value)
	}

	return result
}

func toCoreParameter(parameter *Parameter) core.Parameter {
	return core.Parameter{
		Name:        parameter.Name,
		In:          parameter.In,
		Description: parameter.Description,
		Required:    parameter.Required,
		Schema:      toCoreSchemaRef(parameter.Schema),
	}
}

func toCoreRequestBody(body *RequestBody) *core.RequestBody {
	result := &core.RequestBody{
		Description: body.Description,
		Required:    body.Required,
		Content:     map[string]core.MediaType{},
	}
	for contentType, media := range body.Content {
		result.Content[contentType] = core.MediaType{Schema: toCoreSchemaRef(media.Schema)}
	}
	return result
}

func toCoreResponse(response *Response) core.Response {
	result := core.Response{
		Description: response.Description,
		Content:     map[string]core.MediaType{},
	}
	for contentType, media := range response.Content {
		result.Content[contentType] = core.MediaType{Schema: toCoreSchemaRef(media.Schema)}
	}
	if len(result.Content) == 0 {
		result.Content = nil
	}
	return result
}

func toCoreSchemaRef(schema *SchemaOrReference) *core.Schema {
	if schema == nil {
		return nil
	}
	if schema.Ref != "" {
		return &core.Schema{Ref: schema.Ref}
	}
	if schema.Value == nil {
		return nil
	}
	return toCoreSchema(schema.Value)
}

func toCoreSchema(schema *Schema) *core.Schema {
	if schema == nil {
		return nil
	}

	result := &core.Schema{
		Type:                 schema.Type,
		Format:               schema.Format,
		Description:          schema.Description,
		Nullable:             schema.Nullable,
		Required:             append([]string(nil), schema.Required...),
		AdditionalProperties: schema.AdditionalProperties,
		Enum:                 append([]any(nil), schema.Enum...),
		Example:              schema.Example,
	}

	if schema.Properties != nil {
		result.Properties = make(map[string]*core.Schema, len(schema.Properties))
		for name, property := range schema.Properties {
			result.Properties[name] = toCoreSchemaRef(property)
		}
	}
	if schema.Items != nil {
		result.Items = toCoreSchemaRef(schema.Items)
	}

	return result
}

func fromCoreDocument(document core.Document) Document {
	result := Document{
		OpenAPI: document.OpenAPI,
		Info: Info{
			Title:       document.Info.Title,
			Description: document.Info.Description,
			Version:     document.Info.Version,
		},
		Paths:      map[string]*PathItem{},
		Components: &Components{Schemas: map[string]*SchemaOrReference{}},
		Tags:       make([]Tag, 0, len(document.Tags)),
	}

	for _, server := range document.Servers {
		result.Servers = append(result.Servers, Server{URL: server.URL, Description: server.Description})
	}
	for _, tag := range document.Tags {
		result.Tags = append(result.Tags, Tag{Name: tag.Name, Description: tag.Description})
	}
	for path, item := range document.Paths {
		result.Paths[path] = fromCorePathItem(item)
	}
	for name, schema := range document.Components.Schemas {
		result.Components.Schemas[name] = fromCoreSchema(schema)
	}

	if len(result.Components.Schemas) == 0 {
		result.Components = nil
	}

	return result
}

func fromCorePathItem(item *core.PathItem) *PathItem {
	if item == nil {
		return nil
	}
	return &PathItem{
		Get:     fromCoreOperation(item.Get),
		Put:     fromCoreOperation(item.Put),
		Post:    fromCoreOperation(item.Post),
		Delete:  fromCoreOperation(item.Delete),
		Options: fromCoreOperation(item.Options),
		Head:    fromCoreOperation(item.Head),
		Patch:   fromCoreOperation(item.Patch),
		Trace:   fromCoreOperation(item.Trace),
	}
}

func fromCoreOperation(operation *core.Operation) *Operation {
	if operation == nil {
		return nil
	}
	result := &Operation{
		Tags:        append([]string(nil), operation.Tags...),
		Summary:     operation.Summary,
		Description: operation.Description,
		OperationID: operation.OperationID,
		Deprecated:  operation.Deprecated,
		Responses:   map[string]ResponseOrReference{},
	}
	for _, parameter := range operation.Parameters {
		param := parameter
		result.Parameters = append(result.Parameters, ParameterOrReference{Value: fromCoreParameter(&param)})
	}
	if operation.RequestBody != nil {
		result.RequestBody = &RequestBodyOrReference{Value: fromCoreRequestBody(operation.RequestBody)}
	}
	for code, response := range operation.Responses {
		responseCopy := response
		result.Responses[code] = ResponseOrReference{Value: fromCoreResponse(&responseCopy)}
	}
	return result
}

func fromCoreParameter(parameter *core.Parameter) *Parameter {
	if parameter == nil {
		return nil
	}
	return &Parameter{
		Name:        parameter.Name,
		In:          parameter.In,
		Description: parameter.Description,
		Required:    parameter.Required,
		Schema:      fromCoreSchema(parameter.Schema),
	}
}

func fromCoreRequestBody(body *core.RequestBody) *RequestBody {
	if body == nil {
		return nil
	}
	result := &RequestBody{
		Description: body.Description,
		Required:    body.Required,
		Content:     map[string]MediaType{},
	}
	for contentType, media := range body.Content {
		result.Content[contentType] = MediaType{Schema: fromCoreSchema(media.Schema)}
	}
	return result
}

func fromCoreResponse(response *core.Response) *Response {
	if response == nil {
		return nil
	}
	result := &Response{Description: response.Description}
	if len(response.Content) > 0 {
		result.Content = map[string]MediaType{}
		for contentType, media := range response.Content {
			result.Content[contentType] = MediaType{Schema: fromCoreSchema(media.Schema)}
		}
	}
	return result
}

func fromCoreSchema(schema *core.Schema) *SchemaOrReference {
	if schema == nil {
		return nil
	}
	if schema.Ref != "" {
		return &SchemaOrReference{Ref: schema.Ref}
	}

	result := &Schema{Type: schema.Type, Format: schema.Format, Description: schema.Description, Nullable: schema.Nullable, Required: append([]string(nil), schema.Required...), AdditionalProperties: schema.AdditionalProperties, Enum: append([]any(nil), schema.Enum...), Example: schema.Example}
	if schema.Properties != nil {
		result.Properties = make(map[string]*SchemaOrReference, len(schema.Properties))
		for name, property := range schema.Properties {
			result.Properties[name] = fromCoreSchema(property)
		}
	}
	if schema.Items != nil {
		result.Items = fromCoreSchema(schema.Items)
	}

	return &SchemaOrReference{Value: result}
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

func (w *memoryResponseWriter) Header() http.Header {
	return w.header
}

func (w *memoryResponseWriter) WriteHeader(status int) {
	w.status = status
}

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
