# go-openapi — User Guide

| Field | Value |
|---|---|
| Version | 1.0 |
| Last updated | 2026-07-25 |
| Applies to | `github.com/thebases/go-openapi` current repository layout |
| Audience | Go developers integrating OpenAPI generation and docs UI into an application |

---

## Overview

`go-openapi` helps you generate an OpenAPI 3.0 document in Go and expose a documentation UI for that document. You can use it with the core `openapi` package only, or pair it with a framework integration for Gin, Fiber, Chi, Echo, or Iris.

This guide is for consumers of the module. It focuses on what to import, how to get your first API document working, how to expose `/docs`, and what to check when something does not behave as expected.

---

## Prerequisites

Before you start, make sure you have:

- [ ] Go installed
- [ ] A Go application that already uses one of these routing styles:
  - standard `net/http` usage with manual route registration
  - Gin
  - Fiber
  - Chi
  - Echo
  - Iris
- [ ] Permission to add Go module dependencies to your application
- [ ] At least one API endpoint you want to describe in OpenAPI

If you are using a framework integration module for the first time, you may need to run `go mod tidy` in a network-enabled environment so Go can download that framework's dependencies.

---

## Getting started

Follow these steps for the fastest path to a working OpenAPI document and docs page.

1. Add the core module import to your application:

```go
import openapi "github.com/thebases/go-openapi/openapi"
```

2. Create one shared API document instance during application startup:

```go
api := openapi.New(
    openapi.WithTitle("Merchant API"),
    openapi.WithVersion("1.0.0"),
    openapi.WithDescription("Merchant management endpoints"),
    openapi.WithServer("https://api.example.com", "Production"),
    openapi.WithDocStyle(openapi.DocsSwagger),
)
```

3. Register the response schema, then add at least one operation:

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

