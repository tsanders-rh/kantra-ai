# Batch Processing Optimization Design

## Goals

- Reduce API costs by 50-80% for large migrations
- Reduce execution time by 70-90% through parallelism
- Maintain incident-level state tracking for resume capability
- Keep backward compatibility with existing sequential processing

## Batching Strategies

### 1. Violation-Level Batching (Phase 1 - Implement First)

Group incidents with the **same violation ID** together:

```
Violation: javax-to-jakarta-servlet (23 incidents)
  → Batch 1: incidents 1-10
  → Batch 2: incidents 11-20
  → Batch 3: incidents 21-23

Instead of: 23 API calls
Now:        3 API calls
```

**Benefits**:
- Simplest to implement
- Biggest cost/time savings
- AI sees related patterns together
- Easy to maintain state tracking

**Constraints**:
- Max batch size: 10 incidents (stay under token limits)
- Same violation ID only (predictable pattern)

### 2. File-Level Batching (Phase 2 - Future)

Group all violations in the **same file** together:

```
File: UserServlet.java
  → Violation 1: javax → jakarta (line 3)
  → Violation 2: deprecated API (line 45)
  → One AI call: "Fix all violations in this file"
```

**Benefits**:
- Better file context for AI
- Fewer file reads/writes
- More atomic changes

**Challenges**:
- Different violation types in same prompt
- Harder to track which fix worked
- More complex state tracking

### 3. Parallel Execution (Phase 1 - Implement First)

Process multiple batches concurrently:

```
Workers: 4 parallel goroutines
Batches: 20 total

Instead of: 20 batches × 2 seconds = 40 seconds
Now:        20 batches ÷ 4 workers × 2 seconds = 10 seconds
```

## Architecture

### Provider Interface Extension

```go
// pkg/provider/interface.go

// BatchRequest contains multiple incidents to fix together
type BatchRequest struct {
    Violation   violation.Violation  // Shared violation context
    Incidents   []violation.Incident // Multiple incidents to fix
    FileContent map[string]string    // file path → content
    Language    string               // Shared language
}

// BatchResponse contains fixes for multiple incidents
type BatchResponse struct {
    Fixes      []IncidentFix // One per incident
    Success    bool          // Overall success
    TokensUsed int
    Cost       float64
    Error      error
}

// IncidentFix represents a single fix within a batch
type IncidentFix struct {
    IncidentURI  string  // Which incident this fixes
    Success      bool
    FixedContent string  // Fixed file content
    Explanation  string
    Error        error
}

// Provider interface (extended)
type Provider interface {
    Name() string
    FixViolation(ctx context.Context, req FixRequest) (*FixResponse, error)
    EstimateCost(req FixRequest) (float64, error)
    GeneratePlan(ctx context.Context, req PlanRequest) (*PlanResponse, error)

    // NEW: Batch fixing
    FixBatch(ctx context.Context, req BatchRequest) (*BatchResponse, error)
}
```

### Batch Fixer

```go
// pkg/fixer/batch.go

type BatchFixer struct {
    provider    provider.Provider
    inputDir    string
    dryRun      bool
    maxBatchSize int  // Default: 10
    parallelism  int  // Default: 4
}

func NewBatchFixer(provider provider.Provider, inputDir string, dryRun bool) *BatchFixer

func (bf *BatchFixer) FixViolations(ctx context.Context, violations []violation.Violation) ([]FixResult, error)

// Internal methods
func (bf *BatchFixer) groupByViolation(violations []violation.Violation) map[string][]violation.Incident
func (bf *BatchFixer) createBatches(incidents []violation.Incident, maxSize int) [][]violation.Incident
func (bf *BatchFixer) processBatchesParallel(ctx context.Context, batches []Batch) []FixResult
```

### Configuration

```go
// pkg/executor/types.go

type Config struct {
    // ... existing fields ...

    // Batch processing options
    EnableBatching bool // Enable batch processing (default: true)
    MaxBatchSize   int  // Max incidents per batch (default: 10)
    Parallelism    int  // Concurrent batches (default: 4)
}
```

### CLI Flags

```bash
kantra-ai execute \
  --input=./app \
  --batch-size=10 \          # Max incidents per batch
  --parallelism=4 \          # Concurrent workers
  --disable-batching         # Fall back to sequential
```

