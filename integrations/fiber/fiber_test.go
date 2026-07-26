package openapifiber

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
)

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
	gets   []string
}

func (r *stubFiberV2Router) Add(method, path string, handlers ...stubFiberV2Handler) {
	r.method = method
	r.path = path
	r.calls++
}

func (r *stubFiberV2Router) Get(path string, handlers ...stubFiberV2Handler) {
	r.calls++
	r.gets = append(r.gets, path)
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
	gets    []string
}

func (r *stubFiberV3Router) Add(methods []string, path string, handler stubFiberV3Handler, handlers ...stubFiberV3Handler) {
	r.methods = append([]string(nil), methods...)
	r.path = path
	r.calls++
}

func (r *stubFiberV3Router) Get(path string, handler stubFiberV3Handler, handlers ...stubFiberV3Handler) {
	r.calls++
	r.gets = append(r.gets, path)
}

type stubFiberV3InterfaceRouter struct {
	methods []string
	path    string
	calls   int
	gets    []string
}

func (r *stubFiberV3InterfaceRouter) Add(methods []string, path string, handler any, handlers ...any) {
	r.methods = append([]string(nil), methods...)
	r.path = path
	r.calls++
}

func (r *stubFiberV3InterfaceRouter) Get(path string, handler any, handlers ...any) {
	r.calls++
	r.gets = append(r.gets, path)
}

func TestCallFiberAddSupportsV2Signature(t *testing.T) {
	router := &stubFiberV2Router{}

	if err := callFiberAdd(router, http.MethodGet, "/docs", []any{stubFiberV2Handler(func(ctx *stubFiberV2Ctx) error {
		return nil
	})}); err != nil {
		t.Fatalf("register fiber v2 route: %v", err)
	}

	if router.calls != 1 {
		t.Fatalf("expected 1 fiber v2 route registration, got %d", router.calls)
	}
	if router.method != http.MethodGet {
		t.Fatalf("unexpected fiber v2 method: %s", router.method)
	}
	if router.path != "/docs" {
		t.Fatalf("unexpected fiber v2 path: %s", router.path)
	}
}

func TestCallFiberAddSupportsV3Signature(t *testing.T) {
	router := &stubFiberV3Router{}

	if err := callFiberAdd(router, http.MethodPost, "/docs", []any{stubFiberV3Handler(func(ctx stubFiberV3Ctx) error {
		return nil
	})}); err != nil {
		t.Fatalf("register fiber v3 route: %v", err)
	}

	if router.calls != 1 {
		t.Fatalf("expected 1 fiber v3 route registration, got %d", router.calls)
	}
	if len(router.methods) != 1 || router.methods[0] != http.MethodPost {
		t.Fatalf("unexpected fiber v3 methods: %v", router.methods)
	}
	if router.path != "/docs" {
		t.Fatalf("unexpected fiber v3 path: %s", router.path)
	}
}

func TestMakeFiberHTTPHandlerReplaysStandardHandlerForV2(t *testing.T) {
	router := &stubFiberV2Router{}
	handler, err := makeFiberHTTPHandler(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("body{color:red}"))
	}))
	if err != nil {
		t.Fatalf("make fiber v2 docs handler: %v", err)
	}

	context := &stubFiberV2Ctx{path: "/docs/css/app.css"}
	result := handler.Call([]reflect.Value{reflect.ValueOf(context)})
	if len(result) != 1 || !result[0].IsNil() {
		t.Fatalf("expected nil fiber v2 error result, got %v", result)
	}
	if context.status != http.StatusAccepted {
		t.Fatalf("unexpected fiber v2 status: %d", context.status)
	}
	if string(context.body) != "body{color:red}" {
		t.Fatalf("unexpected fiber v2 body: %q", string(context.body))
	}
	if got := context.header.Get("Content-Type"); got != "text/css; charset=utf-8" {
		t.Fatalf("unexpected fiber v2 content type: %q", got)
	}
}

func TestMakeFiberHTTPHandlerReplaysStandardHandlerForV3(t *testing.T) {
	router := fiber.New()
	handler, err := makeFiberHTTPHandler(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=abc")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"openapi":"3.0.3"}`))
	}))
	if err != nil {
		t.Fatalf("make fiber v3 docs handler: %v", err)
	}

	router.Get("/docs/openapi.json", handler.Interface())

	response, err := router.Test(httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	if err != nil {
		t.Fatalf("serve fiber v3 docs route: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected fiber v3 status: %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read fiber v3 body: %v", err)
	}
	if string(body) != `{"openapi":"3.0.3"}` {
		t.Fatalf("unexpected fiber v3 body: %q", string(body))
	}
}

func TestMountDocsServesDocumentAliasForV2AndV3(t *testing.T) {
	testCases := []struct {
		name   string
		router any
	}{
		{name: "v2", router: &stubFiberV2Router{}},
		{name: "v3", router: &stubFiberV3InterfaceRouter{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mountDocs(
				tc.router,
				"/docs",
				"/openapi.json",
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			)
			if err != nil {
				t.Fatalf("mount docs: %v", err)
			}
		})
	}
}

func TestDocsDocumentAliasPathSupportsRenamedDocuments(t *testing.T) {
	if got := docsDocumentAliasPath("/docs", "/merchant/spec.json"); got != "/docs/spec.json" {
		t.Fatalf("expected renamed document alias, got %q", got)
	}
	if got := docsDocumentAliasPath("/docs", "/docs/spec.json"); got != "" {
		t.Fatalf("expected no alias when document already lives under docs path, got %q", got)
	}
}

func TestMountDocsServesRenamedDocumentAliasForV2AndV3(t *testing.T) {
	testCases := []struct {
		name       string
		router     any
		assertGets func(*testing.T, any)
	}{
		{
			name:   "v2",
			router: &stubFiberV2Router{},
			assertGets: func(t *testing.T, router any) {
				got := router.(*stubFiberV2Router).gets
				want := []string{"/merchant/spec.json", "/docs/spec.json", "/docs", "/docs/*"}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("unexpected fiber v2 mounted paths: got %v want %v", got, want)
				}
			},
		},
		{
			name:   "v3",
			router: &stubFiberV3InterfaceRouter{},
			assertGets: func(t *testing.T, router any) {
				got := router.(*stubFiberV3InterfaceRouter).gets
				want := []string{"/merchant/spec.json", "/docs/spec.json", "/docs", "/docs/*"}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("unexpected fiber v3 mounted paths: got %v want %v", got, want)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mountDocs(
				tc.router,
				"/docs",
				"/merchant/spec.json",
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			)
			if err != nil {
				t.Fatalf("mount docs: %v", err)
			}
			tc.assertGets(t, tc.router)
		})
	}
}
