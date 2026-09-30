// Package ui serves the embedded OpenAPI docs UIs (Swagger, Base, Scalar) and
// the OpenAPI document itself as plain net/http handlers.
package ui

// Provider selects a docs UI theme.
type Provider string

// Built-in docs UI providers. An unknown provider falls back to Swagger.
const (
	Swagger Provider = "swagger"
	Base    Provider = "base"
	Scalar  Provider = "scalar"
)

// Config configures DocsHandler.
type Config struct {
	Provider    Provider
	Title       string
	DocsPath    string
	DocumentURL string
	// CDNBaseURL overrides the Scalar runtime script URL (default: a pinned
	// jsDelivr release). Pin a version or self-host it in production.
	CDNBaseURL string
	// CustomCSS is raw CSS appended after the selected theme's own styles on
	// the docs page, so its rules override the theme at equal specificity.
	// Empty means no extra <style> block is rendered.
	CustomCSS string
	// ContentSecurityPolicy, when set, is sent as the Content-Security-Policy
	// header of the docs page.
	ContentSecurityPolicy string
}
