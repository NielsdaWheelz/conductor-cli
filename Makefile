.PHONY: build test clean install help

# Default target
all: build

# Build the binary
build:
	go build -o agency ./cmd/agency

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-v:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f agency
	go clean

# Install to GOBIN
install:
	go install ./cmd/agency

# Run from source
run:
	go run ./cmd/agency

# Show help
help:
	@echo "available targets:"
	@echo "  build    - build the agency binary"
	@echo "  test     - run tests"
	@echo "  test-v   - run tests with verbose output"
	@echo "  clean    - clean build artifacts"
	@echo "  install  - install to GOBIN"
	@echo "  run      - run from source"
	@echo "  help     - show this help"
