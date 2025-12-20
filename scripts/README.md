# Kantra-AI Scripts

Helper scripts for testing and development.

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

**Step 4: Approval**
- Option to use interactive web UI
- Or manually approve phases in plan.yaml
- Recommends approving just one simple phase for testing

**Step 5: Execution**
- Runs `kantra-ai execute` with:
  - Git commits enabled (per-phase strategy)
  - PR creation enabled (at-end strategy)
  - Real changes (not dry-run)
- Creates commits and pull request

**Step 6: Review**
- Shows created commits
- Lists changed files
- Displays created pull requests
- Shows execution statistics

**Step 7: Cleanup**
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

Modify the kantra command? (y/N)
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

1. **Start Small**: Approve only one low-effort, low-risk phase for your first test

2. **Review Before Cleanup**: Use cleanup option 1 to keep everything, review the PR and commits, then manually clean up later

3. **Iterate Quickly**: Use cleanup option 3 for full cleanup, then run the script again to test changes

4. **Test Different Strategies**: Modify the execute step to test different commit/PR strategies:
   ```bash
   # Per-violation commits and PRs
   ./kantra-ai execute ... --commit-strategy per-violation --pr-strategy per-violation

   # Single commit and PR at end
   ./kantra-ai execute ... --commit-strategy at-end --pr-strategy at-end
   ```

5. **Dry Run First**: Modify the script to add `--dry-run` flag to execution step if you want to preview without making changes

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
