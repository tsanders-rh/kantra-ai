# E2E Test Script - Quick Guide

Automated end-to-end testing for kantra-ai with real codebases.

## Quick Start

```bash
# 1. Build kantra-ai
go build -o kantra-ai ./cmd/kantra-ai

# 2. Set up GitHub token
export GITHUB_TOKEN=$(gh auth token)

# 3. Run the test
./scripts/test-e2e.sh ~/Workspace/boat-fuel-tracker-j2ee
```

## What It Does

Automates the complete workflow:
1. ✅ Runs Konveyor analysis
2. ✅ Creates a test branch
3. ✅ **Choose workflow (Web UI or CLI)**
4. ✅ Generates migration plan
5. ✅ Approval and execution (web or CLI)
6. ✅ Shows results (commits, PRs, stats)
7. ✅ Cleanup options

## The Two Workflows

### Option 1: Web UI (Recommended) 🌐

Everything in your browser:
- Creates plan and launches browser automatically
- Approve phases visually
- Configure settings
- Execute with live monitoring
- See results with PR links

**Flow:**
```
Script → Analysis → Branch → Choose Web UI → Plan created + Browser opens →
You approve & execute → Results → Cleanup
```

### Option 2: CLI Mode

Traditional approach:
- Creates static plan files (YAML + HTML)
- Manual approval in YAML
- Command-line execution
- Flag-based configuration

**Flow:**
```
Script → Analysis → Branch → Choose CLI → Plan created → Manual approval →
CLI execution → Results → Cleanup
```

## Common Usage

**Standard test run:**
```bash
./scripts/test-e2e.sh ~/Workspace/boat-fuel-tracker-j2ee
# Choose option 1 (Web UI) when prompted
# Approve one simple phase in browser
# Click Execute
# Choose cleanup option 3 to reset everything
```

**Quick iteration:**
```bash
# Run test, full cleanup, repeat
./scripts/test-e2e.sh ~/path/to/codebase
# ... test in web UI ...
# Choose cleanup option 3
./scripts/test-e2e.sh ~/path/to/codebase  # Run again immediately
```

**Keep for review:**
```bash
./scripts/test-e2e.sh ~/path/to/codebase
# ... test in web UI ...
# Choose cleanup option 1
gh pr view  # Review the PR
git diff main  # Review changes
```

## Cleanup Options

After execution:
1. **Keep everything** - Review PR and changes later
2. **Partial cleanup** - Close PR, keep local branch
3. **Full cleanup** - Delete everything, ready to run again

## Prerequisites

Install these first:
```bash
# GitHub CLI
brew install gh  # macOS
# or see: https://cli.github.com/

# Konveyor
# See: https://konveyor.io/

# Authenticate GitHub CLI
gh auth login
```

## Troubleshooting

**Port 8080 already in use:**
```bash
lsof -i :8080
kill -9 <PID>
```

**GitHub token not set:**
```bash
export GITHUB_TOKEN=$(gh auth token)
```

**Script can't find kantra-ai:**
```bash
cd /path/to/kantra-ai
go build -o kantra-ai ./cmd/kantra-ai
```

## Tips

💡 **Start small** - Approve just ONE simple phase for your first test

💡 **Use Web UI** - It's the recommended workflow with the best experience

💡 **Full cleanup** - Use option 3 to quickly iterate on tests

💡 **Check GitHub token** - Set it before running to enable PR creation

## What Gets Created

During a test run:
- `kantra-ai-test-TIMESTAMP` branch
- `analysis-output.yaml` file
- `.kantra-ai-plan/` directory
- Git commits on test branch
- GitHub pull request (if token is set)
- `.kantra-ai-state.yaml` execution state

All cleaned up when you choose option 3.

## Example Output

```bash
$ ./scripts/test-e2e.sh ~/Workspace/boat-fuel-tracker-j2ee

==============================================
  Kantra-AI End-to-End Test Script
==============================================

ℹ️  Test codebase: /Users/you/Workspace/boat-fuel-tracker-j2ee
ℹ️  Test branch: kantra-ai-test-20251220-153045

[Analysis runs...]

ℹ️  Step 5: Choose approval and execution workflow...

  1) Interactive Web UI - Approve and execute in browser
  2) CLI Mode - Manual approval and CLI execution

Enter choice (1-2): 1

ℹ️  Launching web UI at http://localhost:8080

[Browser opens, you approve and execute]

✓ Web UI session complete!

ℹ️  Git commits:
  a1b2c3d Fix: javax.servlet → jakarta.servlet migration
  e4f5g6h Fix: EJB 2.x → EJB 3.x annotations

ℹ️  Pull requests created:
  #42 - Migration: Phase 1 - Servlet API Updates
  https://github.com/owner/repo/pull/42

Choose cleanup action:
  1) Keep everything
  2) Close PR, keep local
  3) Full cleanup

Enter choice (1-3): 3

✓ Full cleanup complete!
```

## Need More Help?

- **Full documentation:** [scripts/README.md](README.md)
- **Web UI guide:** [docs/WEB_INTERACTIVE_USAGE.md](../docs/WEB_INTERACTIVE_USAGE.md)
- **Issues:** https://github.com/tsanders-rh/kantra-ai/issues
