package openapifiber

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"

	core "github.com/thebases/go-openapi/core"
)

func mountDocs(router any, docsPath, documentPath string, docsHandler, documentHandler http.Handler) error {
	// Replay standard net/http handlers into Fiber so framework integrations do
	// not need to duplicate docs generation behavior.
	documentRouteHandler, err := makeFiberHTTPHandler(router, documentHandler)
	if err != nil {
		return err
	}
	docsRouteHandler, err := makeFiberHTTPHandler(router, docsHandler)
	if err != nil {
		return err
	}
	if err := callFiberMethod(router, "Get", []any{documentPath}, []any{documentRouteHandler.Interface()}); err != nil {
		return err
	}
	if aliasPath := docsDocumentAliasPath(docsPath, documentPath); aliasPath != "" {
		// Keep a docs-scoped alias for the default document URL so requests under
		// /docs do not fall through to the docs asset handler and return 404.
		if err := callFiberMethod(router, "Get", []any{aliasPath}, []any{documentRouteHandler.Interface()}); err != nil {
			return err
		}
	}
	if err := callFiberMethod(router, "Get", []any{docsPath}, []any{docsRouteHandler.Interface()}); err != nil {
		return err
	}
	if err := callFiberMethod(router, "Get", []any{docsPath + "/*"}, []any{docsRouteHandler.Interface()}); err != nil {
		return err
	}
	return nil
}

var routes = core.RouteRegistrar[any, any]{
	Register: func(router any, method, path string, handlers ...any) error {
		// Fiber v2 and v3 expose different Add signatures, so register against the
		// router's runtime method shape instead of pinning this adapter to one major.
		return callFiberAdd(router, method, path, handlers)
	},
	MountDocs: mountDocs,
}

var docs = core.DocsRegistrar[any]{
	Mount: mountDocs,
}

func Handle(router any, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return routes.Handle(router, api, spec, handlers...)
}

func GET(router any, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return routes.GET(router, api, spec, handlers...)
}

func POST(router any, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return routes.POST(router, api, spec, handlers...)
}

func PUT(router any, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return routes.PUT(router, api, spec, handlers...)
}

func PATCH(router any, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return routes.PATCH(router, api, spec, handlers...)
}

func DELETE(router any, api *core.API, spec core.RouteSpec, handlers ...any) error {
	return routes.DELETE(router, api, spec, handlers...)
}

func MountDocs(router any, api *core.API, docsPath, documentPath string, config core.DocsConfig) error {
	return docs.MountDocs(router, api, docsPath, documentPath, config)
}

func callFiberAdd(router any, method, routePath string, handlers []any) error {
	fixedArgs, err := fiberAddArgs(router, method, routePath)
	if err != nil {
		return err
	}
	return callFiberMethod(router, "Add", fixedArgs, handlers)
}

func fiberAddArgs(router any, method, routePath string) ([]any, error) {
	add := reflect.ValueOf(router).MethodByName("Add")
	if !add.IsValid() {
		return nil, fmt.Errorf("openapi fiber: %T does not expose Add", router)
	}

	methodArg := add.Type().In(0)
	switch {
	case methodArg.Kind() == reflect.String:
		return []any{method, routePath}, nil
	case methodArg.Kind() == reflect.Slice && methodArg.Elem().Kind() == reflect.String:
		return []any{[]string{method}, routePath}, nil
	default:
		return nil, fmt.Errorf("openapi fiber: unsupported Add signature on %T", router)
	}
}

