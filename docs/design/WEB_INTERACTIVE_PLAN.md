# Web-Based Interactive Plan Approval

## Overview

A web-based UI for interactive plan approval that replaces the current CLI-based interactive mode with a rich, visual experience. This provides better visualization, easier navigation, and more powerful features for reviewing and approving migration plans.

## Motivation

The current CLI interactive mode (`kantra-ai plan --interactive`) has several limitations:

- **Linear flow**: Must review phases sequentially, can't easily go back
- **Limited visualization**: No charts, graphs, or syntax highlighting
- **Poor for complex plans**: Difficult to navigate 10+ phases in terminal
- **No preview**: Can't see actual code diffs before approving
- **Single user**: Can't share with team for collaborative review
- **Static**: Can't reorder or modify phases easily

A web-based UI addresses all these limitations while providing a better user experience.

## Architecture

### Option 1: Local Web Server (Recommended)

```bash
kantra-ai plan --analysis=output.yaml --interactive-web
```

**Flow**:
1. Generate plan + HTML report as normal
2. Start local web server (`http://localhost:8080`)
3. Auto-open browser to web UI
4. User interacts with plan (approve/defer/reorder)
5. Decisions saved to `.kantra-ai-plan.yaml`
6. User can execute from web UI or return to CLI

**Benefits**:
- Full integration with kantra-ai
- Can trigger execution directly from web
- Real-time updates during execution
- Can call AI for sample previews
- Saves state automatically

### Option 2: Enhanced Static HTML

Generate interactive HTML with embedded state management:
- Decisions saved to browser localStorage
- Export button to download modified plan YAML
- User imports modified plan to kantra-ai for execution

**Benefits**:
- Simpler implementation (no server needed)
- Can be shared as file
- Works offline

**Drawbacks**:
- Less integrated
- Can't execute directly
- No real-time updates

## User Interface Design

### Dashboard View

