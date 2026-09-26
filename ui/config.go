package docs

type Provider string

const (
	Swagger Provider = "swagger"
	Base    Provider = "base"
	Scalar  Provider = "scalar"
)

type Config struct {
	Provider    Provider
	Title       string
	DocsPath    string
	DocumentURL string
	CDNBaseURL  string
	// CustomCSS is raw CSS appended after the selected theme's own styles on
	// the docs page, so its rules override the theme at equal specificity.
	// Empty means no extra <style> block is rendered.
	CustomCSS string
}
