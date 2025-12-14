# Multi-File Violations Example

This example demonstrates how kantra-ai handles violations across multiple files with different git commit strategies.

## Scenario

A Java web application with **3 files** and **2 violation types**:

### Files & Violations

1. **`src/api/UserController.java`**
   - ✗ `javax-to-jakarta`: Uses old `javax.servlet` imports (4 incidents)

2. **`src/service/DataService.java`**
   - ✗ `javax-to-jakarta`: Uses `javax.servlet.ServletContext` (1 incident)
   - ✗ `hardcoded-credentials`: Has hardcoded database credentials (1 incident)

3. **`src/config/DatabaseConfig.java`**
   - ✗ `hardcoded-credentials`: Has hardcoded passwords (2 incidents)

**Total**: 2 violations, 8 incidents across 3 files

## Commit Strategies Comparison

This example is perfect for understanding how different commit strategies work:

### Strategy 1: `per-incident`
**One commit per file that has violations**

```bash
# Result: 3 commits
Commit 1: Fix javax imports in UserController.java
Commit 2: Fix javax imports and credentials in DataService.java
Commit 3: Fix hardcoded credentials in DatabaseConfig.java
```

**When to use**: When you want fine-grained git history, or when files are unrelated

### Strategy 2: `per-violation`
**One commit per violation type (groups related fixes)**

```bash
# Result: 2 commits
Commit 1: Fix javax-to-jakarta violations (UserController + DataService)
Commit 2: Fix hardcoded-credentials violations (DataService + DatabaseConfig)
```

**When to use**: When you want to group related changes logically

### Strategy 3: `at-end`
**One commit with all fixes**

```bash
# Result: 1 commit
Commit 1: Fix all Konveyor violations (all 3 files, both violation types)
```

**When to use**: For batch migrations or when you don't need granular history

## Testing Each Strategy

### Setup

```bash
# Navigate to kantra-ai root
cd /path/to/kantra-ai

# Build the tool
go build -o kantra-ai ./cmd/kantra-ai

# Make the example a git repo for testing
cd examples/multi-file-violations
git init
git add .
git commit -m "Initial commit with violations"
cd ../..
```

### Test 1: Per-Incident Strategy

```bash
# Using config file
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/.kantra-ai-per-incident.yaml

# Or with CLI flags
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/output.yaml \
  --input=./examples/multi-file-violations \
  --provider=claude \
  --git-commit=per-incident

# Check git history
cd examples/multi-file-violations
git log --oneline
# Expected: 3 commits (one per file)

# Reset for next test
git reset --hard HEAD~3
cd ../..
```

### Test 2: Per-Violation Strategy

```bash
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/.kantra-ai-per-violation.yaml

cd examples/multi-file-violations
git log --oneline --stat
# Expected: 2 commits
#   Commit 1: UserController + DataService (javax fixes)
#   Commit 2: DataService + DatabaseConfig (credential fixes)

git reset --hard HEAD~2
cd ../..
```

### Test 3: At-End Strategy

```bash
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/.kantra-ai-at-end.yaml

cd examples/multi-file-violations
git log --oneline --stat
# Expected: 1 commit with all 3 files

git reset --hard HEAD~1
cd ../..
```

## Dry-Run Mode

Preview what would be fixed without making changes:

```bash
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/output.yaml \
  --input=./examples/multi-file-violations \
  --provider=claude \
  --dry-run
```

## Verification

Compare fixed files with expected results:

```bash
# After running remediation
diff -r ./examples/multi-file-violations/src \
        ./examples/multi-file-violations/expected

# No output = perfect match!
```

## Expected Fixes

### UserController.java
- `javax.servlet.*` → `jakarta.servlet.*`
- All 4 import statements updated

### DataService.java
- `javax.servlet.ServletContext` → `jakarta.servlet.ServletContext`
- Hardcoded credentials → Environment variables (`System.getenv()`)

### DatabaseConfig.java
- Hardcoded passwords → Environment variables with validation
- Added error handling for missing env vars

## Cost Estimation

**Typical costs** (using Claude Sonnet):
- Per file: ~$0.05-0.10
- Total for all 3 files: ~$0.15-0.30

Use `--dry-run` to see cost estimates before applying fixes.

## Success Criteria

✅ All `javax.servlet` imports replaced with `jakarta.servlet`
✅ All hardcoded credentials moved to environment variables
✅ Code structure and logic preserved
✅ Git commits created according to chosen strategy
✅ Files match expected output

## Troubleshooting

**Issue**: AI doesn't remove all credentials
- **Solution**: Check that the Konveyor `output.yaml` correctly identifies all incidents

**Issue**: Git commits not created
- **Solution**: Ensure the example directory is a git repository (`git init`)

**Issue**: Diff shows unexpected changes
- **Solution**: AI providers may format code slightly differently; verify logic is correct

## Configuration Files

Three example config files are provided:

1. **`.kantra-ai-per-incident.yaml`** - 3 commits (one per file)
2. **`.kantra-ai-per-violation.yaml`** - 2 commits (one per violation type)
3. **`.kantra-ai-at-end.yaml`** - 1 commit (all fixes together)

Use them with:
```bash
./kantra-ai remediate --config=examples/multi-file-violations/.kantra-ai-per-violation.yaml
```

Or copy one to `.kantra-ai.yaml` in the example directory and just run:
```bash
./kantra-ai remediate
```

## Pull Request Creation

### Prerequisites

To create pull requests, you need:

1. **GitHub Token** with `repo` scope:
   ```bash
   # Create at: https://github.com/settings/tokens
   export GITHUB_TOKEN=ghp_your_token_here
   ```

2. **GitHub Repository** - This example directory must be:
   - A git repository with commits
   - Connected to a GitHub remote
   - Pushed to GitHub

### Setup for PR Testing

```bash
# Navigate to example directory
cd examples/multi-file-violations

# Initialize as git repo (if not already)
git init
git add .
git commit -m "Initial commit with violations"

# Connect to GitHub (create empty repo first on github.com)
git remote add origin https://github.com/YOUR_USERNAME/kantra-ai-multi-file-test.git
git branch -M main
git push -u origin main

# Return to kantra-ai root
cd ../..
```

### PR Strategy Examples

#### Per-Violation Strategy (2 PRs)

Creates **2 pull requests** (one per violation type):

```bash
# Set environment variables
export GITHUB_TOKEN=ghp_your_token
export ANTHROPIC_API_KEY=sk-ant-your_key  # or OPENAI_API_KEY

# Run with PR creation
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/output.yaml \
  --input=./examples/multi-file-violations \
  --provider=claude \
  --git-commit=per-violation \
  --create-pr
```

**Result:**
- PR #1: Fix javax-to-jakarta violations (UserController + DataService)
- PR #2: Fix hardcoded-credentials violations (DataService + DatabaseConfig)

**Branch names:**
- `kantra-ai/remediation-javax-to-jakarta-001-TIMESTAMP`
- `kantra-ai/remediation-hardcoded-credentials-001-TIMESTAMP`

#### Per-Incident Strategy (3 PRs)

Creates **3 pull requests** (one per file):

```bash
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/output.yaml \
  --input=./examples/multi-file-violations \
  --provider=claude \
  --git-commit=per-incident \
  --create-pr
```

**Result:**
- PR #1: Fix violations in UserController.java
- PR #2: Fix violations in DataService.java
- PR #3: Fix violations in DatabaseConfig.java

#### At-End Strategy (1 PR)

Creates **1 pull request** with all fixes:

```bash
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/output.yaml \
  --input=./examples/multi-file-violations \
  --provider=claude \
  --git-commit=at-end \
  --create-pr
```

**Result:**
- PR #1: Batch remediation of 2 violations (all 3 files)

**Branch name:**
- `kantra-ai/remediation-TIMESTAMP`

### Custom Branch Names

```bash
# Use custom branch prefix
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/output.yaml \
  --input=./examples/multi-file-violations \
  --provider=claude \
  --git-commit=per-violation \
  --create-pr \
  --branch=feature/konveyor-migration
```

### Dry-Run Mode

Preview PRs without creating them:

```bash
./kantra-ai remediate \
  --analysis=./examples/multi-file-violations/output.yaml \
  --input=./examples/multi-file-violations \
  --provider=claude \
  --git-commit=per-violation \
  --create-pr \
  --dry-run
```

This will:
- Show what fixes would be applied
- Show what commits would be created
- Show what PRs would be created
- **Not** actually create any PRs or push to GitHub

### Verifying PRs

After running with `--create-pr`, check your GitHub repository:

1. **View Pull Requests:**
   ```
   https://github.com/YOUR_USERNAME/YOUR_REPO/pulls
   ```

2. **Check PR Content:**
   - Summary section with violation details
   - List of files modified with line numbers
   - AI remediation details (provider, cost, tokens)
   - Link to kantra-ai in footer

3. **Review Changes:**
   - Click "Files changed" tab
   - Verify javax → jakarta replacements
   - Verify credentials → environment variables
   - Check that code logic is preserved

### Cleanup After Testing

```bash
# Close/delete test PRs on GitHub
# Then delete remote branches
git push origin --delete kantra-ai/remediation-*

# Or reset local repository
cd examples/multi-file-violations
git reset --hard HEAD~10  # Reset last 10 commits
git push --force  # Force push reset (use with caution!)
```

## Learning Points

1. **Commit strategies** affect git history organization, not what gets fixed
2. **All strategies** fix the same code - only commit grouping differs
3. **per-violation** is often best for code review (groups related changes)
4. **per-incident** gives most granular history
5. **at-end** is simplest for bulk migrations

## Next Steps

- Try with `--create-pr` to create pull requests for each strategy
- Add `--verify=test` to run tests after fixes (requires Java setup)
- Experiment with `--max-effort` to fix only certain violations
- Use `--categories=mandatory` to filter by violation category