```
┌─────────────────────────────────────────────────────────────┐
│ kantra-ai Migration Plan              [Execute] [Save] [✕] │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  📊 Dashboard                                               │
│  ┌──────────────────┬──────────────────┬─────────────────┐ │
│  │ Total Phases     │ Total Violations │ Estimated Cost  │ │
│  │      5           │       168        │    $4.50        │ │
│  └──────────────────┴──────────────────┴─────────────────┘ │
│                                                              │
│  Progress: [████████████░░░░░░░░] 60% (3 of 5 approved)    │
│                                                              │
│  Complexity Distribution:                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ ████████████ Trivial/Low (95 violations, 89%)       │   │
│  │ ██ Medium (12 violations, 9%)                       │   │
│  │ █ High/Expert (3 violations, 2%)                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━   │
│                                                              │
│  Phase 1: Critical Mandatory Fixes      ✓ Approved         │
│  ┌────────────────────────────────────────────────────┐    │
│  │ 🔴 HIGH RISK  •  mandatory  •  Effort: 5-7         │    │
│  │                                                     │    │
│  │ 23 violations, 456 incidents                       │    │
│  │ Cost: $1.20  •  ~15 minutes                        │    │
│  │                                                     │    │
│  │ Why this grouping:                                 │    │
│  │ High effort mandatory fixes requiring core API     │    │
│  │ refactoring. Should be done first.                 │    │
│  │                                                     │    │
│  │ [View Details ▼] [View Code Diffs] [Modify]       │    │
│  └────────────────────────────────────────────────────┘    │
│  [✓ Approve] [↷ Defer] [⚙️ Settings]                      │
│                                                              │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━   │
│                                                              │
│  Phase 2: Logger Migration              ⏸ Pending          │
│  ┌────────────────────────────────────────────────────┐    │
│  │ 🟡 MEDIUM RISK  •  optional  •  Effort: 3-5        │    │
│  │ [Click to expand...]                                │    │
│  └────────────────────────────────────────────────────┘    │
│  [✓ Approve] [↷ Defer]                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Phase Detail View (Expanded)

```
┌─────────────────────────────────────────────┐
│ Phase 1: Critical Mandatory Fixes           │
│ [Overview] [Violations] [Code Diffs] [AI]   │  ← Tabs
├─────────────────────────────────────────────┤
│                                              │
│ Violations (23):                            │
│ ┌─────────────────────────────────────────┐ │
│ │ ☑ javax-to-jakarta-001 (156 incidents)  │ │  ← Checkboxes to filter
│ │   🟢 Trivial • 95% AI confidence        │ │
│ │   [Preview Sample Fix]                   │ │
│ │                                          │ │
│ │ ☑ servlet-api-upgrade (12 incidents)    │ │
│ │   🟡 Medium • 78% AI confidence         │ │
│ │   [Preview Sample Fix]                   │ │
│ │                                          │ │
│ │ ☐ custom-auth-refactor (1 incident)     │ │  ← Unchecked = exclude
│ │   🔴 Expert • Manual review required    │ │
│ │   💡 Recommend using kai extension       │ │
│ └─────────────────────────────────────────┘ │
│                                              │
│ Selected: 22 violations ($1.15)             │
│ Excluded: 1 violation (manual)              │
│                                              │
│ Bulk Actions:                                │
│ [Select All] [Deselect All]                 │
│ [Select by Complexity: Trivial/Low/Medium]  │
│ [Deselect High/Expert]                       │
└─────────────────────────────────────────────┘
```

### Interactive Code Diff View

```
┌─────────────────────────────────────────────┐
│ src/Controller.java:15                       │
│ [File View] [Diff View] [AI Explanation]    │
├─────────────────────────────────────────────┤
│                                              │
│  12 │ package com.example.app;              │
│  13 │                                        │
│  14 │ public class UserController {         │
│  15 │ - import javax.servlet.HttpServlet;   │  ← Red background
│  15 │ + import jakarta.servlet.HttpServlet; │  ← Green background
│  16 │   import java.util.*;                 │
│  17 │                                        │
│                                              │
│ AI Confidence: ████████░░ 95%               │
│                                              │
│ Explanation:                                 │
│ Simple package rename from javax.* to       │
│ jakarta.*. This is a mechanical change      │
│ required for Jakarta EE 9+ compatibility.   │
│                                              │
│ Risk: Low - Straightforward import change   │
│                                              │
│ [< Previous Incident] [Next Incident >]     │
│ [View Full File] [Open in kai Extension]    │
└─────────────────────────────────────────────┘
```

### Live Execution View

```
┌─────────────────────────────────────────────┐
│ Executing Phase 1 of 3...                   │
├─────────────────────────────────────────────┤
│                                              │
│ [████████████████░░░░] 78% (156/200)        │
│                                              │
│ Recent activity:                             │
│ ✓ Fixed src/Controller.java ($0.02, 0.98)  │
│ ✓ Fixed src/Filter.java ($0.02, 0.95)      │
│ ⚠ Skipped src/Complex.java (conf: 0.65)     │
│ ⏳ Processing src/Service.java...           │
│                                              │
│ Statistics:                                  │
│ Success: 154  •  Skipped: 2  •  Failed: 0   │
│                                              │
│ Cost: $1.56 / $1.80 estimated               │
│ Time: 8 min / ~15 min estimated             │
│                                              │
│ [Pause] [Cancel] [View Detailed Logs]       │
└─────────────────────────────────────────────┘
```

### Execution Summary View

```
┌─────────────────────────────────────────────┐
│ ✓ Execution Complete                        │
├─────────────────────────────────────────────┤
│                                              │
│ Phases executed: 3 of 5                     │
│ Phases deferred: 2 (use kai extension)      │
│                                              │
│ Results:                                     │
│ ✓ Success:  234 incidents fixed             │
│ ⚠ Skipped:   12 incidents (low confidence)  │
│ ✕ Failed:     2 incidents (API errors)      │
│                                              │
│ Total cost:   $4.23 (estimated: $4.50)      │
│ Total time:   32 minutes (estimated: 35)    │
│                                              │
│ Next steps:                                  │
│ • Review 12 skipped fixes in ReviewFile.yaml│
│ • Fix 2 failed incidents manually            │
│ • Use kai extension for deferred phases      │
│   (23 high/expert violations)                │
│                                              │
│ [View Detailed Report] [Create PR]          │
│ [Export Summary] [Close]                     │
└─────────────────────────────────────────────┘
```

## Key Features

### 1. Visual Dashboard
- **Metrics cards**: Total phases, violations, costs, time estimates
- **Progress bars**: Visual approval progress
- **Charts**:
  - Pie chart for complexity distribution
  - Bar chart for violations by category
  - Risk heat map
- **Real-time updates**: Statistics update as user approves/defers

### 2. Rich Phase Management

**Expand/Collapse**:
- All phases collapsed by default
- Click to expand and see details
- Keyboard shortcuts (Space to expand, arrows to navigate)

**Approve/Defer**:
- Checkboxes or toggle buttons
- Visual state indicators (✓ Approved, ↷ Deferred, ⏸ Pending)
- Bulk actions (Approve all, Defer all high-risk)

**Reordering**:
- Drag and drop to reorder phases
- Visual feedback during drag
- Auto-saves changes

### 3. Granular Violation Filtering

Within each phase, allow users to:
- Select/deselect specific violations
- Filter by complexity level
- Filter by confidence threshold
- See cost/time impact of selections in real-time

### 4. Code Diff Viewer

**Features**:
- Syntax highlighting
- Side-by-side or unified diff view
- Navigate between incidents (Previous/Next)
- Jump to specific line numbers
- View full file context
- Copy code snippets

**Integration**:
- "Open in kai Extension" button
- Link to original file in IDE
- Download modified file for preview

### 5. AI Sample Preview

**Live API Preview**:
```
┌─────────────────────────────────────────────┐
│ Preview Sample AI Fixes                      │
├─────────────────────────────────────────────┤
│                                              │
│ Generate 3 sample fixes from this phase     │
│ to preview AI quality before approving?     │
│                                              │
│ Cost: ~$0.03  •  Time: ~10 seconds          │
│ Provider: Claude Sonnet 4                    │
│                                              │
│ Samples: [3▼] (1-10 available)              │
│                                              │
│ [Generate Samples] [Cancel]                  │
└─────────────────────────────────────────────┘
```

**After generation**:
- Show actual AI-generated fixes
- Display confidence scores
- Show AI reasoning/explanation
- Option to approve based on sample quality

### 6. Live Execution

**WebSocket-based updates**:
- Real-time progress bars
- Activity stream (latest fixes applied)
- Running cost/time counters
- Success/skip/fail counts

**Controls**:
- Pause/resume execution
- Cancel with rollback option
- View detailed logs in separate panel
- Export execution log

### 7. Collaborative Features

**Share for Review** (future):
- Generate shareable link
- Team members can comment on phases
- Approval voting
- Export with annotations

### 8. Enhanced Warnings

**High-Risk Phase Alerts**:
```
⚠️  WARNING: HIGH RISK PHASE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
This phase includes complex architectural changes that may
require manual review or modification.

Recommendations:
• Review code diffs before approving
• Consider using kai + VS Code extension for manual review
• Have a backup/branch before executing
• Plan for additional testing time

Risk factors:
• Contains 12 "High" complexity violations
• Contains 3 "Expert" complexity violations
• Affects core authentication logic
• May require domain expertise

[View Detailed Risk Analysis]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 9. Settings Panel

```
┌─────────────────────────────────────────────┐
│ Execution Settings                           │
├─────────────────────────────────────────────┤
│                                              │
│ Confidence Filtering:                        │
│ ☑ Enable confidence threshold filtering     │
│   On low confidence: [Skip ▼]               │
│                                              │
│ Verification:                                │
│ ☑ Run build after fixes                     │
│   Strategy: [at-end ▼]                      │
│   Fail fast: ☑                              │
│                                              │
│ Git Integration:                             │
│ ☑ Create commits                            │
│   Strategy: [per-violation ▼]               │
│ ☐ Create pull request                       │
│                                              │
│ Batch Processing:                            │
│ ☑ Enable batch processing                   │
│   Batch size: [10 ▼]                        │
│   Parallelism: [4 ▼]                        │
│                                              │
│ [Save] [Reset to Defaults]                   │
└─────────────────────────────────────────────┘
```

## Implementation Plan

### Phase 1: MVP (1-2 weeks)

**Core Functionality**:
- Local web server (Go `net/http`)
- Single HTML page with embedded CSS/JS
- Display all phases (expand/collapse)
- Approve/Defer toggles
- Save button → writes to `.kantra-ai-plan.yaml`
- Execute button → runs execution with live updates

**Tech Stack**:
- Backend: Go standard library
- Frontend: Vanilla JavaScript (or lightweight framework)
- Styling: Tailwind CSS or simple Bootstrap
- No build step required

**API Endpoints**:
```
GET  /                    - Serve HTML UI
GET  /api/plan            - Get full plan data
POST /api/phase/approve   - Approve phase
POST /api/phase/defer     - Defer phase
POST /api/plan/save       - Save plan to YAML
POST /api/execute/start   - Start execution
GET  /api/execute/status  - Get execution status
WS   /ws                  - WebSocket for live updates
```

### Phase 2: Enhanced Visualization (1 week)

**Add**:
- Dashboard with metrics cards
- Progress bars and charts
- Visual complexity distribution
- Better styling and layout
- Responsive design for mobile/tablet

**Libraries**:
- Chart.js for visualizations
- Font Awesome for icons

### Phase 3: Code Diff Viewer (1 week)

**Add**:
- Syntax-highlighted diffs
- Navigate between incidents
- View full file context
- Download/export capabilities

**Libraries**:
- Prism.js or Highlight.js for syntax highlighting
- diff2html for diff rendering

### Phase 4: Advanced Features (2 weeks)

**Add**:
- Drag-and-drop phase reordering
- Granular violation filtering within phases
- AI sample preview (live API calls)
- Settings panel for execution options
- Enhanced risk warnings
- Export/import capabilities

### Phase 5: Polish & Optimization (1 week)

**Add**:
- Keyboard shortcuts
- Accessibility improvements (ARIA labels, keyboard navigation)
- Better error handling and user feedback
- Loading states and animations
- Help/documentation tooltips
- Performance optimization for large plans

## Backend Implementation

### Server Structure

```go
// pkg/web/server.go
package web

import (
    "embed"
    "net/http"
    "github.com/gorilla/websocket"
    "github.com/tsanders/kantra-ai/pkg/planfile"
    "github.com/tsanders/kantra-ai/pkg/executor"
)

//go:embed static/*
var staticFiles embed.FS

type PlanServer struct {
    plan     *planfile.Plan
    planPath string
    addr     string
    executor *executor.Executor
    clients  map[*websocket.Conn]bool
}

func NewPlanServer(plan *planfile.Plan, planPath string) *PlanServer {
    return &PlanServer{
        plan:     plan,
        planPath: planPath,
        addr:     "localhost:8080",
        clients:  make(map[*websocket.Conn]bool),
    }
}

func (s *PlanServer) Start() error {
    // Static files
    http.Handle("/static/", http.FileServer(http.FS(staticFiles)))

    // API endpoints
    http.HandleFunc("/", s.handleIndex)
    http.HandleFunc("/api/plan", s.handleGetPlan)
    http.HandleFunc("/api/phase/approve", s.handleApprovePhase)
    http.HandleFunc("/api/phase/defer", s.handleDeferPhase)
    http.HandleFunc("/api/plan/save", s.handleSavePlan)
    http.HandleFunc("/api/execute/start", s.handleExecuteStart)
    http.HandleFunc("/api/execute/status", s.handleExecuteStatus)
    http.HandleFunc("/ws", s.handleWebSocket)

    fmt.Printf("🌐 Opening web interface at http://%s\n", s.addr)
    openBrowser("http://" + s.addr)

    return http.ListenAndServe(s.addr, nil)
}

func (s *PlanServer) handleGetPlan(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(s.plan)
}

func (s *PlanServer) handleApprovePhase(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PhaseID string `json:"phase_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Find and approve phase
    for i := range s.plan.Phases {
        if s.plan.Phases[i].ID == req.PhaseID {
            s.plan.Phases[i].Deferred = false
            break
        }
    }

    w.WriteHeader(http.StatusOK)
}

