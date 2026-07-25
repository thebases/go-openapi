package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
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
