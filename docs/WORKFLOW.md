# Complete Migration Workflow

This document illustrates the end-to-end workflow from Konveyor analysis through kantra-ai automated remediation to manual developer activities.

## Overview

The migration process follows this high-level flow:

1. **Analyze** - Konveyor analyzes application and identifies violations
2. **Plan** (optional) - AI generates phased migration plan with risk assessment
3. **Automate** - kantra-ai fixes trivial/low/medium complexity violations
4. **Manual** - Developers handle high/expert complexity violations
5. **Verify** - Build/test verification ensures fixes work correctly
6. **Integrate** - Git commits and PR creation for review

```mermaid
flowchart TB
    Start([Application to Migrate]) --> KonveyorAnalyze

    %% ===== KONVEYOR ANALYSIS =====
    subgraph Konveyor["🔍 Konveyor Analysis (konveyor/kantra)"]
        KonveyorAnalyze[Run kantra analyze<br/>--input=./app<br/>--output=./analysis]
        KonveyorOutput[/output.yaml<br/>Violations + Incidents/]

        KonveyorAnalyze --> KonveyorOutput
    end

    %% ===== WORKFLOW DECISION =====
    KonveyorOutput --> WorkflowChoice{Migration Size?}
    WorkflowChoice -->|< 20 violations<br/>Quick Fixes| DirectRemediate[Direct Remediation]
    WorkflowChoice -->|20+ violations<br/>Large-Scale| PhasedWorkflow[Phased Migration]

    %% ===== DIRECT REMEDIATION PATH =====
    DirectRemediate --> Remediate

    %% ===== PHASED MIGRATION PATH =====
    subgraph PhasedPlan["📋 Planning Phase (kantra-ai plan)"]
        PlanCmd[kantra-ai plan<br/>--analysis=output.yaml<br/>--provider=claude]
        AIPlanning[AI analyzes violations<br/>Groups by complexity<br/>Assesses risk]
        PlanYAML[/.kantra-ai-plan.yaml<br/>Machine-readable plan/]
        PlanHTML[/.kantra-ai-plan.html<br/>Interactive report/]
        ReviewPlan{Review Plan<br/>HTML Report}
        ApprovePlan[Developer Approves]

        PlanCmd --> AIPlanning
        AIPlanning --> PlanYAML
        AIPlanning --> PlanHTML
        PlanYAML --> ReviewPlan
        PlanHTML --> ReviewPlan
        ReviewPlan -->|Approve| ApprovePlan
        ReviewPlan -->|Revise| PlanCmd
    end

    PhasedWorkflow --> PlanCmd
    ApprovePlan --> ExecuteCmd

    %% ===== REMEDIATION/EXECUTION =====
    subgraph RemediationEngine["🤖 Automated Remediation (kantra-ai remediate/execute)"]
        Remediate[kantra-ai remediate]
        ExecuteCmd[kantra-ai execute<br/>--plan=.kantra-ai-plan.yaml]

        LoadViolations[Load Violations<br/>from Analysis]
        FilterViolations{Apply Filters<br/>category, effort,<br/>violation-ids}

        Remediate --> LoadViolations
        ExecuteCmd --> LoadViolations
        LoadViolations --> FilterViolations

        %% Complexity-based routing
        FilterViolations --> ClassifyComplexity[Classify by<br/>Migration Complexity]

        ClassifyComplexity --> TrivialLow[Trivial/Low<br/>Complexity]
        ClassifyComplexity --> Medium[Medium<br/>Complexity]
        ClassifyComplexity --> HighExpert[High/Expert<br/>Complexity]

        %% Processing for each complexity level
        TrivialLow --> BatchProcess[Batch Processing<br/>Group similar violations<br/>50-80% cost reduction]
        Medium --> BatchProcess
        HighExpert --> ManualReviewFlag[Flag for<br/>Manual Review]

        BatchProcess --> AIFix[AI Provider<br/>Claude/OpenAI<br/>Generate Fix]

        AIFix --> ConfidenceCheck{Confidence<br/>Filtering<br/>Enabled?}

        ConfidenceCheck -->|No| ApplyFix[Apply Fix<br/>to File]
        ConfidenceCheck -->|Yes| CheckThreshold{Confidence >=<br/>Threshold?}

        CheckThreshold -->|Yes<br/>0.70+ trivial<br/>0.75+ low<br/>0.80+ medium| ApplyFix
        CheckThreshold -->|No| LowConfAction{on-low-<br/>confidence<br/>action?}

        LowConfAction -->|skip| SkipFix[Skip Fix<br/>Report as Skipped]
        LowConfAction -->|warn-and-apply| WarnApply[Warn + Apply Fix<br/>Log Warning]
        LowConfAction -->|manual-review-file| WriteReview[Write to<br/>ReviewFileName.yaml]

        WarnApply --> ApplyFix

        ApplyFix --> VerifyEnabled{Verification<br/>Enabled?}
        VerifyEnabled -->|No| GitEnabled
        VerifyEnabled -->|Yes| RunVerify[Run Build/Test<br/>per-fix, per-violation,<br/>or at-end]

        RunVerify --> VerifyResult{Passed?}
        VerifyResult -->|Yes| GitEnabled
        VerifyResult -->|No| RevertFix[Revert Fix<br/>Mark as Failed]

        RevertFix --> FailReport

        GitEnabled{Git Integration<br/>Enabled?}
        GitEnabled -->|Yes| CreateCommit[Create Commit<br/>per-violation,<br/>per-incident,<br/>or at-end]
        GitEnabled -->|No| SuccessReport

        CreateCommit --> PREnabled{Create PR?}
        PREnabled -->|Yes| CreatePR[Create GitHub PR<br/>with Fix Summary]
        PREnabled -->|No| SuccessReport

        CreatePR --> SuccessReport[Success Report<br/>Cost, Tokens,<br/>Fixes Applied]

        SkipFix --> SkipReport
        WriteReview --> ReviewReport
    end

    %% ===== MANUAL DEVELOPMENT ACTIVITIES =====
    subgraph ManualDev["👨‍💻 Manual Development Activities"]
        ManualReviewFlag --> ReviewQueue[Review Queue]
        WriteReview --> ReviewFile[/ReviewFileName.yaml<br/>Low-confidence fixes/]
        HighExpertList[High/Expert<br/>Violations List<br/>from Plan]

        ReviewQueue --> DevReview[Developer Reviews<br/>Complex Violations]
        ReviewFile --> DevReview
        HighExpertList --> DevReview

        DevReview --> ManualCode[Manual Code Changes<br/>Architectural decisions<br/>Domain expertise required]

        ManualCode --> ManualTest[Manual Testing<br/>Verify complex changes]
        ManualTest --> ManualCommit[Create Commit<br/>Manual fixes]
    end

    %% ===== OUTPUTS & REPORTS =====
    subgraph Outputs["📊 Outputs & Reports"]
        SuccessReport
        SkipReport[Skipped Fixes Report]
        ReviewReport[Manual Review Report]
        FailReport[Failed Fixes Report]
        FinalStats[Final Statistics<br/>Total fixes: X<br/>Automated: Y<br/>Manual: Z<br/>Cost: $A.BC]
    end

    SuccessReport --> FinalStats
    SkipReport --> FinalStats
    ReviewReport --> FinalStats
    FailReport --> FinalStats
    ManualCommit --> FinalStats

    %% ===== COMPLETION =====
    FinalStats --> Complete([Migration Complete])

    %% ===== STYLING =====
    classDef konveyorStyle fill:#e1f5ff,stroke:#0077b6,stroke-width:2px
    classDef planStyle fill:#fff3cd,stroke:#ffc107,stroke-width:2px
    classDef autoStyle fill:#d4edda,stroke:#28a745,stroke-width:2px
    classDef manualStyle fill:#f8d7da,stroke:#dc3545,stroke-width:2px
    classDef outputStyle fill:#e2e3e5,stroke:#6c757d,stroke-width:2px
    classDef decisionStyle fill:#cfe2ff,stroke:#0d6efd,stroke-width:2px

    class Konveyor konveyorStyle
    class PhasedPlan planStyle
    class RemediationEngine autoStyle
    class ManualDev manualStyle
    class Outputs outputStyle
    class WorkflowChoice,ReviewPlan,FilterViolations,ConfidenceCheck,CheckThreshold,LowConfAction,VerifyEnabled,VerifyResult,GitEnabled,PREnabled decisionStyle
```