## Implementation Phases

### Phase 1: Core Batching (Days 1-3)

1. **Extend Provider Interface**
   - Add BatchRequest, BatchResponse, IncidentFix types
   - Add FixBatch() method to Provider interface
   - Update MockProvider for tests

2. **Implement Claude Batch Processing**
   - pkg/provider/claude/batch.go
   - Build batch prompt with multiple incidents
   - Parse batch response (JSON array of fixes)
   - Handle partial failures

3. **Create BatchFixer**
   - pkg/fixer/batch.go
   - Group incidents by violation ID
   - Split into batches (max size 10)
   - Process batches with parallelism
   - Maintain incident-level results

4. **Update Executor**
   - Check if batching enabled in config
   - Use BatchFixer instead of Fixer when enabled
   - State tracking still works (incident-level results)

### Phase 2: Testing & Validation (Day 4)

5. **Unit Tests**
   - Test batch grouping logic
   - Test parallel execution
   - Test partial failure handling
   - Test state tracking with batches

6. **Integration Tests**
   - End-to-end with real violations
   - Compare results: batch vs sequential
   - Verify cost/time savings

7. **Performance Benchmarks**
   - Benchmark batch sizes (1, 5, 10, 20)
   - Benchmark parallelism (1, 2, 4, 8)
   - Measure cost/time improvements

### Phase 3: Documentation (Day 5)

8. **Update Documentation**
   - Add batching section to README
   - Update command-line options
   - Add performance comparison examples
   - Document when to enable/disable

## Detailed Design: Claude Batch Implementation

### Batch Prompt Structure

```
You are fixing multiple occurrences of the same violation in a codebase.

VIOLATION: javax-to-jakarta-servlet
DESCRIPTION: Replace javax.servlet with jakarta.servlet

Fix the following incidents:

INCIDENT 1:
File: src/main/java/UserServlet.java
Line: 3
Code context:
```java
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
```

INCIDENT 2:
File: src/main/java/LoginFilter.java
Line: 5
Code context:
```java
import javax.servlet.Filter;
```

For each incident, provide:
1. The fixed code
2. Brief explanation of the change

OUTPUT FORMAT (JSON):
[
  {
    "incident_uri": "file:///src/main/java/UserServlet.java:3",
    "success": true,
    "fixed_content": "...",
    "explanation": "..."
  },
  ...
]
```

### Response Parsing

```go
type ClaudeBatchResponse struct {
    Fixes []struct {
        IncidentURI  string `json:"incident_uri"`
        Success      bool   `json:"success"`
        FixedContent string `json:"fixed_content"`
        Explanation  string `json:"explanation"`
    } `json:"fixes"`
}

func parseBatchResponse(responseText string) ([]IncidentFix, error) {
    // Extract JSON from markdown code blocks
    jsonData := extractJSON(responseText)

    var response ClaudeBatchResponse
    if err := json.Unmarshal(jsonData, &response); err != nil {
        return nil, err
    }

    // Convert to IncidentFix structs
    fixes := make([]IncidentFix, len(response.Fixes))
    for i, f := range response.Fixes {
        fixes[i] = IncidentFix{
            IncidentURI:  f.IncidentURI,
            Success:      f.Success,
            FixedContent: f.FixedContent,
            Explanation:  f.Explanation,
        }
    }

    return fixes, nil
}
```

## Error Handling

### Partial Batch Failures

```go
// If 8 out of 10 fixes succeed:
batchResponse := &BatchResponse{
    Fixes: []IncidentFix{
        {IncidentURI: "file:///a.java:1", Success: true, ...},
        {IncidentURI: "file:///b.java:2", Success: true, ...},
        // ... 6 more successes
        {IncidentURI: "file:///i.java:9", Success: false, Error: ...},
        {IncidentURI: "file:///j.java:10", Success: false, Error: ...},
    },
    Success: false,  // Overall failure if ANY incident failed
}

// State tracking records:
// - 8 completed incidents
// - 2 failed incidents
// - Can resume and retry just the 2 failures
```

### Retry Strategy

```go
// If batch fails entirely (parsing error, API timeout):
// 1. Fall back to sequential processing for that batch
// 2. Process incidents one-by-one
// 3. State tracking continues to work
```

