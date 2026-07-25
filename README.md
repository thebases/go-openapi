# go-openapi

A framework-neutral OpenAPI 3.0 document generation library for Go, with optional docs UI serving and framework-specific route registration helpers.

[![License](https://img.shields.io/github/license/thebases/go-openapi)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/thebases/go-openapi/openapi.svg)](https://pkg.go.dev/github.com/thebases/go-openapi/openapi)
[![Go Version](https://img.shields.io/github/go-mod/go-version/thebases/go-openapi)](https://github.com/thebases/go-openapi/blob/main/go.mod)

You can use `go-openapi` as a small code-first OpenAPI builder, or pair it with one of the integration modules to register application routes and OpenAPI operations together.

> NOTE: the repository follows a multi-module layout.
>
> The primary consumer entrypoint is `github.com/thebases/go-openapi/openapi`.
> Docs UI serving lives in `github.com/thebases/go-openapi/ui`.
> Framework integrations ship as nested modules under `github.com/thebases/go-openapi/integrations/...`.

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

The public API is stable enough for consumer adoption, and the repository is organized around supported module surfaces instead of a single monolithic package.

Supported release surfaces:

- Root module: `github.com/thebases/go-openapi`
- Public OpenAPI facade: `github.com/thebases/go-openapi/openapi`
- Docs UI package path: `github.com/thebases/go-openapi/ui`
- Integration modules:
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
go get github.com/thebases/go-openapi/integrations/{chi|echo|fiber|gin|iris}
```

For application code, import the facade package directly:

```go
import openapi "github.com/thebases/go-openapi/openapi"
```

If you want the docs UI package path directly, import:

```go
import docs "github.com/thebases/go-openapi/ui"
```

The `ui` directory is the real import path. Its Go package name is `docs`.

## Module contents

`go-openapi` exposes a small set of focused public modules.

| Module | Purpose |
| --- | --- |
| `openapi` | Framework-neutral OpenAPI 3.0 document model, schema helpers, validation, and facade namespaces |
| `ui` | Embedded docs UI handlers for `swagger`, `base`, and `scalar` providers |
| `integrations/chi` | Chi route registration helpers that register runtime routes and OpenAPI metadata together |
| `integrations/echo` | Echo route registration helpers |
| `integrations/fiber` | Fiber route registration helpers |
| `integrations/gin` | Gin route registration helpers |
| `integrations/iris` | Iris route registration helpers |

Compatibility notes:

- `openapi` is the primary public API for document generation and route registration.
- `ui` is the supported docs-serving package path for direct imports.
- `integrations/*` modules are first-class supported modules and should build independently.
- `examples/*` modules keep local `replace` directives by design and remain example-only.

## Dependencies

The root module keeps a framework-neutral dependency surface.

- The current minimum supported Go version is `1.25.0`.
- The root module does not require a web framework dependency.
- Framework-specific dependencies belong to the corresponding nested module under `integrations/*`.
- If you are consuming a framework integration for the first time, you may need to run `go mod tidy` in a network-enabled environment so Go can resolve that framework's dependencies.

## Usage

### Core usage

Create one shared API document during application startup:

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
```

Register reusable schemas, then add operations:

```go
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

### Docs UI handlers

You can also build the docs handlers directly:

```go
handler, err := openapi.Docs.Handler(openapi.DocsConfig{
	Provider:    openapi.DocsSwagger,
	Title:       "Merchant API",
	DocumentURL: "/openapi.json",
})
if err != nil {
	panic(err)
}

documentHandler := openapi.Docs.DocumentHandler(api)
```

Available providers:

- `openapi.DocsSwagger`
- `openapi.DocsBase`
- `openapi.DocsScalar`

### Framework integrations

Framework integrations let you register the runtime route and the OpenAPI operation in one call.

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

Chi:

```go
import (
	api "github.com/thebases/go-openapi/integrations/chi"
	openapi "github.com/thebases/go-openapi/openapi"
)

err := api.GET(router, apiDoc, openapi.Route("/merchants/{id}", operation), getMerchant)
```

### Facade namespaces

The `openapi` facade also exposes convenience namespace values for single-import usage:

```go
err := openapi.Gin.GET(router, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
err := openapi.Fiber.GET(app, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
err := openapi.Echo.GET(e, apiDoc, openapi.Route("/merchants/:id", operation), getMerchant)
err := openapi.Iris.GET(app, apiDoc, openapi.Route("/merchants/{id:int}", operation), getMerchant)
err := openapi.Chi.GET(router, apiDoc, openapi.Route("/merchants/{id}", operation), getMerchant)
```

The facade uses exported namespace values such as `openapi.Docs`, `openapi.Gin`, `openapi.Fiber`, `openapi.Chi`, `openapi.Echo`, and `openapi.Iris`.

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
- Supported integration verification: run `go mod tidy` and `go test ./...` inside each `integrations/*` module
- Example verification: run `go test ./...` inside each `examples/*` module
- Consumer verification: validate a clean `go get` flow against the tagged root module and each tagged integration module before publishing

See [RELEASING.md](RELEASING.md) for the full release checklist, tag format, and consumer validation flow.

Current workspace verification status:

- `go list ./...` passes for the root module.
- `go test ./...` passes for the root module packages `openapi` and `ui`.
- `integrations/fiber` currently passes `go test ./...` in this workspace.
- `integrations/chi`, `integrations/echo`, `integrations/gin`, and `integrations/iris` require dependency resolution in a network-enabled environment when `go.sum` entries are missing.

## Roadmap

Near-term priorities:

- continue strengthening framework-neutral core APIs under `openapi`
- keep nested integration modules independently consumable and releasable
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
