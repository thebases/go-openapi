# Changelog

All notable changes to this project are documented in this file.
Versioning follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