func (s *PlanServer) handleSavePlan(w http.ResponseWriter, r *http.Request) {
    if err := planfile.SavePlan(s.plan, s.planPath); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}

func (s *PlanServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }

    s.clients[conn] = true

    // Handle WebSocket messages
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            delete(s.clients, conn)
            conn.Close()
            break
        }
    }
}

func (s *PlanServer) broadcastUpdate(msg ExecutionUpdate) {
    data, _ := json.Marshal(msg)
    for client := range s.clients {
        client.WriteMessage(websocket.TextMessage, data)
    }
}
```

### WebSocket Message Types

```go
type ExecutionUpdate struct {
    Type    string      `json:"type"`    // "progress", "incident", "complete", "error"
    Data    interface{} `json:"data"`
}

type ProgressUpdate struct {
    Phase       int     `json:"phase"`
    Total       int     `json:"total"`
    Incident    int     `json:"incident"`
    TotalInc    int     `json:"total_incidents"`
    Percentage  float64 `json:"percentage"`
}

type IncidentUpdate struct {
    ViolationID string  `json:"violation_id"`
    FilePath    string  `json:"file_path"`
    Status      string  `json:"status"`     // "success", "skipped", "failed"
    Confidence  float64 `json:"confidence"`
    Cost        float64 `json:"cost"`
    Error       string  `json:"error,omitempty"`
}
```

## Frontend Implementation

### HTML Structure

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>kantra-ai Migration Plan</title>
    <link rel="stylesheet" href="/static/css/styles.css">
</head>
<body>
    <div id="app">
        <header>
            <h1>kantra-ai Migration Plan</h1>
            <div class="actions">
                <button id="save-btn">Save</button>
                <button id="execute-btn" class="primary">Execute</button>
            </div>
        </header>

        <div class="dashboard">
            <div class="metric-card">
                <div class="metric-value" id="total-phases">0</div>
                <div class="metric-label">Total Phases</div>
            </div>
            <div class="metric-card">
                <div class="metric-value" id="total-violations">0</div>
                <div class="metric-label">Total Violations</div>
            </div>
            <div class="metric-card">
                <div class="metric-value" id="estimated-cost">$0.00</div>
                <div class="metric-label">Estimated Cost</div>
            </div>
        </div>

        <div id="progress-bar" class="hidden">
            <div class="progress-fill" style="width: 0%"></div>
            <span class="progress-text">0% (0 of 0 approved)</span>
        </div>

        <div id="phases-container"></div>

        <div id="execution-view" class="hidden">
            <!-- Live execution updates go here -->
        </div>
    </div>

    <script src="/static/js/app.js"></script>
</body>
</html>
```

