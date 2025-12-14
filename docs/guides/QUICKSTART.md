# Quick Start Guide

Get kantra-ai running in 5 minutes.

## Prerequisites

```bash
# Required
go 1.21+
export ANTHROPIC_API_KEY=sk-ant-...  # Get from https://console.anthropic.com
# OR
export OPENAI_API_KEY=sk-...         # Get from https://platform.openai.com

# For testing (optional)
kantra  # Install from https://github.com/konveyor/kantra
```

## Installation

```bash
# Clone the repo
git clone https://github.com/yourusername/kantra-ai
cd kantra-ai

# Install dependencies
make deps

# Build
make build
```

## Test with Example

```bash
# Dry run first (see what would happen)
make run-example

# Actually apply the fix
make run-example-real

# Verify it worked
make verify-example

# Reset for next test
make reset-example
```

## Use with Real Konveyor Analysis

### Step 1: Run Konveyor Analysis

```bash
# Analyze your application
kantra analyze \
  --input=/path/to/your/app \
  --output=./analysis \
  --target=quarkus

# This creates: ./analysis/output.yaml
```

### Step 2: Try AI Remediation (Dry Run)

```bash
# See what would be fixed
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=/path/to/your/app \
  --provider=claude \
  --categories=mandatory \
  --max-effort=1 \
  --dry-run
```

### Step 3: Fix a Few Violations

```bash
# Start small - fix just 5 violations
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=/path/to/your/app \
  --provider=claude \
  --categories=mandatory \
  --max-effort=1 \
  --max-cost=1.00
```

### Step 4: Review the Changes

**Option A: Manual Review (No Auto-Commit)**
```bash
# See what changed
git diff

# If good: commit
git add .
git commit -m "AI fixes for mandatory violations (effort ≤ 1)"

# If bad: revert
git checkout .
```

**Option B: Auto-Commit (Recommended)**
```bash
# Let kantra-ai create commits automatically
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=/path/to/your/app \
  --provider=claude \
  --categories=mandatory \
  --max-effort=1 \
  --max-cost=1.00 \
  --git-commit=per-violation

# Review commit history
git log --oneline

# Cherry-pick commits you want to keep
# Or revert specific commits if needed
git revert <commit-hash>
```

## Command Reference

### Basic Usage

```bash
kantra-ai remediate \
  --analysis=<path-to-output.yaml> \
  --input=<path-to-source-code> \
  --provider=<claude|openai>
```

### Common Filters

```bash
# Fix specific violations by ID
--violation-ids=violation-001,violation-042

# Fix by category
--categories=mandatory,optional

# Limit by effort
--max-effort=3

# Limit cost
--max-cost=10.00

# Dry run (no changes)
--dry-run

# Git commit strategies
--git-commit=per-violation   # One commit per violation type
--git-commit=per-incident    # One commit per file fix
--git-commit=at-end          # Single batch commit at end
```

### Provider Selection

```bash
# Use Claude (default)
--provider=claude
--model=claude-sonnet-4-20250514  # Optional: specify model

# Use OpenAI
--provider=openai
--model=gpt-4  # Optional: default is gpt-4
```

## Example Workflows

### Conservative: Fix Easy Wins Only

```bash
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./myapp \
  --provider=claude \
  --categories=mandatory \
  --max-effort=1 \
  --max-cost=5.00
```

### Targeted: Fix Specific Violations

```bash
# Review output.yaml, identify violations to fix
# Then fix just those:
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./myapp \
  --provider=claude \
  --violation-ids=violation-042,violation-103
```

### Provider Comparison

```bash
# Try Claude
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./myapp \
  --provider=claude \
  --violation-ids=violation-001 \
  --dry-run

# Try OpenAI on same violation
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./myapp \
  --provider=openai \
  --violation-ids=violation-001 \
  --dry-run

# Compare results
```

### Git Commit Strategies

```bash
# Per-Violation: One commit per violation type (best for review)
# All fixes for a violation are in one commit
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./myapp \
  --provider=claude \
  --git-commit=per-violation

# Per-Incident: One commit per file fix (most granular)
# Each file gets its own commit
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./myapp \
  --provider=claude \
  --git-commit=per-incident

# At-End: Single batch commit (simplest)
# All fixes in one commit at the end
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./myapp \
  --provider=claude \
  --git-commit=at-end

# Review commits
git log --oneline --graph
git show <commit-hash>  # See details of a specific commit
```

## Tips

1. **Always dry-run first** - See what would happen before spending money
2. **Start small** - Test with 5-10 violations before doing bulk fixes
3. **Use auto-commit** - `--git-commit=per-violation` makes review easier
4. **Review commit history** - Check `git log` to see what was fixed
5. **Track results** - Update VALIDATION.md with your findings
6. **Use cost limits** - Set `--max-cost` to avoid surprises

## Troubleshooting

### API Key Not Found

```bash
# Make sure you've set the environment variable
export ANTHROPIC_API_KEY=sk-ant-your-key-here

# Or add to your shell profile (~/.bashrc, ~/.zshrc)
echo 'export ANTHROPIC_API_KEY=sk-ant-your-key-here' >> ~/.zshrc
```

### File Not Found Errors

```bash
# Make sure paths are correct
--analysis=./analysis/output.yaml    # Path to output.yaml
--input=./myapp                      # Path to source code root
```

### Provider Errors

```bash
# Test your API key
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-4-20250514","max_tokens":10,"messages":[{"role":"user","content":"Hi"}]}'
```

## Next Steps

1. Test on small examples
2. Track results in VALIDATION.md
3. Try on real projects
4. Compare providers
5. Share feedback

## Getting Help

- Check [README.md](./README.md) for full documentation
- Review [DESIGN.md](../design/DESIGN.md) for architecture details
- See [examples/](./examples/) for more test cases
- Open an issue on GitHub
