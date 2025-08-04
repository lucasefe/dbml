.PHONY: build test clean install help

# Build variables
BINARY_NAME=dbml
CMD_DIR=./cmd/dbml
BUILD_DIR=./bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the CLI binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build binaries for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Binaries built in $(BUILD_DIR)/"

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

install: ## Install the CLI binary globally
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(CMD_DIR)

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	golangci-lint run

example: build ## Run example with telephony_core database
	@echo "Running example..."
	./$(BUILD_DIR)/$(BINARY_NAME) --url "postgres://localhost/telephony_core?sslmode=disable" --all-schemas | head -20

example-file: build ## Generate example DBML file
	@echo "Generating example DBML file..."
	./$(BUILD_DIR)/$(BINARY_NAME) --url "postgres://localhost/telephony_core?sslmode=disable" --all-schemas --output example-output.dbml
	@echo "Generated example-output.dbml"

# Development helpers
dev-build: ## Quick development build
	$(GOBUILD) -o $(BINARY_NAME) $(CMD_DIR)

dev-test: ## Quick test run
	$(GOTEST) ./...

release: clean test build-all ## Prepare release (clean, test, build all platforms)
	@echo "Release build complete!"