## Workflow Details

### Phase 1: Konveyor Analysis

Konveyor (kantra) analyzes your application and produces `output.yaml` containing:
- **Violations**: Issues found (e.g., "javax.* → jakarta.*")
- **Incidents**: Specific occurrences (file, line number, code snippet)
- **Metadata**: Category (mandatory/optional), effort level (0-10), migration complexity

### Phase 2: Choose Your Workflow

**Direct Remediation** (< 20 violations):
```bash
kantra-ai remediate --analysis=output.yaml --input=./app
```

**Phased Migration** (20+ violations):
```bash
# Step 1: Generate plan
kantra-ai plan --analysis=output.yaml --input=./app

# Step 2: Review HTML report
# Review .kantra-ai-plan.html in browser

# Step 3: Execute approved plan
kantra-ai execute --plan=.kantra-ai-plan.yaml
```

### Phase 3: Automated Remediation

kantra-ai processes violations based on **migration complexity**:

| Complexity | Auto-Fix Strategy | Default Confidence | Example |
|------------|------------------|-------------------|---------|
| **Trivial** | ✅ Automated (batch) | 0.70 | Package renames: `javax.*` → `jakarta.*` |
| **Low** | ✅ Automated (batch) | 0.75 | Simple API equivalents |
| **Medium** | ✅ Automated (single) | 0.80 | Context-aware changes |
| **High** | ⚠️ Manual review recommended | 0.90 | Architectural changes |
| **Expert** | ❌ Manual review required | 0.95 | Domain expertise needed |

