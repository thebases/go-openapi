package docs

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
	spec "github.com/thebases/go-openapi/openapi"
)

// swaggerUIAssets embeds the vendored Swagger UI bundle so the package can
// serve documentation without requiring runtime files on disk.
//
//go:embed ui/*
var swaggerUIAssets embed.FS

func Register(app *fiber.App, jsonPath string, document func() *spec.Document) {
	docsPath := routeDocsPath(jsonPath)
	trailingDocsPath := docsPath + "/"

	app.Get(jsonPath, func(c fiber.Ctx) error {
		data, err := json.MarshalIndent(document(), "", "  ")
		if err != nil {
			return err
		}

		c.Set(fiber.HeaderContentType, "application/json; charset=utf-8")
		return c.Send(data)
	})

	app.Get(docsPath, func(c fiber.Ctx) error {
		return sendSwaggerUIAsset(c, jsonPath, "index.html")
	})

	app.Get(trailingDocsPath, func(c fiber.Ctx) error {
		return sendSwaggerUIAsset(c, jsonPath, "index.html")
	})

	app.Get(path.Join(docsPath, "*"), func(c fiber.Ctx) error {
		assetPath := strings.TrimPrefix(c.Path(), trailingDocsPath)
		if assetPath == "" {
			assetPath = "index.html"
		}

		return sendSwaggerUIAsset(c, jsonPath, assetPath)
	})
}

func sendSwaggerUIAsset(c fiber.Ctx, jsonPath string, assetPath string) error {
	data, err := swaggerUIFile(assetPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c.SendStatus(http.StatusNotFound)
		}

		return err
	}

	// The bundled initializer targets the Swagger petstore by default, so the
	// served copy must be rewritten to the generated document route.
	if assetPath == "swagger-initializer.js" {
		data = []byte(strings.Replace(
			string(data),
			"https://petstore.swagger.io/v2/swagger.json",
			jsonPath,
			1,
		))
	}

	c.Type(path.Ext(assetPath))
	return c.Send(data)
}

func swaggerUIFile(assetPath string) ([]byte, error) {
	cleanedAssetPath := strings.TrimPrefix(path.Clean("/"+assetPath), "/")
	if cleanedAssetPath == "" || cleanedAssetPath == "." {
		cleanedAssetPath = "index.html"
	}

	return fs.ReadFile(swaggerUIAssets, path.Join("ui", cleanedAssetPath))
}

func NormalizeRoutePath(routePath string) string {
	cleanedPath := path.Clean("/" + strings.TrimSpace(routePath))
	if cleanedPath == "." {
		return "/"
	}

	return cleanedPath
}

func routeDocsPath(jsonPath string) string {
	docsPath := path.Dir(jsonPath)
	if docsPath == "." || docsPath == "/" {
		return "/docs"
	}

	return NormalizeRoutePath(docsPath)
}
