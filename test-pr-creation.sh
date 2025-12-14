#!/bin/bash

# Test PR Creation Feature
# This script sets up a test environment to verify PR creation works end-to-end

set -e

echo "=== PR Creation Test Setup ==="
echo ""

# Check prerequisites
echo "1. Checking prerequisites..."

if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ GITHUB_TOKEN not set"
    echo ""
    echo "Please set your GitHub token:"
    echo "  1. Go to https://github.com/settings/tokens"
    echo "  2. Create a token with 'repo' scope"
    echo "  3. export GITHUB_TOKEN=ghp_your_token_here"
    echo ""
    exit 1
fi

echo "✓ GITHUB_TOKEN is set"

if [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ No AI provider API key set"
    echo "Please set ANTHROPIC_API_KEY or OPENAI_API_KEY"
    exit 1
fi

if [ -n "$ANTHROPIC_API_KEY" ]; then
    PROVIDER="claude"
    echo "✓ Using Claude (ANTHROPIC_API_KEY set)"
else
    PROVIDER="openai"
    echo "✓ Using OpenAI (OPENAI_API_KEY set)"
fi

# Build the tool
echo ""
echo "2. Building kantra-ai..."
go build -o kantra-ai ./cmd/kantra-ai
echo "✓ Build complete"

# Create test repository
TEST_DIR="test-pr-$(date +%s)"
echo ""
echo "3. Creating test repository in $TEST_DIR..."

mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize git
git init
git config user.name "Test User"
git config user.email "test@example.com"

# Copy test case
echo ""
echo "4. Setting up test case (javax-to-jakarta)..."
mkdir -p src
cp ../examples/javax-to-jakarta/src/UserServlet.java src/
cp ../examples/javax-to-jakarta/output.yaml .

# Initial commit
git add .
git commit -m "Initial commit: Test Java servlet for migration"

echo "✓ Test repository created"
echo ""
echo "=== MANUAL STEPS REQUIRED ==="
echo ""
echo "Before we can test PR creation, you need to:"
echo ""
echo "1. Create a test repository on GitHub:"
echo "   - Go to https://github.com/new"
echo "   - Name: kantra-ai-pr-test (or any name)"
echo "   - Make it public or private (your choice)"
echo "   - Do NOT initialize with README"
echo ""
echo "2. Add the remote to this test repo:"
echo "   cd $TEST_DIR"
echo "   git remote add origin https://github.com/YOUR_USERNAME/kantra-ai-pr-test.git"
echo "   git branch -M main"
echo "   git push -u origin main"
echo ""
echo "3. Run the PR creation test:"
echo "   cd .."
echo "   ./test-pr-creation.sh run $TEST_DIR"
echo ""
echo "Current directory: $PWD"

# If "run" command is provided, execute the test
if [ "$1" = "run" ]; then
    if [ -z "$2" ]; then
        echo ""
        echo "Usage: $0 run <test-directory>"
        exit 1
    fi

    TEST_RUN_DIR="$2"

    echo ""
    echo "=== Running PR Creation Test ==="
    echo ""

    # Make sure we're in the kantra-ai root
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    cd "$SCRIPT_DIR"

    # Now cd into the test directory
    if [ ! -d "$TEST_RUN_DIR" ]; then
        echo "❌ Directory not found: $TEST_RUN_DIR"
        echo "Available test directories:"
        ls -d test-pr-* 2>/dev/null || echo "  (none)"
        exit 1
    fi

    cd "$TEST_RUN_DIR"

    # Verify remote is set
    if ! git remote get-url origin > /dev/null 2>&1; then
        echo "❌ No git remote 'origin' found"
        echo "Please add the GitHub remote first (see steps above)"
        exit 1
    fi

    REMOTE_URL=$(git remote get-url origin)
    echo "✓ Remote configured: $REMOTE_URL"

    # Verify it's a GitHub URL
    if [[ ! "$REMOTE_URL" =~ github\.com ]]; then
        echo "❌ Remote is not a GitHub URL: $REMOTE_URL"
        exit 1
    fi

    echo ""
    echo "=== Test 1: Per-Violation Strategy (1 PR for all incidents of same violation) ==="
    echo ""

    ../kantra-ai remediate \
        --analysis=output.yaml \
        --input=src \
        --provider="$PROVIDER" \
        --git-commit=per-violation \
        --create-pr \
        --max-cost=1.00

    echo ""
    echo "=== Test Results ==="
    echo ""
    echo "Check your GitHub repository for the created PR:"
    echo "$REMOTE_URL/pulls"
    echo ""
    echo "Verify:"
    echo "  - PR was created successfully"
    echo "  - Branch name follows pattern: kantra-ai/remediation-<violation-id>-<timestamp>"
    echo "  - PR title is: 'fix: Konveyor violation javax-to-jakarta-001'"
    echo "  - PR body contains violation details, file changes, cost/token info"
    echo "  - Code changes are correct (javax.servlet → jakarta.servlet)"
    echo ""

    # Show what branch was created
    echo "Branches in repository:"
    git branch -a
    echo ""

    # Show commit log
    echo "Recent commits:"
    git log --oneline -5
    echo ""

    echo "=== Cleanup ==="
    echo ""
    echo "To clean up after testing:"
    echo "  - Close/delete the PR on GitHub"
    echo "  - Optionally delete the test repository on GitHub"
    echo "  - Delete local test directory: rm -rf $TEST_RUN_DIR"
    echo ""
fi
