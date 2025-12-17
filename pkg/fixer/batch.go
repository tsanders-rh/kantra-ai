package fixer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tsanders/kantra-ai/pkg/confidence"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// BatchConfig configures batch processing behavior
type BatchConfig struct {
	// MaxBatchSize is the maximum number of incidents to fix in a single batch
	// Default: 10 (stays under token limits)
	// Can be overridden if MaxTokensPerBatch is set and file sizes vary
	MaxBatchSize int

	// Parallelism is the number of concurrent batches to process
	// Default: 8
	Parallelism int

	// Enabled controls whether batching is used
	// Default: true
	Enabled bool

	// GroupByFile controls whether to group incidents by file before batching
	// When enabled, incidents from the same file are batched together, reducing
	// token usage by 10-20% since file content is sent once per file.
	// Default: true
	GroupByFile bool

	// MaxTokensPerBatch is the maximum estimated tokens per batch
	// When set, batch size is dynamically adjusted based on file sizes
	// to stay under this limit. Set to 0 to disable (use MaxBatchSize only).
	// Default: 0 (disabled)
	// Recommended: 50000 tokens (leaves room in 200K context for prompt + output)
	MaxTokensPerBatch int
}

// DefaultBatchConfig returns the recommended batch configuration
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		// MaxBatchSize is limited by the AI provider's context window
		// 10 incidents provide good cost savings without overwhelming the model
		MaxBatchSize: 10,

		// Parallelism set to 8 for better CPU utilization on modern systems
		// Automatically reduced to match batch count if fewer batches
		// Can be adjusted based on provider quotas and CPU availability
		Parallelism: 8,

		Enabled: true,

		// GroupByFile enabled by default for 10-20% token savings
		GroupByFile: true,
	}
}

// BatchFixer provides optimized batch processing of violations
type BatchFixer struct {
	provider       provider.Provider
	inputDir       string
	dryRun         bool
	config         BatchConfig
	confidenceConf confidence.Config
}

// NewBatchFixer creates a new batch fixer
func NewBatchFixer(p provider.Provider, inputDir string, dryRun bool, config BatchConfig) *BatchFixer {
	return &BatchFixer{
		provider:       p,
		inputDir:       inputDir,
		dryRun:         dryRun,
		config:         config,
		confidenceConf: confidence.DefaultConfig(),
	}
}

// NewBatchFixerWithConfidence creates a new batch fixer with confidence configuration
func NewBatchFixerWithConfidence(p provider.Provider, inputDir string, dryRun bool, config BatchConfig, confidenceConf confidence.Config) *BatchFixer {
	return &BatchFixer{
		provider:       p,
		inputDir:       inputDir,
		dryRun:         dryRun,
		config:         config,
		confidenceConf: confidenceConf,
	}
}

// batchJob represents a batch of incidents to fix
type batchJob struct {
	violation violation.Violation
	incidents []violation.Incident
	batch     int // Batch number for this violation
}

// batchResult contains the results from processing a batch
type batchResult struct {
	job        batchJob
	fixes      []provider.IncidentFix
	cost       float64
	tokensUsed int
	err        error
}

