package openapifiber

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	core "github.com/thebases/go-openapi/core"
)

func okOperation() core.Operation {
	return core.Operation{Responses: map[string]core.ResponseOrReference{
		"200": core.JSONResponse("ok", core.StringSchema()),
	}}
}

func get(t *testing.T, app *fiber.App, path string) (int, string) {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return response.StatusCode, string(body)
}

func TestGETRegistersRouteAndDocument(t *testing.T) {
	api := core.New(core.WithTitle("Merchant API"), core.WithDocStyle(core.DocsBase))
	app := fiber.New()

	if err := GET(app, api, core.Route("/merchants/:id", okOperation()), func(c fiber.Ctx) error {
		return c.SendString(c.Params("id"))
	}); err != nil {
		t.Fatalf("register route: %v", err)
	}

	if status, body := get(t, app, "/merchants/7"); status != http.StatusOK || body != "7" {
		t.Fatalf("route: %d %q", status, body)
	}
	if status, body := get(t, app, "/openapi.json"); status != http.StatusOK || !strings.Contains(body, `"/merchants/{id}"`) {
		t.Fatalf("document: %d %q", status, body)
	}
	if status, _ := get(t, app, "/docs/css/app.css"); status != http.StatusOK {
		t.Fatalf("docs asset: %d", status)
	}
}

func TestMountDocsServesRenamedDocumentAlias(t *testing.T) {
	api := core.New()
	app := fiber.New()
	if err := MountDocs(app, api, "/docs", "/merchant/spec.json", core.DocsConfig{Provider: core.DocsBase}); err != nil {
		t.Fatalf("mount docs: %v", err)
	}

	for _, path := range []string{"/merchant/spec.json", "/docs/spec.json"} {
		if status, body := get(t, app, path); status != http.StatusOK || !strings.Contains(body, `"openapi"`) {
			t.Fatalf("%s: %d %q", path, status, body)
		}
	}
}

func TestRegistrationFailureRollsBackDocument(t *testing.T) {
	api := core.New()
	app := fiber.New()
	// Fiber panics on an unsupported handler type; the operation must not
	// stay documented for a route that was never mounted.
	func() {
		defer func() { _ = recover() }()
		_ = GET(app, api, core.Route("/bad", okOperation()), 42)
	}()
	if api.Document().Paths["/bad"] != nil {
		t.Fatal("failed registration left /bad in the document")
	}
}
