# kantra-ai

[![Tests](https://github.com/tsanders-rh/kantra-ai/actions/workflows/test.yml/badge.svg)](https://github.com/tsanders-rh/kantra-ai/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/tsanders-rh/kantra-ai/branch/main/graph/badge.svg)](https://codecov.io/gh/tsanders-rh/kantra-ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsanders-rh/kantra-ai)](https://goreportcard.com/report/github.com/tsanders-rh/kantra-ai)

AI-powered automated remediation for [Konveyor](https://www.konveyor.io/) violations. Use Claude or OpenAI to automatically fix code issues identified during application modernization and migration.

## Features

- **Phased Migration Planning**: AI-generated migration plans with risk assessment and execution order
- **Automated Code Fixes**: AI analyzes violations and applies fixes directly to your source code
- **Batch Processing**: Group similar violations together for 50-80% cost reduction and 70-90% faster execution
- **Multiple AI Providers**: Support for Claude (Anthropic) and OpenAI with easy provider switching
- **Smart Filtering**: Filter by violation category, effort level, or specific violation IDs
- **Resume Capability**: Resume from failures with incident-level state tracking
- **Interactive Approval**: Review and approve phases before execution
- **Git Integration**: Automatic commit creation with configurable strategies (per-violation, per-incident, or batch)
- **GitHub PR Automation**: Automatically create pull requests with detailed fix summaries
- **Build/Test Verification**: Run tests or builds after fixes to ensure they don't break existing functionality
- **Cost Controls**: Set spending limits and track API costs per fix
- **Dry-Run Mode**: Preview changes before applying them
- **Detailed Reporting**: Track success rates, costs, and tokens used

## Quick Start

### Prerequisites

**Required:**
- Go 1.21 or higher
- AI provider API key (Claude or OpenAI)
- Konveyor analysis output (`output.yaml`)

**Optional (for PR creation):**
- GitHub personal access token with `repo` scope

```bash
# Set your AI provider API key
export ANTHROPIC_API_KEY=sk-ant-...  # for Claude
# OR
export OPENAI_API_KEY=sk-...         # for OpenAI

# Optional: Set GitHub token for PR creation
export GITHUB_TOKEN=ghp_...
```

### Installation

```bash
git clone https://github.com/tsanders-rh/kantra-ai
cd kantra-ai
go build -o kantra-ai ./cmd/kantra-ai
```

Or install directly:

```bash
go install github.com/tsanders-rh/kantra-ai/cmd/kantra-ai@latest
```

### Basic Usage

1. **Run Konveyor analysis** on your application:
   ```bash
   kantra analyze --input=./your-app --output=./analysis
   ```

2. **Preview fixes** with dry-run mode:
   ```bash
   ./kantra-ai remediate \
     --analysis=./analysis/output.yaml \
     --input=./your-app \
     --provider=claude \
     --dry-run
   ```

3. **Apply fixes** with a cost limit:
   ```bash
   ./kantra-ai remediate \
     --analysis=./analysis/output.yaml \
     --input=./your-app \
     --provider=claude \
     --max-cost=5.00
   ```

## Two Workflows

kantra-ai supports two workflows depending on the size and complexity of your migration:

### 1. Direct Remediation (Quick Fixes)

For small migrations with < 20 violations, use the `remediate` command for immediate fixes:

```bash
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude
```

**Best for**: Simple migrations, testing, proof-of-concept

### 2. Phased Migration (Large-Scale)

For larger migrations with 20+ violations, use the `plan` → `execute` workflow:

**Step 1: Generate a plan** with AI-powered grouping and risk assessment:

```bash
./kantra-ai plan \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude

# Output:
# Plan saved to: .kantra-ai-plan.yaml
#   Total phases: 3
#   Total violations: 45
#   Estimated cost: $4.30
```

**Step 2: Review and edit the plan** (optional):

```bash
# Review the generated plan
cat .kantra-ai-plan.yaml

# Edit if needed (mark phases as deferred, adjust order, etc.)
vim .kantra-ai-plan.yaml
```

**Step 3: Execute the plan** with progress tracking:

```bash
./kantra-ai execute \
  --input=./your-app \
  --provider=claude

# State is saved to .kantra-ai-state.yaml for resume capability
```

**Step 4: Resume from failures** if needed:

```bash
# If execution fails mid-way, resume from the failure point
./kantra-ai execute \
  --input=./your-app \
  --provider=claude \
  --resume
```

**Benefits of phased migration**:
- **AI-powered grouping**: Violations grouped by risk, category, and effort
- **Incremental execution**: Execute one phase at a time
- **Resume capability**: Continue from where you left off after failures
- **State tracking**: Incident-level progress tracking
- **Interactive approval**: Review phases before execution (with `--interactive`)

### Interactive Phase Approval

For full control over what gets executed, use interactive mode:

```bash
./kantra-ai plan \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --interactive

# Output:
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Phase 1 of 3: Critical Mandatory Fixes - High Effort
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#
# Order:    1
# Risk:     🔴 HIGH
# Category: mandatory
# Effort:   5-7
#
# Why this grouping:
#   These violations require significant refactoring of core APIs.
#   Should be done first but requires careful review.
#
# Violations (2):
#   • javax-to-jakarta-001 (23 incidents)
#   • javax-to-jakarta-002 (18 incidents)
#
# Actions:
#   [a] Approve and continue
#   [d] Defer (skip this phase)
#   [v] View incident details
#   [q] Quit and save plan
#
# Choice:
```

## Configuration File

kantra-ai supports configuration files to avoid repetitive command-line flags. Create a `.kantra-ai.yaml` file in your project directory or home directory:

```yaml
# .kantra-ai.yaml
provider:
  name: claude
  model: claude-sonnet-4-20250514

paths:
  analysis: ./analysis/output.yaml
  input: ./src

limits:
  max-cost: 10.00
  max-effort: 5

filters:
  categories:
    - mandatory
    - optional

git:
  commit-strategy: per-violation
  create-pr: true

verification:
  enabled: true
  type: test
  strategy: at-end
```

**Configuration priority** (highest to lowest):
1. CLI flags (e.g., `--provider=openai`)
2. Configuration file in current directory (`./.kantra-ai.yaml`)
3. Configuration file in home directory (`~/.kantra-ai.yaml`)
4. Built-in defaults

See [.kantra-ai.example.yaml](./.kantra-ai.example.yaml) for a complete configuration example with all available options.

## Usage Examples

### Filtering Violations

```bash
# Only fix mandatory violations
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --categories=mandatory

# Only fix low-effort violations (effort <= 3)
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --max-effort=3

# Fix specific violations by ID
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --violation-ids=javax-to-jakarta-001,log4j-migration-002
```

### Git Commit Strategies

```bash
# One commit per violation type (groups related fixes)
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation

# One commit per file/incident
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-incident

# Single commit with all fixes
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end
```

### Build/Test Verification

```bash
# Run tests after fixes to ensure they don't break anything
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=test

# Run build verification only (faster than tests)
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=build

# Verify after each fix (slow but catches issues immediately)
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation \
  --verify=test \
  --verify-strategy=per-fix

# Custom verification command
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=test \
  --verify-command="make test"

# Continue on verification failures (don't stop at first failure)
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --verify=test \
  --verify-fail-fast=false
```

### GitHub Pull Request Creation

```bash
# Create a single PR with all fixes
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --create-pr

# Create separate PRs per violation type
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation \
  --create-pr

# Customize the branch name
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --create-pr \
  --branch=feature/konveyor-migration
```

## Command-Line Options

### `kantra-ai remediate` - Direct Remediation

| Flag | Description | Example |
|------|-------------|---------|
| `--analysis` | Path to Konveyor output.yaml (required) | `--analysis=./output.yaml` |
| `--input` | Path to source code directory (required) | `--input=./src` |
| `--provider` | AI provider: `claude` or `openai` (default: claude) | `--provider=openai` |
| `--model` | Specific model override (optional) | `--model=gpt-4` |
| `--dry-run` | Preview changes without applying them | `--dry-run` |
| `--max-cost` | Maximum spending limit in USD | `--max-cost=10.00` |
| `--max-effort` | Only fix violations with effort ≤ this value | `--max-effort=5` |
| `--categories` | Filter by category: `mandatory`, `optional`, `potential` | `--categories=mandatory` |
| `--violation-ids` | Comma-separated list of specific violation IDs | `--violation-ids=v001,v002` |
| `--git-commit` | Git commit strategy: `per-violation`, `per-incident`, `at-end` | `--git-commit=per-violation` |
| `--create-pr` | Create GitHub pull request(s) (requires `--git-commit`) | `--create-pr` |
| `--branch` | Custom branch name for PR (default: auto-generated) | `--branch=feature/fixes` |
| `--verify` | Run verification after fixes: `build`, `test` | `--verify=test` |
| `--verify-strategy` | When to verify: `per-fix`, `per-violation`, `at-end` (default: at-end) | `--verify-strategy=per-fix` |
| `--verify-command` | Custom verification command (overrides auto-detection) | `--verify-command="make test"` |
| `--verify-fail-fast` | Stop on first verification failure (default: true) | `--verify-fail-fast=false` |

### `kantra-ai plan` - Generate Migration Plan

| Flag | Description | Example |
|------|-------------|---------|
| `--analysis` | Path to Konveyor output.yaml (required) | `--analysis=./output.yaml` |
| `--input` | Path to source code directory (required) | `--input=./src` |
| `--provider` | AI provider: `claude` (OpenAI not yet supported for planning) | `--provider=claude` |
| `--model` | Specific model override (optional) | `--model=claude-opus-4-20250514` |
| `--output` | Output plan file path (default: .kantra-ai-plan.yaml) | `--output=my-plan.yaml` |
| `--max-phases` | Maximum number of phases (0 = auto, typically 3-5) | `--max-phases=5` |
| `--risk-tolerance` | Risk tolerance: `conservative`, `balanced`, `aggressive` | `--risk-tolerance=conservative` |
| `--categories` | Filter by category | `--categories=mandatory` |
| `--violation-ids` | Filter by specific violation IDs | `--violation-ids=v001,v002` |
| `--max-effort` | Maximum effort level filter | `--max-effort=5` |
| `--interactive` | Enable interactive phase approval | `--interactive` |

### `kantra-ai execute` - Execute Migration Plan

| Flag | Description | Example |
|------|-------------|---------|
| `--plan` | Path to plan file (default: .kantra-ai-plan.yaml) | `--plan=./my-plan.yaml` |
| `--state` | Path to state file (default: .kantra-ai-state.yaml) | `--state=./my-state.yaml` |
| `--input` | Path to source code directory (required) | `--input=./src` |
| `--provider` | AI provider: `claude` or `openai` | `--provider=claude` |
| `--model` | Specific model override (optional) | `--model=gpt-4` |
| `--phase` | Execute specific phase only (e.g., phase-1) | `--phase=phase-1` |
| `--resume` | Resume from last failure | `--resume` |
| `--dry-run` | Preview changes without applying them | `--dry-run` |
| `--git-commit` | Git commit strategy | `--git-commit=per-violation` |
| `--create-pr` | Create GitHub pull request(s) | `--create-pr` |
| `--branch` | Custom branch name for PR | `--branch=feature/fixes` |
| `--verify` | Run verification after fixes | `--verify=test` |
| `--verify-strategy` | When to verify | `--verify-strategy=at-end` |
| `--verify-command` | Custom verification command | `--verify-command="make test"` |
| `--verify-fail-fast` | Stop on first verification failure | `--verify-fail-fast=false` |

## Architecture

```
kantra-ai/
├── cmd/
│   └── kantra-ai/        # CLI entry point (remediate, plan, execute)
├── pkg/
│   ├── violation/        # Konveyor output.yaml parser
│   ├── provider/         # AI provider interface
│   │   ├── claude/       # Claude (Anthropic) implementation
│   │   └── openai/       # OpenAI implementation
│   ├── fixer/            # Code modification engine
│   ├── planner/          # AI-powered plan generation
│   ├── planfile/         # Plan and state YAML management
│   ├── executor/         # Plan execution with resume capability
│   ├── verifier/         # Build/test verification
│   └── gitutil/          # Git & GitHub integration
└── examples/             # Example violations and plans
```

## How It Works

1. **Parse Analysis**: Reads Konveyor's `output.yaml` to identify violations
2. **AI Processing**: Sends violation context to AI provider (Claude/OpenAI)
3. **Apply Fixes**: Applies AI-generated fixes to source files
4. **Verification** (optional): Runs tests or builds to ensure fixes don't break functionality
5. **Git Integration** (optional): Creates commits with meaningful messages
6. **PR Creation** (optional): Opens pull requests on GitHub with detailed summaries
7. **Reporting**: Provides detailed metrics on success rates, costs, and tokens used

## Batch Processing Performance

kantra-ai uses intelligent batch processing to dramatically reduce costs and execution time for large migrations:

**How It Works:**
- Groups similar violations together (same violation ID)
- Processes up to 10 incidents in a single API call
- Runs 4 batches in parallel by default
- Maintains incident-level tracking for resume capability

**Performance Benefits:**
- **50-80% cost reduction**: One API call for 10 violations instead of 10 separate calls
- **70-90% faster execution**: Parallel processing with 4 concurrent workers
- **Better AI context**: AI sees all related violations together for more consistent fixes

**Example:**
```
Without batching: 100 violations × $0.10 = $10.00, ~50 minutes
With batching:     100 violations ÷ 10 × $0.10 = $1.00, ~8 minutes
Savings:          $9.00 (90% cost reduction), 42 minutes (84% faster)
```

**Configuration:**
Batching is enabled by default. To customize:

```yaml
# .kantra-ai.yaml
batch:
  enabled: true        # Enable/disable batching
  max-batch-size: 10   # Max incidents per batch (1-10)
  parallelism: 4       # Concurrent batches (1-8)
```

Or via CLI flags:
```bash
./kantra-ai execute \
  --batch-size=10 \
  --batch-parallelism=4
```

**Note:** Batch processing is currently available for Claude provider only. OpenAI support coming soon.

## Cost Estimation

Typical costs per violation (using Claude Sonnet 4 with batch processing):
- **Simple fixes** (import changes, simple refactoring): $0.002 - $0.01
- **Medium complexity** (API migrations, pattern updates): $0.01 - $0.03
- **Complex fixes** (logic changes, multi-file): $0.03 - $0.10

Use `--dry-run` to get cost estimates before applying fixes.

## GitHub PR Creation

PRs created by kantra-ai include:
- **Detailed summaries** of violations fixed
- **File-by-file breakdown** with line numbers
- **Cost and token metrics** for transparency
- **AI provider information** for traceability
- **Professional formatting** with markdown

Example PR: [See PR template in TESTING.md](./docs/guides/PR-TESTING-GUIDE.md)

## Testing

See [TESTING.md](./docs/guides/TESTING.md) for comprehensive testing instructions, including:
- Setting up test environments
- Testing PR creation with real GitHub repositories
- Troubleshooting common issues

## Contributing

Contributions welcome! Please read [DESIGN.md](./docs/design/DESIGN.md) for architectural details and future plans.

## Roadmap

- [x] Core AI-powered remediation
- [x] Multiple AI provider support (Claude, OpenAI)
- [x] Git commit automation
- [x] GitHub PR creation
- [x] Build/test verification
- [x] Phased migration planning with AI grouping
- [x] Interactive phase approval mode
- [x] Resume capability with state tracking
- [x] Batch processing optimizations (50-80% cost reduction, 70-90% faster)
- [ ] Additional AI providers (Gemini, etc.)
- [ ] Batch processing for OpenAI provider
- [ ] Integration with Konveyor CLI

## License

Apache 2.0

---

**Note**: kantra-ai is an independent tool and is not officially part of the Konveyor project. It's designed to complement Konveyor's analysis capabilities with automated remediation.
