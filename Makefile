GO ?= GO111MODULE=on GOPROXY=https://gocenter.io go

.PHONY: build
build:
	$(GO) build -o dist/stern .

.PHONY: install
install:
	$(GO) install .
