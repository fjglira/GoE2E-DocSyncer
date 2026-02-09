.PHONY: build test test-verbose vet lint clean generate install help

BINARY_NAME := docsyncer
BUILD_DIR := bin
CMD_DIR := ./cmd/docsyncer

# Build the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Install the binary to $GOPATH/bin
install:
	go install $(CMD_DIR)

# Run all tests
test:
	go test ./...

# Run tests with verbose Ginkgo output
test-verbose:
	go test ./... -v -count=1

# Run go vet
vet:
	go vet ./...

# Run golangci-lint (if installed)
lint:
	@which golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

# Remove build artifacts and generated test files
clean:
	rm -rf $(BUILD_DIR)
	rm -rf tests/e2e/generated

# Run the generator against testdata (dry-run)
dry-run: build
	$(BUILD_DIR)/$(BINARY_NAME) generate --config docsyncer.yaml --dry-run --verbose

# Tidy dependencies
tidy:
	go mod tidy

# Build + vet + test
check: vet test build

# Show help
help:
	@echo "Available targets:"
	@echo "  build         Build the docsyncer binary"
	@echo "  install       Install binary to GOPATH/bin"
	@echo "  test          Run all tests"
	@echo "  test-verbose  Run tests with verbose output"
	@echo "  vet           Run go vet"
	@echo "  lint          Run golangci-lint"
	@echo "  clean         Remove build artifacts"
	@echo "  dry-run       Run generator in dry-run mode"
	@echo "  tidy          Run go mod tidy"
	@echo "  check         Run vet + test + build"
