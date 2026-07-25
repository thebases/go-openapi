module github.com/thebases/go-openapi/examples/chi

go 1.25.0

require (
    github.com/go-chi/chi/v5 v5.2.3
    github.com/thebases/go-openapi v0.0.0
    github.com/thebases/go-openapi/integrations/chi v0.0.0
)

replace github.com/thebases/go-openapi => ../..
replace github.com/thebases/go-openapi/integrations/chi => ../../integrations/chi
