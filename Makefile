# go-openapi build and release helpers.
#
# The repository holds one dependency-free root module (core, ui) plus one
# module per framework integration. A release tags the root module first
# (vX.Y.Z), then every integration (integrations/<fw>/vX.Y.Z), because each
# integration's go.mod requires that exact root version (see RELEASING.md).
# examples/* are separate modules with local replace directives and are
# verified but never released.

MODULE         := github.com/thebases/go-openapi
GO             ?= go
INTEGRATIONS   := chi echo fiber gin iris
EXAMPLES       := examples/chi examples/fiber examples/gin
MODULES        := . $(addprefix integrations/,$(INTEGRATIONS)) $(EXAMPLES)
LIBRARIES      := . $(addprefix integrations/,$(INTEGRATIONS))
RELEASE_BRANCH ?= master
REMOTE         ?= origin
GOPROXY_URL    ?= https://proxy.golang.org

# Default to the latest version heading in CHANGELOG.md so `make release`
# tags exactly what the changelog documents. Override with VERSION=vX.Y.Z.
VERSION ?= $(shell sed -n 's/^## \[\(v[0-9][^]]*\)\].*/\1/p' CHANGELOG.md | head -n1)

# Tags pushed for VERSION: the root tag, then one per integration module.
TAGS = $(VERSION) $(foreach fw,$(INTEGRATIONS),integrations/$(fw)/$(VERSION))

.DEFAULT_GOAL := help

.PHONY: help build test test-race vet fmt fmt-check lint vuln tidy tidy-check examples \
        check version release-check tag push-tag publish release

help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# foreach-module runs $(1) inside every module directory in $(2).
define foreach-module
	@for d in $(2); do echo "==> $$d"; (cd $$d && $(1)) || exit 1; done
endef

build: ## Compile every module
	$(call foreach-module,$(GO) build ./...,$(MODULES))

test: ## Run tests in every module
	$(call foreach-module,$(GO) test ./...,$(MODULES))

test-race: ## Run tests with the race detector in every module
	$(call foreach-module,$(GO) test -race -count=1 ./...,$(MODULES))

vet: ## Run go vet in every module
	$(call foreach-module,$(GO) vet ./...,$(MODULES))

fmt: ## Format Go sources in place
	gofmt -s -w .

fmt-check: ## Fail if any Go file is not gofmt-clean
	@out="$$(gofmt -s -l .)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

lint: ## Run staticcheck on the library modules
	$(call foreach-module,$(GO) run honnef.co/go/tools/cmd/staticcheck@latest ./...,$(LIBRARIES))

vuln: ## Run govulncheck on the library modules (use the latest Go patch release)
	$(call foreach-module,$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...,$(LIBRARIES))

tidy: ## go mod tidy in every module
	$(call foreach-module,$(GO) mod tidy,$(MODULES))

# Mirrors the CI "Assert tidy state" step: tidy must be a no-op.
tidy-check: tidy ## Fail if go mod tidy changes any go.mod/go.sum
	git diff --exit-code -- $(addsuffix /go.mod,$(MODULES)) $(wildcard $(addsuffix /go.sum,$(MODULES)))

examples: ## Build and test every example module
	$(call foreach-module,$(GO) build -o /dev/null ./... && $(GO) test ./...,$(EXAMPLES))

check: fmt-check vet build test-race ## Full local verification (matches CI)

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
	@# Each integration must require exactly the root version being released,
	@# otherwise consumers resolve an older core than the one it was tested with.
	@for fw in $(INTEGRATIONS); do \
		grep -q "^\s*$(MODULE) $(VERSION)$$" integrations/$$fw/go.mod || \
			{ echo "integrations/$$fw/go.mod must require $(MODULE) $(VERSION)"; exit 1; }; \
	done
	@for t in $(TAGS); do \
		! git rev-parse -q --verify "refs/tags/$$t" >/dev/null || { echo "tag $$t already exists locally"; exit 1; }; \
		! git ls-remote --exit-code --tags $(REMOTE) "refs/tags/$$t" >/dev/null 2>&1 || { echo "tag $$t already exists on $(REMOTE)"; exit 1; }; \
	done
	@git fetch -q $(REMOTE) $(RELEASE_BRANCH) && \
		test "$$(git rev-parse HEAD)" = "$$(git rev-parse $(REMOTE)/$(RELEASE_BRANCH))" || \
		{ echo "HEAD is not in sync with $(REMOTE)/$(RELEASE_BRANCH); push or pull first"; exit 1; }

tag: release-check check ## Create the annotated release tags locally
	@for t in $(TAGS); do git tag -a $$t -m "$$t" || exit 1; echo "tagged $$t"; done

# The root tag is pushed and published first so the proxy can serve it by the
# time the integration tags (which require it) are fetched.
push-tag: ## Push the release tags to the remote (root first)
	@for t in $(TAGS); do \
		git rev-parse -q --verify "refs/tags/$$t" >/dev/null || { echo "tag $$t does not exist; run 'make tag' first"; exit 1; }; \
		git push $(REMOTE) $$t || exit 1; \
	done

# Asks the module proxy to fetch every new tag so `go get` and pkg.go.dev see
# them immediately instead of on first consumer request.
publish: ## Warm proxy.golang.org / pkg.go.dev for VERSION
	GOPROXY=$(GOPROXY_URL) GOFLAGS=-mod=mod $(GO) list -m $(MODULE)@$(VERSION)
	@for fw in $(INTEGRATIONS); do \
		GOPROXY=$(GOPROXY_URL) GOFLAGS=-mod=mod $(GO) list -m $(MODULE)/integrations/$$fw@$(VERSION) || exit 1; \
	done

release: tag push-tag publish ## Verify, tag, push, and publish VERSION
	@echo "Released $(MODULE)@$(VERSION) and integrations $(INTEGRATIONS)"
