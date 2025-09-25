# Intel GPU Exporter Makefile

# Build targets
.PHONY: all build test  deps fmt vet

# Default target
all: test build

# Build the binary
build:
	go build ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...
	go vet ./...

deps:
	@echo "Downloading dependencies..."
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt -s -w ./...

all:
	test build