# go-openapi build and release helpers.
#
# The repo publishes one root-module tag per release (see RELEASING.md);
# examples/* are separate modules with local replace directives and are
# verified but never released.

MODULE         := github.com/thebases/go-openapi
GO             ?= go
EXAMPLES       := examples/chi examples/fiber examples/gin
RELEASE_BRANCH ?= master
REMOTE         ?= origin
GOPROXY_URL    ?= https://proxy.golang.org

# Default to the latest version heading in CHANGELOG.md so `make release`
# tags exactly what the changelog documents. Override with VERSION=vX.Y.Z.
VERSION ?= $(shell sed -n 's/^## \[\(v[0-9][^]]*\)\].*/\1/p' CHANGELOG.md | head -n1)

.DEFAULT_GOAL := help

.PHONY: help build test test-race vet fmt fmt-check tidy tidy-check examples \
        check version release-check tag push-tag publish release

help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Compile every package in the root module
	$(GO) build ./...

test: ## Run root-module tests
	$(GO) test ./...

test-race: ## Run root-module tests with the race detector
	$(GO) test -race ./...

vet: ## Run go vet on the root module
	$(GO) vet ./...

fmt: ## Format Go sources in place
	gofmt -s -w .

fmt-check: ## Fail if any Go file is not gofmt-clean
	@out="$$(gofmt -s -l .)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

tidy: ## go mod tidy for the root module and examples
	$(GO) mod tidy
	@for d in $(EXAMPLES); do echo "tidy $$d"; (cd $$d && $(GO) mod tidy) || exit 1; done

# Mirrors the CI "Assert tidy state" step: tidy must be a no-op.
tidy-check: tidy ## Fail if go mod tidy changes go.mod/go.sum anywhere
	git diff --exit-code -- go.mod go.sum $(addsuffix /go.mod,$(EXAMPLES)) $(addsuffix /go.sum,$(EXAMPLES))

examples: ## Build and test every example module
	@for d in $(EXAMPLES); do \
		echo "==> $$d"; \
		(cd $$d && $(GO) build -o /dev/null ./... && $(GO) test ./...) || exit 1; \
	done

check: fmt-check vet build test examples ## Full local verification (matches CI)

version: ## Print the version that `make release` would tag
	@echo $(VERSION)

# Guards run before anything outward-facing: a tag that is pushed is cached
# by the module proxy forever, so a bad release cannot be taken back.
release-check:
	@test -n "$(VERSION)" || { echo "VERSION is empty"; exit 1; }
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || \
		{ echo "VERSION '$(VERSION)' is not semver (vX.Y.Z)"; exit 1; }
	@test "$$(git rev-parse --abbrev-ref HEAD)" = "$(RELEASE_BRANCH)" || \
		{ echo "not on $(RELEASE_BRANCH)"; exit 1; }
	@test -z "$$(git status --porcelain)" || \
		{ echo "working tree is dirty; commit or stash first"; exit 1; }
	@grep -q '^## \[$(VERSION)\]' CHANGELOG.md || \
		{ echo "CHANGELOG.md has no '## [$(VERSION)]' section"; exit 1; }
	@! git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null || \
		{ echo "tag $(VERSION) already exists locally"; exit 1; }
	@! git ls-remote --exit-code --tags $(REMOTE) "refs/tags/$(VERSION)" >/dev/null 2>&1 || \
		{ echo "tag $(VERSION) already exists on $(REMOTE)"; exit 1; }
	@git fetch -q $(REMOTE) $(RELEASE_BRANCH) && \
		test "$$(git rev-parse HEAD)" = "$$(git rev-parse $(REMOTE)/$(RELEASE_BRANCH))" || \
		{ echo "HEAD is not in sync with $(REMOTE)/$(RELEASE_BRANCH); push or pull first"; exit 1; }

tag: release-check check ## Create the annotated release tag locally
	git tag -a $(VERSION) -m "$(VERSION)"

push-tag: ## Push the release tag to the remote
	@git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null || \
		{ echo "tag $(VERSION) does not exist; run 'make tag' first"; exit 1; }
	git push $(REMOTE) $(VERSION)

# Asks the module proxy to fetch the new tag so `go get` and pkg.go.dev
# see it immediately instead of on first consumer request.
publish: ## Warm proxy.golang.org / pkg.go.dev for VERSION
	GOPROXY=$(GOPROXY_URL) GOFLAGS=-mod=mod $(GO) list -m $(MODULE)@$(VERSION)
	@curl -fsS "$(GOPROXY_URL)/$(MODULE)/@v/$(VERSION).info" && echo

release: tag push-tag publish ## Verify, tag, push, and publish VERSION
	@echo "Released $(MODULE)@$(VERSION)"
