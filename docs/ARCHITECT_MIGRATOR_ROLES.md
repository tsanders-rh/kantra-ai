# Architect vs Migrator Role Distinction

## Context

From John Matthews:
> We also need to consider if the distinction of Architect and Migrator should be catered to, if that is a real distinction it could be a legit way for us to differentiate for our core market… If this was important then we'd want to get clear in our minds the roles of each and think through if there are ways in the experience to improve for those personas, the angle of having a developer making changes who is not fully trusted and enforcing "motions", "guard rails", etc… that seems to be important

## Role Definitions

### Architect
- **Responsibility**: Migration strategy, risk assessment, approval authority
- **Trust Level**: High - makes decisions about what changes are acceptable
- **Activities**:
  - Review violation reports from Konveyor analysis
  - Design migration approach (phased vs all-at-once)
  - Set confidence thresholds and guard rails
  - Approve/reject AI-generated fixes
  - Define verification requirements (build/test gates)
  - Review high-complexity changes manually

### Migrator
- **Responsibility**: Execution of approved migration plan
- **Trust Level**: Lower - follows architect-approved plan with guard rails
- **Activities**:
  - Execute pre-approved migration phases
  - Monitor progress and handle failures
  - Resume from interruptions
  - Report completion status
  - Escalate when guard rails are hit (low confidence, verification failures)

## kantra-ai vs kai/IDE Plugin

### kantra-ai (CLI Tool)
**Use Case**: Batch migration of large codebases

**Target Personas**:
- Architect: Planning and approving bulk migrations
- Migrator: Executing approved plans at scale

**Key Characteristics**:
- Processes hundreds/thousands of violations
- Operates on entire codebase
- Phased execution over hours/days
- Emphasis on governance, approval, cost control
- State tracking and resume capability
- CI/CD integration potential

**Example Workflow**:
```bash
# Architect: Plan and approve
kantra-ai plan --analysis output.yaml --input /src --interactive
# Reviews 500 violations, approves phases 1-3, defers phase 4

# Migrator: Execute approved plan
kantra-ai execute
# Runs 500 fixes over 2 hours, resumes from failures
```

**Guard Rails Needed**:
- Confidence thresholds prevent unsafe auto-fixes
- Manual review queue for high-complexity changes
- Cost limits prevent runaway AI spending
- Verification gates (build/test) enforce quality
- Audit trail shows who approved what

### kai/IDE Plugin
**Use Case**: Interactive fixing of individual violations during development

**Target Persona**: Developer actively writing/refactoring code

**Key Characteristics**:
- Processes one violation at a time
- Operates in editor context
- Real-time feedback loop (seconds)
- Emphasis on developer productivity, quick iteration
- Integrated with IDE (VSCode, IntelliJ)

**Example Workflow**:
```
1. Developer sees violation in editor gutter
2. Right-click → "Fix with AI"
3. Reviews diff, accepts/rejects
4. Continues coding
```

**Guard Rails Needed**:
- Show confidence score in UI
- Allow per-fix accept/reject (already exists)
- Warn on high-complexity changes
- No batch operations (intentionally)

## Current kantra-ai Features Supporting Role Distinction

### Architect-Oriented Features
✅ **Plan Generation** (`plan` command)
- Review all violations before execution
- See risk levels, cost estimates
- Interactive approval/deferral of phases

✅ **Confidence Threshold Configuration**
- Set global or complexity-based thresholds
- Control what can be auto-applied vs manual review
- `--enable-confidence --min-confidence 0.85`

✅ **Risk Tolerance Settings**
- Conservative/balanced/aggressive planning
- Influences grouping and execution order

✅ **Dry-Run Mode**
- Preview changes without applying
- Architect can validate plan before authorizing

### Migrator-Oriented Features
✅ **Execute Command**
- Run pre-approved plans
- Can't modify plan during execution
- Phase-by-phase progress tracking

✅ **State Tracking**
- Persistent record of what's done
- Resume from failures
- Audit trail of all fixes

✅ **Verification Integration**
- Automatic build/test verification
- Fail-fast on broken builds
- Prevents bad fixes from accumulating

