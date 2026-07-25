# go-openapi

A framework-neutral OpenAPI 3.0 document generation library for Go.

Published module surfaces:

```text
github.com/thebases/go-openapi/openapi
github.com/thebases/go-openapi/ui
github.com/thebases/go-openapi/integrations/chi
github.com/thebases/go-openapi/integrations/echo
github.com/thebases/go-openapi/integrations/fiber
github.com/thebases/go-openapi/integrations/gin
github.com/thebases/go-openapi/integrations/iris
```

The `ui` directory is the real docs-serving package path. Its Go package name is `docs`, so direct imports look like:

```go
import docs "github.com/thebases/go-openapi/ui"
```

Framework integrations under `integrations/*` are first-class supported modules. The `openapi` facade remains available as a compatibility layer. Modules under `examples/*` are example-only local modules; they are verified as sample applications, but they are not supported as installable release surfaces.

## Support policy

- `openapi` is the primary public API for document generation and route registration.
- `ui` is the supported docs-serving package path for direct imports.
- `integrations/*` modules are supported and are expected to build and test independently.
- `examples/*` modules are for local demonstration only and keep local `replace` directives by design.
- The current minimum supported Go version is `1.25.0`, matching the checked-in `go.mod` files.

## Usage guide

For the full consumer guide, including import-path selection, auto-mounted docs behavior, and framework-specific route registration examples, see [documents/using-go-openapi.md](documents/using-go-openapi.md).

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

if err := api.RegisterSchema("Merchant", openapi.InlineSchema(&openapi.Schema{
    Type: "object",
    Properties: map[string]*openapi.SchemaOrReference{
        "id":   openapi.StringSchema(),
        "name": openapi.StringSchema(),
    },
    Required: []string{"id", "name"},
})); err != nil {
    panic(err)
}

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

`openapi.RefSchema("Merchant")` only works after `Merchant` is registered under `components.schemas`.

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

err := api.GET(router, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
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

err := api.GET(app, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
```

Echo:

```go
import (
    api "github.com/thebases/go-openapi/integrations/echo"
    openapi "github.com/thebases/go-openapi/openapi"
)

err := api.GET(e, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
```

Iris:

```go
import (
    api "github.com/thebases/go-openapi/integrations/iris"
    openapi "github.com/thebases/go-openapi/openapi"
)

err := api.GET(app, apiDoc, openapi.Route("/merchants/{id:int}", operation), getMerchant)
```

Compatibility facade:

```go
err := openapi.Gin.GET(router, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
err := openapi.Fiber.GET(app, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
err := openapi.Echo.GET(e, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
err := openapi.Iris.GET(app, apiDoc, openapi.Route("/merchants/{id:int}", operation), getMerchant)
```

Chi:

```go
err := openapi.Chi.GET(router, api, openapi.Route("/merchants/{id}", operation), getMerchant)
```

`openapi.docs` and `openapi.gin` are not valid Go syntax, so the facade uses exported namespace values: `openapi.Docs`, `openapi.Gin`, `openapi.Fiber`, `openapi.Chi`, `openapi.Echo`, and `openapi.Iris`.

## Examples

Each example is a separate module:

- `examples/fiber`
- `examples/chi`
- `examples/gin`

They are intended for local validation and copyable usage samples. They are not versioned as supported consumer modules.

## Release verification

- Root module verification: `go list ./...` and `go test ./...`
- Supported integration verification: run `go mod tidy` and `go test ./...` inside each `integrations/*` module
- Example verification: run `go test ./...` inside each `examples/*` module
- Consumer verification: validate a clean `go get` flow against the tagged root module and each tagged integration module before publishing

See [RELEASING.md](RELEASING.md) for the release checklist, tag format, and consumer validation flow.

## Current verification

- `go list ./...` passes for the root module.
- `go test ./...` passes for the root module packages `openapi` and `ui`.
- `integrations/fiber` currently passes `go test ./...` in this workspace.
- `integrations/chi`, `integrations/echo`, `integrations/gin`, and `integrations/iris` are blocked in this restricted session by missing `go.sum` entries and need a network-enabled `go mod tidy`.





