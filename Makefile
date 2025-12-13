.PHONY: build test run-example clean install help

# Build the binary
build:
	@echo "Building kantra-ai..."
	@go build -o kantra-ai ./cmd/kantra-ai
	@echo "✓ Built: ./kantra-ai"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies ready"

# Run the example test
run-example: build
	@echo "Running example: javax → jakarta"
	@echo "=================================="
	@./kantra-ai remediate \
		--analysis=./examples/javax-to-jakarta/output.yaml \
		--input=./examples/javax-to-jakarta \
		--provider=claude \
		--dry-run
	@echo ""
	@echo "To actually apply the fix, run without --dry-run"

# Run with actual fixes
run-example-real: build
	@echo "⚠ This will modify files in ./examples/javax-to-jakarta/src/"
	@./kantra-ai remediate \
		--analysis=./examples/javax-to-jakarta/output.yaml \
		--input=./examples/javax-to-jakarta \
		--provider=claude

# Reset example to original state
reset-example:
	@echo "Resetting example to original state..."
	@git checkout examples/javax-to-jakarta/src/UserServlet.java 2>/dev/null || echo "No changes to reset"

# Test the fixed file matches expected
verify-example:
	@echo "Verifying fix matches expected output..."
	@diff examples/javax-to-jakarta/src/UserServlet.java \
	      examples/javax-to-jakarta/expected/UserServlet.java \
	&& echo "✓ Fix matches expected output!" \
	|| echo "✗ Fix does not match expected output"

# Install the binary to $GOPATH/bin
install:
	@echo "Installing kantra-ai..."
	@go install ./cmd/kantra-ai
	@echo "✓ Installed to $(go env GOPATH)/bin/kantra-ai"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f kantra-ai
	@echo "✓ Clean"

# Run tests (when we add them)
test:
	@echo "Running tests..."
	@go test ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Formatted"

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@golangci-lint run || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"

# Show help
help:
	@echo "kantra-ai Makefile"
	@echo "=================="
	@echo ""
	@echo "Available targets:"
	@echo "  build           - Build the kantra-ai binary"
	@echo "  deps            - Install Go dependencies"
	@echo "  run-example     - Run the example (dry-run mode)"
	@echo "  run-example-real- Run the example (actually apply fixes)"
	@echo "  verify-example  - Check if fix matches expected output"
	@echo "  reset-example   - Reset example to original state"
	@echo "  install         - Install to GOPATH/bin"
	@echo "  test            - Run tests"
	@echo "  fmt             - Format code"
	@echo "  lint            - Lint code"
	@echo "  clean           - Remove build artifacts"
	@echo "  help            - Show this help"
	@echo ""
	@echo "Example workflow:"
	@echo "  make deps              # Install dependencies"
	@echo "  make build             # Build the binary"
	@echo "  make run-example       # Test with dry-run"
	@echo "  make run-example-real  # Apply fixes"
	@echo "  make verify-example    # Verify correctness"
	@echo "  make reset-example     # Reset for next test"
