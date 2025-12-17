# Confidence Threshold Filtering

kantra-ai supports confidence-based filtering that automatically skips low-confidence fixes based on migration complexity, providing an extra layer of safety for automated remediation.

## Overview

Confidence filtering helps ensure that only high-quality, reliable fixes are automatically applied to your codebase. The AI provider returns a confidence score (0.0-1.0) for each fix, and kantra-ai can use this score combined with migration complexity to determine whether to apply the fix.

**Key Benefits:**
- **Safety**: Prevent low-quality fixes from being applied
- **Complexity-aware**: Different thresholds for different complexity levels
- **Flexible**: Multiple actions when confidence is low (skip, warn, manual review)
- **Integration**: Uses Konveyor's migration complexity metadata

---

## How It Works

kantra-ai uses AI confidence scores (0.0-1.0) combined with Konveyor's migration complexity levels to determine whether to apply a fix.

### Complexity Levels

| Complexity | Expected AI Success | Default Threshold | Description |
|------------|---------------------|-------------------|-------------|
| **trivial** | 95%+ | 0.70 | Mechanical find/replace (e.g., package renames) |
| **low** | 80%+ | 0.75 | Straightforward API equivalents |
| **medium** | 60%+ | 0.80 | Requires context understanding |
| **high** | 30-50% | 0.90 | Architectural changes, manual review recommended |
| **expert** | <30% | 0.95 | Domain expertise required, manual review required |

### Migration Complexity Sources

Complexity is determined from:

1. **Ruleset metadata** (preferred): Konveyor rulesets can include a `migration_complexity` field
2. **Effort-based fallback**: If metadata is missing, kantra-ai maps effort levels (0-10) to complexity:
   - Effort 0-2 → trivial
   - Effort 3-4 → low
   - Effort 5-6 → medium
   - Effort 7-8 → high
   - Effort 9-10 → expert

---

## Configuration

### Enable via Config File

Add to `.kantra-ai.yaml`:

```yaml
confidence:
  enabled: true              # Enable confidence filtering
  on-low-confidence: skip    # skip, warn-and-apply, or manual-review-file

  # Optional: Override default thresholds
  complexity-thresholds:
    trivial: 0.70  # Accept fixes with 70%+ confidence
    low: 0.75      # Accept fixes with 75%+ confidence
    medium: 0.80   # Accept fixes with 80%+ confidence
    high: 0.90     # Require very high confidence for complex changes
    expert: 0.95   # Require near-perfect confidence for expert-level changes
```

### Enable via CLI Flags

**Basic usage** - Skip low-confidence fixes:

```bash
./kantra-ai remediate \
  --enable-confidence \
  --on-low-confidence=skip
```

**Global minimum confidence** - Set a floor across all complexity levels:

```bash
./kantra-ai remediate \
  --enable-confidence \
  --min-confidence=0.85
```

**Custom thresholds** - Override specific complexity levels:

```bash
./kantra-ai remediate \
  --enable-confidence \
  --complexity-threshold="high=0.95,expert=0.98"
```

---

## Actions on Low Confidence

When a fix has confidence below the threshold, kantra-ai can take different actions:

### Skip (Recommended)

Safest option - skip the fix entirely and report why:

```yaml
confidence:
  enabled: true
  on-low-confidence: skip
```

**Output:**
```
  • [2/23] src/ComplexServlet.java:42
  ⚠ Skipped: src/ComplexServlet.java
    Reason: Confidence 0.65 below threshold 0.90 (high complexity)
    To force: --min-confidence=0.65 or --enable-confidence=false
```

### Warn and Apply

Show a warning but apply the fix anyway:

```yaml
confidence:
  enabled: true
  on-low-confidence: warn-and-apply
```

**Output:**
```
  • [2/23] src/ComplexServlet.java:42
  ⚠ Warning: Low confidence fix applied (0.65 < 0.90 threshold)
  ✓ Fixed: src/ComplexServlet.java (cost: $0.12, confidence: 0.65)
```

### Manual Review File

Write low-confidence fixes to a file for manual review:

```yaml
confidence:
  enabled: true
  on-low-confidence: manual-review-file
  manual-review-path: .kantra-ai-manual-review.yaml
```

**Output:**
```
  • [2/23] src/ComplexServlet.java:42
  ⚠ Skipped: Added to manual review file
    Review: .kantra-ai-manual-review.yaml
```

The file contains:
```yaml
- file: src/ComplexServlet.java
  line: 42
  violation_id: javax-to-jakarta-001
  confidence: 0.65
  threshold: 0.90
  complexity: high
  proposed_fix: |
    // AI-generated fix content here
  explanation: "Replaced javax.servlet with jakarta.servlet"
```

---

## Example Scenarios

### Scenario 1: Production Migration (Maximum Safety)

Enable confidence filtering with skip action:

```bash
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --input=src \
  --provider=claude \
  --enable-confidence \
  --on-low-confidence=skip \
  --verify=test
```

**Result:**
- Only high-confidence fixes applied
- Low-confidence fixes skipped and reported
- Tests verify all changes work

### Scenario 2: Rapid Prototyping (Maximum Automation)

