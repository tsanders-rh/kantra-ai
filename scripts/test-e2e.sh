#!/bin/bash
#
# End-to-End Test Script for kantra-ai
#
# This script automates testing of the kantra-ai tool using a real codebase.
# It creates a test branch, runs analysis, creates a simple migration plan,
# launches the interactive web UI for approval and execution, and provides
# cleanup helpers.
#
# Documentation:
#   Quick Start: scripts/test-e2e-README.md
#   Full Guide:  scripts/README.md
#
# Usage:
#   ./scripts/test-e2e.sh [test-codebase-path]
#
# Example:
#   ./scripts/test-e2e.sh ~/Workspace/boat-fuel-tracker-j2ee
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}✗ $1${NC}"
}

prompt_continue() {
    echo
    read -p "Press Enter to continue or Ctrl+C to abort..."
    echo
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing=0

    if ! command -v git &> /dev/null; then
        log_error "git is not installed"
        missing=1
    fi

    if ! command -v gh &> /dev/null; then
        log_error "gh (GitHub CLI) is not installed"
        missing=1
    fi

    if ! command -v kantra &> /dev/null; then
        log_error "kantra is not installed"
        missing=1
    fi

    if [ ! -f "./kantra-ai" ]; then
        log_error "kantra-ai binary not found in current directory"
        log_info "Run 'go build -o kantra-ai ./cmd/kantra-ai' first"
        missing=1
    fi

    if [ $missing -eq 1 ]; then
        exit 1
    fi

    log_success "All prerequisites met"
}

# Parse arguments
TEST_CODEBASE="${1:-$HOME/Workspace/boat-fuel-tracker-j2ee}"

if [ ! -d "$TEST_CODEBASE" ]; then
    log_error "Test codebase not found: $TEST_CODEBASE"
    echo
    echo "Usage: $0 [test-codebase-path]"
    echo "Example: $0 ~/Workspace/boat-fuel-tracker-j2ee"
    exit 1
fi

# Configuration
BRANCH_NAME="kantra-ai-test-$(date +%Y%m%d-%H%M%S)"
ANALYSIS_OUTPUT="analysis-output.yaml"
PLAN_DIR=".kantra-ai-plan"

echo "=============================================="
echo "  Kantra-AI End-to-End Test Script"
echo "=============================================="
echo
log_info "Test codebase: $TEST_CODEBASE"
log_info "Test branch: $BRANCH_NAME"
echo

check_prerequisites

# Step 1: Navigate to test codebase
log_info "Step 1: Navigating to test codebase..."
cd "$TEST_CODEBASE"
log_success "Working directory: $(pwd)"

# Check git status
if [ -n "$(git status --porcelain)" ]; then
    log_warn "Working directory has uncommitted changes"
    git status --short
    echo
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_error "Aborted by user"
        exit 1
    fi
fi

prompt_continue

# Step 2: Create test branch
log_info "Step 2: Creating test branch..."
ORIGINAL_BRANCH=$(git branch --show-current)
log_info "Original branch: $ORIGINAL_BRANCH"

if git show-ref --verify --quiet "refs/heads/$BRANCH_NAME"; then
    log_warn "Branch $BRANCH_NAME already exists, deleting it..."
    git branch -D "$BRANCH_NAME"
fi

git checkout -b "$BRANCH_NAME"
log_success "Created and checked out branch: $BRANCH_NAME"

prompt_continue

# Step 3: Run Konveyor analysis
log_info "Step 3: Running Konveyor analysis..."
log_warn "This will run: kantra analyze --input . --output $ANALYSIS_OUTPUT --target quarkus"
echo
read -p "Modify the kantra command? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Enter your kantra analyze command (or press Enter for default):"
    read -e KANTRA_CMD
    if [ -z "$KANTRA_CMD" ]; then
        KANTRA_CMD="kantra analyze --input . --output $ANALYSIS_OUTPUT --target quarkus"
    fi
