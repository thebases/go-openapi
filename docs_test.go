package openapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestDefaultDocsRoutes(t *testing.T) {
	app := fiber.New()
	api := New(app, Config{
		Title:   "Test API",
		Version: "1.0.0",
	})

	GET(api, "/items/:id", OperationConfig{ID: "getItem"}, func(ctx fiber.Ctx, input struct {
		ID string `path:"id"`
	}) (struct {
		ID string `json:"id"`
	}, error) {
		return struct {
			ID string `json:"id"`
		}{ID: input.ID}, nil
	})

	assertResponseContains(t, app, "/docs", http.StatusOK, "swagger-initializer.js")
	assertResponseContains(t, app, "/docs/swagger-initializer.js", http.StatusOK, "/docs/openapi.json")
	assertResponseContains(t, app, "/docs/openapi.json", http.StatusOK, "\"openapi\": \"3.0.3\"")
}

func TestCustomJSONPathServesSwaggerUIFromSpecDirectory(t *testing.T) {
	app := fiber.New()
	New(app, Config{
		Title:    "Test API",
		Version:  "1.0.0",
		JSONPath: "/reference/openapi.json",
	})

	assertResponseContains(t, app, "/reference/swagger-initializer.js", http.StatusOK, "/reference/openapi.json")
	assertResponseContains(t, app, "/reference/openapi.json", http.StatusOK, "\"info\": {")
}

func TestDocsBrandingDefaultsUseEmbeddedAssets(t *testing.T) {
	app := fiber.New()
	New(app, Config{
		Title:   "Test API",
		Version: "1.0.0",
	})

	assertResponseContains(t, app, "/docs/index.html", http.StatusOK, "./docs/favicon-32x32.png")
	assertResponseContains(t, app, "/docs/index.html", http.StatusOK, "./docs/favicon-16x16.png")
	assertResponseContains(t, app, "/docs/index.css", http.StatusOK, "mask: url(\"logo.svg\") center / contain no-repeat;")
}

func TestDocsBrandingSupportsSVGLogoWithoutDarkLogo(t *testing.T) {
	app := fiber.New()
	New(app, Config{
		Title:   "Test API",
		Version: "1.0.0",
		Logo:    "/assets/brand.svg",
	})

	assertResponseContains(t, app, "/docs/index.css", http.StatusOK, "mask: url(\"/assets/brand.svg\") center / contain no-repeat;")
}

func TestDocsBrandingFallsBackDarkLogoToRasterLogo(t *testing.T) {
	app := fiber.New()
	New(app, Config{
		Title:   "Test API",
		Version: "1.0.0",
		Logo:    "/assets/brand.png",
	})

	assertResponseContains(t, app, "/docs/index.css", http.StatusOK, "background: url(\"/assets/brand.png\") center / contain no-repeat;")
	assertResponseContains(t, app, "/docs/index.css", http.StatusOK, "background-image: url(\"/assets/brand.png\");")
}

func TestDocsBrandingSupportsDedicatedDarkLogoAndFavicon(t *testing.T) {
	app := fiber.New()
	New(app, Config{
		Title:    "Test API",
		Version:  "1.0.0",
		Logo:     "/assets/light.png",
		DarkLogo: "/assets/dark.png",
		Favicon:  "/assets/favicon.png",
	})

	assertResponseContains(t, app, "/docs/index.css", http.StatusOK, "background: url(\"/assets/light.png\") center / contain no-repeat;")
	assertResponseContains(t, app, "/docs/index.css", http.StatusOK, "background-image: url(\"/assets/dark.png\");")
	assertResponseContains(t, app, "/docs/index.html", http.StatusOK, "/assets/favicon.png")
}

func assertResponseContains(t *testing.T, app *fiber.App, target string, wantStatus int, wantBody string) {
	t.Helper()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, target, http.NoBody))
	if err != nil {
		t.Fatalf("request %s: %v", target, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %s: got %d want %d body=%s", target, resp.StatusCode, wantStatus, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", target, err)
	}

	if !strings.Contains(string(body), wantBody) {
		t.Fatalf("body %s missing %q: %s", target, wantBody, body)
	}
}
