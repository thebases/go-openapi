package docs

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	spec "github.com/thebases/go-openapi/openapi"
)

// swaggerUIAssets embeds the vendored Swagger UI bundle so the package can
// serve documentation without requiring runtime files on disk.
//
//go:embed ui/*
var swaggerUIAssets embed.FS

const (
	defaultLogoPath      = "./docs/logo.svg"
	defaultFavicon16Path = "./docs/favicon-16x16.png"
	defaultFavicon32Path = "./docs/favicon-32x32.png"
)

type UIConfig struct {
	Logo     string
	DarkLogo string
	Favicon  string
}

type resolvedUIConfig struct {
	Logo        string
	DarkLogo    string
	Favicon16   string
	Favicon32   string
	LogoIsSVG   bool
	NeedsDarkUI bool
}

func Register(app *fiber.App, jsonPath string, uiConfig UIConfig, document func() *spec.Document) {
	docsPath := routeDocsPath(jsonPath)
	trailingDocsPath := docsPath + "/"
	resolvedConfig := resolveUIConfig(uiConfig)

	app.Get(jsonPath, func(c fiber.Ctx) error {
		data, err := json.MarshalIndent(document(), "", "  ")
		if err != nil {
			return err
		}

		c.Set(fiber.HeaderContentType, "application/json; charset=utf-8")
		return c.Send(data)
	})

	app.Get(docsPath, func(c fiber.Ctx) error {
		return sendSwaggerUIAsset(c, jsonPath, resolvedConfig, "index.html")
	})

	app.Get(trailingDocsPath, func(c fiber.Ctx) error {
		return sendSwaggerUIAsset(c, jsonPath, resolvedConfig, "index.html")
	})

	app.Get(path.Join(docsPath, "*"), func(c fiber.Ctx) error {
		assetPath := strings.TrimPrefix(c.Path(), trailingDocsPath)
		if assetPath == "" {
			assetPath = "index.html"
		}

		return sendSwaggerUIAsset(c, jsonPath, resolvedConfig, assetPath)
	})
}

func sendSwaggerUIAsset(c fiber.Ctx, jsonPath string, uiConfig resolvedUIConfig, assetPath string) error {
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
	if assetPath == "index.html" {
		data = rewriteIndexHTML(data, uiConfig)
	}
	if assetPath == "index.css" {
		data = rewriteIndexCSS(data, uiConfig)
	}

	c.Type(path.Ext(assetPath))
	return c.Send(data)
}

func resolveUIConfig(uiConfig UIConfig) resolvedUIConfig {
	logo := strings.TrimSpace(uiConfig.Logo)
	darkLogo := strings.TrimSpace(uiConfig.DarkLogo)
	favicon := strings.TrimSpace(uiConfig.Favicon)

	// Keep the current embedded assets as the default experience when callers do
	// not opt into custom branding.
	if logo == "" && darkLogo == "" {
		logo = defaultLogoPath
	}
	if logo == "" {
		logo = darkLogo
	}
	if darkLogo == "" && logo != "" && !isSVGAsset(logo) {
		// Raster logos cannot be recolored like the built-in SVG mask, so dark
		// mode must reuse the light logo unless the caller provides a dedicated one.
		darkLogo = logo
	}

	resolved := resolvedUIConfig{
		Logo:        logo,
		DarkLogo:    darkLogo,
		Favicon16:   defaultFavicon16Path,
		Favicon32:   defaultFavicon32Path,
		LogoIsSVG:   isSVGAsset(logo),
		NeedsDarkUI: darkLogo != "",
	}
	if favicon != "" {
		resolved.Favicon16 = favicon
		resolved.Favicon32 = favicon
	}
	if resolved.LogoIsSVG && resolved.DarkLogo == "" {
		resolved.DarkLogo = resolved.Logo
	}

	return resolved
}

func rewriteIndexHTML(data []byte, uiConfig resolvedUIConfig) []byte {
	html := string(data)
	html = strings.ReplaceAll(html, "./docs/favicon-32x32.png", uiConfig.Favicon32)
	html = strings.ReplaceAll(html, "./docs/favicon-16x16.png", uiConfig.Favicon16)
	return []byte(html)
}

func rewriteIndexCSS(data []byte, uiConfig resolvedUIConfig) []byte {
	css := string(data)
	if uiConfig.LogoIsSVG {
		return []byte(strings.ReplaceAll(css, "logo.svg", strings.TrimPrefix(uiConfig.Logo, "./docs/")))
	}

	// Override the embedded mask-based SVG styling so raster logos can be
	// swapped in without editing the vendored Swagger UI bundle on disk.
	var builder strings.Builder
	builder.WriteString(css)
	builder.WriteString("\n.swagger-ui .topbar .topbar-wrapper a svg {")
	builder.WriteString("\n    background: url(\"")
	builder.WriteString(uiConfig.Logo)
	builder.WriteString("\") center / contain no-repeat;")
	builder.WriteString("\n    background-color: transparent;")
	builder.WriteString("\n    -webkit-mask: none;")
	builder.WriteString("\n    mask: none;")
	builder.WriteString("\n}\n")
	builder.WriteString(".swagger-ui .topbar .topbar-wrapper a svg * {")
	builder.WriteString("\n    fill: transparent;")
	builder.WriteString("\n}\n")
	if uiConfig.NeedsDarkUI {
		builder.WriteString("html.dark-mode .swagger-ui .topbar .topbar-wrapper a svg {")
		builder.WriteString("\n    background-image: url(\"")
		builder.WriteString(uiConfig.DarkLogo)
		builder.WriteString("\");")
		builder.WriteString("\n}\n")
	}

	return []byte(builder.String())
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

func isSVGAsset(assetPath string) bool {
	return strings.EqualFold(filepath.Ext(assetPath), ".svg")
}