### Guard Rails Already Implemented
✅ **Confidence Filtering**
- Skip fixes below threshold
- Log skipped fixes for manual review
- Stats tracking (applied vs skipped)

✅ **Complexity-Based Thresholds**
- Trivial: 0.70 (accept more)
- Expert: 0.95 (very high bar)
- Architect can override defaults

✅ **Manual Review Markers**
- High/expert complexity flagged
- `IsHighComplexity()` marks violations
- Can be routed to review file

✅ **Cost Limits**
- `--max-cost` prevents runaway spending
- Per-fix cost tracking
- Estimated vs actual cost reporting

## Potential Enhancements for Role Distinction

### 1. Plan Signing and Trust Model
**Problem**: No enforcement that plan hasn't been modified after architect approval

**Solution**:
```bash
# Architect signs plan after approval
kantra-ai approve-plan .kantra-ai-plan.yaml --sign

# Adds to plan file:
metadata:
  approved_by: "alice@company.com"
  approved_at: "2025-12-14T10:00:00Z"
  signature: "sha256:abc123..."  # Hash of plan content

# Migrator execution verifies signature
kantra-ai execute
# Fails if plan modified after signing
```

**Benefits**:
- Cryptographic proof plan hasn't been tampered with
- Clear accountability for who approved what
- Compliance/audit requirement in regulated industries

### 2. Manual Review Queue
**Problem**: Low-confidence fixes currently just skipped - no workflow to get architect approval

**Solution**:
```bash
# During execution, high-complexity fixes → review file
kantra-ai execute --on-low-confidence manual-review-file

# Creates: .kantra-ai-review.yaml
reviews:
  - violation_id: "javax-to-jakarta-001"
    incident: "file:///src/Controller.java:15"
    confidence: 0.72
    complexity: "expert"
    proposed_fix: |
      - import javax.servlet.http.HttpServlet;
      + import jakarta.servlet.http.HttpServlet;
    status: "pending"  # pending | approved | rejected

# Architect reviews and approves
kantra-ai review approve javax-to-jakarta-001

# Migrator re-runs to apply approved fixes
kantra-ai execute --apply-reviewed
```

**Benefits**:
- Architect can approve specific low-confidence fixes
- Clear workflow for edge cases
- Migrator doesn't need judgment calls

### 3. Role-Based Audit Trail
**Problem**: State file doesn't distinguish who did what

**Solution**:
```yaml
# .kantra-ai-state.yaml
metadata:
  plan_created_by: "alice@company.com"      # Architect
  plan_approved_by: "alice@company.com"     # Architect
  executed_by: "bob@company.com"            # Migrator

audit_log:
  - timestamp: "2025-12-14T10:00:00Z"
    actor: "alice@company.com"
    action: "plan_approved"
    details: "Approved phases 1-3, deferred phase 4"

  - timestamp: "2025-12-14T11:00:00Z"
    actor: "bob@company.com"
    action: "execution_started"
    phase: "phase-1"

  - timestamp: "2025-12-14T11:15:00Z"
    actor: "system"
    action: "verification_failed"
    details: "Build failed after fix xyz"
```

**Benefits**:
- Clear accountability trail
- Compliance/audit requirements
- Debugging (who made what decision)

### 4. Policy Enforcement
**Problem**: Migrator can currently override architect's confidence thresholds via CLI flags

**Solution**:
```yaml
# .kantra-ai.yaml (checked into repo, owned by architect)
confidence:
  enabled: true
  min-confidence: 0.85
  locked: true  # NEW: Prevents CLI override

verification:
  enabled: true
  type: "test"
  locked: true  # NEW: Prevents --verify="" override
```

```bash
# Migrator tries to bypass
kantra-ai execute --min-confidence 0.5

# Tool rejects:
Error: Cannot override locked confidence threshold (set by architect in config)
Current threshold: 0.85 (locked)
Attempted override: 0.5
```

**Benefits**:
- Enforces guard rails
- Prevents accidental or intentional bypassing
- Architect maintains control

### 5. Status and Reporting
**Problem**: No easy way for architect to see execution progress without reading state file

