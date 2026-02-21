# Copyright (C) 2026 megatherium
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

.PHONY: all build run clean lint test fmt vet install help

# Binary name
BINARY_NAME := bdb
BINARY_PATH := ./cmd/blunderbuss

# Build variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: build

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) $(BINARY_PATH)
	@echo "Built: $(BINARY_NAME)"

## run: Build and run the binary
run: build
	./$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean -cache
	@echo "Cleaned build artifacts"

## lint: Run golangci-lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

## test: Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## fmt: Format Go code
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy and verify module dependencies
tidy:
	go mod tidy
	go mod verify

## deps: Download and verify module dependencies
deps:
	go mod download
	go mod verify

## install: Install binary to GOPATH/bin
install: build
	go install $(LDFLAGS) $(BINARY_PATH)

## help: Show this help message
help:
	@echo "Available targets:"
	@awk '/^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$0}' $(MAKEFILE_LIST) | sed 's/:.*## /  /'

# Install development dependencies
dev-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
