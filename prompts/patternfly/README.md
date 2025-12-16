# PatternFly 5 → 6 Migration Prompts

AI-powered prompts for migrating React applications from PatternFly 5 to PatternFly 6 using kantra-ai.

## Overview

PatternFly 6 introduces significant breaking changes:
- **New design token system**: `--pf-v5-*` → `--pf-t--*`
- **Component API changes**: Modal, Button, Table, and more
- **Breakpoint changes**: Pixels → Rems
- **Deprecated components**: Moved to `/deprecated/` paths
- **Logical properties**: Block/inline instead of top/right/bottom/left

## Prompts Included

### 1. `pf5-to-pf6-typescript.txt`
**For**: TypeScript (.ts, .tsx) files
**Handles**:
- Design token updates (CSS variables and React tokens)
- Component import changes
- Component prop migrations
- Deprecated component handling
- TypeScript type updates

**Best for**: Single-fix mode on TypeScript/TSX files

### 2. `pf5-to-pf6-batch.txt`
**For**: Batch processing similar violations
**Handles**:
- Multiple similar design token changes
- Consistent component updates across files
- Bulk import updates

**Best for**: Batch mode when you have many similar violations (e.g., 50+ design token renames)

### 3. `pf5-to-pf6-css.txt`
**For**: CSS/SCSS (.css, .scss) files
**Handles**:
- CSS variable migrations (`--pf-v5-*` → `--pf-t--*`)
- Logical property conversions
- Breakpoint value updates
- Custom CSS override updates

**Best for**: Single-fix mode on CSS/SCSS files

## Configuration

### Option 1: Language-Specific Templates (Recommended)

Add to your `.kantra-ai.yaml`:

```yaml
prompts:
  # Base template for fallback
  single-fix-template: ./prompts/patternfly/pf5-to-pf6-typescript.txt
  batch-fix-template: ./prompts/patternfly/pf5-to-pf6-batch.txt

  # Language-specific overrides
  language-templates:
    typescript:
      single-fix: ./prompts/patternfly/pf5-to-pf6-typescript.txt
      batch-fix: ./prompts/patternfly/pf5-to-pf6-batch.txt

    tsx:
      single-fix: ./prompts/patternfly/pf5-to-pf6-typescript.txt
      batch-fix: ./prompts/patternfly/pf5-to-pf6-batch.txt

    javascript:
      single-fix: ./prompts/patternfly/pf5-to-pf6-typescript.txt
      batch-fix: ./prompts/patternfly/pf5-to-pf6-batch.txt

    jsx:
      single-fix: ./prompts/patternfly/pf5-to-pf6-typescript.txt
      batch-fix: ./prompts/patternfly/pf5-to-pf6-batch.txt

    css:
      single-fix: ./prompts/patternfly/pf5-to-pf6-css.txt

    scss:
      single-fix: ./prompts/patternfly/pf5-to-pf6-css.txt
```

### Option 2: Global Templates (Simpler)

```yaml
prompts:
  single-fix-template: ./prompts/patternfly/pf5-to-pf6-typescript.txt
  batch-fix-template: ./prompts/patternfly/pf5-to-pf6-batch.txt
```

## Usage Examples

### Direct Remediation (Small Projects)

```bash
# Dry-run to preview changes
kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./src \
  --provider=claude \
  --dry-run

# Apply fixes
kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./src \
  --provider=claude \
  --enable-confidence \
  --verify=build
```

### Phased Migration (Large Projects)

```bash
# Step 1: Generate migration plan
kantra-ai plan \
  --analysis=./analysis/output.yaml \
  --input=./src \
  --provider=claude

# Step 2: Review .kantra-ai-plan.html in browser

# Step 3: Execute plan
kantra-ai execute \
  --plan=.kantra-ai-plan.yaml \
  --input=./src \
  --provider=claude \
  --enable-confidence \
  --verify=build \
  --git-commit=per-violation
```

### Filter by Complexity

```bash
# Start with easy wins (design token renames)
kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./src \
  --max-effort=3 \
  --enable-confidence

# Then handle medium complexity (component changes)
kantra-ai remediate \
  --analysis=./analysis/output.yaml \
  --input=./src \
  --categories=mandatory \
  --enable-confidence \
  --on-low-confidence=manual-review-file
```

## Expected Results

