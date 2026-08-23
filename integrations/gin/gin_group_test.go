package openapigin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	core "github.com/thebases/go-openapi/core"
)

func TestGroupRegistersAbsoluteDocPathAndRelativeNativeRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	router := gin.New()

	device := Root(router, api).Group("/device").Group("/config")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/:id", op, func(c *gin.Context) {
		c.String(http.StatusOK, c.Param("id"))
	}); err != nil {
		t.Fatalf("register nested group route: %v", err)
	}

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
	gin.SetMode(gin.TestMode)
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	router := gin.New()

	device := Root(router, api).Group("/device")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/", op, func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	}); err != nil {
		t.Fatalf("register group root route: %v", err)
	}

	if api.Document().Paths["/device"] == nil {
		t.Fatalf("expected /device doc path, got paths: %v", api.Document().Paths)
	}

	// Gin mounts a group's "/" route at "<prefix>/" and 301-redirects the
	// bare prefix there by default, so assert against the route it actually
	// registers rather than gin's redirect behavior.
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/device/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}