**Confidence Filtering** (optional safety layer):
- AI returns confidence score (0.0-1.0) for each fix
- kantra-ai compares against complexity-based threshold
- **Actions for low confidence**:
  - `skip`: Don't apply fix (safest)
  - `warn-and-apply`: Apply with warning
  - `manual-review-file`: Write to ReviewFileName.yaml

**Batch Processing**:
- Groups similar violations together
- Sends up to 10 incidents per batch to AI
- 50-80% cost reduction, 70-90% faster
- Used for trivial/low/medium complexity

### Phase 4: Manual Development

Developers handle:

1. **High/Expert Complexity Violations**
   - Flagged in migration plan
   - Require architectural decisions
   - Need domain expertise

2. **Low-Confidence Fixes** (if manual-review-file enabled)
   - Written to `ReviewFileName.yaml`
   - Developer reviews AI suggestion
   - Decides to apply, modify, or reject

3. **Failed Automated Fixes**
   - Fixes that failed verification
   - Fixes with errors from AI
   - Edge cases AI couldn't handle

### Phase 5: Verification & Integration

**Build/Test Verification** (optional):
```bash
--verify=build  # Run build after fixes
--verify=test   # Run test suite
--verify-strategy=per-fix      # Verify each fix immediately
--verify-strategy=at-end       # Verify once at the end
```

**Git Integration** (optional):
```bash
--git-commit=per-violation  # One commit per violation type
--git-commit=per-incident   # One commit per incident
--git-commit=at-end        # Single commit for all fixes
--create-pr                # Create GitHub PR automatically
```

## Complexity Level Examples

### Trivial (95%+ AI Success, 0.70 threshold)
```java
// Before
import javax.servlet.HttpServlet;

// After
import jakarta.servlet.HttpServlet;
```
**Characteristics**: Mechanical find/replace, no logic changes

### Low (80%+ AI Success, 0.75 threshold)
```java
// Before
Date date = new Date();

// After
LocalDateTime date = LocalDateTime.now();
```
**Characteristics**: Straightforward API equivalents, well-documented migrations

### Medium (60%+ AI Success, 0.80 threshold)
```java
// Before
Properties props = new Properties();
props.load(new FileInputStream("config.properties"));

// After
Properties props = new Properties();
try (InputStream in = Files.newInputStream(Paths.get("config.properties"))) {
    props.load(in);
}
```
**Characteristics**: Requires context understanding, multiple changes, resource management

