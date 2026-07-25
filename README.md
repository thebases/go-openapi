# go-openapi

A framework-neutral OpenAPI 3.0 document generation library for Go.

The main public package is:

```text
github.com/thebases/go-openapi/openapi
```

The shared docs-serving package still exists at `github.com/thebases/go-openapi/docs`. Framework integrations under `integrations/*` are now the preferred route-registration entrypoint, while the `openapi` facade remains available as a compatibility layer.

## Core usage

```go
import (
    "net/http"

    openapi "github.com/thebases/go-openapi/openapi"
)

api := openapi.New(
    openapi.WithTitle("Merchant API"),
    openapi.WithDescription("Merchant management endpoints"),
    openapi.WithVersion("1.0.0"),
    openapi.WithServer("https://api.example.com", "Production"),
    openapi.WithDocStyle(openapi.DocsSwagger),
)

err := api.AddOperation(http.MethodGet, "/merchants/{id}", openapi.Operation{
    OperationID: "getMerchant",
    Summary:     "Get merchant",
    Parameters: []openapi.ParameterOrReference{
        openapi.PathParameter("id", openapi.StringSchema()),
    },
    Responses: map[string]openapi.ResponseOrReference{
        "200": openapi.JSONResponse("Merchant returned", openapi.RefSchema("Merchant")),
    },
})
```

When `WithDocStyle(...)` is set, docs are mounted automatically on:

- `/docs`
- `/openapi.json`

## Facade namespaces

Docs:

```go
handler, err := openapi.Docs.Handler(openapi.DocsConfig{
    Provider:    openapi.DocsSwagger,
    Title:       "Merchant API",
    DocumentURL: "/openapi.json",
})
```

Gin:

```go
import (
    api "github.com/thebases/go-openapi/integrations/gin"
    openapi "github.com/thebases/go-openapi/openapi"
)

apiDoc := openapi.New(
    openapi.WithTitle("Merchant API"),
    openapi.WithVersion("1.0.0"),
    openapi.WithDocStyle(openapi.DocsSwagger),
)

err := api.GET(router, apiDoc, "/merchants/:id", operation, getMerchant)
```

Fiber:

```go
import (
    api "github.com/thebases/go-openapi/integrations/fiber"
    openapi "github.com/thebases/go-openapi/openapi"
)

apiDoc := openapi.New(
    openapi.WithTitle("Merchant API"),
    openapi.WithVersion("1.0.0"),
    openapi.WithDocStyle(openapi.DocsSwagger),
)

err := api.GET(app, apiDoc, "/merchants/:id", operation, getMerchant)
```

Compatibility facade:

```go
err := openapi.Gin.GET(router, apiDoc, "/merchants/:id", operation, getMerchant)
err := openapi.Fiber.GET(app, apiDoc, "/merchants/:id", operation, getMerchant)
```

Chi:

```go
err := openapi.Chi.GET(router, api, "/merchants/{id}", operation, getMerchant)
```

`openapi.docs` and `openapi.gin` are not valid Go syntax, so the facade uses exported namespace values: `openapi.Docs`, `openapi.Gin`, `openapi.Fiber`, and `openapi.Chi`.

## Examples

Each example is a separate module:

- `examples/fiber`
- `examples/chi`
- `examples/gin`

## Current verification

- `go test ./openapi ./docs` passes.
- The nested integration/example modules still need their own `go.sum` written with `go mod tidy` or `go test ./...` in a network-enabled environment.
