SHELL:=/usr/bin/env bash

.PHONY: build
build:
	go build -o dist/stern .

TOOLS_BIN_DIR := $(CURDIR)/hack/tools/bin
GORELEASER_VERSION ?= v2.12.0
GORELEASER := $(TOOLS_BIN_DIR)/goreleaser
GOLANGCI_LINT_VERSION ?= v2.4.0
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
VALIDATE_KREW_MAIFEST_VERSION ?= v0.4.5
VALIDATE_KREW_MAIFEST := $(TOOLS_BIN_DIR)/validate-krew-manifest
YQ_VERSION ?= v4.48.1
YQ := $(TOOLS_BIN_DIR)/yq

$(GORELEASER):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

$(GOLANGCI_LINT):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(VALIDATE_KREW_MAIFEST):
	GOBIN=$(TOOLS_BIN_DIR) go install sigs.k8s.io/krew/cmd/validate-krew-manifest@$(VALIDATE_KREW_MAIFEST_VERSION)

$(YQ):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/mikefarah/yq/v4@$(YQ_VERSION)

.PHONY: build-cross
build-cross: $(GORELEASER)
	$(GORELEASER) build --snapshot --clean

.PHONY: test
test: lint
	go test -v ./...

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix

README_FILE ?= ./README.md

.PHONY: update-readme
update-readme:
	go run hack/update-readme/update-readme.go $(README_FILE)

.PHONY: verify-readme
verify-readme:
	./hack/verify-readme.sh

.PHONY: validate-krew-manifest
validate-krew-manifest: $(VALIDATE_KREW_MAIFEST)
	$(VALIDATE_KREW_MAIFEST) -manifest dist/krew/stern.yaml -skip-install

.PHONY: dist
dist: $(GORELEASER) $(YQ)
	cat .goreleaser.yaml | $(YQ) '.builds[0].goos = ["linux"], .builds[].goarch = ["amd64"]' | $(GORELEASER) release -f- --clean --skip=publish --snapshot

.PHONY: dist-all
dist-all: $(GORELEASER)
	$(GORELEASER) release --clean --skip=publish --snapshot

.PHONY: release
release: $(GORELEASER)
	$(GORELEASER) release --clean --skip=validate

.PHONY: clean
clean: clean-tools clean-dist

.PHONY: clean-tools
clean-tools:
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: clean-dist
clean-dist:
	rm -rf ./dist
