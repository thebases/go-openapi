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

- The repository publishes one root-module tag per release, for example `v0.0.6`.
- The same root tag covers `core`, `ui`, and every `integrations/*` package.
- Tags are annotated, follow [Semantic Versioning](https://semver.org/), and are cut from `master`.
- The version lives only in the git tag and in `CHANGELOG.md`. There is no version constant in Go code.
- A pushed tag is cached by the Go module proxy permanently. Never move or re-push a published tag; fix forward with a new patch version instead.

## Release flow

The `Makefile` automates the whole flow. `VERSION` defaults to the first `## [vX.Y.Z]` heading in `CHANGELOG.md`.

1. Move the `## [Unreleased]` notes in `CHANGELOG.md` under a new `## [vX.Y.Z] - YYYY-MM-DD` heading, and update any version shown in `README.md`.
2. Run the local checks:

   ```bash
   make check        # gofmt, vet, build, test, examples (same as CI)
   make tidy-check   # go mod tidy must be a no-op everywhere
   ```

3. Commit and push `master`, and confirm the `Release Readiness` workflow is green.
4. Publish:

   ```bash
   make version      # confirm the version that will be tagged
   make release      # or: make release VERSION=vX.Y.Z
   ```

`make release` runs these steps and stops at the first failure:

| Step | Target | What it does |
| --- | --- | --- |
| 1 | `release-check` | Checks that the version is semver, you are on `master`, the working tree is clean, `CHANGELOG.md` has a `## [VERSION]` section, the tag does not exist locally or on `origin`, and `HEAD` matches `origin/master`. |
| 2 | `check` | Runs the full local verification again. |
| 3 | `tag` | Creates the annotated tag. |
| 4 | `push-tag` | Pushes the tag to `origin`. |
| 5 | `publish` | Asks `proxy.golang.org` to fetch the version, so `go get` and pkg.go.dev see it immediately. |

If a step after `tag` fails, rerun the remaining targets individually (`make push-tag`, then `make publish`) instead of `make release`, because `release-check` will now refuse the existing tag.

## Pre-tag checklist

1. `make check` and `make tidy-check` pass, and the CI workflow in `.github/workflows/release-readiness.yml` is green for the root module and the nested example modules.
2. Read `README.md` as a new consumer and confirm every documented import path, option, and default exists exactly as written.
3. Confirm the docs-serving package path is still `github.com/thebases/go-openapi/ui`.
4. Confirm examples remain example-only and still compile with their local `replace` directives.
5. Confirm `core.New(...).JSON()` produces a valid document for all three supported `WithOpenAPIVersion` values (`0`=3.0.4, `1`=3.1.1, `2`=3.2.0, the default), and that 3.0.4 output has no 3.1/3.2-only fields (`jsonSchemaDialect`, `webhooks`, `components.pathItems`, `license.identifier`, `nullable`-as-type-array).

## Consumer validation

After publishing, validate the root module from a clean temporary directory:

```bash
mkdir -p /tmp/root-check && cd /tmp/root-check
go mod init example.com/root-check
go get github.com/thebases/go-openapi@v0.0.6
```

Validate at least one integration package from a clean temporary module:

```bash
mkdir -p /tmp/chi-check && cd /tmp/chi-check
go mod init example.com/chi-check
go get github.com/thebases/go-openapi/integrations/chi@v0.0.6
```

Repeat the same pattern for `echo`, `fiber`, `gin`, and `iris` as needed. Confirm that <https://pkg.go.dev/github.com/thebases/go-openapi@v0.0.6> lists the new version (it can take a few minutes).

Validate the nested example modules from their own directories, because they remain separate modules with local `replace` directives.

## Release notes

Every public tag should include release notes (the matching `CHANGELOG.md` section is the source) that state:

- the root module version
- the package surfaces covered by that root release
- the supported OpenAPI spec versions (3.0.4, 3.1.1, 3.2.0) and the default (3.2.0)
- any import-path or API compatibility notes
- the minimum supported Go version (currently 1.25)
- any known limitations that remain after the release
