package core

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type stubEchoHandler func(*stubEchoContext) error

type stubEchoResponse struct {
	Writer http.ResponseWriter
}

type stubEchoContext struct {
	request  *http.Request
	response *stubEchoResponse
}

func (c *stubEchoContext) Request() *http.Request {
	return c.request
}

func (c *stubEchoContext) Response() *stubEchoResponse {
	return c.response
}

type stubEchoRouter struct {
	method string
	path   string
	calls  int
}

func (r *stubEchoRouter) Add(method, path string, handlers ...stubEchoHandler) {
	r.method = method
	r.path = path
	r.calls++
}

func (r *stubEchoRouter) GET(path string, handler stubEchoHandler, middleware ...stubEchoHandler) {
	r.calls++
}

type stubIrisHandler func(*stubIrisContext)

type stubIrisContext struct {
	writer  http.ResponseWriter
	request *http.Request
}

func (c *stubIrisContext) ResponseWriter() http.ResponseWriter {
	return c.writer
}

func (c *stubIrisContext) Request() *http.Request {
	return c.request
}

type stubIrisRouter struct {
	method string
	path   string
	calls  int
}

func (r *stubIrisRouter) Handle(method, path string, handlers ...stubIrisHandler) {
	r.method = method
	r.path = path
	r.calls++
}

func (r *stubIrisRouter) Get(path string, handler stubIrisHandler, middleware ...stubIrisHandler) {
	r.calls++
}

type stubFiberV2Handler func(*stubFiberV2Ctx) error

type stubFiberV2Ctx struct {
	path     string
	header   http.Header
	body     []byte
	status   int
	appended map[string][]string
}

func (c *stubFiberV2Ctx) Path() string {
	return c.path
}

func (c *stubFiberV2Ctx) Append(key, value string) {
	if c.appended == nil {
		c.appended = map[string][]string{}
	}
	c.appended[key] = append(c.appended[key], value)
}

func (c *stubFiberV2Ctx) Set(key, value string) {
	if c.header == nil {
		c.header = http.Header{}
	}
	c.header.Set(key, value)
}

func (c *stubFiberV2Ctx) Status(status int) *stubFiberV2Ctx {
	c.status = status
	return c
}

func (c *stubFiberV2Ctx) Send(body []byte) error {
	c.body = append([]byte(nil), body...)
	return nil
}

type stubFiberV2Router struct {
	method string
	path   string
	calls  int
}

func (r *stubFiberV2Router) Add(method, path string, handlers ...stubFiberV2Handler) {
	r.method = method
	r.path = path
	r.calls++
}

func (r *stubFiberV2Router) Get(path string, handlers ...stubFiberV2Handler) {
	r.calls++
}

type stubFiberV3Handler func(stubFiberV3Ctx) error

type stubFiberV3State struct {
	header   http.Header
	body     []byte
	status   int
	appended map[string][]string
}

type stubFiberV3Ctx struct {
	path  string
	state *stubFiberV3State
}

func (c stubFiberV3Ctx) Path() string {
	return c.path
}

func (c stubFiberV3Ctx) Append(key, value string) {
	if c.state.appended == nil {
		c.state.appended = map[string][]string{}
	}
	c.state.appended[key] = append(c.state.appended[key], value)
}

func (c stubFiberV3Ctx) Set(key, value string) {
	if c.state.header == nil {
		c.state.header = http.Header{}
	}
	c.state.header.Set(key, value)
}

func (c stubFiberV3Ctx) Status(status int) stubFiberV3Ctx {
	c.state.status = status
	return c
}

func (c stubFiberV3Ctx) Send(body []byte) error {
	c.state.body = append([]byte(nil), body...)
	return nil
}

type stubFiberV3Router struct {
	methods []string
	path    string
	calls   int
}

func (r *stubFiberV3Router) Add(methods []string, path string, handler stubFiberV3Handler, handlers ...stubFiberV3Handler) {
	r.methods = append([]string(nil), methods...)
	r.path = path
	r.calls++
}

func (r *stubFiberV3Router) Get(path string, handler stubFiberV3Handler, handlers ...stubFiberV3Handler) {
	r.calls++
}

type stubFiberV3InterfaceRouter struct {
	methods []string
	path    string
	calls   int
}

func (r *stubFiberV3InterfaceRouter) Add(methods []string, path string, handler any, handlers ...any) {
	r.methods = append([]string(nil), methods...)
	r.path = path
	r.calls++
}

func (r *stubFiberV3InterfaceRouter) Get(path string, handler any, handlers ...any) {
	r.calls++
}

func TestEchoFacadeRegistersOperationAndRoute(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubEchoRouter{}
	spec := Route("/merchants/:id", Operation{
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("ok", StringSchema()),
		},
	})

	err := Echo.GET(router, api, spec, func(ctx *stubEchoContext) error {
		return nil
	})
	if err != nil {
		t.Fatalf("register echo route: %v", err)
	}
	if router.calls != 1 {
		t.Fatalf("expected 1 echo route registration, got %d", router.calls)
	}
	if router.method != http.MethodGet {
		t.Fatalf("unexpected echo method: %s", router.method)
	}
	if router.path != "/merchants/:id" {
		t.Fatalf("unexpected echo path: %s", router.path)
	}
	if api.Document().Paths["/merchants/{id}"] == nil {
		t.Fatal("expected echo canonical path in document")
	}
}

