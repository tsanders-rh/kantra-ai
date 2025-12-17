# PR Creation Manual Testing Guide

This guide walks you through testing the PR creation feature end-to-end with a real GitHub repository.

## Prerequisites

### 1. GitHub Token
Create a personal access token with `repo` scope:

```bash
# 1. Go to: https://github.com/settings/tokens
# 2. Click "Generate new token (classic)"
# 3. Grant 'repo' scope (full repository access)
# 4. Copy the token and export it:

export GITHUB_TOKEN=ghp_your_token_here
```

### 2. AI Provider API Key
Make sure you have either Claude or OpenAI set up:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
# OR
export OPENAI_API_KEY=sk-...
```

## Quick Start (Automated Setup)

### Step 1: Run Setup Script

```bash
./test-pr-creation.sh
```

This will:
- Check your environment variables
- Build kantra-ai
- Create a test directory with a git repo
- Copy the javax-to-jakarta test case

### Step 2: Create GitHub Test Repository

The script will pause and ask you to:

1. **Create a new repository on GitHub:**
   - Go to https://github.com/new
   - Name: `kantra-ai-pr-test` (or any name)
   - Visibility: Public or Private (your choice)
   - **Do NOT** initialize with README, .gitignore, or license

2. **Add the remote and push:**
   ```bash
   cd test-pr-TIMESTAMP  # The directory created by the script
   git remote add origin https://github.com/YOUR_USERNAME/kantra-ai-pr-test.git
   git branch -M main
   git push -u origin main
   ```

### Step 3: Run the Test

```bash
cd ..  # Back to kantra-ai root
./test-pr-creation.sh run test-pr-TIMESTAMP
```

This will:
- Run kantra-ai with PR creation enabled
- Apply the fixes
- Create commits
- Push to GitHub
- Create a pull request

## What to Verify

After the test runs, check the GitHub repository:

### 1. Pull Request Created
Go to: `https://github.com/YOUR_USERNAME/kantra-ai-pr-test/pulls`

**Verify:**
- ✓ PR exists
- ✓ PR is open (not draft)
- ✓ Branch name: `kantra-ai/remediation-javax-to-jakarta-001-TIMESTAMP`
- ✓ Title: `fix: Konveyor violation javax-to-jakarta-001`

### 2. PR Content
Open the PR and check:

**Summary section:**
- ✓ Violation ID: javax-to-jakarta-001
- ✓ Category and effort displayed
- ✓ Description present

**Changes section:**
- ✓ Shows number of incidents fixed
- ✓ Lists files modified with line numbers
- ✓ Example: `src/UserServlet.java:3, 4, 5, 6, 7`

**AI Remediation Details:**
- ✓ Provider name (claude or openai)
- ✓ Total cost shown (e.g., $0.0234)
- ✓ Total tokens shown (e.g., 1,245)

**Footer:**
- ✓ Link to kantra-ai repository

### 3. Inline Comments (Low-Confidence Fixes)
If using `--pr-comment-threshold`, check for inline review comments:

**Verify:**
- ✓ Comments appear on specific lines with low-confidence fixes
- ✓ Comment includes warning emoji (⚠️) and confidence percentage
- ✓ Comment explains the violation and requests careful review
- ✓ Comments only appear for fixes below the threshold

### 4. Code Changes
Click on "Files changed" tab:

**Verify:**
- ✓ All `javax.servlet` imports → `jakarta.servlet`
- ✓ Changes are syntactically correct
- ✓ No unrelated changes

### 5. Branch and Commits
Check the branch:

**Verify:**
- ✓ Branch exists on remote
- ✓ Commits have proper messages
- ✓ Original branch (main) unchanged

## Manual Testing (No Script)

If you prefer to test manually:

### 1. Create Test Repository

```bash
# Create directory
mkdir pr-test
cd pr-test

# Initialize git
git init
git config user.name "Your Name"
git config user.email "your.email@example.com"

# Copy test files
cp ../examples/javax-to-jakarta/src/UserServlet.java .
cp ../examples/javax-to-jakarta/output.yaml .

# Initial commit
git add .
git commit -m "Initial commit"

# Add GitHub remote (create repo on GitHub first)
git remote add origin https://github.com/YOUR_USERNAME/test-repo.git
git push -u origin main
```

### 2. Run kantra-ai with PR Creation

```bash
cd ..  # Back to kantra-ai root

./kantra-ai remediate \
  --analysis=pr-test/output.yaml \
  --input=pr-test \
  --provider=claude \
  --git-commit=per-violation \
  --create-pr
```

## Testing Different Strategies

### Per-Violation Strategy (Default)
One PR per violation type (groups all incidents of same violation):

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation \
  --create-pr
```

**Expected:** 1 PR with all javax-to-jakarta fixes

### Per-Incident Strategy
One PR per file/incident:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-incident \
  --create-pr
```

**Expected:** Multiple PRs (one for each import line fix)

### At-End Strategy
Single PR with all fixes:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=at-end \
  --create-pr
```

**Expected:** 1 PR with all violations combined

### With Inline Comments for Low-Confidence Fixes
Add inline review comments for fixes with confidence below 0.8 (80%):

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --git-commit=per-violation \
  --create-pr \
  --pr-comment-threshold=0.8
```

**Expected:**
- PR created as normal
- Inline comments appear on lines where fixes have confidence < 0.8
- Comments include warning emoji, confidence %, violation info, and review guidance
- High-confidence fixes (≥ 0.8) have no comments

**Note:** Set to 0 to disable inline comments (default behavior)

## Common Issues to Watch For

### Issue 1: Branch Already Exists
**Error:** `A pull request already exists for this branch`

**Fix:** Use different branch name or delete existing branch:
```bash
git push origin --delete old-branch-name
```

### Issue 2: Push Permission Denied
**Error:** `Permission denied (publickey)` or `403 Forbidden`

**Causes:**
- Wrong GitHub token
- Token lacks `repo` scope
- Remote URL uses SSH but no SSH key configured

**Fix:** Use HTTPS remote and ensure token has repo scope

### Issue 3: No Commits to Create PR
**Error:** `No commits between main and branch`

**Cause:** Fixes weren't applied or committed

**Debug:**
```bash
git log --oneline -5  # Check commits
git diff main..branch  # Check what changed
```

### Issue 4: API Rate Limit
**Error:** `API rate limit exceeded`

**Fix:** Wait an hour or use authenticated requests (already done)

## Cleanup After Testing

1. **Close/Delete PRs:**
   - Go to PR page
   - Click "Close pull request" or "Delete"

2. **Delete Branches:**
   ```bash
   git push origin --delete branch-name
   ```

3. **Delete Test Repository (optional):**
   - Go to repo Settings → Danger Zone → Delete repository

4. **Delete Local Test Directory:**
   ```bash
   rm -rf test-pr-TIMESTAMP
   ```

## What We're Looking For

During this manual test, we want to identify:

1. **Bugs:**
   - Does it create PRs successfully?
   - Are branch names valid?
   - Do pushes work?
   - Any errors or crashes?

2. **UX Issues:**
   - Are error messages clear?
   - Is progress visible?
   - What happens on failures?

3. **Edge Cases:**
   - What if branch exists?
   - What if PR already exists?
   - What if push fails?

4. **PR Quality:**
   - Is PR content helpful?
   - Are code changes correct?
   - Is metadata accurate?

## Reporting Issues

If you find bugs, note:
- Exact command run
- Full error output
- Git status before/after
- GitHub repo state
- Expected vs actual behavior

Let's use this information to fix bugs and improve error messages!
