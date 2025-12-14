package planfile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const PlanVersion = "1.0"

// LoadPlan reads a plan from a YAML file
func LoadPlan(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan Plan
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan file: %w", err)
	}

	if err := ValidatePlan(&plan); err != nil {
		return nil, fmt.Errorf("invalid plan: %w", err)
	}

	return &plan, nil
}

// SavePlan writes a plan to a YAML file
func SavePlan(plan *Plan, path string) error {
	if err := ValidatePlan(plan); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}

	data, err := yaml.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	return nil
}

// NewPlan creates a new plan with default values
func NewPlan(provider string, totalViolations int) *Plan {
	return &Plan{
		Version: PlanVersion,
		Metadata: PlanMetadata{
			Provider:        provider,
			TotalViolations: totalViolations,
		},
		Phases: make([]Phase, 0),
	}
}

// GetPhaseByID returns a phase by its ID
func (p *Plan) GetPhaseByID(phaseID string) (*Phase, error) {
	for i := range p.Phases {
		if p.Phases[i].ID == phaseID {
			return &p.Phases[i], nil
		}
	}
	return nil, fmt.Errorf("phase not found: %s", phaseID)
}

// GetActivePhases returns phases that are not deferred
func (p *Plan) GetActivePhases() []Phase {
	active := make([]Phase, 0)
	for _, phase := range p.Phases {
		if !phase.Deferred {
			active = append(active, phase)
		}
	}
	return active
}

// GetTotalIncidents returns the total number of incidents across all active phases
func (p *Plan) GetTotalIncidents() int {
	total := 0
	for _, phase := range p.GetActivePhases() {
		for _, violation := range phase.Violations {
			total += violation.IncidentCount
		}
	}
	return total
}

// GetTotalCost returns the estimated total cost for all active phases
func (p *Plan) GetTotalCost() float64 {
	total := 0.0
	for _, phase := range p.GetActivePhases() {
		total += phase.EstimatedCost
	}
	return total
}