func TestIrisFacadeRegistersOperationAndRoute(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubIrisRouter{}
	spec := Route("/merchants/{id:int}", Operation{
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("ok", StringSchema()),
		},
	})

	err := Iris.GET(router, api, spec, func(ctx *stubIrisContext) {})
	if err != nil {
		t.Fatalf("register iris route: %v", err)
	}
	if router.calls != 1 {
		t.Fatalf("expected 1 iris route registration, got %d", router.calls)
	}
	if router.method != http.MethodGet {
		t.Fatalf("unexpected iris method: %s", router.method)
	}
	if router.path != "/merchants/{id:int}" {
		t.Fatalf("unexpected iris path: %s", router.path)
	}
	if api.Document().Paths["/merchants/{id}"] == nil {
		t.Fatal("expected iris canonical path in document")
	}
}

func TestFiberFacadeRegistersOperationAndRouteForV2(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubFiberV2Router{}
	spec := Route("/merchants/:id", Operation{
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("ok", StringSchema()),
		},
	})

	err := Fiber.GET(router, api, spec, func(ctx *stubFiberV2Ctx) error {
		return nil
	})
	if err != nil {
		t.Fatalf("register fiber v2 route: %v", err)
	}
	if router.calls != 1 {
		t.Fatalf("expected 1 fiber v2 route registration, got %d", router.calls)
	}
	if router.method != http.MethodGet {
		t.Fatalf("unexpected fiber v2 method: %s", router.method)
	}
	if router.path != "/merchants/:id" {
		t.Fatalf("unexpected fiber v2 path: %s", router.path)
	}
	if api.Document().Paths["/merchants/{id}"] == nil {
		t.Fatal("expected fiber v2 canonical path in document")
	}
}

func TestFiberFacadeRegistersOperationAndRouteForV3(t *testing.T) {
	api := New(WithTitle("Merchant API"), WithVersion("1.0.0"))
	router := &stubFiberV3Router{}
	spec := Route("/merchants/:id", Operation{
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("ok", StringSchema()),
		},
	})

	err := Fiber.GET(router, api, spec, func(ctx stubFiberV3Ctx) error {
		return nil
	})
	if err != nil {
		t.Fatalf("register fiber v3 route: %v", err)
	}
	if router.calls != 1 {
		t.Fatalf("expected 1 fiber v3 route registration, got %d", router.calls)
	}
	if len(router.methods) != 1 || router.methods[0] != http.MethodGet {
		t.Fatalf("unexpected fiber v3 methods: %v", router.methods)
	}
	if router.path != "/merchants/:id" {
		t.Fatalf("unexpected fiber v3 path: %s", router.path)
	}
	if api.Document().Paths["/merchants/{id}"] == nil {
		t.Fatal("expected fiber v3 canonical path in document")
	}
}

func TestMakeEchoHTTPHandlerReplaysStandardHandler(t *testing.T) {
	router := &stubEchoRouter{}
	handler, err := makeEchoHTTPHandler(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("echo-docs"))
	}))
	if err != nil {
		t.Fatalf("make echo docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	context := &stubEchoContext{
		request: httptest.NewRequest(http.MethodGet, "/docs", nil),
		response: &stubEchoResponse{
			Writer: recorder,
		},
	}

	result := handler.Call([]reflect.Value{reflect.ValueOf(context)})
	if len(result) != 1 || !result[0].IsNil() {
		t.Fatalf("expected nil echo error result, got %v", result)
	}
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected echo status: %d", recorder.Code)
	}
	if recorder.Body.String() != "echo-docs" {
		t.Fatalf("unexpected echo body: %q", recorder.Body.String())
	}
}

func TestMakeIrisHTTPHandlerReplaysStandardHandler(t *testing.T) {
	router := &stubIrisRouter{}
	handler, err := makeIrisHTTPHandler(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("iris-docs"))
	}))
	if err != nil {
		t.Fatalf("make iris docs handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	context := &stubIrisContext{
		writer:  recorder,
		request: httptest.NewRequest(http.MethodGet, "/docs", nil),
	}

	handler.Call([]reflect.Value{reflect.ValueOf(context)})
	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected iris status: %d", recorder.Code)
	}
	if recorder.Body.String() != "iris-docs" {
		t.Fatalf("unexpected iris body: %q", recorder.Body.String())
	}
}

func TestMakeFiberHTTPHandlerReplaysStandardHandlerForV2(t *testing.T) {
	router := &stubFiberV2Router{}
	handler, err := makeFiberHTTPHandler(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	if err != nil {
		t.Fatalf("make fiber v2 docs handler: %v", err)
	}

	context := &stubFiberV2Ctx{path: "/docs"}
	result := handler.Call([]reflect.Value{reflect.ValueOf(context)})
	if len(result) != 1 || !result[0].IsNil() {
		t.Fatalf("expected nil fiber v2 error result, got %v", result)
	}
	if context.status != http.StatusAccepted {
		t.Fatalf("unexpected fiber v2 status: %d", context.status)
	}
	if string(context.body) != `{"ok":true}` {
		t.Fatalf("unexpected fiber v2 body: %q", string(context.body))
	}
	if got := context.header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected fiber v2 content type: %q", got)
	}
}

func TestMakeFiberHTTPHandlerReplaysStandardHandlerForV3(t *testing.T) {
	router := fiber.New()
	handler, err := makeFiberHTTPHandler(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=abc")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("fiber-docs"))
	}))
	if err != nil {
		t.Fatalf("make fiber v3 docs handler: %v", err)
	}

	router.Get("/docs", handler.Interface())

	response, err := router.Test(httptest.NewRequest(http.MethodGet, "/docs", nil))
	if err != nil {
		t.Fatalf("serve fiber v3 docs route: %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected fiber v3 status: %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read fiber v3 body: %v", err)
	}
	if string(body) != "fiber-docs" {
		t.Fatalf("unexpected fiber v3 body: %q", string(body))
	}
}
