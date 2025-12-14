# Contributing to kantra-ai

## Current Phase: Validation

We're in the **validation phase** - proving that AI can successfully fix Konveyor violations.

**What we need right now:**
1. Test results from real violations
2. Success rate data across different violation types
3. Cost data from different AI providers
4. Feedback on what works and what doesn't

## How to Contribute

### 1. Run Validation Tests

The most valuable contribution right now is **testing and data collection**.

```bash
# Test on your Konveyor violations
./kantra-ai remediate \
  --analysis=./your-analysis/output.yaml \
  --input=./your-app \
  --provider=claude \
  --max-cost=5.00

# Document results in VALIDATION.md
```

**What to track:**
- Which violation types worked well?
- Which failed?
- What was the cost?
- How accurate were the fixes?

### 2. Report Issues

Found a bug or unexpected behavior? [Open an issue](https://github.com/yourusername/kantra-ai/issues)

Include:
- kantra-ai version
- AI provider used
- Violation type
- Error message or unexpected behavior
- Example violation (if possible)

### 3. Improve Prompting

AI quality depends heavily on prompts. See:
- `pkg/provider/claude/claude.go` - `buildPrompt()`
- `pkg/provider/openai/openai.go` - `buildPrompt()`

Try different prompting strategies and share results.

### 4. Add Test Cases

More examples help validate the tool:

```bash
# Add to examples/
examples/
  your-violation-type/
    output.yaml          # Violation definition
    src/                 # Original code
    expected/            # Expected fix
    README.md            # Test description
```

### 5. Code Contributions

For the MVP, we're keeping it minimal. Focus on:
- Bug fixes
- Better error handling
- Improved prompts
- Provider implementations
- Test coverage

**Not yet:**
- Git integration (post-validation)
- Advanced features (post-validation)
- UI improvements (post-validation)

## Development Setup

```bash
# Clone and setup
git clone https://github.com/yourusername/kantra-ai
cd kantra-ai
./scripts/setup.sh

# Make changes
# ... edit code ...

# Test
make build
make run-example

# Format
make fmt

# Submit PR
git checkout -b my-fix
git commit -am "Fix: description"
git push origin my-fix
```

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Keep functions focused and small
- Add comments for non-obvious logic
- Update documentation for user-facing changes

## Testing

```bash
# Run tests
make test

# Test with examples
make run-example
make verify-example
```

## Documentation

Update documentation when changing behavior:
- README.md - User-facing changes
- QUICKSTART.md - Usage examples
- VALIDATION.md - Test results
- Code comments - Implementation details

## Questions?

Open an issue or discussion on GitHub.

## After Validation

Once we validate that AI can successfully fix violations (>60% success rate), we'll expand scope to include:
- Git workflow
- Advanced features
- Production polish

At that point, we'll update contribution guidelines for the next phase.

## License

By contributing, you agree that your contributions will be licensed under Apache 2.0.
