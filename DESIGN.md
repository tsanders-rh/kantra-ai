# kantra-ai Design

This document outlines the full vision for kantra-ai. The current MVP is focused on validation only.

## Full Design Document

See the comprehensive design document: [AI Batch Remediation Design](./docs/FULL_DESIGN.md)

## Current MVP Scope

**Phase 0: Validation (Weeks 1-2)**

The current implementation is intentionally minimal to validate the core hypothesis:
> "Can AI successfully fix Konveyor violations at reasonable cost and quality?"

**What's implemented:**
- ✅ Parse Konveyor output.yaml
- ✅ Send violations to AI (Claude or OpenAI)
- ✅ Apply fixes to files
- ✅ Track costs and success rate
- ✅ Basic filtering (by ID, category, effort)

**What's NOT implemented (yet):**
- ❌ Git commits
- ❌ PR creation
- ❌ Advanced cost controls
- ❌ Verification (build/test)
- ❌ Resume capability
- ❌ Multiple commit strategies
- ❌ Kai solution server integration
- ❌ Ollama local models
- ❌ Progress UI

## MVP Architecture

```
kantra-ai (MVP)
│
├── cmd/kantra-ai/           # CLI entry point
│   └── main.go              # Cobra command structure
│
├── pkg/
│   ├── violation/           # Parse output.yaml
│   │   ├── types.go         # Violation data structures
│   │   └── parser.go        # YAML parsing
│   │
│   ├── provider/            # AI provider abstraction
│   │   ├── interface.go     # Provider interface
│   │   ├── claude/          # Claude (Anthropic) implementation
│   │   └── openai/          # OpenAI implementation
│   │
│   └── fixer/               # Apply fixes to files
│       └── fixer.go         # File modification logic
│
└── examples/                # Test cases
    └── javax-to-jakarta/    # Simple migration test
```

## Evolution Plan

### After Validation Succeeds (Week 3+)

**Phase 1: Git Integration**
- Commit per violation
- Branch creation
- Basic PR creation

**Phase 2: Safety & Quality**
- Syntax validation
- Build verification
- Test verification
- Rollback on failure

**Phase 3: Advanced Features**
- Kai solution server integration
- Cost controls and limits
- Resume capability
- Better error handling

**Phase 4: Production Polish**
- Progress UI
- Better reporting
- Configuration files
- Documentation

### Integration with kantra

**Option A: Merge into kantra**
```bash
# Becomes a subcommand
kantra analyze ...
kantra remediate --ai ...
```

**Option B: Stay separate**
```bash
# kubectl-style plugin pattern
kantra analyze ...
kantra-ai remediate ...
```

Decision will be made based on:
- Validation results
- Team feedback
- User preferences
- Maintenance considerations

## Key Design Principles

1. **Konveyor's Value = Analysis Quality**
   - Konveyor produces structured, AI-ready violations
   - This is the hard part that makes everything else possible

2. **Pluggable AI Providers**
   - No vendor lock-in
   - Users choose their preferred AI
   - Competition improves quality

3. **Git-Native Workflow**
   - All changes are commits
   - Easy review in GitHub/GitLab
   - Cherry-pick what you want

4. **Safety First**
   - Always reviewable
   - Easy to rollback
   - Cost controls
   - Verification options

5. **Validation Before Investment**
   - Prove it works before building everything
   - Real metrics, not assumptions
   - Fast iteration

## Success Criteria

**For MVP (Phase 0):**
- ✅ >60% fix success rate (stretch: >75%)
- ✅ Average cost <$0.15 per violation
- ✅ Fixes are syntactically valid
- ✅ Clear data on what works vs. what doesn't

**For Production:**
- >80% success rate on common migrations
- <$0.10 per violation on average
- User adoption and positive feedback
- Integration with Konveyor ecosystem

## References

- [Full Design Document](./docs/FULL_DESIGN.md)
- [Validation Results](./VALIDATION.md)
- [Konveyor Project](https://konveyor.io)
- [kantra](https://github.com/konveyor/kantra)
