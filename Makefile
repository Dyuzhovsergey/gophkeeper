APP_NAME := gophkeeper
MODULE := github.com/Dyuzhovsergey/gophkeeper

VERSION ?= dev
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

BUILDINFO_PKG := $(MODULE)/internal/buildinfo

LDFLAGS := -X $(BUILDINFO_PKG).Version=$(VERSION) \
           -X $(BUILDINFO_PKG).Date=$(BUILD_DATE) \
           -X $(BUILDINFO_PKG).Commit=$(COMMIT)

CLIENT_BIN := bin/client/$(APP_NAME)
SERVER_BIN := bin/server/$(APP_NAME)-server

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make test"
	@echo "  make build"
	@echo "  make build-client"
	@echo "  make build-server"
	@echo "  make run-client-version"
	@echo "  make run-server"
	@echo "  make build-client-linux"
	@echo "  make build-client-windows"
	@echo "  make build-client-darwin"

.PHONY: test
test:
	go test ./...

.PHONY: build
build: build-server build-client

.PHONY: build-server
build-server:
	mkdir -p bin/server
	go build -ldflags="$(LDFLAGS)" -o $(SERVER_BIN) ./cmd/server

.PHONY: build-client
build-client:
	mkdir -p bin/client
	go build -ldflags="$(LDFLAGS)" -o $(CLIENT_BIN) ./cmd/client

.PHONY: run-server
run-server:
	go run -ldflags="$(LDFLAGS)" ./cmd/server

.PHONY: run-client-version
run-client-version:
	go run -ldflags="$(LDFLAGS)" ./cmd/client version

.PHONY: build-client-linux
build-client-linux:
	mkdir -p bin/client
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/client/$(APP_NAME)-linux-amd64 ./cmd/client

.PHONY: build-client-windows
build-client-windows:
	mkdir -p bin/client
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/client/$(APP_NAME)-windows-amd64.exe ./cmd/client

.PHONY: build-client-darwin
build-client-darwin:
	mkdir -p bin/client
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/client/$(APP_NAME)-darwin-amd64 ./cmd/client