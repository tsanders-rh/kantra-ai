# Command-Line Reference

Complete reference for all kantra-ai commands and flags.

## Commands

kantra-ai provides three main commands:

- **`remediate`** - Direct remediation for quick fixes
- **`plan`** - Generate migration plan with AI-powered grouping
- **`execute`** - Execute a previously generated plan

---

## `kantra-ai remediate`

Direct remediation for small migrations with < 20 violations.

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--analysis` | Path to Konveyor output.yaml | `--analysis=./output.yaml` |
| `--input` | Path to source code directory | `--input=./src` |

### Provider Options

| Flag | Description | Example |
|------|-------------|---------|
| `--provider` | AI provider: `claude`, `openai`, `groq`, `ollama`, `together`, `anyscale`, `perplexity`, `openrouter`, `lmstudio` (default: claude) | `--provider=openai` |
| `--model` | Specific model override (optional) | `--model=gpt-4` |

### Filtering Options

| Flag | Description | Example |
|------|-------------|---------|
| `--categories` | Filter by category: `mandatory`, `optional`, `potential` | `--categories=mandatory` |
| `--max-effort` | Only fix violations with effort ≤ this value | `--max-effort=5` |
| `--violation-ids` | Comma-separated list of specific violation IDs | `--violation-ids=v001,v002` |

### Cost Controls

| Flag | Description | Example |
|------|-------------|---------|
| `--max-cost` | Maximum spending limit in USD | `--max-cost=10.00` |
| `--dry-run` | Preview changes without applying them | `--dry-run` |

### Git Integration

| Flag | Description | Example |
|------|-------------|---------|
| `--git-commit` | Git commit strategy: `per-violation`, `per-incident`, `at-end` | `--git-commit=per-violation` |
| `--create-pr` | Create GitHub pull request(s) (requires `--git-commit`) | `--create-pr` |
| `--branch` | Custom branch name for PR (default: auto-generated) | `--branch=feature/fixes` |

### Verification Options

| Flag | Description | Example |
|------|-------------|---------|
| `--verify` | Run verification after fixes: `build`, `test` | `--verify=test` |
| `--verify-strategy` | When to verify: `per-fix`, `per-violation`, `at-end` (default: at-end) | `--verify-strategy=per-fix` |
| `--verify-command` | Custom verification command (overrides auto-detection) | `--verify-command="make test"` |
| `--verify-fail-fast` | Stop on first verification failure (default: true) | `--verify-fail-fast=false` |

### Confidence Filtering

| Flag | Description | Example |
|------|-------------|---------|
| `--enable-confidence` | Enable confidence-based filtering | `--enable-confidence` |
| `--min-confidence` | Global minimum confidence threshold (0.0-1.0) | `--min-confidence=0.85` |
| `--on-low-confidence` | Action for low confidence: `skip`, `warn-and-apply`, `manual-review-file` | `--on-low-confidence=skip` |
| `--complexity-threshold` | Custom thresholds per complexity level | `--complexity-threshold="high=0.95,expert=0.98"` |

### Batch Processing

| Flag | Description | Example |
|------|-------------|---------|
| `--batch-size` | Max incidents per batch (1-10, default: 10) | `--batch-size=10` |
| `--batch-parallelism` | Concurrent batches (1-8, default: 4) | `--batch-parallelism=4` |

---

## `kantra-ai plan`

Generate AI-powered migration plan with risk assessment and phase grouping.

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--analysis` | Path to Konveyor output.yaml | `--analysis=./output.yaml` |
| `--input` | Path to source code directory | `--input=./src` |

### Provider Options

| Flag | Description | Example |
|------|-------------|---------|
| `--provider` | AI provider (currently only `claude` supported for planning) | `--provider=claude` |
| `--model` | Specific model override (optional) | `--model=claude-opus-4-20250514` |

### Plan Configuration

| Flag | Description | Example |
|------|-------------|---------|
| `--output` | Output directory path (default: .kantra-ai-plan) | `--output=my-plan-dir` |
| `--max-phases` | Maximum number of phases (0 = auto, typically 3-5) | `--max-phases=5` |
| `--risk-tolerance` | Risk tolerance: `conservative`, `balanced`, `aggressive` | `--risk-tolerance=conservative` |