## Token Limit Management

```go
const (
    MaxTokensPerBatch     = 8000  // Claude input limit
    EstimatedTokensPerIncident = 500  // Conservative estimate
    MaxIncidentsPerBatch  = 10    // 500 * 10 = 5000 tokens (safe margin)
)

func estimateBatchTokens(incidents []violation.Incident) int {
    total := 0
    for _, inc := range incidents {
        // Estimate: prompt template + file context + incident details
        total += len(inc.Message) + len(inc.FileContext) + 300
    }
    return total / 4 // Rough token estimate (4 chars per token)
}

func createSafeBatch(incidents []violation.Incident, maxTokens int) [][]violation.Incident {
    batches := [][]violation.Incident{}
    currentBatch := []violation.Incident{}
    currentTokens := 0

    for _, inc := range incidents {
        incTokens := estimateBatchTokens([]violation.Incident{inc})

        if currentTokens + incTokens > maxTokens {
            batches = append(batches, currentBatch)
            currentBatch = []violation.Incident{inc}
            currentTokens = incTokens
        } else {
            currentBatch = append(currentBatch, inc)
            currentTokens += incTokens
        }
    }

    if len(currentBatch) > 0 {
        batches = append(batches, currentBatch)
    }

    return batches
}
```

## Performance Characteristics

### Expected Improvements

**Scenario: 100 incidents, same violation type**

| Metric | Sequential | Batched (10/batch, 4 workers) | Improvement |
|--------|-----------|-------------------------------|-------------|
| API Calls | 100 | 10 | 90% reduction |
| Execution Time | 200 seconds | 5 seconds | 97.5% faster |
| Cost | $5.00 | $1.50 | 70% cheaper |

**Why batching is cheaper:**
- Shared prompt overhead (violation description sent once)
- Shared context (AI understands pattern once)
- Reduced API overhead (fewer requests)

**Why parallelism helps:**
- Network latency overlap
- CPU utilization (waiting for API = idle time)
- Better throughput

### When Batching Doesn't Help

- **Single incident**: No batching benefit
- **Diverse violations**: Each unique violation needs its own pattern
- **Small files**: File I/O dominates over API time
- **Rate limiting**: API provider limits concurrent requests

## Backward Compatibility

```go
// Old code still works (uses sequential processing):
fixer := fixer.New(provider, inputPath, dryRun)
result, err := fixer.FixIncident(ctx, violation, incident)

// New code opts into batching:
config := executor.Config{
    EnableBatching: true,  // New field, defaults to true
    MaxBatchSize:   10,
    Parallelism:    4,
}
```

## Configuration Defaults

```go
const (
    DefaultEnableBatching = true   // On by default
    DefaultMaxBatchSize   = 10     // Conservative for token limits
    DefaultParallelism    = 4      // Balance throughput and rate limits
)
```

## Monitoring & Metrics

```go
type BatchMetrics struct {
    TotalBatches      int
    TotalIncidents    int
    SuccessfulBatches int
    FailedBatches     int
    AverageBatchSize  float64
    AverageTime       time.Duration
    TotalCost         float64
}

// Log at end of execution:
// Batch Metrics:
//   Batches:  10
//   Incidents: 95
//   Success rate: 95%
//   Avg batch size: 9.5
//   Total cost: $1.45 (70% savings vs sequential)
```

## Future Enhancements

1. **Adaptive Batch Sizing**
   - Start with max size
   - Reduce if failures occur
   - Increase if success rate high

2. **Smart Grouping**
   - ML-based similarity detection
   - Group beyond violation ID
   - Context-aware batching

3. **Caching**
   - Cache fix patterns
   - Apply cached patterns without API call
   - 90%+ cost reduction for repeated patterns

4. **File-Level Batching**
   - All violations in one file → one API call
   - Better file context
   - More atomic changes

## Success Criteria

- ✅ 50-80% cost reduction for migrations with 20+ incidents
- ✅ 70-90% execution time reduction with parallelism
- ✅ All tests pass (including new batch tests)
- ✅ State tracking works correctly (resume capability intact)
- ✅ Backward compatible (sequential mode still works)
- ✅ Documentation updated