### JavaScript Application

```javascript
// static/js/app.js
class PlanApp {
    constructor() {
        this.plan = null;
        this.ws = null;
        this.init();
    }

    async init() {
        await this.loadPlan();
        this.render();
        this.attachEventListeners();
    }

    async loadPlan() {
        const response = await fetch('/api/plan');
        this.plan = await response.json();
    }

    render() {
        this.renderDashboard();
        this.renderPhases();
        this.updateProgress();
    }

    renderDashboard() {
        document.getElementById('total-phases').textContent = this.plan.phases.length;
        document.getElementById('total-violations').textContent =
            this.plan.phases.reduce((sum, p) => sum + p.violations.length, 0);
        document.getElementById('estimated-cost').textContent =
            this.formatCost(this.plan.phases.reduce((sum, p) => sum + p.estimated_cost, 0));
    }

    renderPhases() {
        const container = document.getElementById('phases-container');
        container.innerHTML = this.plan.phases.map(phase => this.renderPhase(phase)).join('');
    }

    renderPhase(phase) {
        const status = phase.deferred ? 'deferred' : 'approved';
        const riskIcon = this.getRiskIcon(phase.risk);

        return `
            <div class="phase ${status}" data-phase-id="${phase.id}">
                <div class="phase-header">
                    <h3>${phase.name}</h3>
                    <span class="status-badge">${status}</span>
                </div>
                <div class="phase-meta">
                    <span class="risk ${phase.risk}">${riskIcon} ${phase.risk.toUpperCase()}</span>
                    <span>${phase.category}</span>
                    <span>Effort: ${phase.effort_range[0]}-${phase.effort_range[1]}</span>
                </div>
                <div class="phase-stats">
                    <span>${phase.violations.length} violations</span>
                    <span>${this.formatCost(phase.estimated_cost)}</span>
                    <span>~${phase.estimated_duration_minutes} minutes</span>
                </div>
                <p class="phase-explanation">${phase.explanation}</p>
                <div class="phase-actions">
                    <button class="approve-btn" onclick="app.approvePhase('${phase.id}')">
                        ✓ Approve
                    </button>
                    <button class="defer-btn" onclick="app.deferPhase('${phase.id}')">
                        ↷ Defer
                    </button>
                    <button class="view-btn" onclick="app.viewDetails('${phase.id}')">
                        View Details
                    </button>
                </div>
            </div>
        `;
    }

    async approvePhase(phaseId) {
        await fetch('/api/phase/approve', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ phase_id: phaseId })
        });

        // Update local state
        const phase = this.plan.phases.find(p => p.id === phaseId);
        phase.deferred = false;

        this.render();
    }

    async deferPhase(phaseId) {
        await fetch('/api/phase/defer', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ phase_id: phaseId })
        });

        // Update local state
        const phase = this.plan.phases.find(p => p.id === phaseId);
        phase.deferred = true;

        this.render();
    }

    async savePlan() {
        await fetch('/api/plan/save', { method: 'POST' });
        alert('Plan saved successfully!');
    }

    async executePhases() {
        // Start execution
        await fetch('/api/execute/start', { method: 'POST' });

        // Show execution view
        document.getElementById('execution-view').classList.remove('hidden');
        document.getElementById('phases-container').classList.add('hidden');

        // Connect WebSocket for live updates
        this.connectWebSocket();
    }

    connectWebSocket() {
        this.ws = new WebSocket(`ws://${window.location.host}/ws`);

        this.ws.onmessage = (event) => {
            const update = JSON.parse(event.data);
            this.handleExecutionUpdate(update);
        };
    }

    handleExecutionUpdate(update) {
        switch (update.type) {
            case 'progress':
                this.updateExecutionProgress(update.data);
                break;
            case 'incident':
                this.addIncidentUpdate(update.data);
                break;
            case 'complete':
                this.showExecutionSummary(update.data);
                break;
            case 'error':
                this.showExecutionError(update.data);
                break;
        }
    }

    updateProgress() {
        const approved = this.plan.phases.filter(p => !p.deferred).length;
        const total = this.plan.phases.length;
        const percentage = (approved / total) * 100;

        const progressBar = document.querySelector('.progress-fill');
        const progressText = document.querySelector('.progress-text');

        progressBar.style.width = `${percentage}%`;
        progressText.textContent = `${Math.round(percentage)}% (${approved} of ${total} approved)`;

        document.getElementById('progress-bar').classList.remove('hidden');
    }

    attachEventListeners() {
        document.getElementById('save-btn').addEventListener('click', () => this.savePlan());
        document.getElementById('execute-btn').addEventListener('click', () => this.executePhases());
    }

    formatCost(cost) {
        return `$${cost.toFixed(2)}`;
    }

    getRiskIcon(risk) {
        const icons = {
            'high': '🔴',
            'medium': '🟡',
            'low': '🟢'
        };
        return icons[risk] || '⚪';
    }
}