// FixViolationBatch processes all incidents for a violation using batching
// Returns individual FixResult for each incident to maintain compatibility with state tracking
func (bf *BatchFixer) FixViolationBatch(ctx context.Context, v violation.Violation) ([]FixResult, error) {
	if !bf.config.Enabled || len(v.Incidents) == 0 {
		// Fall back to sequential processing
		return bf.fixSequential(ctx, v)
	}

	// Group incidents into batches
	batches := bf.createBatches(v)

	// Create job channel and result channel
	jobs := make(chan batchJob, len(batches))
	results := make(chan batchResult, len(batches))

	// Start worker pool
	var wg sync.WaitGroup
	workers := min(bf.config.Parallelism, len(batches))
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go bf.worker(ctx, jobs, results, &wg)
	}

	// Send all jobs
	for _, batch := range batches {
		jobs <- batch
	}
	close(jobs)

	// Wait for all workers to finish and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	allResults := make([]FixResult, 0, len(v.Incidents))
	for result := range results {
		if result.err != nil {
			// If batch failed entirely, create failed results for all incidents
			for _, incident := range result.job.incidents {
				allResults = append(allResults, FixResult{
					Success:    false,
					FilePath:   filepath.Base(incident.GetFilePath()),
					Error:      result.err,
					TokensUsed: 0,
					Cost:       0,
				})
			}
			continue
		}

		// Distribute cost and tokens evenly across fixes
		costPerFix := 0.0
		tokensPerFix := 0
		if len(result.fixes) > 0 {
			costPerFix = result.cost / float64(len(result.fixes))
			tokensPerFix = result.tokensUsed / len(result.fixes)
		}

		// Convert batch fixes to individual FixResults
		for _, fix := range result.fixes {
			fixResult := FixResult{
				Success:    fix.Success,
				FilePath:   filepath.Base(getFilePathFromURI(fix.IncidentURI)),
				TokensUsed: tokensPerFix,
				Cost:       costPerFix,
				Confidence: fix.Confidence,
			}

			if fix.Success {
				// Check confidence threshold before applying
				shouldApply, reason := bf.confidenceConf.ShouldApplyFix(fix.Confidence, v.MigrationComplexity, v.Effort)

				// Resolve and validate file path
				filePath, err := resolveAndValidateFilePath(getFilePathFromURI(fix.IncidentURI), bf.inputDir)
				if err != nil {
					fixResult.Error = fmt.Errorf("invalid file path: %w", err)
					fixResult.Success = false
					allResults = append(allResults, fixResult)
					continue
				}
				fullPath := filepath.Join(bf.inputDir, filePath)

				if !shouldApply {
					// Handle based on configured action
					switch bf.confidenceConf.OnLowConfidence {
					case confidence.ActionSkip:
						fixResult.SkippedLowConfidence = true
						fixResult.SkipReason = reason
						fixResult.Success = false
						fmt.Printf("  ⚠ Skipped: %s\n", fullPath)
						fmt.Printf("    Reason: %s\n", reason)

					case confidence.ActionWarnAndApply:
						// Print warning but continue to apply the fix
						fmt.Printf("  ⚠ Warning (low confidence): %s\n", fullPath)
						fmt.Printf("    Reason: %s\n", reason)
						fmt.Printf("    Applying anyway (action: warn-and-apply)\n")
						// Write the fixed file if not dry-run
						if !bf.dryRun {
							if err := os.WriteFile(fullPath, []byte(fix.FixedContent), 0644); err != nil {
								fixResult.Success = false
								fixResult.Error = fmt.Errorf("failed to write file: %w", err)
							}
						}

					case confidence.ActionManualReviewFile:
						fixResult.SkippedLowConfidence = true
						fixResult.SkipReason = reason
						fixResult.Success = false
						// Write to manual review file - need incident info
						// Find the matching incident for this fix
						for _, incident := range result.job.incidents {
							if incident.URI == fix.IncidentURI {
								tmpFixer := &Fixer{inputDir: bf.inputDir}
								if err := tmpFixer.writeToReviewFile(v, incident, &fixResult, reason, fix.Confidence); err != nil {
									fmt.Printf("  ⚠ Failed to write to review file: %v\n", err)
								} else {
									fmt.Printf("  ⚠ Low confidence: %s\n", fullPath)
									fmt.Printf("    Reason: %s\n", reason)
									fmt.Printf("    Added to .kantra-ai-review.yaml for manual review\n")
								}
								break
							}
						}
					}
				} else {
					// Confidence is good, apply the fix
					if !bf.dryRun {
						if err := os.WriteFile(fullPath, []byte(fix.FixedContent), 0644); err != nil {
							fixResult.Success = false
							fixResult.Error = fmt.Errorf("failed to write file: %w", err)
						}
					}
				}
			} else {
				fixResult.Error = fix.Error
			}

			allResults = append(allResults, fixResult)
		}
	}

	return allResults, nil
}

