#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Usage information
usage() {
    echo "Usage: $0 [setup|run TEST_DIR]"
    echo ""
    echo "Commands:"
    echo "  setup          - Create test directory and prepare for PR testing"
    echo "  run TEST_DIR   - Run kantra-ai with PR creation in TEST_DIR"
    echo ""
    echo "Examples:"
    echo "  $0 setup"
    echo "  $0 run test-pr-1234567890"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    echo -e "${GREEN}Checking prerequisites...${NC}"

    # Check for required environment variables
    if [ -z "$GITHUB_TOKEN" ]; then
        echo -e "${RED}✗ GITHUB_TOKEN not set${NC}"
        echo ""
        echo "To create a GitHub token:"
        echo "  1. Go to: https://github.com/settings/tokens"
        echo "  2. Click 'Generate new token (classic)'"
        echo "  3. Grant 'repo' scope"
        echo "  4. Export: export GITHUB_TOKEN=ghp_your_token_here"
        exit 1
    else
        echo -e "${GREEN}✓ GITHUB_TOKEN set${NC}"
    fi

    # Check for AI provider API key
    if [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$OPENAI_API_KEY" ]; then
        echo -e "${RED}✗ No AI provider API key found${NC}"
        echo ""
        echo "Set one of:"
        echo "  export ANTHROPIC_API_KEY=sk-ant-..."
        echo "  export OPENAI_API_KEY=sk-..."
        exit 1
    else
        if [ -n "$ANTHROPIC_API_KEY" ]; then
            echo -e "${GREEN}✓ ANTHROPIC_API_KEY set (using Claude)${NC}"
            PROVIDER="claude"
        else
            echo -e "${GREEN}✓ OPENAI_API_KEY set (using OpenAI)${NC}"
            PROVIDER="openai"
        fi
    fi

    # Check for git
    if ! command -v git &> /dev/null; then
        echo -e "${RED}✗ git not found${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ git installed${NC}"
    fi

    # Check for go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}✗ go not found${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ go installed${NC}"
    fi

    echo ""
}

# Build kantra-ai
build_kantra_ai() {
    echo -e "${GREEN}Building kantra-ai...${NC}"
    go build -o kantra-ai ./cmd/kantra-ai
    echo -e "${GREEN}✓ Build complete${NC}"
    echo ""
}

# Setup test directory
setup_test() {
    TIMESTAMP=$(date +%s)
    TEST_DIR="test-pr-$TIMESTAMP"

    echo -e "${GREEN}Creating test directory: $TEST_DIR${NC}"

    # Create test directory
    mkdir -p "$TEST_DIR/src"

    # Copy test case from examples
    if [ -d "examples/javax-to-jakarta" ]; then
        cp examples/javax-to-jakarta/src/UserServlet.java "$TEST_DIR/src/"
        cp examples/javax-to-jakarta/output.yaml "$TEST_DIR/"
        echo -e "${GREEN}✓ Copied javax-to-jakarta test case${NC}"
    else
        echo -e "${YELLOW}⚠ examples/javax-to-jakarta not found, creating sample files${NC}"

        # Create sample Java file with javax imports
        cat > "$TEST_DIR/src/UserServlet.java" <<'EOF'
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import javax.servlet.ServletException;
import java.io.IOException;

public class UserServlet extends HttpServlet {
    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response)
            throws ServletException, IOException {
        response.getWriter().println("Hello World");
    }
}
EOF

        # Create sample output.yaml
        cat > "$TEST_DIR/output.yaml" <<'EOF'
---
violations:
  - name: javax-to-jakarta-001
    id: javax-to-jakarta-001
    description: "Replace javax.servlet with jakarta.servlet"
    category: mandatory
    effort: 1
    incidents:
      - uri: "file:///src/UserServlet.java"
        line_number: 1
        message: "Replace javax.servlet.http.HttpServlet with jakarta.servlet.http.HttpServlet"
      - uri: "file:///src/UserServlet.java"
        line_number: 2
        message: "Replace javax.servlet.http.HttpServletRequest with jakarta.servlet.http.HttpServletRequest"
      - uri: "file:///src/UserServlet.java"
        line_number: 3
        message: "Replace javax.servlet.http.HttpServletResponse with jakarta.servlet.http.HttpServletResponse"
      - uri: "file:///src/UserServlet.java"
        line_number: 4
        message: "Replace javax.servlet.ServletException with jakarta.servlet.ServletException"
EOF
    fi

    # Initialize git repository
    echo -e "${GREEN}Initializing git repository...${NC}"
    cd "$TEST_DIR"
    git init
    git add .
    git commit -m "Initial commit with javax violations"
    cd ..

    echo ""
    echo -e "${GREEN}✓ Test directory created: $TEST_DIR${NC}"
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
    echo "1. Create a new repository on GitHub:"
    echo "   https://github.com/new"
    echo ""
    echo "   Name: kantra-ai-pr-test (or any name)"
    echo "   Visibility: Public or Private"
    echo "   Do NOT initialize with README, .gitignore, or license"
    echo ""
    echo "2. Add the remote and push:"
    echo ""
    echo "   cd $TEST_DIR"
    echo "   git remote add origin https://github.com/YOUR_USERNAME/kantra-ai-pr-test.git"
    echo "   git branch -M main"
    echo "   git push -u origin main"
    echo "   cd .."
    echo ""
    echo "3. Run the test:"
    echo ""
    echo "   ./test-pr-creation.sh run $TEST_DIR"
    echo ""
}

