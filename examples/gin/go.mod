module github.com/thebases/go-openapi/examples/gin

go 1.25.0

require (
    github.com/gin-gonic/gin v1.10.1
    github.com/thebases/go-openapi v0.0.0
    github.com/thebases/go-openapi/integrations/gin v0.0.0
)

replace github.com/thebases/go-openapi => ../..
replace github.com/thebases/go-openapi/integrations/gin => ../../integrations/gin
