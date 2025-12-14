# Implementation Plan: Interactive Plan → Execute Workflow

## Overview

Add `kantra-ai plan` and `kantra-ai execute` commands to enable AI-powered migration planning with phased execution, progress tracking, and resume capability.

## User Requirements Summary

- **Two modes**: Interactive CLI prompts (--interactive) OR file-based editing
- **AI Analysis**: Risk assessment, grouping explanations, execution order, cost/effort estimates
- **Phase Organization**: Group by category (mandatory/optional/potential) then effort level
- **State Tracking**: Separate .kantra-ai-state.yaml file for execution progress
- **Key Features**: Approve/reject phases, edit plans, execute one phase at a time, resume from failures

## Architecture

### New Package Structure

```
pkg/planner/       # Plan generation, AI analysis, interactive approval
pkg/executor/      # Plan execution, state management, resume logic
pkg/planfile/      # YAML persistence and validation
```

### Extended Provider Interface

```go
type Provider interface {
    // ... existing methods ...
    GeneratePlan(ctx context.Context, req PlanRequest) (*PlanResponse, error)
}
```

## Plan YAML Schema (.kantra-ai-plan.yaml)

```yaml
version: "1.0"
metadata:
  created_at: "2025-12-14T10:30:00Z"
  provider: "claude"
  total_violations: 45

phases:
  - id: "phase-1"
    name: "Critical Mandatory Fixes - High Effort"
    order: 1
    risk: "high"                    # low | medium | high
    category: "mandatory"
    effort_range: [5, 7]
    explanation: |                  # AI explains WHY these are grouped
      These violations require significant refactoring of core APIs.
      Should be done first but requires careful review.

    violations:
      - violation_id: "javax-to-jakarta-001"
        description: "Replace javax.servlet with jakarta.servlet"
        category: "mandatory"
        effort: 7
        incident_count: 23
        incidents:
          - uri: "file:///src/Controller.java"
            line_number: 15
            message: "Replace javax.servlet.http.HttpServlet"

    estimated_cost: 2.45
    estimated_duration_minutes: 15
    deferred: false                 # User can edit to true to skip phase
```

## State YAML Schema (.kantra-ai-state.yaml)

```yaml
version: "1.0"
plan_file: ".kantra-ai-plan.yaml"
started_at: "2025-12-14T11:00:00Z"
updated_at: "2025-12-14T11:15:00Z"

execution_summary:
  total_phases: 3
  completed_phases: 1
  pending_phases: 2
  total_cost: 2.45

phases:
  - phase_id: "phase-1"
    status: "completed"             # pending | in_progress | completed | failed
    started_at: "2025-12-14T11:00:00Z"
    completed_at: "2025-12-14T11:15:00Z"
    fixes_applied: 41
    cost: 2.45

violations:
  "javax-to-jakarta-001":
    status: "completed"
    incidents:
      "file:///src/Controller.java:15":
        status: "completed"
        cost: 0.12
        timestamp: "2025-12-14T11:05:00Z"

last_failure:                       # Track resume point
  phase_id: "phase-1"
  violation_id: "javax-to-jakarta-002"
  incident_uri: "file:///src/model/Order.java"
  error: "AI provider timeout"
```

## CLI Commands

### kantra-ai plan

**Flags:**
- `--analysis` - Path to Konveyor output.yaml (required)
- `--input` - Source code path (required)
- `--provider` - AI provider (default: claude)
- `--output` - Output plan file (default: .kantra-ai-plan.yaml)
- `--interactive` - Enable interactive approval mode
- `--violation-ids` - Filter violations
- `--categories` - Filter categories
- `--max-effort` - Maximum effort level
- `--max-phases` - Hint for number of phases (0=auto)
- `--risk-tolerance` - conservative | balanced | aggressive

**Interactive Mode Flow:**

```
kantra-ai plan --interactive

Loading analysis... ✓ 45 violations
Analyzing violations and generating plan...
✓ Generated 3-phase migration plan

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Phase 1 of 3: Critical Mandatory Fixes - High Effort
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Order:    1
Risk:     🔴 HIGH
Category: mandatory
Effort:   5-7

Why this grouping:
  These violations require significant refactoring of core APIs.
  Group includes javax.* to jakarta.* namespace changes in
  critical paths. Should be done first but requires review.

Violations (2):
  • javax-to-jakarta-001 (23 incidents)
  • javax-to-jakarta-002 (18 incidents)

Estimated cost: $2.45
Estimated time: ~15 minutes

Actions:
  [a] Approve and continue
  [d] Defer (skip this phase)
  [v] View incident details
  [q] Quit and save plan

Choice: a

✓ Phase 1 approved
[... repeat for remaining phases ...]

Plan saved to: .kantra-ai-plan.yaml

Summary:
  Total phases:     3
  Approved:         2
  Deferred:         1
  Estimated cost:   $3.65

Next steps:
  • Execute all:   kantra-ai execute
  • Execute phase: kantra-ai execute --phase phase-1
```

