# Usage Examples

Practical examples for common kantra-ai workflows.

## Table of Contents

- [Quick Start Examples](#quick-start-examples)
- [Filtering Violations](#filtering-violations)
- [Git Commit Strategies](#git-commit-strategies)
- [Build/Test Verification](#buildtest-verification)
- [GitHub Pull Request Creation](#github-pull-request-creation)
- [Batch Processing](#batch-processing)
- [Confidence Filtering](#confidence-filtering)
- [Provider Selection](#provider-selection)
- [Phased Migration Workflow](#phased-migration-workflow)

---

## Quick Start Examples

### Preview Changes (Dry Run)

Preview what fixes would be applied without modifying any files:

```bash
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --dry-run
```

### Apply All Fixes with Cost Limit

Apply fixes with a maximum spending cap:

```bash
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --max-cost=5.00
```

### Quick Fix with Auto-Commit

Fix violations and automatically commit changes:

```bash
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --git-commit=at-end
```

---

## Filtering Violations

### By Category

Fix only mandatory violations:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --categories=mandatory
```

Fix mandatory and optional (exclude potential):

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --categories=mandatory,optional
```

### By Effort Level

Fix only low-effort violations (effort ≤ 3):

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --max-effort=3
```

### By Specific Violation IDs

Fix only specific violations:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --violation-ids=javax-to-jakarta-001,log4j-migration-002
```

### Combined Filters

Combine multiple filters:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --categories=mandatory \
  --max-effort=5 \
  --max-cost=10.00
```

---

## Git Commit Strategies

### Per-Violation (Recommended)

Create one commit per violation type, grouping related fixes together:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation
```

**Output:**
```
commit abc123 Fix: javax-to-jakarta-001 (23 incidents)
commit def456 Fix: log4j-migration-002 (15 incidents)
commit ghi789 Fix: servlet-api-upgrade-003 (8 incidents)
```

### Per-Incident

Create one commit per file/incident (most granular):

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-incident
```

**Output:**
```
commit abc123 Fix: javax-to-jakarta-001 in Controller.java
commit def456 Fix: javax-to-jakarta-001 in Service.java
commit ghi789 Fix: javax-to-jakarta-001 in Repository.java
```

### At End (Single Commit)

Create one commit with all fixes:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end
```

**Output:**
```
commit abc123 Fix: Applied 46 fixes across 3 violations
```

---

## Build/Test Verification

### Run Tests After Fixes

Verify fixes don't break tests:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=test
```

### Run Build Verification Only

Faster than tests, just checks compilation:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=build
```

### Verify After Each Fix

Catch issues immediately (slower but safer):

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation \
  --verify=test \
  --verify-strategy=per-fix
```

### Custom Verification Command

Use your own verification command:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=test \
  --verify-command="make test"
```

### Continue on Verification Failures

Don't stop at first failure:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=test \
  --verify-fail-fast=false
```

---

## GitHub Pull Request Creation

### Single PR with All Fixes

Create one PR containing all fixes:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --create-pr
```

### Separate PRs Per Violation

Create individual PRs for each violation type:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation \
  --create-pr
```

**Output:**
```
PR #1: Fix: javax-to-jakarta-001 (23 incidents)
PR #2: Fix: log4j-migration-002 (15 incidents)
PR #3: Fix: servlet-api-upgrade-003 (8 incidents)
```

### Custom Branch Name

Specify a custom branch name:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --create-pr \
  --branch=feature/konveyor-migration
```

### PR with Verification

Create PR after running tests:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=test \
  --create-pr
```

---

## Batch Processing

### Enable Batch Processing

Batch processing is enabled by default, but you can customize it:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --batch-size=10 \
  --batch-parallelism=4
```

### Disable Batching

Process violations one at a time:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --batch-size=1
```

### Aggressive Batching

Maximum batching for fastest execution:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --batch-size=10 \
  --batch-parallelism=8
```

---

## Confidence Filtering

### Skip Low-Confidence Fixes

Only apply fixes the AI is confident about:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --enable-confidence \
  --on-low-confidence=skip
```

### Warn But Apply

Show warnings for low confidence but apply anyway:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --enable-confidence \
  --on-low-confidence=warn-and-apply
```

### Global Minimum Confidence

Set a minimum confidence threshold for all fixes:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --enable-confidence \
  --min-confidence=0.85
```

### Custom Complexity Thresholds

Require higher confidence for complex changes:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --enable-confidence \
  --complexity-threshold="high=0.95,expert=0.98"
```

---

## Provider Selection

### Claude (Anthropic)

Use Claude for highest quality fixes:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=claude \
  --model=claude-sonnet-4-20250514
```

### OpenAI

Use GPT-4 or GPT-3.5:

```bash
export OPENAI_API_KEY=sk-...
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=openai \
  --model=gpt-4
```

### Groq (Fast Inference)

Ultra-fast inference with Groq:

```bash
export OPENAI_API_KEY=gsk_...
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=groq \
  --model=llama-3.1-70b-versatile
```

### Ollama (Local Models)

Run locally for free:

```bash
ollama serve  # Start Ollama server
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=ollama \
  --model=codellama
```

### Together AI (Open Source Models)

Use open source models via Together AI:

```bash
export OPENAI_API_KEY=...
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=together \
  --model=meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
```

---

## Phased Migration Workflow

### Generate Plan

Create an AI-powered migration plan:

```bash
./kantra-ai plan \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude
```

**Output:**
```
📦 Batching 44 violations into smaller groups...
   Split into 3 batches

   Progress [████████████████████████████████████████] 100% | Batch 3/3 | Complete

✓ Generated 4 final phases

Created directory: .kantra-ai-plan/
Plan saved to:     .kantra-ai-plan/plan.yaml
HTML report:       .kantra-ai-plan/plan.html
  Total phases: 4
  Total violations: 44
  Estimated cost: $4.30
```

### Generate Plan with Interactive Web UI

Launch web-based planner:

```bash
./kantra-ai plan \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --interactive-web
```

**Output:**
```
🌐 Starting web interface at http://localhost:8080
   Browser opens automatically
```

### Generate Plan with CLI Approval

Use terminal-based phase approval:

```bash
./kantra-ai plan \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --interactive
```

### Execute Plan

Run the generated plan:

```bash
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --input=./your-app \
  --provider=claude
```

### Execute Specific Phase Only

Run just one phase:

```bash
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --input=./your-app \
  --provider=claude \
  --phase=phase-1
```

### Resume After Failure

Continue from where you left off:

```bash
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --input=./your-app \
  --provider=claude \
  --resume
```

### Execute with Verification

Run tests after each phase:

```bash
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --input=./your-app \
  --provider=claude \
  --verify=test \
  --verify-strategy=per-violation
```

---

## Complete Workflow Examples

### Safe Production Migration

Maximum safety with verification and manual review:

```bash
# Step 1: Generate plan with conservative settings
./kantra-ai plan \
  --analysis=output.yaml \
  --input=src \
  --provider=claude \
  --risk-tolerance=conservative \
  --categories=mandatory

# Step 2: Review plan
open .kantra-ai-plan/plan.html

# Step 3: Execute with confidence filtering and testing
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --input=src \
  --provider=claude \
  --enable-confidence \
  --on-low-confidence=skip \
  --verify=test \
  --verify-strategy=per-violation \
  --git-commit=per-violation \
  --create-pr
```

### Fast Development Iteration

Quick fixes for development:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=groq \
  --max-effort=3 \
  --git-commit=at-end
```

### Cost-Optimized Migration

Minimize costs with local models:

```bash
ollama serve

./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=ollama \
  --model=codellama \
  --batch-size=10 \
  --git-commit=at-end
```

---

## See Also

- [CLI Reference](./CLI_REFERENCE.md) - Complete command-line flag reference
- [AI Providers](./AI_PROVIDERS.md) - Detailed provider comparison
- [Confidence Filtering](./CONFIDENCE_FILTERING.md) - Confidence threshold guide
- [Quick Start](./QUICKSTART.md) - Getting started guide
