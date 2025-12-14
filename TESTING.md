# Testing kantra-ai

## Quick Start

### 1. Ensure API key is set
Make sure you have `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` set in your environment (e.g., in `.zshrc` or `.bashrc`).

### 2. Create test environment
```bash
# Simple test (no git)
./test-setup.sh

# With git repo for testing git-commit features
./test-setup.sh --with-git
```

### 3. Run test commands

The setup script will output the exact commands to run. Here are the main scenarios:

#### Dry-run (safe, no changes)
```bash
./kantra-ai remediate \
  --analysis=test-run-TIMESTAMP/output.yaml \
  --input=test-run-TIMESTAMP/src \
  --provider=claude \
  --dry-run
```

#### Actually apply fixes
```bash
./kantra-ai remediate \
  --analysis=test-run-TIMESTAMP/output.yaml \
  --input=test-run-TIMESTAMP/src \
  --provider=claude \
  --max-cost=1.00
```

#### With git commits
```bash
./kantra-ai remediate \
  --analysis=test-run-TIMESTAMP/output.yaml \
  --input=test-run-TIMESTAMP/src \
  --provider=claude \
  --git-commit=per-violation
```

**Git commit strategies:**
- `per-violation` - One commit per violation ID (groups all incidents)
- `per-incident` - One commit per file/incident
- `at-end` - Single batch commit with all fixes

## Manual Test (without script)

If you prefer to set up manually:

```bash
# 1. Use existing example
cd /Users/tsanders/Workspace/kantra-ai

# 2. Dry-run test
./kantra-ai remediate \
  --analysis=examples/javax-to-jakarta/output.yaml \
  --input=examples/javax-to-jakarta/src \
  --provider=claude \
  --dry-run

# 3. Real run (will modify files in examples/)
./kantra-ai remediate \
  --analysis=examples/javax-to-jakarta/output.yaml \
  --input=examples/javax-to-jakarta/src \
  --provider=claude
```

## Verifying Results

After running (non-dry-run), compare with expected:
```bash
diff test-run-TIMESTAMP/src/UserServlet.java \
     test-run-TIMESTAMP/expected/UserServlet.java
```

Expected changes:
- `javax.servlet` → `jakarta.servlet` (all imports)

## Cost Estimates

For the javax-to-jakarta example:
- ~1 violation with 4-5 incidents (import statements)
- Estimated cost: $0.05 - $0.15 total
- Uses Claude Sonnet by default

## Available Flags

```bash
--analysis       Path to Konveyor output.yaml (required)
--input          Path to source code (required)
--provider       AI provider: claude, openai (default: claude)
--model          Specific model override (optional)
--dry-run        Show what would happen without making changes
--max-cost       Stop if cost exceeds this amount (USD)
--max-effort     Only fix violations with effort <= this number
--categories     Filter by category: mandatory, optional, potential
--violation-ids  Comma-separated list of specific violation IDs
--git-commit     Enable git commits: per-violation, per-incident, at-end
--create-pr      Create GitHub pull request(s) (requires --git-commit and GITHUB_TOKEN)
--branch         Custom branch name for PR (default: kantra-ai/remediation-TIMESTAMP)
```

## Examples with Filters

```bash
# Only mandatory violations
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --categories=mandatory

# Only low-effort fixes
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --max-effort=3

# Specific violations
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --violation-ids=javax-to-jakarta-001,another-violation-id

# Budget control
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --max-cost=2.00
```

## Testing PR Creation

### Setup GitHub Token

First, create a GitHub personal access token:
1. Go to https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Grant the `repo` scope (full repository access)
4. Copy the token and export it:

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

### Testing with Real Repository

```bash
# 1. Fork or create a test repository on GitHub
# 2. Clone it locally
git clone https://github.com/your-username/test-repo.git
cd test-repo

# 3. Run kantra analysis
kantra analyze --input=. --output=analysis

# 4. Test PR creation with at-end strategy (single PR)
./kantra-ai remediate \
  --analysis=analysis/output.yaml \
  --input=. \
  --git-commit=at-end \
  --create-pr

# 5. Check the created PR on GitHub
# Visit: https://github.com/your-username/test-repo/pulls
```

### PR Creation Strategies

```bash
# Single PR with all fixes
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --create-pr

# Multiple PRs (one per violation type)
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation \
  --create-pr

# Multiple PRs (one per file/incident)
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-incident \
  --create-pr

# Custom branch name
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --create-pr \
  --branch=feature/migration-fixes
```

### What to Expect

When PR creation succeeds, you'll see:
```
Creating pull request(s)...

✓ Created 1 pull request(s):
  - PR #42: https://github.com/owner/repo/pull/42
```

The PR will include:
- Detailed description of violations fixed
- List of files modified
- Cost and token usage statistics
- Automatic link back to kantra-ai

### Troubleshooting PR Creation

**Error: Missing GITHUB_TOKEN**
```
--create-pr requires GITHUB_TOKEN environment variable
```
Solution: Export your GitHub token as shown above.

**Error: Invalid permissions**
```
⚠ PR creation failed: Insufficient permissions
  Your token needs 'repo' scope
```
Solution: Regenerate your token with the `repo` scope.

**Error: Not a GitHub repository**
```
failed to create GitHub client: not a valid GitHub URL
```
Solution: Ensure the repository has a GitHub remote (`git remote -v`).

**Error: PR already exists**
```
⚠ PR creation failed: Validation Failed - A pull request already exists
```
Solution: Use a different branch name with `--branch` or close the existing PR.
```