### Filtering Options

| Flag | Description | Example |
|------|-------------|---------|
| `--categories` | Filter by category | `--categories=mandatory` |
| `--violation-ids` | Filter by specific violation IDs | `--violation-ids=v001,v002` |
| `--max-effort` | Maximum effort level filter | `--max-effort=5` |

### Interactive Modes

| Flag | Description | Example |
|------|-------------|---------|
| `--interactive` | Enable CLI-based phase approval | `--interactive` |
| `--interactive-web` | Launch web-based interactive planner | `--interactive-web` |
| `--port` | Port for web interface (default: 8080) | `--port=3000` |

---

## `kantra-ai execute`

Execute a previously generated migration plan.

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--plan` | Path to plan file (default: .kantra-ai-plan/plan.yaml) | `--plan=./my-plan.yaml` |
| `--input` | Path to source code directory | `--input=./src` |

### Provider Options

| Flag | Description | Example |
|------|-------------|---------|
| `--provider` | AI provider: `claude`, `openai`, etc. | `--provider=claude` |
| `--model` | Specific model override (optional) | `--model=gpt-4` |

### Execution Options

| Flag | Description | Example |
|------|-------------|---------|
| `--phase` | Execute specific phase only (e.g., phase-1) | `--phase=phase-1` |
| `--resume` | Resume from last failure | `--resume` |
| `--state` | Path to state file (default: .kantra-ai-state.yaml) | `--state=./my-state.yaml` |
| `--dry-run` | Preview changes without applying them | `--dry-run` |

### Git Integration

| Flag | Description | Example |
|------|-------------|---------|
| `--git-commit` | Git commit strategy | `--git-commit=per-violation` |
| `--create-pr` | Create GitHub pull request(s) | `--create-pr` |
| `--branch` | Custom branch name for PR | `--branch=feature/fixes` |

### Verification Options

| Flag | Description | Example |
|------|-------------|---------|
| `--verify` | Run verification after fixes | `--verify=test` |
| `--verify-strategy` | When to verify | `--verify-strategy=at-end` |
| `--verify-command` | Custom verification command | `--verify-command="make test"` |
| `--verify-fail-fast` | Stop on first verification failure | `--verify-fail-fast=false` |

### Batch Processing

| Flag | Description | Example |
|------|-------------|---------|
| `--batch-size` | Max incidents per batch (1-10, default: 10) | `--batch-size=10` |
| `--batch-parallelism` | Concurrent batches (1-8, default: 4) | `--batch-parallelism=4` |

---

## Environment Variables

kantra-ai uses environment variables for sensitive configuration:

### AI Provider API Keys

```bash
# Claude (Anthropic)
export ANTHROPIC_API_KEY=sk-ant-...

# OpenAI
export OPENAI_API_KEY=sk-...

# Groq
export OPENAI_API_KEY=gsk_...

# Together AI, Anyscale, Perplexity, OpenRouter
export OPENAI_API_KEY=...

# GitHub (for PR creation)
export GITHUB_TOKEN=ghp_...
```

### Configuration File Locations

```bash
# Override default config file location
export KANTRA_AI_CONFIG=/path/to/.kantra-ai.yaml
```

---

## Configuration Priority

kantra-ai uses the following priority order (highest to lowest):

1. **CLI flags** - Explicit command-line arguments
2. **Environment variables** - API keys and config paths
3. **Project config file** - `./.kantra-ai.yaml` in current directory
4. **User config file** - `~/.kantra-ai.yaml` in home directory
5. **Built-in defaults** - Hardcoded fallback values

---

## Exit Codes

kantra-ai uses standard exit codes:

- **0** - Success
- **1** - General error (validation, file I/O, etc.)
- **2** - API error (rate limits, authentication, etc.)
- **3** - Verification failure (tests/build failed)

---

## See Also

- [Usage Examples](./USAGE_EXAMPLES.md) - Common usage patterns
- [Quick Start](./QUICKSTART.md) - Getting started guide
- [Configuration](../../.kantra-ai.example.yaml) - Full configuration file example