// Initialize app
const app = new PlanApp();
```

## Benefits Over CLI Interactive

1. **Better Visualization**
   - Charts and graphs for complexity distribution
   - Visual progress indicators
   - Syntax-highlighted code diffs
   - Color-coded risk indicators

2. **Easier Navigation**
   - Click between phases (non-linear)
   - Expand/collapse for overview
   - Jump to specific violations
   - Drag-and-drop reordering

3. **More Control**
   - Granular violation filtering
   - Preview sample AI fixes before approving
   - Modify execution settings on the fly
   - Pause/resume execution

4. **Collaborative**
   - Share URL with team members
   - Export plan with annotations
   - View together on shared screen
   - Future: Multi-user approval voting

5. **Better for Complex Plans**
   - Handle 10+ phases easily
   - See entire plan at once
   - Quick overview of all phases
   - Easy to compare phase costs/risks

6. **Mobile Friendly**
   - Responsive design for tablets
   - Touch-friendly controls
   - Review plans on the go

7. **Live Feedback**
   - Real-time execution progress
   - See fixes as they happen
   - Running cost/time counters
   - Immediate error feedback

## Future Enhancements

### Multi-User Collaboration
- Share plan via URL or export
- Team members can comment on phases
- Approval voting (2/3 approvals required)
- Activity log (who approved what, when)

### Advanced Analytics
- Violation trends over time
- Cost analysis by category
- Success rate predictions
- Historical comparison

### Integration with kai Extension
- "Open in VS Code" button for violations
- Send high-complexity violations to kai
- Import kai fixes back to plan
- Unified workflow

### AI-Powered Recommendations
- Suggest phase groupings
- Recommend which phases to defer
- Predict success rates
- Identify risky combinations

### Export & Reporting
- PDF report generation
- CSV export for spreadsheet analysis
- Markdown summary for documentation
- Integration with project management tools

## Open Questions

1. **Authentication**: Do we need user authentication for multi-user scenarios?
2. **Persistence**: Should we use a database for plan history/versioning?
3. **Remote Access**: Should the server be accessible outside localhost?
4. **Framework Choice**: React/Vue for better UX vs vanilla JS for simplicity?
5. **Deployment**: Package as standalone binary with embedded UI or separate?

## Success Metrics

- **Adoption**: % of users who try web UI vs CLI interactive
- **Satisfaction**: User feedback scores
- **Efficiency**: Time saved reviewing plans (vs CLI)
- **Error Reduction**: Fewer mistakes in phase approval
- **Completion Rate**: % of plans fully reviewed vs abandoned

## References

- Current CLI interactive: `pkg/planner/interactive.go`
- HTML report generation: `pkg/planner/report.go`
- Plan file format: `pkg/planfile/types.go`
- Executor: `pkg/executor/executor.go`
