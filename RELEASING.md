# Releasing go-openapi

This repository has one root module with multiple public packages. A public release is only valid when the documentation, import paths, and root tag all describe the same install surface.

## Supported release surfaces

- Root module: `github.com/thebases/go-openapi`
- Public packages:
  - `github.com/thebases/go-openapi/core`
  - `github.com/thebases/go-openapi/ui`
  - `github.com/thebases/go-openapi/integrations/chi`
  - `github.com/thebases/go-openapi/integrations/echo`
  - `github.com/thebases/go-openapi/integrations/fiber`
  - `github.com/thebases/go-openapi/integrations/gin`
  - `github.com/thebases/go-openapi/integrations/iris`

Example modules under `examples/*` are not release artifacts.

## Tagging strategy

- The repository publishes one root-module tag per release, for example `v0.0.1`.
- The same root tag covers `core`, `ui`, and every `integrations/*` package.

## Pre-tag checklist

1. Run the CI workflow in `.github/workflows/release-readiness.yml` and confirm the root module plus the nested example modules pass.
2. Run `go mod tidy` in the root module, then commit any `go.mod` or `go.sum` drift.
3. Read `README.md` as a new consumer and confirm every documented import path exists exactly as written.
4. Confirm the docs-serving package path is still `github.com/thebases/go-openapi/ui`.
5. Confirm examples remain example-only and still compile with their local `replace` directives.

## Consumer validation

Validate the root module from a clean temporary directory:

```powershell
New-Item -ItemType Directory -Force tmp-root | Out-Null
Set-Location tmp-root
go mod init example.com/root-check
go get github.com/thebases/go-openapi@v0.0.1
```

Validate at least one integration package from a clean temporary module:

```powershell
New-Item -ItemType Directory -Force tmp-chi | Out-Null
Set-Location tmp-chi
go mod init example.com/chi-check
go get github.com/thebases/go-openapi/integrations/chi@v0.0.1
```

Repeat the same pattern for `echo`, `fiber`, `gin`, and `iris` as needed.

Validate the nested example modules from their own directories because they remain separate modules with local `replace` directives.

## Release notes

Every public tag should include release notes that state:

- the root module version
- the package surfaces covered by that root release
- any import-path or API compatibility notes
- the minimum supported Go version
- any known limitations that remain after the release
