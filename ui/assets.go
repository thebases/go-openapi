package docs

import "embed"

// uiAssets keeps the vendored Swagger UI bundle local to the module so docs do
// not depend on external CDNs or drift away from the checked-in assets.
//
//go:embed theme
var uiAssets embed.FS
