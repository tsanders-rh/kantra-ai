package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tsanders/kantra-ai/pkg/config"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/gitutil"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/provider/claude"
	"github.com/tsanders/kantra-ai/pkg/provider/openai"
	"github.com/tsanders/kantra-ai/pkg/ux"
	"github.com/tsanders/kantra-ai/pkg/verifier"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

var (
	analysisPath        string
	inputPath           string
	providerName        string
	violationIDs        string
	categories          string
	maxEffort           int
	maxCost             float64
	dryRun              bool
	model               string
	gitCommitStrategy   string
	createPR            bool
	branchName          string
	verify              string
	verifyStrategy      string
	verifyCommand       string
	verifyFailFast      bool
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
	remediateCmd.Flags().StringVar(&verify, "verify", "", "Verification type: build, test (runs after fixes to ensure they don't break build/tests)")
	remediateCmd.Flags().StringVar(&verifyStrategy, "verify-strategy", "at-end", "When to verify: per-fix, per-violation, at-end")
	remediateCmd.Flags().StringVar(&verifyCommand, "verify-command", "", "Custom verification command (overrides auto-detection)")
	remediateCmd.Flags().BoolVar(&verifyFailFast, "verify-fail-fast", true, "Stop on first verification failure")

	// MarkFlagRequired only errors if flag doesn't exist, which can't happen here
	_ = remediateCmd.MarkFlagRequired("analysis")
	_ = remediateCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(remediateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRemediate(cmd *cobra.Command, args []string) error {
	// Load configuration from file (if exists)
	cfg := config.LoadOrDefault()

	// Apply config file values for flags that weren't explicitly set
	// CLI flags take precedence over config file values
	if analysisPath == "" && cfg.Paths.Analysis != "" {
		analysisPath = cfg.Paths.Analysis
	}
	if inputPath == "" && cfg.Paths.Input != "" {
		inputPath = cfg.Paths.Input
	}
	if providerName == "claude" && cfg.Provider.Name != "" { // "claude" is the flag default
		providerName = cfg.Provider.Name
	}
	if model == "" && cfg.Provider.Model != "" {
		model = cfg.Provider.Model
	}
	if violationIDs == "" && len(cfg.Filters.ViolationIDs) > 0 {
		violationIDs = strings.Join(cfg.Filters.ViolationIDs, ",")
	}
	if categories == "" && len(cfg.Filters.Categories) > 0 {
		categories = strings.Join(cfg.Filters.Categories, ",")
	}
	if maxEffort == 0 && cfg.Limits.MaxEffort > 0 {
		maxEffort = cfg.Limits.MaxEffort
	}
	if maxCost == 0 && cfg.Limits.MaxCost > 0 {
		maxCost = cfg.Limits.MaxCost
	}
	if gitCommitStrategy == "" && cfg.Git.CommitStrategy != "" {
		gitCommitStrategy = cfg.Git.CommitStrategy
	}
	if !createPR && cfg.Git.CreatePR {
		createPR = cfg.Git.CreatePR
	}
	if branchName == "" && cfg.Git.BranchPrefix != "" {
		branchName = cfg.Git.BranchPrefix
	}
	if verify == "" && cfg.Verification.Enabled {
		verify = cfg.Verification.Type
	}
	if verifyStrategy == "at-end" && cfg.Verification.Strategy != "" { // "at-end" is the flag default
		verifyStrategy = cfg.Verification.Strategy
	}
	if verifyCommand == "" && cfg.Verification.Command != "" {
		verifyCommand = cfg.Verification.Command
	}
	// For verify-fail-fast, only apply config if it differs from default (true)
	if verifyFailFast && !cfg.Verification.FailFast {
		verifyFailFast = cfg.Verification.FailFast
	}
	if !dryRun && cfg.DryRun {
		dryRun = cfg.DryRun
	}

	ux.PrintHeader("kantra-ai remediate")

	// Load violations
	spinner := ux.NewSpinner(fmt.Sprintf("Loading analysis from %s...", analysisPath))
	spinner.Start()

	analysis, err := violation.LoadAnalysis(analysisPath)
	if err != nil {
		spinner.StopWithError(fmt.Sprintf("Failed to load analysis: %v", err))
		return fmt.Errorf("failed to load analysis: %w", err)
	}

	spinner.StopWithSuccess(fmt.Sprintf("Loaded %d violations", len(analysis.Violations)))
	fmt.Println()

	// Initialize git tracker if requested
	var commitTracker *gitutil.CommitTracker
	var verifiedTracker *gitutil.VerifiedCommitTracker
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

		// Check if verification is requested
		if verify != "" {
			// Parse verification configuration
			verifyType, err := verifier.ParseVerificationType(verify)
			if err != nil {
				return err
			}

			verifyStrat, err := verifier.ParseVerificationStrategy(verifyStrategy)
			if err != nil {
				return err
			}

			verifyConfig := verifier.Config{
				Type:          verifyType,
				Strategy:      verifyStrat,
				WorkingDir:    inputPath,
				CustomCommand: verifyCommand,
				FailFast:      verifyFailFast,
				SkipOnDryRun:  dryRun,
			}

			verifiedTracker, err = gitutil.NewVerifiedCommitTracker(strategy, inputPath, providerName, verifyConfig)
			if err != nil {
				return fmt.Errorf("failed to initialize verification: %w", err)
			}
			commitTracker = verifiedTracker.GetCommitTracker()
			ux.PrintSuccess("Git commits enabled (%s strategy)", gitCommitStrategy)
			ux.PrintSuccess("Verification enabled (%s, %s strategy)", verify, verifyStrategy)
			fmt.Println()
		} else {
			commitTracker = gitutil.NewCommitTracker(strategy, inputPath, providerName)
			ux.PrintSuccess("Git commits enabled (%s strategy)", gitCommitStrategy)
			fmt.Println()
		}
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

		ux.PrintSuccess("PR creation enabled (%s strategy)", gitCommitStrategy)
		fmt.Println()
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
	provSpinner := ux.NewSpinner(fmt.Sprintf("Initializing %s provider...", providerName))
	provSpinner.Start()

	prov, err := createProvider(providerName, model)
	if err != nil {
		provSpinner.StopWithError(fmt.Sprintf("Failed to initialize provider: %v", err))
		return fmt.Errorf("failed to create provider: %w", err)
	}

	provSpinner.StopWithSuccess(fmt.Sprintf("%s provider ready", providerName))
	fmt.Println()

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
	ux.PrintSection("Fixing violations")

	ctx := context.Background()
	totalCost := 0.0
	totalTokens := 0
	successCount := 0
	failCount := 0
	startTime := time.Now()

	// Count total incidents for progress bar
	totalIncidents := 0
	for _, v := range filtered {
		totalIncidents += len(v.Incidents)
	}

	// Create progress bar
	var bar *progressbar.ProgressBar
	if ux.IsTerminal() && !dryRun {
		bar = ux.NewProgressBar(totalIncidents, "Progress")
	}

	for i, v := range filtered {
		fmt.Printf("\n%s [%d/%d] Violation: %s (%s)\n",
			ux.Bold("→"), i+1, len(filtered), ux.Info(v.ID), ux.Dim(v.Category))
		fmt.Printf("  %s %s\n", ux.Dim("Description:"), v.Description)
		fmt.Printf("  %s %d\n", ux.Dim("Incidents:"), len(v.Incidents))

		// Fix each incident
		for j, incident := range v.Incidents {
			filePath := incident.GetFilePath()
			fmt.Printf("  %s [%d/%d] %s:%d\n",
				ux.Dim("•"), j+1, len(v.Incidents), filePath, incident.LineNumber)

			result, err := fix.FixIncident(ctx, v, incident)
			if bar != nil {
				_ = bar.Add(1) // Ignore progress bar errors
			}

			if err != nil {
				ux.PrintError("    Failed: %v", err)
				failCount++
				continue
			}

			if result.Success {
				successCount++
				totalCost += result.Cost
				totalTokens += result.TokensUsed

				// Track for git commit if enabled
				if commitTracker != nil && !dryRun {
					// Use verified tracker if verification is enabled
					if verifiedTracker != nil {
						if err := verifiedTracker.TrackFix(v, incident, result); err != nil {
							ux.PrintWarning("    Git commit/verification failed: %v", err)
						}
					} else {
						if err := commitTracker.TrackFix(v, incident, result); err != nil {
							ux.PrintWarning("    Git commit failed: %v", err)
						}
					}
				}

				// Track for PR if enabled
				if prTracker != nil && !dryRun {
					if err := prTracker.TrackForPR(v, incident, result); err != nil {
						ux.PrintWarning("    PR tracking failed: %v", err)
					}
				}

				// Check if we've exceeded max cost
				if maxCost > 0 && totalCost >= maxCost {
					ux.PrintWarning("\nMax cost ($%.2f) reached. Stopping.", maxCost)
					goto summary
				}
			} else {
				failCount++
				ux.PrintError("    Failed: %v", result.Error)
			}
		}
	}

	// Finish progress bar
	if bar != nil {
		_ = bar.Finish() // Ignore progress bar errors
		fmt.Println()
	}

summary:
	// Finalize git commits if enabled
	if commitTracker != nil && !dryRun {
		// Use verified tracker if verification is enabled
		if verifiedTracker != nil {
			if err := verifiedTracker.Finalize(); err != nil {
				ux.PrintWarning("\nFinal git commit/verification failed: %v", err)
			}
			// Print verification stats
			stats := verifiedTracker.GetStats()
			if stats.TotalVerifications > 0 {
				ux.PrintSection("Verification Summary")
				fmt.Printf("  Total verifications: %s\n", ux.Bold(fmt.Sprintf("%d", stats.TotalVerifications)))
				fmt.Printf("  %s Passed: %s\n", ux.Success("✓"), ux.Success(fmt.Sprintf("%d", stats.PassedVerifications)))
				if stats.FailedVerifications > 0 {
					fmt.Printf("  %s Failed: %s\n", ux.Error("✗"), ux.Error(fmt.Sprintf("%d", stats.FailedVerifications)))
					fmt.Printf("  %s Fixes skipped due to failures: %s\n",
						ux.Warning("⚠"), ux.Warning(fmt.Sprintf("%d", stats.SkippedFixes)))
				}
				fmt.Println()
			}
		} else {
			if err := commitTracker.Finalize(); err != nil {
				ux.PrintWarning("\nFinal git commit failed: %v", err)
			}
		}
	}

	// Create pull requests if enabled
	if prTracker != nil && !dryRun {
		prSpinner := ux.NewSpinner("Creating pull request(s)...")
		prSpinner.Start()

		if err := prTracker.Finalize(); err != nil {
			prSpinner.Stop()
			// Format error message based on error type
			ghErr, ok := err.(*gitutil.GitHubError)
			if ok {
				switch ghErr.StatusCode {
				case 401:
					ux.PrintWarning("\nPR creation failed: Invalid GITHUB_TOKEN")
					fmt.Println("  Verify your token at: https://github.com/settings/tokens")
					fmt.Println("  Make sure it hasn't expired")
				case 403:
					ux.PrintWarning("\nPR creation failed: Insufficient permissions")
					fmt.Println("  Your token needs 'repo' scope")
					fmt.Println("  Regenerate at: https://github.com/settings/tokens")
				default:
					// Default case - most errors now have helpful messages from pr_tracker
					ux.PrintWarning("\nPR creation failed: %v", err)
				}
			} else {
				// Non-GitHub API errors (git operations, etc.) - pass through as-is
				ux.PrintWarning("\nPR creation failed: %v", err)
			}
		} else {
			prSpinner.Stop()
			// Print created PRs
			prs := prTracker.GetCreatedPRs()
			ux.PrintSuccess("\nCreated %d pull request(s):", len(prs))
			for _, pr := range prs {
				if pr.ViolationID != "" {
					fmt.Printf("  %s PR #%d (%s): %s\n",
						ux.Success("→"), pr.Number, ux.Info(pr.ViolationID), ux.Dim(pr.URL))
				} else {
					fmt.Printf("  %s PR #%d: %s\n",
						ux.Success("→"), pr.Number, ux.Dim(pr.URL))
				}
			}
		}
	}

	duration := time.Since(startTime)

	ux.PrintHeader("Summary")

	// Print summary as a table
	rows := [][]string{
		{ux.Success("✓") + " Successful fixes:", ux.Success(fmt.Sprintf("%d", successCount))},
		{ux.Error("✗") + " Failed fixes:", ux.Error(fmt.Sprintf("%d", failCount))},
		{"💰 Total cost:", ux.FormatCost(totalCost)},
		{"🎫 Total tokens:", ux.FormatTokens(totalTokens)},
		{"⏱  Duration:", ux.FormatDuration(duration)},
	}

	if successCount > 0 {
		avgCost := totalCost / float64(successCount)
		avgTokens := totalTokens / successCount
		rows = append(rows, []string{
			"📊 Average per fix:",
			fmt.Sprintf("%s (%s tokens)", ux.FormatCost(avgCost), ux.FormatTokens(avgTokens)),
		})
	}

	ux.PrintSummaryTable(rows)

	if dryRun {
		fmt.Println()
		ux.PrintWarning("DRY-RUN mode - no changes were made")
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
