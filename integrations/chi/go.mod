module github.com/thebases/go-openapi/integrations/chi

go 1.25.0

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/thebases/go-openapi v1.0.0
)

// Local development resolves core from this repository. Consumers ignore
// replace directives and get the tagged root release required above, so the
// root module must be tagged before this module (see RELEASING.md).
replace github.com/thebases/go-openapi => ../..
