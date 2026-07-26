# go-openapi

A framework-neutral OpenAPI 3.0 document generation library for Go, with optional docs UI serving and framework-specific route registration helpers.

[![License](https://img.shields.io/github/license/thebases/go-openapi)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/thebases/go-openapi/core.svg)](https://pkg.go.dev/github.com/thebases/go-openapi/core)
[![Go Version](https://img.shields.io/github/go-mod/go-version/thebases/go-openapi)](https://github.com/thebases/go-openapi/blob/main/go.mod)

You can use `go-openapi` as a small code-first OpenAPI builder, or pair it with one of the integration packages to register application routes and OpenAPI operations together.

> NOTE: the repository follows a single-module layout.
>
> The primary consumer entrypoint is `github.com/thebases/go-openapi/core`.
> Docs UI serving lives in `github.com/thebases/go-openapi/ui`.
> Framework integrations ship as packages under `github.com/thebases/go-openapi/integrations/...`.

## Start here

If you are integrating `go-openapi` into an application, begin with the consumer guide:

- [Using go-openapi](documents/using-go-openapi.md)

That guide covers import-path selection, first-run setup, docs mounting behavior, and framework-specific integration patterns.

## Contents

- [go-openapi](#go-openapi)
  - [Start here](#start-here)
  - [Contents](#contents)
  - [Status](#status)
  - [Import this library in your project](#import-this-library-in-your-project)
  - [Module contents](#module-contents)
  - [Dependencies](#dependencies)
  - [Usage](#usage)
    - [Core usage](#core-usage)
    - [Docs UI handlers](#docs-ui-handlers)
    - [Framework integrations](#framework-integrations)
    - [Facade namespaces](#facade-namespaces)
  - [Examples](#examples)
  - [Release verification](#release-verification)
  - [Roadmap](#roadmap)
  - [Change log](#change-log)
  - [Licensing](#licensing)
  - [Other documentation](#other-documentation)

## Status

The public API is stable enough for consumer adoption, and the repository is organized as one root Go module with focused public packages.

Supported release surfaces:

- Root module: `github.com/thebases/go-openapi`
- Public core package: `github.com/thebases/go-openapi/core`
- Docs UI package path: `github.com/thebases/go-openapi/ui`
- Integration packages:
  - `github.com/thebases/go-openapi/integrations/chi`
  - `github.com/thebases/go-openapi/integrations/echo`
  - `github.com/thebases/go-openapi/integrations/fiber`
  - `github.com/thebases/go-openapi/integrations/gin`
  - `github.com/thebases/go-openapi/integrations/iris`

Example modules under `examples/*` are intended for local validation and copyable samples. They are not supported release artifacts.

## Import this library in your project

Use the module that matches the surface you need:

```bash
go get github.com/thebases/go-openapi
```

Compatibility note: the primary import path moved from `github.com/thebases/go-openapi/openapi` to `github.com/thebases/go-openapi/core`. Integration imports stay under `github.com/thebases/go-openapi/integrations/...`, but they now resolve from the same root module release instead of nested module tags.

For application code, import the core package directly:

```go
import core "github.com/thebases/go-openapi/core"
```

If you want the docs UI package path directly, import:

```go
import docs "github.com/thebases/go-openapi/ui"
```

The `ui` directory is the real import path. Its Go package name is `docs`.

## Module contents

`go-openapi` exposes a small set of focused public packages.

| Module | Purpose |
| --- | --- |
| `core` | Framework-neutral OpenAPI 3.0 document model, schema helpers, validation, and namespace helpers |
| `ui` | Embedded docs UI handlers for `swagger`, `base`, and `scalar` providers |
| `integrations/chi` | Chi route registration helpers that register runtime routes and OpenAPI metadata together |
| `integrations/echo` | Echo route registration helpers |
| `integrations/fiber` | Fiber v2/v3 route registration helpers |
| `integrations/gin` | Gin route registration helpers |
| `integrations/iris` | Iris route registration helpers |

Compatibility notes:

- `core` is the primary public API for document generation and route registration.
- `ui` is the supported docs-serving package path for direct imports.
- `integrations/*` are first-class supported packages in the root module.
- `examples/*` modules keep local `replace` directives by design and remain example-only.

## Dependencies

The root module keeps a single release surface while integrations pull framework-specific dependencies only when you import them.

- The current minimum supported Go version is `1.25.0`.
- Framework integrations are released with the root module instead of separate nested module tags.
- If you are consuming a framework integration for the first time, you may need to run `go mod tidy` in a network-enabled environment so Go can resolve that framework's dependencies.

## Usage

### Core usage

Create one shared API document during application startup:

```go
import (
	"net/http"

	core "github.com/thebases/go-openapi/core"
)

api := core.New(
	core.WithTitle("Merchant API"),
	core.WithDescription("Merchant management endpoints"),
	core.WithVersion("1.0.0"),
	core.WithServer("https://api.example.com", "Production"),
	core.WithDocStyle(core.DocsSwagger),
)
```

Register reusable schemas, then add operations:

```go
if err := api.RegisterSchema("Merchant", core.InlineSchema(&core.Schema{
	Type: "object",
	Properties: map[string]*core.SchemaOrReference{
		"id":   core.StringSchema(),
		"name": core.StringSchema(),
	},
	Required: []string{"id", "name"},
})); err != nil {
	panic(err)
}

err := api.AddOperation(http.MethodGet, "/merchants/{id}", core.Operation{
	OperationID: "getMerchant",
	Summary:     "Get merchant",
	Parameters: []core.ParameterOrReference{
		core.PathParameter("id", core.StringSchema()),
	},
	Responses: map[string]core.ResponseOrReference{
		"200": core.JSONResponse("Merchant returned", core.RefSchema("Merchant")),
	},
})
```

`core.RefSchema("Merchant")` only works after `Merchant` is registered under `components.schemas`.

When `WithDocStyle(...)` is set, docs are mounted automatically on:

- `/docs`
- `/openapi.json`

### Docs UI handlers

You can also build the docs handlers directly:

```go
handler, err := core.Docs.Handler(core.DocsConfig{
	Provider:    core.DocsSwagger,
	Title:       "Merchant API",
	DocumentURL: "/openapi.json",
})
if err != nil {
	panic(err)
}

documentHandler := core.Docs.DocumentHandler(api)
```

Available providers:

- `core.DocsSwagger`
- `core.DocsBase`
- `core.DocsScalar`

### Framework integrations

Framework integrations let you register the runtime route and the OpenAPI operation in one call.

Gin:

```go
import (
	api "github.com/thebases/go-openapi/integrations/gin"
	core "github.com/thebases/go-openapi/core"
)

apiDoc := core.New(
	core.WithTitle("Merchant API"),
	core.WithVersion("1.0.0"),
	core.WithDocStyle(core.DocsSwagger),
)

err := api.GET(router, apiDoc, core.Route("/merchants/:id", operation), getMerchant)
```

Fiber:

```go
import (
	api "github.com/thebases/go-openapi/integrations/fiber"
	core "github.com/thebases/go-openapi/core"
)

err := api.GET(app, apiDoc, core.Route("/merchants/:id", operation), getMerchant)
```

`integrations/fiber` now accepts both Fiber v2 and Fiber v3 routers and handlers. Fiber keeps the same adapter import path, so your application continues importing `github.com/thebases/go-openapi/integrations/fiber` while using whichever Fiber major your service already depends on.

Echo:

```go
import (
	api "github.com/thebases/go-openapi/integrations/echo"
	core "github.com/thebases/go-openapi/core"
)

err := api.GET(e, apiDoc, core.Route("/merchants/:id", operation), getMerchant)
```

Iris:

```go
import (
	api "github.com/thebases/go-openapi/integrations/iris"
	core "github.com/thebases/go-openapi/core"
)

err := api.GET(app, apiDoc, core.Route("/merchants/{id:int}", operation), getMerchant)
```

Chi:

```go
import (
	api "github.com/thebases/go-openapi/integrations/chi"
	core "github.com/thebases/go-openapi/core"
)

err := api.GET(router, apiDoc, core.Route("/merchants/{id}", operation), getMerchant)
```

### Namespace helpers

The `core` package also exposes convenience namespace values for single-import usage:

```go
err := core.Gin.GET(router, apiDoc, core.Route("/merchants/:id", operation), getMerchant)
err := core.Fiber.GET(app, apiDoc, core.Route("/merchants/:id", operation), getMerchant)
err := core.Echo.GET(e, apiDoc, core.Route("/merchants/:id", operation), getMerchant)
err := core.Iris.GET(app, apiDoc, core.Route("/merchants/{id:int}", operation), getMerchant)
err := core.Chi.GET(router, apiDoc, core.Route("/merchants/{id}", operation), getMerchant)
```

These helpers use exported namespace values such as `core.Docs`, `core.Gin`, `core.Fiber`, `core.Chi`, `core.Echo`, and `core.Iris`.

## Examples

Each example is a separate module:

- `examples/chi`
- `examples/fiber`
- `examples/gin`

Run an example from its own directory:

```bash
go run .
```

The examples expose:

```text
http://localhost:3000/docs
http://localhost:3000/openapi.json
```

## Release verification

- Root module verification: `go list ./...` and `go test ./...`
- Supported integration verification: run `go test ./...` from the repo root so `integrations/*` are verified as ordinary packages in the root module
- Example verification: run `go test ./...` inside each `examples/*` module
- Consumer verification: validate a clean `go get` flow against the tagged root module and confirm package imports for `core`, `ui`, and `integrations/*` resolve from that single release

See [RELEASING.md](RELEASING.md) for the full release checklist, tag format, and consumer validation flow.

Current workspace verification status:

- `go list ./...` passes for the root module.
- `go test ./...` passes for the root module, including `core`, `ui`, and `integrations/*`.
- `go test ./...` also passes in `examples/chi`, `examples/fiber`, and `examples/gin`.

## Roadmap

Near-term priorities:

- continue strengthening framework-neutral core APIs under `core`
- keep integration packages easy to consume from the root module release
- expand validation coverage and schema-generation edge case handling
- improve docs UX and example coverage across supported integrations

## Change log

This repository does not currently publish a standalone `CHANGELOG.md`.

For release history and tagged versions, use the repository releases page.

## Licensing

This library ships under the Apache 2.0 license. See [LICENSE](LICENSE).

## Other documentation

- [documents/using-go-openapi.md](documents/using-go-openapi.md)
- [RELEASING.md](RELEASING.md)
