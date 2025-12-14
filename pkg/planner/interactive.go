package planner

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/ux"
)

// InteractiveApproval presents phases to the user for approval
type InteractiveApproval struct {
	plan   *planfile.Plan
	reader *bufio.Reader
}

// NewInteractiveApproval creates a new interactive approval session
func NewInteractiveApproval(plan *planfile.Plan) *InteractiveApproval {
	return &InteractiveApproval{
		plan:   plan,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Run executes the interactive approval flow
func (ia *InteractiveApproval) Run() error {
	fmt.Println()
	ux.PrintHeader("Interactive Plan Approval")
	fmt.Printf("Generated %d-phase migration plan\n", len(ia.plan.Phases))
	fmt.Println()

	approved := 0
	deferred := 0

	for i := range ia.plan.Phases {
		phase := &ia.plan.Phases[i]

		// Display phase details
		ia.displayPhase(phase, i+1, len(ia.plan.Phases))

		// Get user choice
		for {
			choice := ia.promptChoice()

			switch choice {
			case "a":
				phase.Deferred = false
				approved++
				ux.PrintSuccess("✓ Phase %d approved", phase.Order)
				fmt.Println()
				goto nextPhase

			case "d":
				phase.Deferred = true
				deferred++
				ux.PrintWarning("↷ Phase %d deferred (will be skipped)", phase.Order)
				fmt.Println()
				goto nextPhase

			case "v":
				ia.displayViolationDetails(phase)
				// Continue loop to show choices again

			case "q":
				fmt.Println()
				ux.PrintWarning("Quitting approval process...")
				return ia.showSummary(approved, deferred, i+1)

			default:
				ux.PrintError("Invalid choice. Please enter a, d, v, or q")
			}
		}

	nextPhase:
	}

	return ia.showSummary(approved, deferred, len(ia.plan.Phases))
}

// displayPhase shows phase information
func (ia *InteractiveApproval) displayPhase(phase *planfile.Phase, current, total int) {
	// Header
	fmt.Println(strings.Repeat("━", 70))
	fmt.Printf("Phase %d of %d: %s\n", current, total, phase.Name)
	fmt.Println(strings.Repeat("━", 70))
	fmt.Println()

	// Metadata
	fmt.Printf("Order:    %d\n", phase.Order)
	fmt.Printf("Risk:     %s\n", ia.formatRisk(phase.Risk))
	fmt.Printf("Category: %s\n", phase.Category)
	fmt.Printf("Effort:   %d-%d\n", phase.EffortRange[0], phase.EffortRange[1])
	fmt.Println()

	// Explanation
	fmt.Println("Why this grouping:")
	explanation := strings.TrimSpace(phase.Explanation)
	for _, line := range strings.Split(explanation, "\n") {
		fmt.Printf("  %s\n", strings.TrimSpace(line))
	}
	fmt.Println()

	// Violations summary
	fmt.Printf("Violations (%d):\n", len(phase.Violations))
	for _, v := range phase.Violations {
		fmt.Printf("  • %s (%d incidents)\n", v.ViolationID, v.IncidentCount)
	}
	fmt.Println()

	// Cost and time
	fmt.Printf("Estimated cost: %s\n", ux.FormatCost(phase.EstimatedCost))
	fmt.Printf("Estimated time: ~%d minutes\n", phase.EstimatedDurationMinutes)
	fmt.Println()
}

// displayViolationDetails shows detailed incident information
func (ia *InteractiveApproval) displayViolationDetails(phase *planfile.Phase) {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("Detailed Incident View")
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println()

	for _, v := range phase.Violations {
		fmt.Printf("Violation: %s\n", v.ViolationID)
		fmt.Printf("Description: %s\n", v.Description)
		fmt.Printf("Category: %s | Effort: %d\n", v.Category, v.Effort)
		fmt.Println()

		if len(v.Incidents) > 0 {
			fmt.Printf("Incidents (%d):\n", len(v.Incidents))
			displayCount := len(v.Incidents)
			if displayCount > 10 {
				displayCount = 10
			}

			for i := 0; i < displayCount; i++ {
				incident := v.Incidents[i]
				fmt.Printf("  %d. %s:%d\n", i+1, incident.GetFilePath(), incident.LineNumber)
				if incident.Message != "" {
					fmt.Printf("     %s\n", incident.Message)
				}
			}

			if len(v.Incidents) > 10 {
				fmt.Printf("  ... and %d more incidents\n", len(v.Incidents)-10)
			}
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 70))
	fmt.Println()
}

// promptChoice asks the user for their choice
func (ia *InteractiveApproval) promptChoice() string {
	fmt.Println("Actions:")
	fmt.Println("  [a] Approve and continue")
	fmt.Println("  [d] Defer (skip this phase)")
	fmt.Println("  [v] View incident details")
	fmt.Println("  [q] Quit and save plan")
	fmt.Println()
	fmt.Print("Choice: ")

	input, err := ia.reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(input))
}

// showSummary displays the final summary
func (ia *InteractiveApproval) showSummary(approved, deferred, reviewed int) error {
	fmt.Println()
	ux.PrintHeader("Approval Summary")

	rows := [][]string{
		{"Total phases:", fmt.Sprintf("%d", len(ia.plan.Phases))},
		{"Reviewed:", ux.Success(fmt.Sprintf("%d", reviewed))},
		{"Approved:", ux.Success(fmt.Sprintf("%d", approved))},
		{"Deferred:", ux.FormatWarning(fmt.Sprintf("%d", deferred))},
	}

	// Calculate estimated cost for approved phases only
	approvedCost := 0.0
	for _, phase := range ia.plan.Phases {
		if !phase.Deferred {
			approvedCost += phase.EstimatedCost
		}
	}

	if approved > 0 {
		rows = append(rows, []string{"Estimated cost:", ux.FormatCost(approvedCost)})
	}

	ux.PrintSummaryTable(rows)

	return nil
}

// formatRisk returns a colored risk indicator
func (ia *InteractiveApproval) formatRisk(risk planfile.RiskLevel) string {
	switch risk {
	case planfile.RiskHigh:
		return ux.Error("🔴 HIGH")
	case planfile.RiskMedium:
		return ux.FormatWarning("🟡 MEDIUM")
	case planfile.RiskLow:
		return ux.Success("🟢 LOW")
	default:
		return string(risk)
	}
}
