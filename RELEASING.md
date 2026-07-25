# Releasing go-openapi

This repository has one root module and multiple nested modules. A public release is only valid when the documentation, module paths, and tags all describe the same install surface.

## Supported release surfaces

- Root module: `github.com/thebases/go-openapi`
- Integration modules:
  - `github.com/thebases/go-openapi/integrations/chi`
  - `github.com/thebases/go-openapi/integrations/echo`
  - `github.com/thebases/go-openapi/integrations/fiber`
  - `github.com/thebases/go-openapi/integrations/gin`
  - `github.com/thebases/go-openapi/integrations/iris`

Example modules under `examples/*` are not release artifacts.

## Tagging strategy

- Root module tags use the normal module tag form, for example `v0.0.1`.
- Nested integration modules must use module-path-prefixed tags, for example:
  - `integrations/chi/v0.0.1`
  - `integrations/echo/v0.0.1`
  - `integrations/fiber/v0.0.1`
  - `integrations/gin/v0.0.1`
  - `integrations/iris/v0.0.1`

Use the same semantic version across the root and supported integration modules for the first public release unless maintainers intentionally want separate version streams.

## Pre-tag checklist

1. Run the CI workflow in `.github/workflows/release-readiness.yml` and confirm every module passes.
2. Run `go mod tidy` in the root module and each supported integration module, then commit any `go.mod` or `go.sum` drift.
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

Validate each supported integration module in its own clean temporary module:

```powershell
New-Item -ItemType Directory -Force tmp-chi | Out-Null
Set-Location tmp-chi
go mod init example.com/chi-check
go get github.com/thebases/go-openapi/integrations/chi@integrations/chi/v0.0.1
```

Repeat the same pattern for `echo`, `fiber`, `gin`, and `iris`.

## Release notes

Every public tag should include release notes that state:

- the root module version
- the integration module versions published with it
- any import-path or API compatibility notes
- the minimum supported Go version
- any known limitations that remain after the release
