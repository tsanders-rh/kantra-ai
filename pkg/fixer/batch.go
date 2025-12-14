package fixer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// BatchConfig configures batch processing behavior
type BatchConfig struct {
	// MaxBatchSize is the maximum number of incidents to fix in a single batch
	// Default: 10 (stays under token limits)
	MaxBatchSize int

	// Parallelism is the number of concurrent batches to process
	// Default: 4
	Parallelism int

	// Enabled controls whether batching is used
	// Default: true
	Enabled bool
}

// DefaultBatchConfig returns the recommended batch configuration
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MaxBatchSize: 10,
		Parallelism:  4,
		Enabled:      true,
	}
}

// BatchFixer provides optimized batch processing of violations
type BatchFixer struct {
	provider provider.Provider
	inputDir string
	dryRun   bool
	config   BatchConfig
}

// NewBatchFixer creates a new batch fixer
func NewBatchFixer(p provider.Provider, inputDir string, dryRun bool, config BatchConfig) *BatchFixer {
	return &BatchFixer{
		provider: p,
		inputDir: inputDir,
		dryRun:   dryRun,
		config:   config,
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
			}

			if fix.Success {
				// Write the fixed file if not dry-run
				if !bf.dryRun {
					filePath := bf.resolveFilePath(getFilePathFromURI(fix.IncidentURI))
					fullPath := filepath.Join(bf.inputDir, filePath)
					if err := os.WriteFile(fullPath, []byte(fix.FixedContent), 0644); err != nil {
						fixResult.Success = false
						fixResult.Error = fmt.Errorf("failed to write file: %w", err)
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
func (bf *BatchFixer) createBatches(v violation.Violation) []batchJob {
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
		filePath := bf.resolveFilePath(incident.GetFilePath())
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
		filePath := bf.resolveFilePath(job.incidents[0].GetFilePath())
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

// resolveFilePath resolves a file path to be relative to inputDir
func (bf *BatchFixer) resolveFilePath(filePath string) string {
	// Make it relative to input directory if it's absolute
	if filepath.IsAbs(filePath) {
		// Try to make it relative to inputDir
		absInputDir, _ := filepath.Abs(bf.inputDir)
		if strings.HasPrefix(filePath, absInputDir) {
			filePath = strings.TrimPrefix(filePath, absInputDir)
			filePath = strings.TrimPrefix(filePath, string(filepath.Separator))
		} else {
			// Path looks absolute but doesn't match input dir
			// This happens with URIs like file:///src/file.java
			// Strip leading slash(es) to make it relative
			filePath = strings.TrimLeft(filePath, string(filepath.Separator))
		}
	}
	return filePath
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
