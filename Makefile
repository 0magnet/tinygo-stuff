.DEFAULT_GOAL := help
.PHONY: help format tidy lint vet test check install-linters docs

# The targets that matter are `format` and `check`, and they mean the same
# thing here as in 0pcom/skywire, which is the reference for these repos.
# .golangci.yml is copied from there.

PROJECT_BASE := github.com/0magnet/tinygo-stuff
OPTS ?= GO111MODULE=on

# Packages this toolchain cannot build at all — firmware for a different
# target, say. Not the same as code that merely does not build for this host:
# js/wasm is handled below by running the checks again in that context, which
# is the better answer whenever it is available. Empty in most repos.
SKIP ?= /firmware
# Directories rather than import paths, because golangci-lint resolves a bare
# import path against the working directory and then cannot find it.
PKGS = $(shell go list -f '{{.Dir}}' ./... 2>/dev/null $(if $(SKIP),| grep -vE '$(SKIP)'))

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

tidy: ## Tidy dependencies
	${OPTS} go mod tidy -v

format: tidy ## Format the code. Needs goimports (make install-linters)
	@if grep -qE '^(replace|exclude)' go.mod; then \
		echo "ERROR: go.mod contains replace or exclude directives which break go install @version"; \
		grep -E '^(replace|exclude)' go.mod; \
		exit 1; \
	fi
	${OPTS} goimports -w -local ${PROJECT_BASE} $(shell go list -f '{{.Dir}}' ./... 2>/dev/null | grep -v /vendor/)

lint: ## Run golangci-lint. Needs it installed (make install-linters)
	command -v golangci-lint || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	golangci-lint --version
	@# Some of these repos are entirely js/wasm-tagged, so the host context has
	@# nothing in it and linting it is an error rather than a pass.
	@if [ -n "$(PKGS)" ]; then \
		CGO_ENABLED=0 ${OPTS} golangci-lint run -c .golangci.yml $(PKGS); \
	else \
		echo '--- nothing builds for this host; skipping the host pass'; \
	fi
	@# A host run cannot see js/wasm-tagged files, so anything only they use
	@# reads as dead — and anything wrong inside them is never checked at all.
	@if grep -rlq '^//go:build js' --include='*.go' . 2>/dev/null; then \
		echo '--- again in the js/wasm build context'; \
		CGO_ENABLED=0 GOOS=js GOARCH=wasm ${OPTS} golangci-lint run -c .golangci.yml ./...; \
	fi

vet: ## Run go vet
	@if [ -n "$(PKGS)" ]; then \
		CGO_ENABLED=0 ${OPTS} go vet $(PKGS); \
	fi
	@if grep -rlq '^//go:build js' --include='*.go' . 2>/dev/null; then \
		CGO_ENABLED=0 GOOS=js GOARCH=wasm ${OPTS} go vet ./...; \
	fi

test: ## Run tests
	@if [ -n "$(PKGS)" ]; then \
		${OPTS} go test $(PKGS); \
	else \
		echo 'nothing builds for this host; no tests to run'; \
	fi

check: lint vet test ## Run linters, vet and tests

install-linters: ## Install the linters
	${OPTS} go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	${OPTS} go install golang.org/x/tools/cmd/goimports@latest

docs: ## Regenerate the dependency graph and code counts in the README
	./gendocs.sh
