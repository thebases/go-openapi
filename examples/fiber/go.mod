module github.com/thebases/go-openapi/examples/fiber

go 1.25.0

require (
    github.com/gofiber/fiber/v3 v3.0.0
    github.com/thebases/go-openapi v0.0.0
    github.com/thebases/go-openapi/integrations/fiber v0.0.0
)

replace github.com/thebases/go-openapi => ../..
replace github.com/thebases/go-openapi/integrations/fiber => ../../integrations/fiber