**File-Based Mode (Default):**

```
kantra-ai plan

Loading analysis... ✓ 45 violations
Analyzing violations and generating plan... ✓

Plan saved to: .kantra-ai-plan.yaml

Summary:
  Total phases:     3
  Total violations: 45
  Estimated cost:   $4.30

Next steps:
  • Review plan:   cat .kantra-ai-plan.yaml
  • Edit if needed: vim .kantra-ai-plan.yaml
  • Execute:       kantra-ai execute
```

### kantra-ai execute

**Flags:**
- `--plan` - Plan file path (default: .kantra-ai-plan.yaml)
- `--input` - Source code path (required)
- `--phase` - Execute specific phase (e.g., phase-1)
- `--dry-run` - Preview without applying
- `--git-commit` - Git commit strategy (per-violation | per-incident | at-end)
- `--create-pr` - Create GitHub PRs
- `--verify` - Verification type (build | test)
- `--verify-strategy` - When to verify (per-fix | per-violation | at-end)

**Execution Flow:**

```
kantra-ai execute

Loading plan from .kantra-ai-plan.yaml...
Loading state from .kantra-ai-state.yaml...

Execution Plan:
  Phase 1: Critical Mandatory Fixes (41 incidents)
  Phase 2: Mandatory Medium Effort (12 incidents) [DEFERRED]
  Phase 3: Optional Improvements (8 incidents)

Starting execution...

Phase 1: Critical Mandatory Fixes - High Effort
  ✓ Fixed: Controller.java:15 (cost: $0.12)
  ✓ Fixed: Servlet.java:20 (cost: $0.11)
  [Progress: 2/41]
  ...
```

**Resume from Failure:**

```
kantra-ai execute

Loading plan...
Loading state...

⚠ Previous execution incomplete
  Last failure: phase-1 (violation javax-to-jakarta-002)
  41 incidents completed
  1 incident failed
  85 incidents pending

Resume from failure point? [y/N]: y

Resuming execution...

Phase 1: Critical Mandatory Fixes
  ↻ Retrying failed incident: model/Order.java:12
  ✓ Fixed: model/Order.java:12 (cost: $0.08)
  ...
```

## AI Provider Implementation

### Extend pkg/provider/claude/claude.go

Add `GeneratePlan()` method that:
1. Builds specialized prompt for plan generation
2. Calls Claude API with higher token limit (8192)
3. Parses structured response (JSON array of phases)
4. Calculates cost estimates per phase
5. Returns `PlanResponse` with phases and metadata

**Prompt Structure:**

```
You are a migration planning expert. Analyze violations and create
a phased migration plan.

VIOLATIONS:
[JSON representation of violations]

REQUIREMENTS:
1. Group into 3-5 logical phases
2. Prioritize by: category > effort > risk
3. For each phase provide:
   - Clear name
   - Risk level (low/medium/high)
   - Explanation of WHY grouped together
   - Recommended execution order
   - Violation IDs to include

OUTPUT FORMAT: JSON array of phases
```

## Implementation Components

### pkg/planner/planner.go
- Main orchestrator for plan generation
- Loads violations, filters, calls AI
- Handles both interactive and file-based modes
- Validates and saves plan YAML

### pkg/planner/interactive.go
- Interactive approval flow
- User prompts for approve/defer/view
- Phase detail display
- Progress tracking through phases

### pkg/executor/executor.go
- Main execution engine
- Loads plan and state
- Filters phases to execute
- Orchestrates fixes using existing fixer.Fixer
- Updates state after each fix

### pkg/executor/state.go
- State file management
- Read/write .kantra-ai-state.yaml
- Track phase and incident status
- Provide resume capability

### pkg/planfile/plan.go
- Plan YAML serialization/deserialization
- Schema validation
- Version compatibility

## Error Handling

