# kantra-ai

[![Tests](https://github.com/tsanders-rh/kantra-ai/actions/workflows/test.yml/badge.svg)](https://github.com/tsanders-rh/kantra-ai/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/tsanders-rh/kantra-ai/branch/main/graph/badge.svg)](https://codecov.io/gh/tsanders-rh/kantra-ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsanders-rh/kantra-ai)](https://goreportcard.com/report/github.com/tsanders-rh/kantra-ai)

AI-powered automated remediation for [Konveyor](https://www.konveyor.io/) violations. Use Claude or OpenAI to automatically fix code issues identified during application modernization and migration.

## Features

- **Automated Code Fixes**: AI analyzes violations and applies fixes directly to your source code
- **Multiple AI Providers**: Support for Claude (Anthropic) and OpenAI with easy provider switching
- **Smart Filtering**: Filter by violation category, effort level, or specific violation IDs
- **Git Integration**: Automatic commit creation with configurable strategies (per-violation, per-incident, or batch)
- **GitHub PR Automation**: Automatically create pull requests with detailed fix summaries
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

## Architecture

```
kantra-ai/
├── cmd/
│   └── kantra-ai/        # CLI entry point
├── pkg/
│   ├── violation/        # Konveyor output.yaml parser
│   ├── provider/         # AI provider interface
│   │   ├── claude/       # Claude (Anthropic) implementation
│   │   └── openai/       # OpenAI implementation
│   ├── fixer/            # Code modification engine
│   └── gitutil/          # Git & GitHub integration
└── examples/             # Example violations and test cases
```

## How It Works

1. **Parse Analysis**: Reads Konveyor's `output.yaml` to identify violations
2. **AI Processing**: Sends violation context to AI provider (Claude/OpenAI)
3. **Apply Fixes**: Applies AI-generated fixes to source files
4. **Git Integration** (optional): Creates commits with meaningful messages
5. **PR Creation** (optional): Opens pull requests on GitHub with detailed summaries
6. **Reporting**: Provides detailed metrics on success rates, costs, and tokens used

## Cost Estimation

Typical costs per violation (using Claude Sonnet 3.5):
- **Simple fixes** (import changes, simple refactoring): $0.01 - $0.05
- **Medium complexity** (API migrations, pattern updates): $0.05 - $0.15
- **Complex fixes** (logic changes, multi-file): $0.15 - $0.50

Use `--dry-run` to get cost estimates before applying fixes.

## GitHub PR Creation

PRs created by kantra-ai include:
- **Detailed summaries** of violations fixed
- **File-by-file breakdown** with line numbers
- **Cost and token metrics** for transparency
- **AI provider information** for traceability
- **Professional formatting** with markdown

Example PR: [See PR template in TESTING.md](./PR-TESTING-GUIDE.md)

## Testing

See [TESTING.md](./TESTING.md) for comprehensive testing instructions, including:
- Setting up test environments
- Testing PR creation with real GitHub repositories
- Troubleshooting common issues

## Contributing

Contributions welcome! Please read [DESIGN.md](./DESIGN.md) for architectural details and future plans.

## Roadmap

- [x] Core AI-powered remediation
- [x] Multiple AI provider support (Claude, OpenAI)
- [x] Git commit automation
- [x] GitHub PR creation
- [ ] Additional AI providers (Gemini, etc.)
- [ ] Build/test verification
- [ ] Interactive fix review mode
- [ ] Batch processing optimizations
- [ ] Integration with Konveyor CLI

## License

Apache 2.0

---

**Note**: kantra-ai is an independent tool and is not officially part of the Konveyor project. It's designed to complement Konveyor's analysis capabilities with automated remediation.
