# Performance Optimization Roadmap

## Completed Optimizations ✅

### 1. File-Based Batching (Implemented)
**Impact**: 10-20% token reduction
**Status**: ✅ Shipped

Groups incidents by file before batching, ensuring file content is sent once per file instead of redundantly across batches.

- Implementation: `pkg/fixer/batch.go`
- Default: Enabled via `GroupByFile: true` in `DefaultBatchConfig()`
- Automatically detects when file grouping provides benefits

### 2. Increased Parallelism (Implemented)
**Impact**: 2x speedup potential
**Status**: ✅ Shipped

Increased default parallel workers from 4 to 8 for better CPU utilization on modern systems.

- Implementation: `pkg/fixer/batch.go`
- Default: `Parallelism: 8` in `DefaultBatchConfig()`
- Automatically scales down if fewer batches available

### 3. Incremental Processing (Implemented)
**Impact**: 100% time/cost savings on already-fixed incidents
**Status**: ✅ Shipped

Automatically skips incidents that have already been successfully fixed, enabling:
- Resume interrupted runs without redoing work
- Iterative workflows with incremental fixes
- Zero cost for re-running on partially fixed codebases

- Implementation: `pkg/planfile/state.go`, `pkg/executor/executor.go`
- CLI Flag: `--resume`
- Default: Enabled when state file exists
- Tracks completion at incident-level granularity
- Reports skipped incidents in execution summary

**Combined Impact**: ~2-3x overall speedup with 10-20% cost savings, plus 100% savings on resume

---

## Future Optimizations (Deferred)

### 4. Prompt Caching
**Impact**: 20-30% additional cost reduction
**Status**: 🔄 Deferred - Waiting for stable Anthropic SDK

#### Why Deferred:
- Current Anthropic SDK (v0.2.0-alpha.4) lacks stable prompt caching support
- SDK v1.19.0 has caching but introduces breaking API changes
- Would require rewriting ~500+ lines of Claude provider code
- Current optimizations already provide substantial performance gains

#### Implementation Plan (When Ready):
1. **Upgrade Anthropic SDK** to stable v1.x with prompt caching support
2. **Update all Claude provider code** for new SDK API:
   - Remove `anthropic.F()` wrappers
   - Update `MessageNewParams` structure
   - Migrate to new system prompt API
   - Handle new Usage field structure
3. **Implement Caching Strategy**:
   - Cache static system instructions (unchanged across requests)
   - Cache violation context (same for all incidents of one violation)
   - Keep incident details uncached (varies per batch)
4. **Update Cost Calculations**:
   - Cache writes: $3.75/1M (25% premium over base)
   - Cache reads: $0.30/1M (90% discount from base $3/1M)
   - Regular input: $3/1M
   - Output: $15/1M

#### Technical Details:
```go
// System blocks with cache control (future implementation)
systemBlocks := []anthropic.TextBlockParam{
    // Static instructions - cached across all requests
    {
        Text: staticInstructions,
        Type: anthropic.F(anthropic.TextBlockParamTypeText),
    },
    // Violation context - cached for same violation
    {
        Text:         violationContext,
        Type:         anthropic.F(anthropic.TextBlockParamTypeText),
        CacheControl: anthropic.F(anthropic.NewCacheControlEphemeralParam()),
    },
}
```

#### References:
- [Anthropic Prompt Caching Docs](https://platform.claude.com/docs/en/build-with-claude/prompt-caching)
- [Prompt Caching Announcement](https://www.anthropic.com/news/prompt-caching)
- [Token-Saving Updates](https://www.anthropic.com/news/token-saving-updates)

---

## Other Future Optimization Opportunities

### 5. Token Estimation & Pre-batching
**Impact**: Better batch consistency, prevent context limit errors
**Complexity**: Medium

Currently batches by count (10 incidents max). Could batch by estimated token count for more consistent API usage.

### 6. Response Streaming
**Impact**: Better UX for large batches
**Complexity**: Low

Show fixes as they arrive instead of waiting for complete batch. Anthropic SDK supports streaming.

### 7. Concurrent File I/O
**Impact**: Minor improvement for large codebases
**Complexity**: Low

Pre-read files for upcoming batches in parallel. Currently reads sequentially per batch.

---

## Performance Metrics Tracking

To measure optimization effectiveness:

1. **Token Usage**: Monitor `cache_read_input_tokens` and `cache_creation_input_tokens` when caching is implemented
2. **Batch Efficiency**: Track file grouping metrics (already logged)
3. **Throughput**: Measure incidents processed per minute
4. **Cost Per Incident**: Track average cost per fixed incident

## Next Steps

1. Monitor SDK stability - check for v1.x stable release
2. Evaluate if additional optimizations are needed based on user feedback
3. Consider prompt caching implementation when SDK is stable