else
    KANTRA_CMD="kantra analyze --input . --output $ANALYSIS_OUTPUT --target quarkus"
fi

log_info "Running: $KANTRA_CMD"
eval "$KANTRA_CMD"
log_success "Analysis complete: $ANALYSIS_OUTPUT"

prompt_continue

# Step 4: Create migration plan
log_info "Step 4: Creating migration plan..."
log_info "Running: kantra-ai plan --analysis $ANALYSIS_OUTPUT --input ."

# Create a simple plan with just one low-effort phase
./kantra-ai plan --analysis "$ANALYSIS_OUTPUT" --input .

if [ ! -d "$PLAN_DIR" ]; then
    log_error "Plan directory not created: $PLAN_DIR"
    exit 1
fi

log_success "Plan created in: $PLAN_DIR"
echo
log_info "Plan files:"
ls -lh "$PLAN_DIR/"
echo

# Show plan summary
if [ -f "$PLAN_DIR/plan.yaml" ]; then
    log_info "Plan summary:"
    grep -E "^(id:|name:|status:|violations:)" "$PLAN_DIR/plan.yaml" | head -20
fi

prompt_continue

# Step 5: Choose workflow mode
log_info "Step 5: Choose approval and execution workflow..."
echo
echo "How would you like to approve phases and execute?"
echo "  1) Interactive Web UI - Approve and execute in browser with live monitoring"
echo "  2) CLI Mode - Manually approve in plan.yaml, then execute via command line"
echo
read -p "Enter choice (1-2): " -n 1 -r
echo
echo

if [[ $REPLY =~ ^[Yy1]$ ]]; then
    # Interactive Web Workflow
    log_info "Starting interactive web planner for approval and execution..."
    echo
    log_info "In the web UI you can:"
    echo "  ✓ Review the migration plan with visual charts"
    echo "  ✓ Approve/defer phases interactively"
    echo "  ✓ Configure settings (git, PR, verification, etc.)"
    echo "  ✓ Execute with live progress monitoring"
    echo "  ✓ View execution results and created PRs"
    echo
    log_warn "Recommended: Approve just ONE simple phase for testing"
    echo

    # Check for GitHub token upfront
    if [ -z "$GITHUB_TOKEN" ]; then
        log_warn "GITHUB_TOKEN not set. PR creation will be disabled."
        echo "  To enable PR creation: export GITHUB_TOKEN=\$(gh auth token)"
        echo
    fi

    prompt_continue

    log_info "Launching web UI at http://localhost:8080"
    log_info "Running: kantra-ai plan --analysis $ANALYSIS_OUTPUT --input . --interactive-web"
    echo

    # This will block until the user closes the web UI
    ./kantra-ai plan --analysis "$ANALYSIS_OUTPUT" --input . --interactive-web

    log_success "Web UI session complete!"
    echo
    log_info "Check .kantra-ai-state.yaml for execution results"

else
    # CLI Workflow
    log_info "Manual approval and CLI execution workflow..."
    echo
    log_info "Step 5a: Approve a phase manually"
    echo "  - Open: file://$PWD/$PLAN_DIR/plan.html"
    echo "  - Or edit: $PLAN_DIR/plan.yaml"
    echo "  - Set one phase status to 'approved'"
    echo
    prompt_continue

    log_info "Step 5b: Run execution with PR creation..."
    echo
    log_info "Configuration:"
    echo "  - Git commits: enabled (per-phase)"
    echo "  - PR creation: enabled (at-end)"
    echo "  - Dry run: disabled (will make real changes)"
    echo

    read -p "Proceed with execution? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_warn "Skipping execution"
    else
        # Check for GitHub token
        if [ -z "$GITHUB_TOKEN" ]; then
            log_warn "GITHUB_TOKEN not set. PR creation will fail."
            echo "  Set it with: export GITHUB_TOKEN=\$(gh auth token)"
            prompt_continue
        fi

        log_info "Running execution with PR creation..."
        ./kantra-ai execute \
            --analysis "$ANALYSIS_OUTPUT" \
            --input . \
            --plan "$PLAN_DIR" \
            --provider claude \
            --git-commit \
            --commit-strategy per-phase \
            --create-pr \
            --pr-strategy at-end \
            --pr-branch-prefix "kantra-ai-test"

        log_success "Execution complete!"
    fi
