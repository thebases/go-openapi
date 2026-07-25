# GO-OPENAPI

A from-scratch OpenAPI 3.0 document generation library for Go and Fiber.

This project intentionally does **not** depend on any OpenAPI generator or schema-generation library. It implements:

- A native Go model of the OpenAPI 3.0 document
- Go `reflect`-based schema generation
- Typed Fiber route registration
- OpenAPI path, parameter, request-body, response, schema, component, and security generation
- JSON document output
- Basic semantic validation

## Goal

Application code:

```go
openapi.GET(
    api,
    "/merchants/:id",
    openapi.OperationConfig{
        ID:      "getMerchant",
        Summary: "Get merchant",
        Tags:    []string{"Merchants"},
    },
    func(ctx fiber.Ctx, input GetMerchantInput) (Merchant, error) {
        return Merchant{
            ID:     input.ID,
            Name:   "The Base Store",
            Status: "active",
        }, nil
    },
)
```

Produces an OpenAPI 3.0 document and Swagger UI at:

```text
GET /docs
GET /docs/openapi.json
```

## Design constraints

- No dependency on `kin-openapi`
- No dependency on `swaggo`
- No dependency on `swaggest`
- No dependency on `openapi3gen`
- OpenAPI 3.0.x only in the initial implementation
- JSON output first
- Fiber integration kept separate from reflection and document modeling

## Package layout

```text
openapi/
├── api.go
├── components.go
├── document.go
├── errors.go
├── extension.go
├── marshal.go
├── operation.go
├── parameter.go
├── path.go
├── reference.go
├── reflector.go
├── route.go
├── schema.go
├── security.go
├── validate.go
├── example/
│   └── main.go
├── architecture.md
├── implementation-notes.md
└── context.md
```

## Current implementation status

This ZIP is an implementation scaffold and reference design. It contains the core code required to continue development, but it has not been compiled against a pinned Fiber v3 release in this environment.

Before publishing a release:

1. Pin the exact Fiber v3 version.
2. Run `go mod tidy`.
3. Fix any Fiber v3 API differences.
4. Add unit tests.
5. Add golden-file tests for generated OpenAPI JSON.
6. Validate generated documents with an independent OpenAPI validator in CI.

## Recommended first release scope

- OpenAPI 3.0.3
- JSON document output
- GET, POST, PUT, PATCH, DELETE
- Path, query, header, and cookie parameters
- JSON request bodies
- JSON responses
- Primitive, struct, slice, array, map, and `time.Time` reflection
- Component schemas
- Recursive schema support
- Bearer authentication
- Basic semantic validation