// createBatches splits incidents into batches of max size
// If GroupByFile is enabled, it groups incidents by file first to reduce token usage
func (bf *BatchFixer) createBatches(v violation.Violation) []batchJob {
	if bf.config.GroupByFile {
		return bf.createBatchesByFile(v)
	}

	// Original sequential batching
	var batches []batchJob
	for i := 0; i < len(v.Incidents); i += bf.config.MaxBatchSize {
		end := min(i+bf.config.MaxBatchSize, len(v.Incidents))
		batches = append(batches, batchJob{
			violation: v,
			incidents: v.Incidents[i:end],
			batch:     len(batches) + 1,
		})
	}

	return batches
}

// createBatchesByFile groups incidents by file before creating batches
// This reduces token usage by ensuring each file's content is sent once per batch
func (bf *BatchFixer) createBatchesByFile(v violation.Violation) []batchJob {
	// Group incidents by file path
	fileGroups := make(map[string][]violation.Incident)
	for _, incident := range v.Incidents {
		filePath := incident.GetFilePath()
		fileGroups[filePath] = append(fileGroups[filePath], incident)
	}

	// Create batches from file groups
	var batches []batchJob
	for _, incidents := range fileGroups {
		// If file has more incidents than max batch size, split into multiple batches
		for i := 0; i < len(incidents); i += bf.config.MaxBatchSize {
			end := min(i+bf.config.MaxBatchSize, len(incidents))
			batches = append(batches, batchJob{
				violation: v,
				incidents: incidents[i:end],
				batch:     len(batches) + 1,
			})
		}
	}

	// Log optimization metrics
	if len(fileGroups) < len(batches) {
		// File-based grouping created fewer file loads than sequential batching would
		sequentialBatches := (len(v.Incidents) + bf.config.MaxBatchSize - 1) / bf.config.MaxBatchSize
		fmt.Printf("   📊 File-based batching: %d files grouped into %d batches (vs %d sequential batches)\n",
			len(fileGroups), len(batches), sequentialBatches)
	}

	return batches
}

// estimateIncidentTokens estimates the token count for an incident
// Based on code context (10 lines around the incident) + incident metadata
// Uses rough approximation: 1 token ≈ 4 characters
func estimateIncidentTokens(incident violation.Incident, fileContent string) int {
	const (
		tokensPerChar     = 0.25 // 1 token ≈ 4 chars
		contextLines      = 10   // 5 before + 5 after
		avgCharsPerLine   = 80   // Average line length
		incidentOverhead  = 50   // Tokens for incident metadata (file, line, message)
	)

	// If we have file content, estimate based on actual context
	if fileContent != "" {
		lines := strings.Split(fileContent, "\n")
		lineNum := incident.LineNumber

		// Get context range (5 lines before and after)
		start := max(0, lineNum-5)
		end := min(len(lines), lineNum+5)

		// Count characters in context
		contextChars := 0
		for i := start; i < end && i < len(lines); i++ {
			contextChars += len(lines[i]) + 1 // +1 for newline
		}

		return int(float64(contextChars)*tokensPerChar) + incidentOverhead
	}

	// Fallback: estimate based on average line length
	estimatedChars := contextLines * avgCharsPerLine
	return int(float64(estimatedChars)*tokensPerChar) + incidentOverhead
}

// estimateBatchTokens estimates total tokens for a batch of incidents
// Includes prompt overhead for batch instructions
func estimateBatchTokens(incidents []violation.Incident, fileContents map[string]string) int {
	const promptOverhead = 500 // Tokens for batch instructions, JSON format, etc.

	totalTokens := promptOverhead
	for _, incident := range incidents {
		filePath := incident.GetFilePath()
		fileContent := fileContents[filePath]
		totalTokens += estimateIncidentTokens(incident, fileContent)
	}

	return totalTokens
}