err := api.AddOperation("GET", "/merchants/{id}", openapi.Operation{
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

If you use `openapi.RefSchema("Merchant")`, register that schema first so the generated document contains `#/components/schemas/Merchant`.

4. Start your application.

5. Open the generated endpoints in your browser:

```text
/docs
/openapi.json
```

You should now see an interactive documentation page at `/docs` and the generated OpenAPI JSON at `/openapi.json`.

---

## How to choose the right module

Use the module that matches the task you are trying to complete.

| If you want to... | Use this import path |
|---|---|
| Build and export the OpenAPI document | `github.com/thebases/go-openapi/openapi` |
| Serve the docs UI directly as `net/http` handlers | `github.com/thebases/go-openapi/ui` |
| Register Gin routes and OpenAPI metadata together | `github.com/thebases/go-openapi/integrations/gin` |
| Register Fiber routes and OpenAPI metadata together | `github.com/thebases/go-openapi/integrations/fiber` |
| Register Chi routes and OpenAPI metadata together | `github.com/thebases/go-openapi/integrations/chi` |
| Register Echo routes and OpenAPI metadata together | `github.com/thebases/go-openapi/integrations/echo` |
| Register Iris routes and OpenAPI metadata together | `github.com/thebases/go-openapi/integrations/iris` |

**Important:** the docs-serving directory path is `ui`, but the Go package name is `docs`.

Example:

```go
import docs "github.com/thebases/go-openapi/ui"
```

---

## How to register routes with a framework integration

If you use one of the supported framework integrations, you can register the runtime route and the OpenAPI operation in one call.

### Gin

1. Import the Gin integration:

```go
import (
    openapigin "github.com/thebases/go-openapi/integrations/gin"
    openapi "github.com/thebases/go-openapi/openapi"
)
```

2. Create an `openapi.Operation` value.
3. Register the route:

```go
err := openapigin.GET(
    router,
    api,
    openapi.Route("/merchants/:id", operation),
    getMerchant,
)
```

**What you will see:** the Gin route is registered, and the OpenAPI document stores the canonical path version of the endpoint.

### Fiber

```go
err := openapifiber.GET(
    app,
    api,
    openapi.Route("/merchants/:id", operation),
    getMerchant,
)
```

### Chi

```go
err := openapichi.GET(
    router,
    api,
    openapi.Route("/merchants/{id}", operation),
    getMerchant,
)
```

### Echo

```go
err := openapiecho.GET(
    e,
    api,
    openapi.Route("/merchants/:id", operation),
    getMerchant,
)
```

### Iris

```go
err := openapiiris.GET(
    app,
    api,
    openapi.Route("/merchants/{id:int}", operation),
    getMerchant,
)
```

> **Note:** Gin, Fiber, and Echo use colon-style parameters such as `:id`. Chi already uses `{id}`. Iris typed parameters such as `{id:int}` are normalized before they are written into the OpenAPI document.

---

## How to serve the docs UI manually

Use this approach when you want full control over where the docs endpoints are mounted.

1. Build a docs handler:

```go
handler, err := openapi.Docs.Handler(openapi.DocsConfig{
    Provider:    openapi.DocsSwagger,
    Title:       "Merchant API",
    DocumentURL: "/openapi.json",
})
```

2. Build a document JSON handler:

```go
documentHandler := openapi.Docs.DocumentHandler(api)
```

3. Mount both handlers in your router.

**What you will see:** your router will expose one endpoint for the HTML docs UI and one endpoint for the OpenAPI JSON document.

If you prefer the lower-level package directly, use:

```go
import docs "github.com/thebases/go-openapi/ui"
```

Then create the handler with:

```go
handler, err := docs.DocsHandler(docs.Config{
    Provider:    docs.Swagger,
    Title:       "Merchant API",
    DocumentURL: "/openapi.json",
})
```

---

## How automatic docs mounting works

If you set `openapi.WithDocStyle(...)` when you create the API instance, docs mounting is enabled automatically.

```go
api := openapi.New(
    openapi.WithTitle("Merchant API"),
    openapi.WithVersion("1.0.0"),
    openapi.WithDocStyle(openapi.DocsSwagger),
)
```

When you register the first route for a specific router, the library also mounts:

- `/docs`
- `/openapi.json`

Use manual `MountDocs(...)` only when you need a custom docs path or a provider override.

Example:

```go
err := openapigin.MountDocs(router, api, "/internal/docs", "/internal/openapi.json", openapi.DocsConfig{
    Provider: openapi.DocsScalar,
    Title:    "Internal Merchant API",
})
```

---

## Settings and configuration

| Setting | Description | Default | When to change |
|---|---|---|---|
| `WithTitle(...)` | Sets the API title shown in the generated document and UI | `API` | Change it for every real application |
| `WithVersion(...)` | Sets the API version string | `0.0.1` | Change it to your release or schema version |
| `WithDescription(...)` | Adds a description to the API metadata | empty | Change it when you want richer docs metadata |
| `WithServer(url, description)` | Adds a server entry to the OpenAPI document | none | Change it when you want the document to advertise one or more environments |
| `WithDocStyle(...)` | Enables docs auto-mounting and selects the default UI provider | disabled | Change it when you want `/docs` and `/openapi.json` exposed automatically |
| `DocsConfig.Provider` | Chooses the docs UI for a manual mount | inherited from API docs style or package default | Change it when a specific mount should use Swagger, Base, or Scalar |
| `DocsConfig.Title` | Overrides the page title for a manual docs mount | inherited from API title when available | Change it when a docs page needs a different label |
| `DocsConfig.DocumentURL` | Tells the docs UI where to fetch the OpenAPI JSON | `/openapi.json` in most standard mounts | Change it when your JSON is mounted elsewhere |

---

## Understanding the published modules

| Module type | What it means |
|---|---|
| Root public API: `openapi` | The main package for document creation, schema registration, JSON output, and facade helpers |
| Docs package: `ui` | The docs-serving package path; imported in Go code as `docs` |
| Integration modules: `integrations/*` | Supported nested modules for framework-specific route registration |
| Example modules: `examples/*` | Local examples for learning and validation; not supported as installable release surfaces |

---

## Troubleshooting

### "I opened `/docs`, but nothing is there"
**Why it happens:** docs auto-mounting only happens when `WithDocStyle(...)` is enabled or when you mount docs handlers manually.

**What to do:**
1. Check that your API instance was created with `openapi.WithDocStyle(...)`.
2. If not, either add `WithDocStyle(...)` or mount the docs handlers yourself.
3. Restart the application and open `/docs` again.

---

### "`/openapi.json` is missing"
**Why it happens:** the JSON handler was not mounted, or docs auto-mounting never ran because no route registration happened on that router.

**What to do:**
1. Confirm that you registered at least one route through the same router object.
2. If you are using manual docs mounting, make sure you mounted both the docs handler and the document handler.
3. Open `/openapi.json` directly in the browser to verify it returns JSON.

---

### "My framework path uses `:id`, but the document shows `{id}`"
**Why it happens:** this is expected behavior. The runtime router keeps its native syntax, but the stored OpenAPI path is normalized into OpenAPI format.

**What to do:** keep using the native route format required by your framework.

---

### "`go mod tidy` fails for an integration module"
**Why it happens:** nested integration modules pull framework-specific dependencies and may need network access to resolve them.

**What to do:**
1. Run `go mod tidy` in a network-enabled environment.
2. Run it inside the specific integration module directory you are using.
3. Re-run your build or tests after the dependency download completes.

---

### "The docs page loads, but it cannot find the JSON document"
**Why it happens:** `DocumentURL` does not match the actual JSON route.

**What to do:**
1. Confirm the exact URL where your application serves the JSON document.
2. Set `DocumentURL` to that path.
3. Refresh the docs page and verify the network request succeeds.

---

## Frequently asked questions

**Q: Which package should I import first?**
A: Start with `github.com/thebases/go-openapi/openapi`. Add `ui` or an `integrations/*` module only if you need docs serving or framework-specific route registration.

**Q: Do I need to use a framework integration?**
A: No. You can use `openapi` by itself and register operations manually.

**Q: Is `ui` a different package from `docs`?**
A: The directory path is `ui`, but the Go package name is `docs`.

**Q: Are the example modules supported for production imports?**
A: No. `examples/*` are for local learning and validation only.

**Q: What are the supported integration modules?**
A: `integrations/chi`, `integrations/echo`, `integrations/fiber`, `integrations/gin`, and `integrations/iris`.

**Q: When should I use `MountDocs(...)`?**
A: Use it when you need a custom docs route, a custom JSON route, or a provider override for a specific mount.

---

## Reference

### Default routes

| Route | Purpose |
|---|---|
| `/docs` | Interactive documentation UI |
| `/openapi.json` | Generated OpenAPI JSON document |

### Supported docs providers

| Provider | Usage |
|---|---|
| `openapi.DocsSwagger` | Standard Swagger UI |
| `openapi.DocsBase` | Base bundled UI |
| `openapi.DocsScalar` | Scalar-based UI |

### Related files in this repository

- [README.md](../README.md)
- [examples/fiber/README.md](../examples/fiber/README.md)
- [examples/chi/README.md](../examples/chi/README.md)
- [examples/gin/README.md](../examples/gin/README.md)
- [RELEASING.md](../RELEASING.md)

---

## Getting help

- Start with [README.md](../README.md)
- Use the example modules under `examples/*` to validate expected setup
- If nested modules fail to resolve locally, retry in a network-enabled environment
