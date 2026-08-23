package openapiiris

import (
	"net/http"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	core "github.com/thebases/go-openapi/core"
)

func TestGroupRegistersAbsoluteDocPathAndRelativeNativeRoute(t *testing.T) {
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	app := iris.New()

	device := Root(app, api).Group("/device").Group("/config")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/{id}", op, func(ctx iris.Context) {
		ctx.WriteString(ctx.Params().Get("id"))
	}); err != nil {
		t.Fatalf("register nested group route: %v", err)
	}

	// Iris's own path syntax is already "{id}"-shaped, so the doc key must
	// match it exactly, unlike ":id"/"{id:type}" frameworks that need rewriting.
	if api.Document().Paths["/device/config/{id}"] == nil {
		t.Fatalf("expected absolute doc path /device/config/{id}, got paths: %v", api.Document().Paths)
	}
	if api.Document().Paths["/{id}"] != nil {
		t.Fatal("did not expect a relative doc path to be registered")
	}

	httptest.New(t, app).GET("/device/config/42").Expect().Status(http.StatusOK).Body().IsEqual("42")
}

func TestGroupParamStyleTrimsIrisTypeMacro(t *testing.T) {
	api := core.New(core.WithTitle("Device API"), core.WithVersion("1.0.0"))
	app := iris.New()

	device := Root(app, api).Group("/device")

	op := core.Operation{
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("ok", core.StringSchema()),
		},
	}
	if err := device.GET("/{id:int}", op, func(ctx iris.Context) {
		ctx.WriteString(ctx.Params().Get("id"))
	}); err != nil {
		t.Fatalf("register group route with typed param: %v", err)
	}

	if api.Document().Paths["/device/{id}"] == nil {
		t.Fatalf("expected /device/{id} doc path with type macro trimmed, got paths: %v", api.Document().Paths)
	}

	httptest.New(t, app).GET("/device/42").Expect().Status(http.StatusOK).Body().IsEqual("42")
}
