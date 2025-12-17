# GitHub Status Checks for Verification

kantra-ai can report verification results (build/test) as GitHub Status Checks that appear directly in pull request UIs.

## Overview

When verification is enabled (`--verify=build` or `--verify=test`) and a GitHub token is provided, kantra-ai automatically reports verification results as commit statuses. These appear in the PR interface and can be configured as required checks for merging.

## Benefits

1. **Visible in PR UI**: Verification results appear alongside other CI checks
2. **Merge Requirements**: Can be configured as required checks to prevent merging failing changes
3. **Status Tracking**: See verification status (pending/success/failure) without checking logs
4. **Integration**: Works seamlessly with existing GitHub workflows

## How It Works

### Status States

kantra-ai reports four possible states:

- **pending** - Verification is running
- **success** - Verification passed
- **failure** - Verification failed (tests/build failed)
- **error** - Verification encountered an error (e.g., timeout, missing dependencies)

### Status Contexts

Status checks appear with these contexts:

- `kantra-ai/verify-build` - For build verification (`--verify=build`)
- `kantra-ai/verify-test` - For test verification (`--verify=test`)

## Usage

### Basic Usage

```bash
# Enable verification with PR creation
kantra-ai execute \
  --create-pr \
  --verify=test

# Status checks are automatically reported when:
# 1. --create-pr is enabled (provides GitHub token)
# 2. --verify is enabled
# 3. Verification runs
```

### Configuration

Status check reporting is **automatic** when both conditions are met:
1. GitHub token is available (via `--create-pr` or `GITHUB_TOKEN` environment variable)
2. Verification is enabled (`--verify=build` or `--verify=test`)

No additional configuration is required.

### Verification Strategies

Status checks work with all verification strategies:

#### Per-Fix Strategy
```bash
kantra-ai execute \
  --verify=build \
  --verify-strategy=per-fix \
  --create-pr
```
- Reports status after each individual fix
- Useful for catching issues immediately
- More API calls to GitHub

#### Per-Violation Strategy
```bash
kantra-ai execute \
  --verify=test \
  --verify-strategy=per-violation \
  --create-pr
```
- Reports status after all fixes for each violation
- Balances verification frequency with API usage

#### At-End Strategy (Default)
```bash
kantra-ai execute \
  --verify=test \
  --verify-strategy=at-end \
  --create-pr
```
- Reports status once after all fixes are applied
- Most efficient for API usage
- Best for batch remediation

## GitHub PR Integration

### Setting Up Required Checks

To require verification before merging:

1. Go to repository **Settings** → **Branches**
2. Edit branch protection rules for your main branch
3. Enable "Require status checks to pass before merging"
4. Search for and select:
   - `kantra-ai/verify-build` (if using `--verify=build`)
   - `kantra-ai/verify-test` (if using `--verify=test`)
5. Save changes

Now PRs created by kantra-ai won't be mergeable until verification passes.

### Example PR Status Display

When kantra-ai creates a PR with verification enabled, you'll see:

```
kantra-ai/verify-build
  ✓ Build verification passed (2.3s)

kantra-ai/verify-test
  ✓ Test verification passed (15.2s)
```

If verification fails:

```
kantra-ai/verify-test
  ✗ Test verification failed
```

## Troubleshooting

### Status checks not appearing

**Cause**: GitHub token may not have required permissions

**Solution**: Ensure your GitHub token has `repo:status` scope:
```bash
# Check token permissions
gh auth status

# Refresh token if needed
gh auth refresh -s repo
```

### "Warning: failed to report status to GitHub"

**Cause**: API error or network issue

**Impact**: Verification still runs, but status isn't reported to GitHub

**Solution**:
- Check GitHub token is valid
- Verify network connectivity
- Check GitHub API status: https://githubstatus.com

### Status check shows "error" state

**Cause**: Verification encountered an unexpected error

**Solution**:
- Check console output for error details
- Verify project has required build files (go.mod, pom.xml, etc.)
- Check `--verify-command` if using custom verification

## Advanced Configuration

### Custom Verification Command

When using custom verification commands, status checks still work:

```bash
kantra-ai execute \
  --verify=build \
  --verify-command="make test-and-build" \
  --create-pr
```

The status will be reported as `kantra-ai/verify-build` regardless of the custom command.

### Disabling Status Checks

Status checks are automatically enabled when both verification and PR creation are enabled. To disable them:

- Don't use `--create-pr` (verification will still run locally)
- Or don't use `--verify` (no verification, no status checks)

There is currently no flag to explicitly disable status checks while keeping PR creation and verification enabled.

## Implementation Details

### API Usage

- Uses GitHub [Commit Status API](https://docs.github.com/en/rest/commits/statuses)
- Creates status on HEAD commit of PR branch
- Automatically retries transient failures (503, 502, 504)
- Warnings logged if status reporting fails (doesn't block verification)

### Status Check Lifecycle

1. **Before verification runs**: Creates `pending` status
2. **After verification succeeds**: Updates to `success` status with duration
3. **After verification fails**: Updates to `failure` status
4. **On verification error**: Updates to `error` status

### Performance Impact

- Status API calls are asynchronous
- Failures to report status don't block execution
- Minimal overhead (< 500ms per verification)

## Examples

### Example 1: Simple PR with Build Verification

```bash
kantra-ai execute \
  --input=/path/to/java/project \
  --verify=build \
  --create-pr \
  --pr-strategy=per-violation
```

**Result**: Each violation gets its own PR with build verification status check

### Example 2: Batch PR with Test Verification

```bash
kantra-ai execute \
  --input=/path/to/go/project \
  --verify=test \
  --verify-strategy=at-end \
  --create-pr \
  --pr-strategy=at-end
```

**Result**: Single PR with all fixes, one test verification status check reported after all fixes applied

### Example 3: Fail-Fast Mode

```bash
kantra-ai execute \
  --input=/path/to/project \
  --verify=test \
  --verify-fail-fast \
  --create-pr
```

**Result**: If verification fails, execution stops and PR shows failed status check

## Related Documentation

- [Verification Documentation](./VERIFICATION.md)
- [Pull Request Documentation](./PULL_REQUESTS.md)
- [GitHub Commit Status API](https://docs.github.com/en/rest/commits/statuses)
- [GitHub Branch Protection](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
