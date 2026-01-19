.PHONY: help test build build-arm64 clean

# Go binary - use full path if go is not in PATH
GO ?= $(shell which go 2>/dev/null || echo /home/orion/go/bin/go)

# Default target
help:
	@echo "ORION Edge Agent (Go) - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build       - Build orion-edge binary for local platform"
	@echo "  make build-arm64 - Cross-compile for Pi 4 (ARM64)"
	@echo "  make test        - Run tests with coverage"
	@echo "  make clean       - Remove binaries"
	@echo "  make help        - Display this help message"

# Run tests with coverage
test:
	@echo "Running tests with race detector and coverage..."
	$(GO) test ./... -v -race -coverprofile=coverage.out
	@echo ""
	@echo "Coverage report:"
	$(GO) tool cover -func=coverage.out

# Build the orion-edge binary for local platform
build:
	@echo "Building orion-edge for local platform..."
	@mkdir -p bin
	$(GO) build -o bin/orion-edge ./cmd/orion-edge
	@echo "Binary created at: bin/orion-edge"

# Cross-compile for Pi 4 (ARM64)
build-arm64:
	@echo "Building orion-edge for ARM64 (Pi 4)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -o bin/orion-edge-arm64 ./cmd/orion-edge
	@echo "Binary created at: bin/orion-edge-arm64"

# Clean binaries
clean:
	@echo "Cleaning binaries..."
	rm -rf bin/ coverage.out
	@echo "Clean complete"
