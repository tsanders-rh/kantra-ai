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
