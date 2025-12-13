package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/provider/claude"
	"github.com/tsanders/kantra-ai/pkg/provider/openai"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

var (
	analysisPath  string
	inputPath     string
	providerName  string
	violationIDs  string
	categories    string
	maxEffort     int
	maxCost       float64
	dryRun        bool
	model         string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kantra-ai",
		Short: "AI-powered remediation for Konveyor violations",
		Long: `kantra-ai applies AI-powered fixes to violations found by Konveyor analysis.

This is an MVP focused on validation: proving that AI can successfully fix
Konveyor violations at reasonable cost and quality.`,
	}

	remediateCmd := &cobra.Command{
		Use:   "remediate",
		Short: "Remediate violations using AI",
		RunE:  runRemediate,
	}

	remediateCmd.Flags().StringVar(&analysisPath, "analysis", "", "Path to Konveyor analysis output.yaml (required)")
	remediateCmd.Flags().StringVar(&inputPath, "input", "", "Path to application source code (required)")
	remediateCmd.Flags().StringVar(&providerName, "provider", "claude", "AI provider: claude, openai")
	remediateCmd.Flags().StringVar(&violationIDs, "violation-ids", "", "Comma-separated violation IDs to fix")
	remediateCmd.Flags().StringVar(&categories, "categories", "", "Comma-separated categories: mandatory, optional, potential")
	remediateCmd.Flags().IntVar(&maxEffort, "max-effort", 0, "Maximum effort level (0 = no limit)")
	remediateCmd.Flags().Float64Var(&maxCost, "max-cost", 0, "Maximum cost in USD (0 = no limit)")
	remediateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	remediateCmd.Flags().StringVar(&model, "model", "", "AI model to use (provider-specific)")

	remediateCmd.MarkFlagRequired("analysis")
	remediateCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(remediateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRemediate(cmd *cobra.Command, args []string) error {
	fmt.Println("kantra-ai remediate")
	fmt.Println("===================")
	fmt.Println()

	// Load violations
	fmt.Printf("Loading analysis from %s...\n", analysisPath)
	analysis, err := violation.LoadAnalysis(analysisPath)
	if err != nil {
		return fmt.Errorf("failed to load analysis: %w", err)
	}
	fmt.Printf("✓ Loaded %d violations\n\n", len(analysis.Violations))

	// Parse filters
	var idFilter []string
	if violationIDs != "" {
		idFilter = strings.Split(violationIDs, ",")
	}

	var catFilter []string
	if categories != "" {
		catFilter = strings.Split(categories, ",")
	}

	// Apply filters
	filtered := analysis.FilterViolations(idFilter, catFilter, maxEffort)
	fmt.Printf("After filtering: %d violations\n", len(filtered))

	if len(filtered) == 0 {
		fmt.Println("No violations to fix.")
		return nil
	}

	// Initialize provider
	fmt.Printf("Initializing %s provider...\n", providerName)
	prov, err := createProvider(providerName, model)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	fmt.Printf("✓ Provider ready\n\n")

	// Estimate cost
	if !dryRun {
		totalEstimate := 0.0
		for _, v := range filtered {
			for _, incident := range v.Incidents {
				req := provider.FixRequest{
					Violation: v,
					Incident:  incident,
				}
				cost, _ := prov.EstimateCost(req)
				totalEstimate += cost
			}
		}
		fmt.Printf("Estimated cost: $%.2f\n", totalEstimate)
		if maxCost > 0 && totalEstimate > maxCost {
			return fmt.Errorf("estimated cost ($%.2f) exceeds max-cost ($%.2f)", totalEstimate, maxCost)
		}
		fmt.Println()
	}

	// Create fixer
	fix := fixer.New(prov, inputPath, dryRun)

	// Fix violations
	fmt.Println("Fixing violations...")
	fmt.Println()

	ctx := context.Background()
	totalCost := 0.0
	totalTokens := 0
	successCount := 0
	failCount := 0
	startTime := time.Now()

	for i, v := range filtered {
		fmt.Printf("[%d/%d] Violation: %s (%s)\n", i+1, len(filtered), v.ID, v.Category)
		fmt.Printf("  Description: %s\n", v.Description)
		fmt.Printf("  Incidents: %d\n", len(v.Incidents))

		// Fix each incident
		for j, incident := range v.Incidents {
			fmt.Printf("  [%d/%d] %s:%d\n", j+1, len(v.Incidents), incident.GetFilePath(), incident.LineNumber)

			result, err := fix.FixIncident(ctx, v, incident)
			if err != nil {
				fmt.Printf("    ✗ Failed: %v\n", err)
				failCount++
				continue
			}

			if result.Success {
				successCount++
				totalCost += result.Cost
				totalTokens += result.TokensUsed

				// Check if we've exceeded max cost
				if maxCost > 0 && totalCost >= maxCost {
					fmt.Printf("\n⚠ Max cost ($%.2f) reached. Stopping.\n", maxCost)
					goto summary
				}
			} else {
				failCount++
				fmt.Printf("    ✗ Failed: %v\n", result.Error)
			}
		}
		fmt.Println()
	}

summary:
	duration := time.Since(startTime)

	fmt.Println("Summary")
	fmt.Println("=======")
	fmt.Printf("✓ Successful fixes: %d\n", successCount)
	fmt.Printf("✗ Failed fixes: %d\n", failCount)
	fmt.Printf("💰 Total cost: $%.4f\n", totalCost)
	fmt.Printf("🎫 Total tokens: %d\n", totalTokens)
	fmt.Printf("⏱  Duration: %s\n", duration.Round(time.Second))

	if successCount > 0 {
		avgCost := totalCost / float64(successCount)
		avgTokens := totalTokens / successCount
		fmt.Printf("📊 Average per fix: $%.4f (%d tokens)\n", avgCost, avgTokens)
	}

	if dryRun {
		fmt.Println("\n⚠ DRY-RUN mode - no changes were made")
	}

	return nil
}

func createProvider(name string, model string) (provider.Provider, error) {
	config := provider.Config{
		Name:        name,
		Model:       model,
		Temperature: 0.2,
	}

	switch name {
	case "claude":
		return claude.New(config)
	case "openai":
		return openai.New(config)
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
