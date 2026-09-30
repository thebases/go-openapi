# Changelog

All notable changes to this project are documented in this file.
Versioning follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [v1.0.0] - 2026-09-30

First stable release. The public API of `core`, `ui`, and every `integrations/*` module now follows Semantic Versioning: no breaking changes until v2 (see "API stability" in `README.md`). Every module is tagged at `v1.0.0`. Upgrading from v0.0.x? See "Migrating to v1.0.0" in `README.md`.

### Breaking

- **Module split.** Each `integrations/<fw>` directory is now its own Go module, tagged `integrations/<fw>/vX.Y.Z`. The root module (`core`, `ui`) has no third-party requires, so installing one integration no longer pulls every framework into `go.sum`. Import paths are unchanged.
- **Fiber v2 dropped.** `integrations/fiber` is typed against Fiber v3 (`fiber.Router`). This removes ~300 lines of reflection.
- The `ui` package is now named `ui` (it was `docs` in directory `ui`).
- `core.Group` has one type parameter (`Group[H]`), and group creation errors are returned from `Handle` or `Err()` instead of panicking (Fiber).
- `WithOpenAPIVersion` takes a `core.SpecVersion`; unknown versions make `JSON()` fail with `ErrUnsupportedVersion`.
- A route needs at least one handler (`ErrNoHandler`). Chi and Echo reject extra handlers instead of silently dropping them.
- `ValidateDocument(nil)` returns an error.

### Changed (generated output)

- Go `int`/`int64` map to `format: int64`, `uint32` to `int64` with `minimum: 0`, and `uint`/`uint64` to `integer` with `minimum: 0` and no format. `int` used to be `int32`, which truncates 64-bit IDs and amounts in generated clients.
- Embedded structs are flattened with `encoding/json` rules. Fields promoted through an embedded pointer are optional.
- `enum`, `example`, and `default` tags are converted to the field's type.
- Anonymous structs are inlined instead of becoming invalid component names.
- Generic type names are sanitized (`Page[User]` → `Page_User`).
- The docs UI is auto-mounted only on the root of a `Root(...)` group tree, not under every group.
- The Swagger page links its bundles as versioned, immutable assets. The HTML response is ~2 KB instead of ~2 MB.
- The Scalar runtime is pinned to `@scalar/api-reference@1.72.2`.
- `/openapi.json` and embedded assets send an `ETag` and answer `If-None-Match` with `304`.
- `Document.Validate()` reports all problems at once (`errors.Join`), in a deterministic order. It also checks undeclared path template variables, duplicate `(name, in)` parameters, response keys, component names, and dangling local `$ref`s.

### Added

- `core.Adapter[H]` / `core.GroupAdapter[H]` interfaces, plus `core.Handle`, `core.MountDocs`, `core.NewRootGroup`, `core.DocsMount`. Each integration exports `NewAdapter(router)`.
- `core.WithDescriptionFS(fs.FS)` for embedded Markdown descriptions. Paths are sandboxed with `fs.ValidPath`, and `..` is rejected with `ErrInvalidDescription`.
- `API.Snapshot()` returns a deep-copied document or the rendering error, for fail-fast boots.
- `core.SchemaNamer` / `core.RequiredPolicy` hooks on `Reflector` (`Namer`, `Required`).
- `ui.Config.ContentSecurityPolicy`.
- Sentinel errors: `ErrNoHandler`, `ErrDuplicateComponent`, `ErrInvalidComponentName`, `ErrUnsupportedVersion`, `ErrInvalidDescription`, `ErrInvalidTag`.
- CI tests every module on its minimum Go and on stable, with `-race`, gofmt, vet, `staticcheck`, and `govulncheck`. `make lint`/`make vuln` are available locally.
- Golden OpenAPI documents for 3.0.4, 3.1.1, and 3.2.0 (`core/testdata/golden`, regenerate with `go test ./core -run Golden -update`). They pin the exact output, and each one must pass `Validate()`.
- Runnable pkg.go.dev examples (`Example`, `ExampleReflector`, `ExampleNewRootGroup`, `ExampleHandle`, `ExampleWithOpenAPIVersion`).
- Doc comments on every exported identifier.

### Removed

- The reflection-based `core.Chi`, `core.Gin`, `core.Fiber`, `core.Echo`, and `core.Iris` facades (`any`-typed routers and handlers, runtime type errors). Use the typed `integrations/<fw>` packages. `core.Docs.Handler` / `core.Docs.DocumentHandler` remain.
- `core.RouteRegistrar`, `core.GroupRegistrar`, `core.DocsRegistrar`, and `core.NewGroup`. Implement `core.Adapter` / `core.GroupAdapter`, then call `core.Handle`, `core.MountDocs`, or `core.NewRootGroup`.
- `Reflector.Visiting`, which was unused.
- `core.ValidateDocument`'s silent acceptance of `nil`.