**Solution**:
```bash
kantra-ai status

Migration Status:
  Plan: .kantra-ai-plan.yaml (signed by alice@company.com)
  Executed by: bob@company.com

  Phase 1: ✅ Completed (41/41 fixes applied)
  Phase 2: 🔄 In Progress (5/12 fixes applied)
  Phase 3: ⏸️  Pending
  Phase 4: ⏭️  Deferred (awaiting architect approval)

  Guard Rails:
    Confidence threshold: 0.85 (locked)
    Verification: enabled (test)
    Skipped fixes: 3 (low confidence) → See .kantra-ai-review.yaml

  Progress: 46/53 fixes (86.7%)
  Cost: $2.45 / $4.30 estimated
```

**Benefits**:
- Quick visibility for architect
- Clear distinction of roles
- Highlights guard rail triggers

## Differentiation vs Competitors

### Value Proposition
**"kantra-ai lets architects maintain control while enabling migrators to execute safely"**

### Competitive Differentiation
| Feature | kantra-ai | GitHub Copilot | Cursor | Codemod |
|---------|-----------|----------------|--------|---------|
| Batch migration | ✅ Hundreds/thousands of files | ❌ One file at a time | ❌ One file at a time | ✅ Via AST transforms |
| Architect approval workflow | 🟡 Planned | ❌ No | ❌ No | ❌ No |
| Confidence thresholds | ✅ Yes | ❌ No | ❌ No | ❌ No |
| Manual review queue | 🟡 Planned | ❌ No | ❌ No | ❌ No |
| State tracking & resume | ✅ Yes | ❌ No | ❌ No | ❌ No |
| Cost controls | ✅ Yes | ❌ No | ❌ No | N/A |
| Audit trail | 🟡 Basic (planned: enhanced) | ❌ No | ❌ No | ❌ No |
| Policy enforcement | 🟡 Planned | ❌ No | ❌ No | ❌ No |

### Key Differentiator
**Governance-first AI migration tool for enterprises**

- Trust model: Architect sets guard rails, migrator executes within bounds
- Compliance: Audit trail, approval workflow, policy enforcement
- Safety: Confidence thresholds, verification gates, manual review queue
- Scale: Process entire codebase with state tracking and resume

## Open Questions

1. **Market Validation**
   - Do real organizations have distinct architect/migrator roles?
   - Is this more common in large enterprises vs startups?
   - Would this increase adoption by reducing fear of AI mistakes?

2. **Technical Feasibility**
   - How to implement plan signing (GPG keys? OAuth identity?)?
   - Should policy enforcement use file permissions or cryptographic locks?
   - What's the right UX for manual review queue?

3. **Workflow Integration**
   - Should architects approve in CLI or web UI?
   - Integration with existing approval systems (Jira, ServiceNow)?
   - How to handle distributed teams (async approval)?

4. **Compliance Requirements**
   - SOC2/ISO27001 audit trail requirements?
   - Separation of duties enforcement?
   - Immutable audit logs?

## Recommendations

### Short Term (Next 2-3 months)
1. Validate role distinction with 3-5 enterprise customers
2. Implement basic audit trail in state file (who approved, who executed)
3. Add `kantra-ai status` command for architect visibility
4. Document architect vs migrator workflows in README

### Medium Term (3-6 months)
5. Implement manual review queue (`--on-low-confidence manual-review-file`)
6. Add plan signing for tamper detection
7. Implement policy locking (prevent CLI override of config)
8. Add `kantra-ai review` command for approving low-confidence fixes

### Long Term (6-12 months)
9. Web UI for plan review and approval
10. Integration with enterprise approval systems
11. Enhanced audit trail with immutable logs
12. SOC2/compliance certification for audit features

## Conclusion

The architect/migrator distinction is a **strong differentiator** for kantra-ai in the enterprise market. It addresses:

1. **Trust concerns**: Guard rails prevent unsafe AI changes
2. **Governance requirements**: Audit trail and approval workflow
3. **Risk management**: Architect maintains control at scale
4. **Compliance**: Separation of duties for regulated industries

This positions kantra-ai as the **enterprise-grade migration tool** vs consumer-focused coding assistants (Copilot, Cursor) and script-based automation (Codemod).

The distinction also clearly separates kantra-ai (bulk migration orchestration) from kai/IDE plugin (interactive development assistance).