### High Success Rate (95%+)
- Design token renames (`--pf-v5-*` → `--pf-t--*`)
- React token updates (`global_FontSize_lg` → `t_global_font_size_lg`)
- Simple import updates
- Straightforward component prop changes

### Medium Success Rate (75-85%)
- Component migrations (Modal, Table, etc.)
- Deprecated component handling
- CSS override updates
- Logical property conversions

### Manual Review Recommended (<75%)
- Custom component wrappers around PatternFly
- Complex CSS overrides
- Breakpoint logic requiring pixel → rem conversion
- Components with extensive customization

## Confidence Filtering

Use confidence filtering to automatically skip uncertain fixes:

```yaml
# .kantra-ai.yaml
confidence:
  enabled: true
  on-low-confidence: skip  # or 'manual-review-file'

  # Higher thresholds for PatternFly due to breaking changes
  complexity-thresholds:
    trivial: 0.85   # Design token renames
    low: 0.85       # Simple component updates
    medium: 0.90    # Component migrations
    high: 0.95      # Complex custom code
    expert: 0.98    # Extensive customization
```

## Common Violations

Based on PatternFly 6 codemods, expect these violation types:

1. **Design Token Updates** (Most Common)
   - CSS variables: `--pf-v5-*` → `--pf-t--*`
   - React tokens: `*_*_*` → `t_*_*_*`
   - Estimated: 500-1000+ occurrences in medium apps

2. **Component Import Changes**
   - Deprecated components → `/deprecated/` path
   - New component implementations
   - Estimated: 50-200 occurrences

3. **Component Prop Changes**
   - Button `isDisabled` behavior
   - Modal implementation updates
   - Table expansion props
   - Estimated: 20-100 occurrences

4. **CSS/SCSS Overrides**
   - Custom styling using PF variables
   - Logical property conversions
   - Estimated: 100-500 occurrences

## Cost Estimates

Using Claude Sonnet 4 with batch processing:

| Violation Type | Count | Est. Cost | Est. Time |
|----------------|-------|-----------|-----------|
| Design tokens (batch) | 500 | $0.50-$1.50 | 5-10 min |
| Component updates | 100 | $2.00-$5.00 | 10-15 min |
| CSS overrides | 200 | $1.00-$3.00 | 8-12 min |
| **Total** | **800** | **$3.50-$9.50** | **23-37 min** |

*Actual costs vary based on file size and complexity*

## Testing

After migration, run PatternFly's official codemods to verify:

```bash
# Install PF codemods
npm install -g @patternfly/pf-codemods

# Run codemods to check for missed updates
npx @patternfly/pf-codemods ./src --v6

# Review any remaining violations
```

## Troubleshooting

### Low Confidence Scores

If you're seeing low confidence scores (<0.70):
1. Check that prompts are loaded correctly
2. Verify file language detection is correct
3. Review complex custom code that may need manual work
4. Consider using `--on-low-confidence=manual-review-file`

### Incorrect Fixes

If fixes are wrong:
1. Review the specific violation type in Konveyor analysis
2. Check if the component has special customization
3. Refine prompts based on your codebase patterns
4. Use `--dry-run` to preview before applying

### Missing Fixes

If some violations aren't fixed:
1. Check filters (categories, effort levels)
2. Verify violation IDs in plan
3. Look for errors in execution logs
4. Some violations may be flagged for manual review

## Resources

**Official PatternFly 6 Documentation**:
- [Upgrade Guide](https://www.patternfly.org/get-started/upgrade/)
- [Design Tokens](https://www.patternfly.org/tokens/about-tokens/)
- [Release Highlights](https://www.patternfly.org/get-started/release-highlights/)
- [PatternFly Codemods](https://github.com/patternfly/pf-codemods)

**kantra-ai Documentation**:
- [Prompt Customization Guide](../../docs/guides/PROMPT_CUSTOMIZATION.md)
- [Workflow Documentation](../../docs/WORKFLOW.md)
- [Main README](../../README.md)

## Contributing

Found patterns that work well for PatternFly migrations?
- Add them to these prompts
- Share examples in GitHub discussions
- Create PRs with improvements

## Questions?

- **Technical issues**: Open a GitHub issue
- **PatternFly specific**: Check [PatternFly docs](https://www.patternfly.org)
- **Migration help**: PatternFly community forums