### Fixed

- A second type with the same name (e.g. `billing.Error` and `auth.Error`) silently reused the first type's schema. It now gets a package-qualified component (`auth_Error`).
- An operation stayed in the document when native registration failed or panicked. Registration is now all-or-nothing.
- `API.Document()` returned internal maps, so callers could mutate state without the lock. It now returns a deep copy.
- `AddOperation` with an unsupported method left an empty path entry.
- Specification extensions (`Extensions` / `x-*`) were never serialized.
- `/openapi.json` re-walked, re-read Markdown files, and re-marshaled the document under an exclusive lock on every request. It now serves a per-revision cache: 24 µs vs 3.9 ms for 200 operations.
- Titles were HTML-escaped twice in the Swagger and Scalar pages (`A &amp;amp; B`).
- Invalid doc paths from unnamed wildcards (`/*`, Fiber `+`), optional params (`:id?`), and Fiber constraints (`:id<int>`).
- `Group("merchants")` without a leading slash produced a relative doc path.
- Malformed numeric tags (`minimum:"abc"`) were silently ignored. They now return `ErrInvalidTag`.
- Compiled example binaries were committed to the repository.

## [v0.0.6] - 2026-09-26

### Added

- `Makefile` with build/test/vet/tidy/example checks (`make check`) and a guarded release flow (`make release VERSION=vX.Y.Z`) that verifies a clean tree, the release branch, a matching `CHANGELOG.md` section, and an unused tag before tagging, pushing, and warming the Go module proxy.
- `CustomCSS` option for the docs UI: `ui.Config.CustomCSS` / `core.DocsConfig.CustomCSS` per mount, and `core.WithCustomCSS(...)` for every docs mount of an API. The CSS is rendered in a `<style id="docs-custom-css">` block after the theme styles for Swagger, Base, and Scalar; a literal `</style` is escaped so it cannot break out of the block.

### Changed

- `README.md` rewritten as the onboarding guide with `core.DocsBase` as the default sample: a runnable quick start, a step-by-step guide (struct reflection and tag reference, operations, security, per-framework routing, groups, docs mounting, Markdown descriptions, custom CSS, spec versions, export), production notes, and troubleshooting.
- `RELEASING.md` documents the `make release` flow, bash consumer-validation commands, and the never-move-a-published-tag rule.
- CI (`release-readiness.yml`) now runs on pushes to `master`, the repository's default branch (it was listening on `main`).
- Base docs UI: Mermaid 11.17.2 is now vendored under `ui/theme/base/js/vendor/` (embedded, MIT license alongside) and lazy-loaded from the docs asset path only when a page contains a diagram, so diagrams render offline and under a same-origin CSP. The CDN script tag was removed.
- Base docs UI markdown: code fences follow CommonMark more closely — ``` or ~~~ fences of 3+ characters, the language is the first word of the info string (so ```` ```mermaid {config} ```` is recognised, case-insensitive), and a fence only closes on a matching fence of equal or greater length.

### Fixed

- `documents/using-go-openapi.md` stated the wrong `WithVersion` default; it is `0.0.0`.
- `examples/gin/go.sum` drift that made the CI tidy check fail; two `core` files brought to `gofmt -s` form.
- Base docs UI: a Mermaid diagram that fails to render, or a runtime that fails to load, now shows an inline error panel with the diagram source instead of failing silently (console only). One bad diagram no longer affects the others on the page.

## [v0.0.5] - 2026-08-23

### Added

- Route grouping / nested sub-router support across all framework integrations (Fiber, Gin, Echo, Chi, Iris) via `core.Group` and a per-integration `Root(router, api)` helper, so nested groups register the correct absolute OpenAPI path while still using framework-relative route paths.
- Documentation for grouped/nested routes in `documents/using-go-openapi.md`, with usage examples per integration.
- Group-registration tests for Chi, Echo, Fiber, Gin, and Iris integrations.

### Changed

- Try-It panel: sending a request now shows a "Request" detail block (URL, headers, body) alongside the "Response" block instead of a bare spinner, so both stay visible together after the response arrives.
- Sidebar toggle button is now hidden at desktop widths (>1100px, aligned with the sidebar's own overlay breakpoint) instead of >750px.

## [v0.0.4] - 2026-07-26

### Added

- Multi-OpenAPI-spec support in core; OpenAPI 3.2.0 is now the default spec version.
- Markdown and Mermaid rendering support in the docs UI.

## [v0.0.3] - 2026-07-26

### Fixed

- Auto-alias handling for renamed files inside the docs subtree.

## [v0.0.2] - 2026-07-26

### Changed

- Fiber v2/v3 support.

### Fixed

- Refactor and bug fixes; removed unused Basecoat CSS.

## [v0.0.1] - 2026-07-25

### Added

- Initial release: Fiber, Gin, Echo, Chi, and Iris integrations with generated OpenAPI docs UI.
