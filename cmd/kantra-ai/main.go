package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tsanders/kantra-ai/pkg/confidence"
	"github.com/tsanders/kantra-ai/pkg/config"
	"github.com/tsanders/kantra-ai/pkg/executor"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/gitutil"
	"github.com/tsanders/kantra-ai/pkg/planner"
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

	// Plan command flags
	planOutputPath      string
	planMaxPhases       int
	planRiskTolerance   string
	planInteractive     bool

	// Execute command flags
	executePlanPath     string
	executeStatePath    string
	executePhaseID      string
	executeResume       bool

	// Confidence threshold flags
	confidenceEnabled   bool
	minConfidence       float64
	onLowConfidence     string
	complexityThreshold string // format: "level=threshold,level=threshold"
)

func main() {
	rootCmd = &cobra.Command{
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
	remediateCmd.Flags().BoolVar(&confidenceEnabled, "enable-confidence", false, "Enable confidence threshold filtering")
	remediateCmd.Flags().Float64Var(&minConfidence, "min-confidence", 0.0, "Global minimum confidence threshold (0.0-1.0, overrides complexity thresholds)")
	remediateCmd.Flags().StringVar(&onLowConfidence, "on-low-confidence", "skip", "Action on low confidence: skip, warn-and-apply, manual-review-file")
	remediateCmd.Flags().StringVar(&complexityThreshold, "complexity-threshold", "", "Override thresholds: trivial=0.7,low=0.75,medium=0.8,high=0.9,expert=0.95")

	// MarkFlagRequired only errors if flag doesn't exist, which can't happen here
	_ = remediateCmd.MarkFlagRequired("analysis")
	_ = remediateCmd.MarkFlagRequired("input")

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate a phased migration plan",
		Long: `Generate an AI-powered phased migration plan from Konveyor violations.

The plan command analyzes violations and creates a structured plan with multiple
phases that can be reviewed, edited, and executed incrementally.`,
		RunE: runPlan,
	}

	planCmd.Flags().StringVar(&analysisPath, "analysis", "", "Path to Konveyor analysis output.yaml (required)")
	planCmd.Flags().StringVar(&inputPath, "input", "", "Path to application source code (required)")
	planCmd.Flags().StringVar(&providerName, "provider", "claude", "AI provider: claude (openai not yet supported for planning)")
	planCmd.Flags().StringVar(&planOutputPath, "output", ".kantra-ai-plan.yaml", "Output plan file path")
	planCmd.Flags().IntVar(&planMaxPhases, "max-phases", 0, "Maximum number of phases (0 = auto, typically 3-5)")
	planCmd.Flags().StringVar(&planRiskTolerance, "risk-tolerance", "balanced", "Risk tolerance: conservative, balanced, aggressive")
	planCmd.Flags().StringVar(&violationIDs, "violation-ids", "", "Comma-separated violation IDs to include")
	planCmd.Flags().StringVar(&categories, "categories", "", "Comma-separated categories: mandatory, optional, potential")
	planCmd.Flags().IntVar(&maxEffort, "max-effort", 0, "Maximum effort level (0 = no limit)")
	planCmd.Flags().StringVar(&model, "model", "", "AI model to use (provider-specific)")
	planCmd.Flags().BoolVar(&planInteractive, "interactive", false, "Enable interactive phase approval")

	_ = planCmd.MarkFlagRequired("analysis")
	_ = planCmd.MarkFlagRequired("input")

	executeCmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute a migration plan",
		Long: `Execute a migration plan generated by the plan command.

The execute command loads a plan file and executes the phases, tracking progress
in a state file. Supports resuming from failures and executing specific phases.`,
		RunE: runExecute,
	}

	executeCmd.Flags().StringVar(&executePlanPath, "plan", ".kantra-ai-plan.yaml", "Path to plan file")
	executeCmd.Flags().StringVar(&executeStatePath, "state", ".kantra-ai-state.yaml", "Path to state file")
	executeCmd.Flags().StringVar(&inputPath, "input", "", "Path to application source code (required)")
	executeCmd.Flags().StringVar(&providerName, "provider", "claude", "AI provider: claude, openai")
	executeCmd.Flags().StringVar(&model, "model", "", "AI model to use (provider-specific)")
	executeCmd.Flags().StringVar(&executePhaseID, "phase", "", "Execute specific phase (e.g., phase-1)")
	executeCmd.Flags().BoolVar(&executeResume, "resume", false, "Resume from last failure")
	executeCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without applying changes")
	executeCmd.Flags().StringVar(&gitCommitStrategy, "git-commit", "", "Git commit strategy: per-violation, per-incident, at-end")
	executeCmd.Flags().BoolVar(&createPR, "create-pr", false, "Create GitHub pull request(s)")
	executeCmd.Flags().StringVar(&branchName, "branch", "", "Branch name for PR")
	executeCmd.Flags().StringVar(&verify, "verify", "", "Verification type: build, test")
	executeCmd.Flags().StringVar(&verifyStrategy, "verify-strategy", "at-end", "When to verify: per-fix, per-violation, at-end")
	executeCmd.Flags().StringVar(&verifyCommand, "verify-command", "", "Custom verification command")
	executeCmd.Flags().BoolVar(&verifyFailFast, "verify-fail-fast", true, "Stop on first verification failure")
	executeCmd.Flags().BoolVar(&confidenceEnabled, "enable-confidence", false, "Enable confidence threshold filtering")
	executeCmd.Flags().Float64Var(&minConfidence, "min-confidence", 0.0, "Global minimum confidence threshold (0.0-1.0, overrides complexity thresholds)")
	executeCmd.Flags().StringVar(&onLowConfidence, "on-low-confidence", "skip", "Action on low confidence: skip, warn-and-apply, manual-review-file")
	executeCmd.Flags().StringVar(&complexityThreshold, "complexity-threshold", "", "Override thresholds: trivial=0.7,low=0.75,medium=0.8,high=0.9,expert=0.95")

	_ = executeCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(remediateCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(executeCmd)

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

	// Build confidence configuration
	confidenceConf, err := buildConfidenceConfig(cfg)
	if err != nil {
		return fmt.Errorf("invalid confidence configuration: %w", err)
	}

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

	// Create fixer with confidence configuration
	fix := fixer.NewWithConfidence(prov, inputPath, dryRun, confidenceConf)

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

func runPlan(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	ux.PrintHeader("Generating Migration Plan")

	// Create provider
	prov, err := createProvider(providerName, model)
	if err != nil {
		return err
	}

	fmt.Printf("📋 Analysis: %s\n", analysisPath)
	fmt.Printf("📂 Input: %s\n", inputPath)
	fmt.Printf("🤖 Provider: %s\n", prov.Name())
	fmt.Printf("📝 Output: %s\n", planOutputPath)
	fmt.Println()

	// Parse filters
	var violationIDList []string
	if violationIDs != "" {
		violationIDList = strings.Split(violationIDs, ",")
	}

	var categoryList []string
	if categories != "" {
		categoryList = strings.Split(categories, ",")
	}

	// Create planner
	plannerConfig := planner.Config{
		AnalysisPath:  analysisPath,
		InputPath:     inputPath,
		Provider:      prov,
		OutputPath:    planOutputPath,
		MaxPhases:     planMaxPhases,
		RiskTolerance: planRiskTolerance,
		Categories:    categoryList,
		ViolationIDs:  violationIDList,
		MaxEffort:     maxEffort,
		Interactive:   planInteractive,
	}

	p := planner.New(plannerConfig)

	// Generate plan
	fmt.Println("Analyzing violations and generating plan...")
	fmt.Println()

	ctx := context.Background()
	result, err := p.Generate(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	duration := time.Since(startTime)

	// Print success message
	ux.PrintHeader("Plan Generated Successfully")

	rows := [][]string{
		{"📝 Plan file:", ux.Success(result.PlanPath)},
		{"📊 Total phases:", ux.Success(fmt.Sprintf("%d", result.TotalPhases))},
		{"💰 Estimated total cost:", ux.FormatCost(result.TotalCost)},
		{"💵 Plan generation cost:", ux.FormatCost(result.GenerateCost)},
		{"🎫 Tokens used:", ux.FormatTokens(result.TokensUsed)},
		{"⏱  Duration:", ux.FormatDuration(duration)},
	}

	ux.PrintSummaryTable(rows)

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review plan:    cat %s\n", result.PlanPath)
	fmt.Printf("  • Edit if needed: vim %s\n", result.PlanPath)
	fmt.Println("  • Execute:        kantra-ai execute")

	return nil
}

func runExecute(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	ux.PrintHeader("Executing Migration Plan")

	// Load configuration from file (if exists)
	cfg := config.LoadOrDefault()

	// Create provider
	prov, err := createProvider(providerName, model)
	if err != nil {
		return err
	}

	fmt.Printf("📋 Plan: %s\n", executePlanPath)
	fmt.Printf("📊 State: %s\n", executeStatePath)
	fmt.Printf("📂 Input: %s\n", inputPath)
	fmt.Printf("🤖 Provider: %s\n", prov.Name())
	fmt.Println()

	// Build confidence configuration
	confidenceConf, err := buildConfidenceConfig(cfg)
	if err != nil {
		return fmt.Errorf("invalid confidence configuration: %w", err)
	}

	// Create executor config
	executorConfig := executor.Config{
		PlanPath:         executePlanPath,
		StatePath:        executeStatePath,
		InputPath:        inputPath,
		Provider:         prov,
		PhaseID:          executePhaseID,
		DryRun:           dryRun,
		GitCommit:        gitCommitStrategy,
		CreatePR:         createPR,
		Progress:         &ux.ConsoleProgressWriter{},
		Resume:           executeResume,
		ConfidenceConfig: confidenceConf,
	}

	// Create executor
	exec, err := executor.New(executorConfig)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Execute plan
	ctx := context.Background()
	result, err := exec.Execute(ctx)
	if err != nil {
		ux.PrintError("Execution failed: %v", err)
		if result != nil {
			printExecutionSummary(result, time.Since(startTime))
		}
		return err
	}

	duration := time.Since(startTime)
	printExecutionSummary(result, duration)

	if dryRun {
		fmt.Println()
		ux.PrintWarning("DRY-RUN mode - no changes were made")
	}

	return nil
}

func printExecutionSummary(result *executor.Result, duration time.Duration) {
	ux.PrintHeader("Execution Summary")

	rows := [][]string{
		{"📊 Total phases:", ux.Success(fmt.Sprintf("%d", result.TotalPhases))},
		{"✓ Completed phases:", ux.Success(fmt.Sprintf("%d", result.CompletedPhases))},
		{"→ Executed phases:", ux.Info(fmt.Sprintf("%d", result.ExecutedPhases))},
		{ux.Success("✓") + " Successful fixes:", ux.Success(fmt.Sprintf("%d", result.SuccessfulFixes))},
		{ux.Error("✗") + " Failed fixes:", ux.Error(fmt.Sprintf("%d", result.FailedFixes))},
		{"💰 Total cost:", ux.FormatCost(result.TotalCost)},
		{"🎫 Total tokens:", ux.FormatTokens(result.TotalTokens)},
		{"⏱  Duration:", ux.FormatDuration(duration)},
	}

	if result.SuccessfulFixes > 0 {
		avgCost := result.TotalCost / float64(result.SuccessfulFixes)
		avgTokens := result.TotalTokens / result.SuccessfulFixes
		rows = append(rows, []string{
			"📈 Average per fix:",
			fmt.Sprintf("%s (%s tokens)", ux.FormatCost(avgCost), ux.FormatTokens(avgTokens)),
		})
	}

	ux.PrintSummaryTable(rows)

	fmt.Println()
	fmt.Printf("📊 State saved to: %s\n", result.StatePath)
}

func createProvider(name string, model string) (provider.Provider, error) {
	config := provider.Config{
		Name:        name,
		Model:       model,
		Temperature: 0.2,
	}

	// Check if this is a provider preset (groq, ollama, etc.)
	if preset, ok := provider.ProviderPresets[name]; ok {
		// Use OpenAI provider with custom base URL
		config.BaseURL = preset.BaseURL

		// Use preset's default model if no model specified
		if config.Model == "" {
			config.Model = preset.DefaultModel
		}

		return openai.New(config)
	}

	switch name {
	case "claude":
		return claude.New(config)
	case "openai":
		return openai.New(config)
	default:
		return nil, fmt.Errorf("unknown provider: %s (available: claude, openai, groq, together, anyscale, perplexity, ollama, lmstudio, openrouter)", name)
	}
}

// buildConfidenceConfig creates a confidence.Config from config file and CLI flags
// CLI flags override config file values
func buildConfidenceConfig(cfg *config.Config) (confidence.Config, error) {
	// Start with config file settings
	confidenceConf := cfg.Confidence.ToConfidenceConfig()

	// CLI flags override config file
	if cmd, _, err := rootCmd.Find(os.Args[1:]); err == nil && cmd.Flags().Changed("enable-confidence") {
		confidenceConf.Enabled = confidenceEnabled
	}

	if cmd, _, err := rootCmd.Find(os.Args[1:]); err == nil && cmd.Flags().Changed("min-confidence") {
		if minConfidence < 0.0 || minConfidence > 1.0 {
			return confidenceConf, fmt.Errorf("--min-confidence must be between 0.0 and 1.0")
		}
		if minConfidence > 0 {
			// Apply global minimum to all thresholds
			for level := range confidenceConf.Thresholds {
				confidenceConf.Thresholds[level] = minConfidence
			}
			confidenceConf.Default = minConfidence
		}
	}

	if cmd, _, err := rootCmd.Find(os.Args[1:]); err == nil && cmd.Flags().Changed("on-low-confidence") {
		switch onLowConfidence {
		case "skip":
			confidenceConf.OnLowConfidence = confidence.ActionSkip
		case "warn-and-apply":
			confidenceConf.OnLowConfidence = confidence.ActionWarnAndApply
		case "manual-review-file":
			confidenceConf.OnLowConfidence = confidence.ActionManualReviewFile
		default:
			return confidenceConf, fmt.Errorf("invalid --on-low-confidence value: %s (must be: skip, warn-and-apply, manual-review-file)", onLowConfidence)
		}
	}

	if cmd, _, err := rootCmd.Find(os.Args[1:]); err == nil && cmd.Flags().Changed("complexity-threshold") {
		// Parse complexity-threshold flag: "trivial=0.7,low=0.75,..."
		pairs := strings.Split(complexityThreshold, ",")
		for _, pair := range pairs {
			parts := strings.Split(strings.TrimSpace(pair), "=")
			if len(parts) != 2 {
				return confidenceConf, fmt.Errorf("invalid --complexity-threshold format: %s (expected: level=threshold)", pair)
			}
			level := strings.TrimSpace(parts[0])
			thresholdStr := strings.TrimSpace(parts[1])

			threshold, err := strconv.ParseFloat(thresholdStr, 64)
			if err != nil {
				return confidenceConf, fmt.Errorf("invalid threshold value for %s: %s", level, thresholdStr)
			}
			if threshold < 0.0 || threshold > 1.0 {
				return confidenceConf, fmt.Errorf("threshold for %s must be between 0.0 and 1.0", level)
			}

			confidenceConf.Thresholds[level] = threshold
		}
	}

	return confidenceConf, nil
}

var rootCmd *cobra.Command
