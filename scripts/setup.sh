#!/bin/bash
set -e

echo "Setting up kantra-ai development environment..."
echo ""

# Check Go version
echo "Checking Go version..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✓ Go $GO_VERSION found"
echo ""

# Install dependencies
echo "Installing Go dependencies..."
go mod download
go mod tidy
echo "✓ Dependencies installed"
echo ""

# Build the binary
echo "Building kantra-ai..."
go build -o kantra-ai ./cmd/kantra-ai
echo "✓ Binary built: ./kantra-ai"
echo ""

# Check for API keys
echo "Checking API keys..."
if [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠ No API keys found."
    echo ""
    echo "To use kantra-ai, you need to set at least one:"
    echo ""
    echo "For Claude (recommended):"
    echo "  export ANTHROPIC_API_KEY=sk-ant-your-key-here"
    echo "  Get your key from: https://console.anthropic.com"
    echo ""
    echo "For OpenAI:"
    echo "  export OPENAI_API_KEY=sk-your-key-here"
    echo "  Get your key from: https://platform.openai.com"
    echo ""
else
    if [ -n "$ANTHROPIC_API_KEY" ]; then
        echo "✓ ANTHROPIC_API_KEY is set"
    fi
    if [ -n "$OPENAI_API_KEY" ]; then
        echo "✓ OPENAI_API_KEY is set"
    fi
    echo ""
fi

echo "Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Test with example: make run-example"
echo "  2. Read QUICKSTART.md for usage guide"
echo "  3. Start validation with real violations"
echo ""