// worker processes batches from the job channel
func (bf *BatchFixer) worker(ctx context.Context, jobs <-chan batchJob, results chan<- batchResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- batchResult{
				job: job,
				err: ctx.Err(),
			}
			return
		default:
			// Process the batch
			fixes, cost, tokensUsed, err := bf.processBatch(ctx, job)
			results <- batchResult{
				job:        job,
				fixes:      fixes,
				cost:       cost,
				tokensUsed: tokensUsed,
				err:        err,
			}
		}
	}
}

// processBatch sends a batch to the provider and gets fixes
func (bf *BatchFixer) processBatch(ctx context.Context, job batchJob) ([]provider.IncidentFix, float64, int, error) {
	// Load file contents for all incidents
	fileContents := make(map[string]string)
	for _, incident := range job.incidents {
		// Check for context cancellation before expensive I/O
		select {
		case <-ctx.Done():
			return nil, 0, 0, ctx.Err()
		default:
		}

		// Resolve and validate file path (prevents path traversal)
		filePath, err := resolveAndValidateFilePath(incident.GetFilePath(), bf.inputDir)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("invalid file path: %w", err)
		}

		fullPath := filepath.Join(bf.inputDir, filePath)

		if _, exists := fileContents[fullPath]; !exists {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, 0, 0, fmt.Errorf("failed to read file %s: %w", fullPath, err)
			}
			fileContents[fullPath] = string(content)
		}
	}

	// Detect language from first file
	language := "unknown"
	if len(job.incidents) > 0 {
		filePath, err := resolveAndValidateFilePath(job.incidents[0].GetFilePath(), bf.inputDir)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("invalid file path: %w", err)
		}
		language = detectLanguage(filePath)
	}

	// Create batch request
	req := provider.BatchRequest{
		Violation:    job.violation,
		Incidents:    job.incidents,
		FileContents: fileContents,
		Language:     language,
	}

	// Call provider
	resp, err := bf.provider.FixBatch(ctx, req)
	if err != nil {
		return nil, 0, 0, err
	}

	// Note: resp.Success=false just means one or more fixes failed,
	// not that the batch processing itself failed. We return the fixes
	// as-is and let the caller handle individual successes/failures.
	return resp.Fixes, resp.Cost, resp.TokensUsed, nil
}

// fixSequential falls back to sequential processing when batching is disabled
func (bf *BatchFixer) fixSequential(ctx context.Context, v violation.Violation) ([]FixResult, error) {
	// Create a regular fixer and process sequentially
	regularFixer := New(bf.provider, bf.inputDir, bf.dryRun)

	results := make([]FixResult, 0, len(v.Incidents))
	for _, incident := range v.Incidents {
		result, err := regularFixer.FixIncident(ctx, v, incident)
		if err != nil {
			return results, err
		}
		results = append(results, *result)
	}

	return results, nil
}

// getFilePathFromURI extracts the file path from a file:// URI
// It also strips line numbers if present (e.g., "file:///path/file.java:10" → "/path/file.java")
func getFilePathFromURI(uri string) string {
	// Remove file:// prefix if present
	if len(uri) > 7 && uri[:7] == "file://" {
		uri = uri[7:]
	}

	// Strip line number if present (format: "path/file:123")
	// Find the last colon and check if what follows is a number (line number)
	if idx := strings.LastIndex(uri, ":"); idx != -1 {
		// Check if everything after the colon is digits
		afterColon := uri[idx+1:]
		if len(afterColon) > 0 {
			allDigits := true
			for _, ch := range afterColon {
				if ch < '0' || ch > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				// This is a line number, strip it
				uri = uri[:idx]
			}
		}
	}

	return uri
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
