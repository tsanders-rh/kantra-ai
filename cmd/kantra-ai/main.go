package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/gitutil"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/provider/claude"
	"github.com/tsanders/kantra-ai/pkg/provider/openai"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

var (
	analysisPath      string
	inputPath         string
	providerName      string
	violationIDs      string
	categories        string
	maxEffort         int
	maxCost           float64
	dryRun            bool
	model             string
	gitCommitStrategy string
	createPR          bool
	branchName        string
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
	remediateCmd.Flags().StringVar(&gitCommitStrategy, "git-commit", "", "Git commit strategy: per-violation, per-incident, at-end")
	remediateCmd.Flags().BoolVar(&createPR, "create-pr", false, "Create GitHub pull request(s) after remediation (requires --git-commit)")
	remediateCmd.Flags().StringVar(&branchName, "branch", "", "Branch name for PR (default: kantra-ai/remediation-TIMESTAMP)")

	// MarkFlagRequired only errors if flag doesn't exist, which can't happen here
	_ = remediateCmd.MarkFlagRequired("analysis")
	_ = remediateCmd.MarkFlagRequired("input")

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

	// Initialize git tracker if requested
	var commitTracker *gitutil.CommitTracker
	if gitCommitStrategy != "" {
		if !gitutil.IsGitInstalled() {
			return fmt.Errorf("--git-commit requires git to be installed")
		}
		if !gitutil.IsGitRepository(inputPath) {
			return fmt.Errorf("--git-commit requires input directory to be a git repository")
		}

		strategy, err := gitutil.ParseStrategy(gitCommitStrategy)
		if err != nil {
			return err
		}

		commitTracker = gitutil.NewCommitTracker(strategy, inputPath, providerName)
		fmt.Printf("✓ Git commits enabled (%s strategy)\n\n", gitCommitStrategy)
	}

	// Initialize PR tracker if requested
	var prTracker *gitutil.PRTracker
	if createPR {
		// Validate prerequisites
		if gitCommitStrategy == "" {
			return fmt.Errorf("--create-pr requires --git-commit to be set")
		}

		// Check for GitHub token (not required in dry-run mode)
		githubToken := os.Getenv("GITHUB_TOKEN")
		if githubToken == "" && !dryRun {
			return fmt.Errorf("--create-pr requires GITHUB_TOKEN environment variable\n\n" +
				"To set up:\n" +
				"  1. Create a token at: https://github.com/settings/tokens\n" +
				"  2. Grant 'repo' scope\n" +
				"  3. Export: export GITHUB_TOKEN=your_token_here")
		}

		// Parse PR strategy from commit strategy
		prStrategy, err := gitutil.ParsePRStrategy(gitCommitStrategy)
		if err != nil {
			return err
		}

		// Generate branch name if not provided
		if branchName == "" {
			branchName = fmt.Sprintf("kantra-ai/remediation-%d", time.Now().Unix())
		}

		// Initialize PR tracker
		prConfig := gitutil.PRConfig{
			Strategy:     prStrategy,
			BranchPrefix: branchName,
			GitHubToken:  githubToken,
			DryRun:       dryRun,
		}

		progress := &gitutil.StdoutProgressWriter{}
		prTracker, err = gitutil.NewPRTracker(prConfig, inputPath, providerName, progress)
		if err != nil {
			return fmt.Errorf("failed to initialize PR tracker: %w", err)
		}

		fmt.Printf("✓ PR creation enabled (%s strategy)\n\n", gitCommitStrategy)
	}

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

				// Track for git commit if enabled
				if commitTracker != nil && !dryRun {
					if err := commitTracker.TrackFix(v, incident, result); err != nil {
						fmt.Printf("    ⚠ Git commit failed: %v\n", err)
					}
				}

				// Track for PR if enabled
				if prTracker != nil && !dryRun {
					if err := prTracker.TrackForPR(v, incident, result); err != nil {
						fmt.Printf("    ⚠ PR tracking failed: %v\n", err)
					}
				}

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
	// Finalize git commits if enabled
	if commitTracker != nil && !dryRun {
		if err := commitTracker.Finalize(); err != nil {
			fmt.Printf("\n⚠ Final git commit failed: %v\n", err)
		}
	}

	// Create pull requests if enabled
	if prTracker != nil && !dryRun {
		fmt.Println("\nCreating pull request(s)...")
		if err := prTracker.Finalize(); err != nil {
			// Format error message based on error type
			ghErr, ok := err.(*gitutil.GitHubError)
			if ok {
				switch ghErr.StatusCode {
				case 401:
					fmt.Printf("\n⚠ PR creation failed: Invalid GITHUB_TOKEN\n")
					fmt.Println("  Verify your token at: https://github.com/settings/tokens")
					fmt.Println("  Make sure it hasn't expired")
				case 403:
					fmt.Printf("\n⚠ PR creation failed: Insufficient permissions\n")
					fmt.Println("  Your token needs 'repo' scope")
					fmt.Println("  Regenerate at: https://github.com/settings/tokens")
				default:
					// Default case - most errors now have helpful messages from pr_tracker
					fmt.Printf("\n⚠ PR creation failed: %v\n", err)
				}
			} else {
				// Non-GitHub API errors (git operations, etc.) - pass through as-is
				fmt.Printf("\n⚠ PR creation failed: %v\n", err)
			}
		} else {
			// Print created PRs
			prs := prTracker.GetCreatedPRs()
			fmt.Printf("\n✓ Created %d pull request(s):\n", len(prs))
			for _, pr := range prs {
				if pr.ViolationID != "" {
					fmt.Printf("  - PR #%d (%s): %s\n", pr.Number, pr.ViolationID, pr.URL)
				} else {
					fmt.Printf("  - PR #%d: %s\n", pr.Number, pr.URL)
				}
			}
		}
	}

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
