package openapichi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	core "github.com/thebases/go-openapi/core"
)

func TestGroupRegistersAbsoluteDocPathAndRelativeNativeRoute(t *testing.T) {
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	router := chi.NewRouter()

	device := Root(router, api).Group("/device").Group("/config")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/{id}", op, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chi.URLParam(r, "id")))
	}); err != nil {
		t.Fatalf("register nested group route: %v", err)
	}

	// Chi's own routing pattern syntax is already "{id}"-shaped, so the doc key
	// must match it exactly, and it's the resolved absolute path (Mount-based
	// sub-routers have no queryable prefix to reflect it back out of).
	if api.Document().Paths["/device/config/{id}"] == nil {
		t.Fatalf("expected absolute doc path /device/config/{id}, got paths: %v", api.Document().Paths)
	}
	if api.Document().Paths["/{id}"] != nil {
		t.Fatal("did not expect a relative doc path to be registered")
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/device/config/42", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if response.Body.String() != "42" {
		t.Fatalf("unexpected body: %q", response.Body.String())
	}
}

func TestGroupRootRouteCollapsesOntoPrefix(t *testing.T) {
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	router := chi.NewRouter()

	device := Root(router, api).Group("/device")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/", op, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}); err != nil {
		t.Fatalf("register group root route: %v", err)
	}

	if api.Document().Paths["/device"] == nil {
		t.Fatalf("expected /device doc path, got paths: %v", api.Document().Paths)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/device/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}
