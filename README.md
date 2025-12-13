# kantra-ai

[![Tests](https://github.com/tsanders-rh/kantra-ai/actions/workflows/test.yml/badge.svg)](https://github.com/tsanders-rh/kantra-ai/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/tsanders-rh/kantra-ai/branch/main/graph/badge.svg)](https://codecov.io/gh/tsanders-rh/kantra-ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/tsanders-rh/kantra-ai)](https://goreportcard.com/report/github.com/tsanders-rh/kantra-ai)

AI-powered remediation for Konveyor violations.

## 🎯 Current Phase: Validation (Week 1-2)

**Goal:** Answer the critical question: *"Can AI actually fix Konveyor violations well enough to be useful?"*

### Success Criteria
- [ ] Test on 20-30 real violations from actual projects
- [ ] Achieve >60% fix success rate (stretch: >75%)
- [ ] Average cost < $0.15 per violation
- [ ] Fixes are syntactically valid
- [ ] Document what works and what doesn't

### What This MVP Does
- Reads Konveyor `output.yaml` files
- Sends violations to AI provider (Claude or OpenAI)
- Applies fixes to source files
- Tracks success/failure and costs
- Creates git commits (optional, with configurable strategies)

## 🚀 Quick Start

### Prerequisites
```bash
# Required
go 1.21+
export ANTHROPIC_API_KEY=sk-ant-...  # or OPENAI_API_KEY

# For testing
kantra analyze --input=./your-app --output=./analysis
```

### Installation
```bash
git clone https://github.com/yourusername/kantra-ai
cd kantra-ai
go build -o kantra-ai ./cmd/kantra-ai
```

### Usage
```bash
# Fix all violations (dry-run first!)
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --dry-run

# Actually apply fixes
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --max-cost=5.00

# Fix specific violations for testing
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --violation-ids=violation-001,violation-002

# Auto-commit fixes (one commit per violation)
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --git-commit=per-violation

# Auto-commit fixes (one commit per file)
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --git-commit=per-incident

# Auto-commit fixes (single batch commit at end)
./kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --git-commit=at-end
```

## 📊 Validation Tracking

See [VALIDATION.md](./VALIDATION.md) for test results and metrics.

## 🏗️ Architecture (MVP)

```
cmd/
  kantra-ai/          # CLI entry point
pkg/
  violation/          # Parse output.yaml
  provider/           # AI provider interface
    claude/           # Claude implementation
    openai/           # OpenAI implementation
  fixer/              # Apply fixes to files
  gitutil/            # Git commit integration
examples/             # Test violations
validation/           # Test results tracking
```

## 🔮 Future Plans

After validation succeeds:
- ✅ Git workflow (commits per violation/incident/batch)
- PR creation automation
- Cost controls and limits
- Multiple providers and comparison
- Verification (syntax check, build, tests)
- Better error handling
- Eventual integration into kantra

See [DESIGN.md](./DESIGN.md) for full vision.

## 📝 License

Apache 2.0
