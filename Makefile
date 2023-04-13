SHELL:=/usr/bin/env bash

.PHONY: build
build:
	go build -o dist/stern .

TOOLS_BIN_DIR := $(CURDIR)/hack/tools/bin
GORELEASER_VERSION ?= v1.17.0
GORELEASER := $(TOOLS_BIN_DIR)/goreleaser
GOLANGCI_LINT_VERSION ?= v1.52.2
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
VALIDATE_KREW_MAIFEST_VERSION ?= v0.4.3
VALIDATE_KREW_MAIFEST := $(TOOLS_BIN_DIR)/validate-krew-manifest
GORELEASER_FILTER_VERSION ?= v0.3.0
GORELEASER_FILTER := $(TOOLS_BIN_DIR)/goreleaser-filter

$(GORELEASER):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)

$(GOLANGCI_LINT):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(VALIDATE_KREW_MAIFEST):
	GOBIN=$(TOOLS_BIN_DIR) go install sigs.k8s.io/krew/cmd/validate-krew-manifest@$(VALIDATE_KREW_MAIFEST_VERSION)

$(GORELEASER_FILTER):
	GOBIN=$(TOOLS_BIN_DIR) go install github.com/t0yv0/goreleaser-filter@$(GORELEASER_FILTER_VERSION)

.PHONY: build-cross
build-cross: $(GORELEASER)
	$(GORELEASER) build --snapshot --rm-dist

.PHONY: test
test: fmt vet lint
	go test -v ./...

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

README_FILE ?= ./README.md

.PHONY: update-readme
update-readme:
	go run hack/update-readme/update-readme.go $(README_FILE)

.PHONY: verify-readme
verify-readme:
	./hack/verify-readme.sh

.PHONY: validate-krew-manifest
validate-krew-manifest: $(VALIDATE_KREW_MAIFEST)
	$(VALIDATE_KREW_MAIFEST) -manifest dist/stern.yaml -skip-install

.PHONY: dist
dist: $(GORELEASER) $(GORELEASER_FILTER)
	cat .goreleaser.yaml | $(GORELEASER_FILTER) -goos $(shell go env GOOS) -goarch $(shell go env GOARCH) | $(GORELEASER) release -f- --rm-dist --skip-publish --snapshot

.PHONY: dist-all
dist-all: $(GORELEASER)
	$(GORELEASER) release --rm-dist --skip-publish --snapshot

.PHONY: release
release: $(GORELEASER)
	$(GORELEASER) release --rm-dist --skip-validate

.PHONY: clean
clean: clean-tools clean-dist

.PHONY: clean-tools
clean-tools:
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: clean-dist
clean-dist:
	rm -rf ./dist