### High (30-50% AI Success, 0.90 threshold)
```java
// Requires architectural decisions
// - Should we use reactive patterns?
// - What error handling strategy?
// - How to handle state migration?
```
**Characteristics**: Architectural changes, multiple valid approaches, significant refactoring

**Recommendation**: Manual review required

### Expert (<30% AI Success, 0.95 threshold)
```java
// Requires domain expertise
// - Business logic understanding
// - Performance optimization decisions
// - Security/compliance requirements
// - Integration with proprietary systems
```
**Characteristics**: Domain-specific knowledge, business logic, complex integrations

**Recommendation**: Manual development required

## Configuration Examples

### Maximum Safety (Production)
```yaml
# .kantra-ai.yaml
confidence:
  enabled: true
  on-low-confidence: skip
  complexity-thresholds:
    medium: 0.85  # Higher threshold for medium
    high: 0.95    # Very high for complex
    expert: 0.98  # Near-perfect for expert

verification:
  enabled: true
  type: test
  strategy: per-fix
  fail-fast: true

git:
  commit-strategy: per-violation
  create-pr: true
```

### Rapid Development (POC)
```yaml
# .kantra-ai.yaml
confidence:
  enabled: false  # Maximize automation

verification:
  enabled: false  # Skip verification for speed

git:
  commit-strategy: at-end
```

### Balanced Approach (Recommended)
```yaml
# .kantra-ai.yaml
confidence:
  enabled: true
  on-low-confidence: manual-review-file  # Collect for review

verification:
  enabled: true
  type: build
  strategy: at-end  # Fast verification

git:
  commit-strategy: per-violation
  create-pr: true
```

## Cost & Performance Optimization

**Batch Processing** automatically groups similar violations:

| Metric | Single Processing | Batch Processing (10x) | Savings |
|--------|------------------|----------------------|---------|
| **API Calls** | 100 violations = 100 calls | 100 violations = 10 calls | 90% fewer |
| **Cost** | $10.00 | $2.00-$5.00 | 50-80% |
| **Time** | 15 minutes | 2-5 minutes | 70-90% faster |

**When Batch Processing is Used**:
- Trivial complexity violations
- Low complexity violations
- Medium complexity violations (if similar enough)

**When Single Processing is Used**:
- High complexity violations (if auto-fixed)
- Expert complexity violations (if auto-fixed)
- Violations with unique context requirements

## Best Practices

### 1. Start Conservative
```bash
# First run: Dry-run to preview
kantra-ai remediate --dry-run --enable-confidence

# Second run: Limited scope
kantra-ai remediate --categories=mandatory --max-cost=5.00

# Third run: Full automation
kantra-ai remediate --enable-confidence --verify=build
```

### 2. Use Phased Approach for Large Migrations
```bash
# Step 1: Plan
kantra-ai plan --categories=mandatory

# Step 2: Review plan (HTML report)
# Step 3: Execute phase by phase
kantra-ai execute --plan=.kantra-ai-plan.yaml --phases=1
kantra-ai execute --plan=.kantra-ai-plan.yaml --phases=2
```

### 3. Combine Automation with Manual Review
```bash
# Auto-fix low complexity, flag high complexity for review
kantra-ai remediate \
  --enable-confidence \
  --on-low-confidence=manual-review-file \
  --verify=build

# Review: ReviewFileName.yaml contains low-confidence fixes
# Plan: HTML report shows high/expert violations for manual work
```

### 4. Use Verification for Critical Code
```bash
# Production code: Verify each fix
kantra-ai remediate \
  --verify=test \
  --verify-strategy=per-fix \
  --fail-fast

# Test code: Verify at end (faster)
kantra-ai remediate \
  --input=./tests \
  --verify=build \
  --verify-strategy=at-end
```

## Summary

This workflow enables:
- **Automation**: Handle 60-95% of violations automatically (trivial/low/medium)
- **Safety**: Confidence filtering prevents low-quality automated fixes
- **Efficiency**: Batch processing reduces costs by 50-80%
- **Control**: Manual review for complex violations requiring expertise
- **Quality**: Optional build/test verification ensures fixes work
- **Integration**: Git commits and PR creation for team review

**Result**: Faster, safer, more cost-effective migrations with clear separation between automated fixes and manual development work.