# Run kantra-ai with PR creation
run_test() {
    TEST_DIR=$1

    if [ -z "$TEST_DIR" ]; then
        echo -e "${RED}Error: TEST_DIR not provided${NC}"
        usage
    fi

    if [ ! -d "$TEST_DIR" ]; then
        echo -e "${RED}Error: Directory $TEST_DIR does not exist${NC}"
        exit 1
    fi

    # Check if remote is configured
    cd "$TEST_DIR"
    if ! git remote get-url origin &> /dev/null; then
        echo -e "${RED}Error: No git remote 'origin' configured${NC}"
        echo ""
        echo "Please add a remote first:"
        echo "  git remote add origin https://github.com/YOUR_USERNAME/kantra-ai-pr-test.git"
        exit 1
    fi

    REMOTE_URL=$(git remote get-url origin)
    echo -e "${GREEN}Remote URL: $REMOTE_URL${NC}"
    cd ..

    echo ""
    echo -e "${GREEN}Running kantra-ai with PR creation...${NC}"
    echo ""

    # Determine provider
    if [ -n "$ANTHROPIC_API_KEY" ]; then
        PROVIDER="claude"
    else
        PROVIDER="openai"
    fi

    # Run kantra-ai
    ./kantra-ai remediate \
        --analysis="$TEST_DIR/output.yaml" \
        --input="$TEST_DIR" \
        --provider="$PROVIDER" \
        --git-commit=per-violation \
        --create-pr

    EXIT_CODE=$?

    echo ""
    if [ $EXIT_CODE -eq 0 ]; then
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}Test Complete!${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo ""
        echo "Check the pull request on GitHub:"

        # Extract owner/repo from remote URL
        if [[ $REMOTE_URL =~ github\.com[:/]([^/]+)/([^/\.]+) ]]; then
            OWNER="${BASH_REMATCH[1]}"
            REPO="${BASH_REMATCH[2]}"
            echo "  https://github.com/$OWNER/$REPO/pulls"
        else
            echo "  (Check your repository's Pull Requests tab)"
        fi

        echo ""
        echo "Verify:"
        echo "  ✓ PR exists and is open"
        echo "  ✓ Branch name contains 'kantra-ai/remediation'"
        echo "  ✓ Title: 'fix: Konveyor violation javax-to-jakarta-001'"
        echo "  ✓ PR body includes summary, changes, and AI details"
        echo "  ✓ Files show jakarta.servlet instead of javax.servlet"
    else
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}Test Failed${NC}"
        echo -e "${RED}========================================${NC}"
        echo ""
        echo "Exit code: $EXIT_CODE"
        echo "Check the error messages above for details."
    fi
}

# Main script logic
case "${1:-setup}" in
    setup)
        check_prerequisites
        build_kantra_ai
        setup_test
        ;;
    run)
        check_prerequisites
        run_test "$2"
        ;;
    *)
        usage
        ;;
esac