Disable confidence filtering to apply all fixes:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=claude
```

**Result:**
- All fixes applied regardless of confidence
- Fastest execution
- May require more manual cleanup

### Scenario 3: High-Risk Codebase (Extra Caution)

Increase thresholds for complex changes:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=claude \
  --enable-confidence \
  --complexity-threshold="medium=0.85,high=0.95,expert=0.98"
```

**Result:**
- Trivial/low fixes: normal thresholds
- Medium fixes: 85%+ confidence required
- High/expert fixes: 95%+ confidence required

### Scenario 4: Review Workflow (Collect Low-Confidence Fixes)

Use manual review file to collect fixes for human review:

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=src \
  --provider=claude \
  --enable-confidence \
  --on-low-confidence=manual-review-file
```

**Result:**
- High-confidence fixes applied automatically
- Low-confidence fixes written to review file
- Developer reviews and applies manually

---

## Phased Migration Integration

Confidence filtering works seamlessly with phased migrations:

### Plan Generation

High/expert complexity violations are automatically flagged for manual review in plans:

```bash
./kantra-ai plan \
  --analysis=output.yaml \
  --input=src \
  --provider=claude
```

**Output (plan.yaml):**
```yaml
phases:
  - id: phase-1
    name: "Critical Mandatory Fixes - High Effort"
    risk: high
    violations:
      - id: complex-refactoring-001
        complexity: high
        manual_review_recommended: true  # Flagged for review
```

### Plan Execution

Enable confidence filtering during execution:

```bash
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --input=src \
  --provider=claude \
  --enable-confidence \
  --on-low-confidence=manual-review-file
```

**Result:**
- High-confidence fixes in high-risk phases applied automatically
- Low-confidence fixes collected for manual review
- Best of both worlds: automation + safety

---

## Best Practices

### 1. Start Conservative

Begin with strict settings and loosen as needed:

```yaml
confidence:
  enabled: true
  on-low-confidence: skip
  complexity-thresholds:
    high: 0.95
    expert: 0.98
```

### 2. Use with Verification

Combine confidence filtering with testing:

```bash
./kantra-ai remediate \
  --enable-confidence \
  --on-low-confidence=skip \
  --verify=test \
  --verify-fail-fast=false
```

### 3. Review Skipped Fixes

Examine what was skipped to understand AI limitations:

```bash
# Check logs for skipped fixes
grep "Skipped:" kantra-ai.log

# Or use manual review file
./kantra-ai remediate \
  --enable-confidence \
  --on-low-confidence=manual-review-file
cat .kantra-ai-manual-review.yaml
```

### 4. Adjust Thresholds Based on Results

If too many fixes are skipped:
- Lower thresholds slightly
- Review if rulesets have accurate complexity metadata

If too many poor fixes applied:
- Increase thresholds
- Enable confidence filtering if not already enabled

### 5. Different Settings for Different Phases

Use stricter settings for critical code:

```bash
# Phase 1: Critical code (high standards)
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --phase=phase-1 \
  --enable-confidence \
  --min-confidence=0.90

# Phase 2: Non-critical code (more lenient)
./kantra-ai execute \
  --plan=.kantra-ai-plan/plan.yaml \
  --phase=phase-2 \
  --enable-confidence \
  --min-confidence=0.75
```

---

## Troubleshooting

### Too Many Fixes Being Skipped

**Problem:** Most fixes are being skipped as low-confidence.

**Solutions:**
1. Check if complexity metadata is accurate in rulesets
2. Lower thresholds slightly:
   ```bash
   --complexity-threshold="high=0.85,expert=0.90"
   ```
3. Use `warn-and-apply` to see what would be skipped:
   ```bash
   --on-low-confidence=warn-and-apply --dry-run
   ```

### Low-Quality Fixes Still Applied

**Problem:** Fixes with low confidence are being applied.

**Solutions:**
1. Enable confidence filtering:
   ```bash
   --enable-confidence
   ```
2. Increase thresholds:
   ```bash
   --min-confidence=0.85
   ```
3. Use stricter complexity thresholds:
   ```bash
   --complexity-threshold="medium=0.85,high=0.95,expert=0.98"
   ```

### Unclear Why a Fix Was Skipped

**Problem:** Want to see the proposed fix even though it was skipped.

**Solution:** Use manual review file:
```bash
--on-low-confidence=manual-review-file
cat .kantra-ai-manual-review.yaml
```

---

## Default Behavior

**Important:** Confidence filtering is **disabled by default** for backward compatibility.

To enable, you must explicitly set `--enable-confidence` or add it to your config file.

**Default settings when enabled:**
```yaml
confidence:
  enabled: false  # Must explicitly enable
  on-low-confidence: skip
  complexity-thresholds:
    trivial: 0.70
    low: 0.75
    medium: 0.80
    high: 0.90
    expert: 0.95
```

---

## See Also

- [Usage Examples](./USAGE_EXAMPLES.md) - Examples using confidence filtering
- [CLI Reference](./CLI_REFERENCE.md) - Complete flag reference
- [Quick Start](./QUICKSTART.md) - Getting started guide
