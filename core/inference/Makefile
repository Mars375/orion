# ORION Inference Subsystem Makefile

.PHONY: all build build-worker build-router test test-unit test-integration clean lint fmt

# Build output directory
BUILD_DIR := bin

# Go build flags
LDFLAGS := -ldflags="-s -w"
CGO_ENABLED := 0

all: build

# Build all binaries
build: build-worker build-router

# Build worker binary
build-worker:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/orion-inference-worker ./cmd/orion-inference-worker

# Build router binary
build-router:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/orion-inference-router ./cmd/orion-inference-router

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	go test -v ./...

# Run integration tests (requires Redis)
test-integration:
	go test -v -tags=integration ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Tidy dependencies
tidy:
	go mod tidy

# Cross-compile for ARM (Raspberry Pi)
build-arm:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/orion-inference-worker-arm64 ./cmd/orion-inference-worker
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/orion-inference-router-arm64 ./cmd/orion-inference-router

# Show help
help:
	@echo "ORION Inference Subsystem"
	@echo ""
	@echo "Available targets:"
	@echo "  build          - Build all binaries"
	@echo "  build-worker   - Build worker binary only"
	@echo "  build-router   - Build router binary only"
	@echo "  build-arm      - Cross-compile for ARM64 (Raspberry Pi)"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-integration - Run integration tests (requires Redis)"
	@echo "  clean          - Remove build artifacts"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy dependencies"
