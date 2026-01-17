.PHONY: help generate test build clean

# Default target
help:
	@echo "ORION Bus (Go) - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make generate  - Generate Go structs from JSON Schema files (Plan 03)"
	@echo "  make test      - Run tests with coverage"
	@echo "  make build     - Build orion-bus binary"
	@echo "  make clean     - Remove generated files and binaries"
	@echo "  make help      - Display this help message"

# Generate Go structs from JSON Schema files
# NOTE: go-jsonschema tools (atombender/omissis) require Go 1.23+
# Current environment has Go 1.22, so code generation is deferred
# Runtime validation with ContractValidator is operational and sufficient for Phase 4.1
generate:
	@echo "Code generation from JSON Schema"
	@echo ""
	@echo "LIMITATION: go-jsonschema tools require Go 1.23+ (current: 1.22)"
	@echo ""
	@echo "Options:"
	@echo "  1. Upgrade to Go 1.23+ and run:"
	@echo "     go install github.com/atombender/go-jsonschema@latest"
	@echo "     gojsonschema --package contracts --output pkg/contracts/generated.go ../contracts/*.schema.json"
	@echo ""
	@echo "  2. Use runtime validation only (current approach)"
	@echo "     ContractValidator validates all messages against JSON Schema"
	@echo "     Generated structs provide compile-time safety but are optional"
	@echo ""
	@echo "Current status: Runtime validation operational, code generation pending Go upgrade"

# Run tests with coverage
test:
	@echo "Running tests with race detector and coverage..."
	go test ./... -v -race -coverprofile=coverage.out
	@echo ""
	@echo "Coverage report:"
	go tool cover -func=coverage.out

# Build the orion-bus binary
build:
	@echo "Building orion-bus..."
	@mkdir -p bin
	go build -o bin/orion-bus cmd/orion-bus/main.go
	@echo "Binary created at: bin/orion-bus"

# Clean generated files and binaries
clean:
	@echo "Cleaning generated files and binaries..."
	rm -rf pkg/contracts/*.go bin/ coverage.out
	@echo "Clean complete"