func callFiberMethod(router any, name string, fixedArgs []any, handlers []any) error {
	method := reflect.ValueOf(router).MethodByName(name)
	if !method.IsValid() {
		return fmt.Errorf("openapi fiber: %T does not expose %s", router, name)
	}
	methodType := method.Type()
	if !methodType.IsVariadic() {
		return fmt.Errorf("openapi fiber: %T.%s is not variadic", router, name)
	}

	requiredArgs := methodType.NumIn() - 1
	useFixedHandler := len(fixedArgs)+1 == requiredArgs
	if len(fixedArgs) != requiredArgs && !useFixedHandler {
		return fmt.Errorf("openapi fiber: %T.%s signature mismatch", router, name)
	}
	if useFixedHandler && len(handlers) == 0 {
		return fmt.Errorf("openapi fiber: %T.%s requires at least one handler", router, name)
	}

	values := make([]reflect.Value, 0, len(fixedArgs)+len(handlers))
	for index, arg := range fixedArgs {
		value, err := assignValue(arg, methodType.In(index))
		if err != nil {
			return err
		}
		values = append(values, value)
	}
	if useFixedHandler {
		value, err := assignValue(handlers[0], methodType.In(len(fixedArgs)))
		if err != nil {
			return err
		}
		values = append(values, value)
		handlers = handlers[1:]
	}

	variadicType := methodType.In(methodType.NumIn() - 1).Elem()
	for _, handler := range handlers {
		value, err := assignValue(handler, variadicType)
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
	return reflect.Value{}, fmt.Errorf("openapi fiber: cannot use %T as %s", value, target)
}

func makeFiberHTTPHandler(router any, handler http.Handler) (reflect.Value, error) {
	get := reflect.ValueOf(router).MethodByName("Get")
	if !get.IsValid() {
		return reflect.Value{}, fmt.Errorf("openapi fiber: %T does not expose Get", router)
	}

	handlerType := get.Type().In(1)
	if handlerType.Kind() == reflect.Slice {
		handlerType = handlerType.Elem()
	}

	return reflect.MakeFunc(handlerType, func(args []reflect.Value) []reflect.Value {
		err := replayFiberHTTP(handler, args[0])
		if handlerType.NumOut() == 0 {
			return nil
		}
		if err == nil {
			return []reflect.Value{reflect.Zero(handlerType.Out(0))}
		}
		return []reflect.Value{reflect.ValueOf(err)}
	}), nil
}

func replayFiberHTTP(handler http.Handler, ctx reflect.Value) error {
	// Build a minimal request from the Fiber context path so the embedded docs
	// handlers resolve assets and document aliases the same way across majors.
	request := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: callStringMethod(ctx, "Path")}}
	recorder := &fiberHTTPWriter{header: http.Header{}}
	handler.ServeHTTP(recorder, request)

	// Replay headers through Fiber's mutators so Set-Cookie keeps all values
	// while ordinary headers stay single-valued like the upstream docs response.
	for key, values := range recorder.header {
		if len(values) == 0 {
			continue
		}
		if strings.EqualFold(key, "Set-Cookie") {
			for _, value := range values {
				callMethod(ctx, "Append", key, value)
			}
			continue
		}
		callMethod(ctx, "Set", key, values[0])
	}

	statusResult := callMethod(ctx, "Status", recorder.statusOrOK())
	return callErrorMethod(statusResult, "Send", recorder.body)
}

func callStringMethod(target reflect.Value, name string) string {
	result := callMethod(target, name)
	if !result.IsValid() || result.Kind() != reflect.String {
		return ""
	}
	return result.String()
}

func callMethod(target reflect.Value, name string, args ...any) reflect.Value {
	method := target.MethodByName(name)
	if !method.IsValid() {
		return reflect.Value{}
	}

	values := make([]reflect.Value, 0, len(args))
	for index, arg := range args {
		value, err := assignValue(arg, method.Type().In(index))
		if err != nil {
			return reflect.Value{}
		}
		values = append(values, value)
	}

	results := method.Call(values)
	if len(results) == 0 {
		return reflect.Value{}
	}
	return results[0]
}

func callErrorMethod(target reflect.Value, name string, args ...any) error {
	if !target.IsValid() {
		return fmt.Errorf("openapi fiber: missing %s receiver", name)
	}

	method := target.MethodByName(name)
	if !method.IsValid() {
		return fmt.Errorf("openapi fiber: %T does not expose %s", target.Interface(), name)
	}

	values := make([]reflect.Value, 0, len(args))
	for index, arg := range args {
		value, err := assignValue(arg, method.Type().In(index))
		if err != nil {
			return err
		}
		values = append(values, value)
	}

	results := method.Call(values)
	if len(results) == 0 || results[0].IsNil() {
		return nil
	}
	return results[0].Interface().(error)
}

type fiberHTTPWriter struct {
	header http.Header
	body   []byte
	status int
}

func (w *fiberHTTPWriter) Header() http.Header {
	return w.header
}

func (w *fiberHTTPWriter) WriteHeader(status int) {
	w.status = status
}

func (w *fiberHTTPWriter) Write(p []byte) (int, error) {
	w.body = append(w.body, p...)
	return len(p), nil
}

func (w *fiberHTTPWriter) statusOrOK() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func docsDocumentAliasPath(docsPath, documentPath string) string {
	if documentPath != "/openapi.json" {
		return ""
	}
	trimmedDocsPath := strings.TrimRight(docsPath, "/")
	if trimmedDocsPath == "" || trimmedDocsPath == "/" {
		return ""
	}
	aliasPath := trimmedDocsPath + "/" + path.Base(documentPath)
	if aliasPath == documentPath {
		return ""
	}
	return aliasPath
}
