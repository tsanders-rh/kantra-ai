# Kantra-AI Scripts

Helper scripts for testing and development.

> **Quick Start:** See [test-e2e-README.md](test-e2e-README.md) for a simple getting started guide.

## test-e2e.sh

End-to-end test script for validating kantra-ai functionality with a real codebase.

### Purpose

This script automates the entire kantra-ai workflow:
1. Creates a test branch
2. Runs Konveyor analysis
3. Creates a migration plan
4. Approves a simple phase
5. Executes fixes with git commits and PR creation
6. Provides cleanup options

### Prerequisites

- `git` - Version control
- `gh` - GitHub CLI (for PR management)
- `kantra` - Konveyor analysis tool
- `kantra-ai` - Built binary in project root
- GitHub token with PR permissions

### Setup

```bash
# Build kantra-ai
go build -o kantra-ai ./cmd/kantra-ai

# Configure GitHub CLI
gh auth login

# Set GitHub token for PR creation
export GITHUB_TOKEN=$(gh auth token)

# Optional: Set your preferred AI provider API key
export ANTHROPIC_API_KEY=your-key-here
```

### Usage

```bash
# Use default test codebase (~/Workspace/boat-fuel-tracker-j2ee)
./scripts/test-e2e.sh

# Or specify a different codebase
./scripts/test-e2e.sh ~/path/to/your/java/project
```

### What It Does

**Step 1: Setup**
- Validates prerequisites
- Checks git status
- Creates timestamped test branch

**Step 2: Analysis**
- Runs `kantra analyze` on the codebase
- Generates analysis output YAML
- Customizable command via interactive prompt

**Step 3: Planning**
- Runs `kantra-ai plan` to create migration plan
- Generates plan.yaml and plan.html
- Shows plan summary

**Step 4: Workflow Selection**
Choose between two workflows:

**Option 1: Interactive Web UI** (Recommended)
- Launches web UI at http://localhost:8080
- Approve/defer phases with visual charts
- Configure git, PR, and verification settings
- Execute with live progress monitoring
- View results in-browser with PR links

**Option 2: CLI Mode** (Traditional)
- Manually approve phases in plan.yaml
- Run execution via command line
- Configure via CLI flags

**Step 5: Review**
- Shows created commits
- Lists changed files
- Displays created pull requests
- Shows execution statistics

**Step 6: Cleanup**
Three cleanup options:
1. **Keep everything** - Review at your leisure
2. **Close PR, keep local** - Clean up GitHub, keep local branch
3. **Full cleanup** - Delete everything, return to original state

### Example Session

```bash
$ ./scripts/test-e2e.sh ~/Workspace/boat-fuel-tracker-j2ee

==============================================
  Kantra-AI End-to-End Test Script
==============================================

ℹ️  Test codebase: /Users/you/Workspace/boat-fuel-tracker-j2ee
ℹ️  Test branch: kantra-ai-test-20251220-143022

ℹ️  Step 1: Navigating to test codebase...
✓ Working directory: /Users/you/Workspace/boat-fuel-tracker-j2ee

Press Enter to continue or Ctrl+C to abort...

ℹ️  Step 2: Creating test branch...
✓ Created and checked out branch: kantra-ai-test-20251220-143022

ℹ️  Step 3: Running Konveyor analysis...
⚠️  This will run: kantra analyze --input . --output analysis-output.yaml --target quarkus

Modify the kantra command? (y/N) n

[Analysis runs...]

ℹ️  Step 4: Creating migration plan...
[Plan created]

ℹ️  Step 5: Choose approval and execution workflow...

How would you like to approve phases and execute?
  1) Interactive Web UI - Approve and execute in browser with live monitoring
  2) CLI Mode - Manually approve in plan.yaml, then execute via command line

Enter choice (1-2): 1

ℹ️  Starting interactive web planner for approval and execution...

In the web UI you can:
  ✓ Review the migration plan with visual charts
  ✓ Approve/defer phases interactively
  ✓ Configure settings (git, PR, verification, etc.)
  ✓ Execute with live progress monitoring
  ✓ View execution results and created PRs

⚠️  Recommended: Approve just ONE simple phase for testing

Press Enter to continue...

ℹ️  Launching web UI at http://localhost:8080

[Browser opens, you approve phases and execute in the UI]

✓ Web UI session complete!
...
```

### Cleanup Commands

If you exit the script early or want to clean up manually:

```bash
# Close any open PRs for test branches
gh pr list --search "kantra-ai-test" --state open
gh pr close <PR_NUMBER>

# Delete remote test branches
git push origin --delete kantra-ai-test-20251220-143022

# Delete local test branches
git branch -D kantra-ai-test-20251220-143022

# Return to main branch
git checkout main

# Clean up test files
rm -f analysis-output.yaml
rm -rf .kantra-ai-plan
rm -f .kantra-ai-state.yaml
```

### Tips

1. **Use Interactive Web UI**: Choose option 1 for the best experience - you can approve phases, configure all settings, and execute with live monitoring all in your browser

2. **Start Small**: Approve only one low-effort, low-risk phase for your first test

3. **Review Before Cleanup**: Use cleanup option 1 to keep everything, review the PR and commits, then manually clean up later

4. **Iterate Quickly**: Use cleanup option 3 for full cleanup, then run the script again to test changes

5. **Test Different Strategies**: In the web UI, use the Settings panel to test different commit/PR strategies:
   - Per-violation commits and PRs
   - Per-phase commits with single PR at end
   - Single commit and PR after all fixes

6. **Monitor Live Progress**: When using the web UI, watch execution progress in real-time with the activity log and statistics

### Troubleshooting

**"kantra-ai binary not found"**
```bash
cd /path/to/kantra-ai
go build -o kantra-ai ./cmd/kantra-ai
```

**"gh is not installed"**
```bash
# macOS
brew install gh

# Linux
# See: https://github.com/cli/cli/blob/trunk/docs/install_linux.md
```

**"GITHUB_TOKEN not set"**
```bash
export GITHUB_TOKEN=$(gh auth token)
```

**"PR creation failed"**
- Ensure you have push access to the repository
- Verify GitHub token has PR creation permissions
- Check if repository has branch protection rules

### Customization

Edit the script to customize:
- Default Konveyor analysis command (line ~140)
- AI provider (line ~280, change `--provider claude`)
- Commit strategy (line ~282)
- PR strategy (line ~284)
- PR branch prefix (line ~285)

### Related

- Main documentation: `docs/USAGE.md`
- Interactive web UI: `docs/WEB_INTERACTIVE_USAGE.md`
- PR creation guide: `docs/PR_CREATION.md`
