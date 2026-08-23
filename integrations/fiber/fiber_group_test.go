package openapifiber

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	core "github.com/thebases/go-openapi/core"
)

func TestGroupRegistersAbsoluteDocPathAndRelativeNativeRoute(t *testing.T) {
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	app := fiber.New()

	device := Root(app, api).Group("/device").Group("/config")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/:id", op, func(c fiber.Ctx) error {
		return c.SendString(c.Params("id"))
	}); err != nil {
		t.Fatalf("register nested group route: %v", err)
	}

	// The absolute path must be the doc key, not "/{id}" (which would collide
	// with any other group's "GET /:id" route under a different prefix).
	if api.Document().Paths["/device/config/{id}"] == nil {
		t.Fatalf("expected absolute doc path /device/config/{id}, got paths: %v", api.Document().Paths)
	}
	if api.Document().Paths["/{id}"] != nil {
		t.Fatal("did not expect a relative doc path to be registered")
	}

	// The native router must still receive the request under the mounted
	// absolute prefix, since Fiber's Group prepends it internally.
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/device/config/42", nil))
	if err != nil {
		t.Fatalf("serve request: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
}

func TestGroupRootRouteCollapsesOntoPrefix(t *testing.T) {
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	app := fiber.New()

	device := Root(app, api).Group("/device")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/", op, func(c fiber.Ctx) error {
		return c.SendString("ok")
	}); err != nil {
		t.Fatalf("register group root route: %v", err)
	}

	if api.Document().Paths["/device"] == nil {
		t.Fatalf("expected /device doc path, got paths: %v", api.Document().Paths)
	}

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/device", nil))
	if err != nil {
		t.Fatalf("serve request: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
}