fi

# Step 6: Review results
log_info "Step 6: Review results..."
echo
log_info "What was created:"
echo

# Show commits
log_info "Git commits:"
git log --oneline "$ORIGINAL_BRANCH..$BRANCH_NAME"
echo

# Show files changed
log_info "Files changed:"
git diff --stat "$ORIGINAL_BRANCH"
echo

# Show PRs
log_info "Pull requests created:"
gh pr list --head "$BRANCH_NAME"
echo

# Show state file
if [ -f ".kantra-ai-state.yaml" ]; then
    log_info "Execution state:"
    grep -E "^(total_|executed_|successful_|failed_)" .kantra-ai-state.yaml
    echo
fi

prompt_continue

# Step 7: Cleanup
log_info "Step 7: Cleanup options..."
echo
echo "Choose cleanup action:"
echo "  1) Keep everything for review"
echo "  2) Close PR and delete remote branch (keep local)"
echo "  3) Full cleanup (close PR, delete branch, return to original)"
echo
read -p "Enter choice (1-3): " -n 1 -r
echo
echo

case $REPLY in
    1)
        log_info "Keeping everything for review"
        log_info "To review:"
        echo "  - View PR: gh pr view"
        echo "  - View commits: git log"
        echo "  - View diff: git diff $ORIGINAL_BRANCH"
        ;;
    2)
        log_info "Closing PR and deleting remote branch..."
        PR_NUMBER=$(gh pr list --head "$BRANCH_NAME" --json number --jq '.[0].number')
        if [ -n "$PR_NUMBER" ]; then
            gh pr close "$PR_NUMBER"
            log_success "Closed PR #$PR_NUMBER"
        fi

        if git ls-remote --heads origin "$BRANCH_NAME" | grep -q "$BRANCH_NAME"; then
            git push origin --delete "$BRANCH_NAME"
            log_success "Deleted remote branch: $BRANCH_NAME"
        fi

        log_info "Local branch kept for review: $BRANCH_NAME"
        ;;
    3)
        log_info "Full cleanup..."

        # Close PR
        PR_NUMBER=$(gh pr list --head "$BRANCH_NAME" --json number --jq '.[0].number')
        if [ -n "$PR_NUMBER" ]; then
            gh pr close "$PR_NUMBER"
            log_success "Closed PR #$PR_NUMBER"
        fi

        # Delete remote branch
        if git ls-remote --heads origin "$BRANCH_NAME" | grep -q "$BRANCH_NAME"; then
            git push origin --delete "$BRANCH_NAME"
            log_success "Deleted remote branch: $BRANCH_NAME"
        fi

        # Return to original branch
        git checkout "$ORIGINAL_BRANCH"
        log_success "Returned to branch: $ORIGINAL_BRANCH"

        # Delete local branch
        git branch -D "$BRANCH_NAME"
        log_success "Deleted local branch: $BRANCH_NAME"

        # Clean up files
        rm -f "$ANALYSIS_OUTPUT"
        rm -rf "$PLAN_DIR"
        rm -f ".kantra-ai-state.yaml"
        log_success "Cleaned up analysis files"

        log_success "Full cleanup complete!"
        ;;
    *)
        log_info "No cleanup performed"
        ;;
esac

echo
log_success "Test script complete!"
echo
log_info "Summary:"
echo "  Test codebase: $TEST_CODEBASE"
echo "  Test branch: $BRANCH_NAME"
echo "  Current branch: $(git branch --show-current)"
