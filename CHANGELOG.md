# Changelog

All notable changes to this project are documented in this file.
Versioning follows [Semantic Versioning](https://semver.org/).

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