| Scenario | Handling |
|----------|----------|
| AI timeout during plan generation | Retry once, fallback to simple grouping by category |
| Invalid plan YAML | Validation with detailed error messages |
| State file corruption | Create backup, allow manual recovery |
| Fix failure mid-execution | Mark failed in state, continue remaining |
| User abort (Ctrl+C) | Graceful shutdown, save state |
| Plan manually edited | Validate before execution |

## Implementation Order

### Phase 1: Infrastructure (Days 1-2)
1. Create package structure (planner, executor, planfile)
2. Define data structures (Plan, Phase, ExecutionState types)
3. Implement YAML serialization for plan and state
4. Add validation logic

### Phase 2: Plan Generation (Days 3-4)
5. Extend Provider interface with GeneratePlan
6. Implement Claude.GeneratePlan with prompt
7. Build planner.go core logic
8. Add plan command to main.go (file-based mode)
9. Test plan generation with real violations

### Phase 3: Interactive Mode (Day 5)
10. Implement interactive.go approval flow
11. Add CLI prompts (approve/defer/view)
12. Integrate with plan command (--interactive flag)

### Phase 4: Execution Engine (Days 6-7)
13. Implement executor.go core logic
14. Build state.go for state management
15. Add resume capability
16. Integrate with existing fixer.Fixer
17. Add execute command to main.go

### Phase 5: Testing & Polish (Days 8-9)
18. Unit tests for planner package
19. Unit tests for executor package
20. Integration test: plan → execute → resume
21. Update README and documentation
22. Add example plans to examples/

## Testing Strategy

### Unit Tests
- Plan generation with various violation sets
- Phase grouping logic (category, effort)
- Interactive approval flow
- State management operations
- Resume from different failure points
- YAML serialization/validation

### Integration Tests
- End-to-end: generate plan → execute all → verify state
- Execute specific phase → verify only that phase runs
- Fail mid-phase → resume → complete
- Interactive approval → execute approved only
- Plan with git commits → verify commits created

### Manual Testing
- Large plans (100+ violations) - AI grouping quality
- Network failures - state persistence
- Manual plan editing - validation catches errors
- Deferred phases - correctly skipped

## Backward Compatibility

Existing `kantra-ai remediate` command remains unchanged. Users can:
- Continue using `remediate` for immediate fixes
- Adopt `plan → execute` gradually
- Mix both approaches as needed

**Recommended Usage:**
- **Small migrations (< 20 violations)**: Use `remediate`
- **Large migrations (> 20 violations)**: Use `plan` then `execute`

## Critical Files

### New Files (16 files)
- `pkg/planner/planner.go` - Main plan generation logic
- `pkg/planner/types.go` - Plan data structures
- `pkg/planner/interactive.go` - Interactive approval
- `pkg/planner/grouping.go` - Violation grouping algorithms
- `pkg/executor/executor.go` - Execution engine
- `pkg/executor/state.go` - State management
- `pkg/executor/resume.go` - Resume logic
- `pkg/planfile/plan.go` - Plan YAML I/O
- `pkg/planfile/state.go` - State YAML I/O
- `pkg/planfile/validation.go` - Schema validation
- `pkg/planner/planner_test.go` - Planner tests
- `pkg/executor/executor_test.go` - Executor tests
- `pkg/planfile/plan_test.go` - Plan file tests

### Modified Files (3 files)
- `pkg/provider/interface.go` - Add GeneratePlan method
- `pkg/provider/claude/claude.go` - Implement GeneratePlan
- `cmd/kantra-ai/main.go` - Add plan and execute commands

### Example Files
- `examples/planning-workflow/README.md` - Usage guide
- `examples/planning-workflow/.kantra-ai-plan.yaml` - Example plan
- `examples/planning-workflow/.kantra-ai-state.yaml` - Example state

## Configuration Extensions

Add to `.kantra-ai.yaml`:

```yaml
plan:
  output-file: ".kantra-ai-plan.yaml"
  interactive: false
  max-phases: 5
  risk-tolerance: "balanced"

execute:
  state-file: ".kantra-ai-state.yaml"
  resume-on-failure: true
```

## Success Criteria

- ✅ Generate plans with AI-powered grouping and explanations
- ✅ Interactive mode allows approve/reject per phase
- ✅ File-based mode allows manual YAML editing
- ✅ Execute entire plan or specific phases
- ✅ State tracking persists across runs
- ✅ Resume from any failure point
- ✅ Git commits and PRs work with phased execution
- ✅ Clear error messages and recovery instructions
- ✅ Comprehensive tests covering all workflows
