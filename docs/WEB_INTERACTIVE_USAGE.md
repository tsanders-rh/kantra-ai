# Web Interactive Planner Usage Guide

This guide shows how to use the web-based interactive planner for kantra-ai, both locally and in a Podman container.

## Overview

The web interactive planner provides a modern web UI for reviewing and approving migration phases, replacing the CLI-based interactive mode with a visual interface featuring:

- **Dashboard with metrics** - Real-time statistics and progress tracking
- **Visual phase cards** - Color-coded risk indicators and detailed violation breakdowns
- **Interactive approval** - Approve/defer phases with one click
- **Code diff viewer** - Syntax-highlighted before/after code comparisons
- **Live execution** - Real-time progress updates via WebSocket
- **Drag-and-drop reordering** - Reorganize phase execution order
- **Advanced filtering** - Select/deselect specific violations within phases
- **Execution controls** - Start, cancel, and monitor execution from the browser
- **Keyboard shortcuts** - Power-user features for faster workflow
- **Export/Import** - Save and load plan configurations

## Local Usage (Without Container)

### Prerequisites
- Go 1.22 or later
- Access to Konveyor analysis output
- AI provider configured (Claude or OpenAI)

### Steps

1. **Generate a plan with web interface:**
   ```bash
   kantra-ai plan \
     --analysis output.yaml \
     --input /path/to/source \
     --interactive-web
   ```

2. **The web server will start automatically:**
   ```
   🌐 Starting web interface at http://localhost:8080
   ```

3. **Your browser will open automatically** to the web UI. If not, navigate to `http://localhost:8080`

4. **Review and approve phases:**
   - View the dashboard with plan metrics and charts
   - Click "Approve" or "Defer" for each phase
   - Expand phase details to see violations and code diffs
   - Drag phases to reorder execution priority
   - Click "Save" to persist your decisions to `.kantra-ai-plan.yaml`

5. **Execute approved phases (optional):**
   - Click "Execute" to run fixes directly from the web UI
   - Monitor real-time progress in the activity log
   - Cancel execution at any time if needed
   - View detailed summary when complete

6. **Stop the server:**
   - Press `Ctrl+C` in the terminal
   - Your changes are saved to the plan file

7. **Or execute from CLI:**
   ```bash
   kantra-ai execute --plan .kantra-ai-plan.yaml --input /path/to/source
   ```

## Using with Podman

### Prerequisites
- Podman installed (`brew install podman` on macOS, or see [Podman installation](https://podman.io/getting-started/installation))
- Konveyor analysis output file

### Option 1: Build and Run Container

1. **Build the container image:**
   ```bash
   podman build -t kantra-ai:latest .
   ```

2. **Run the plan command in container:**
   ```bash
   podman run -it --rm \
     -v $(pwd)/output.yaml:/data/output.yaml:ro \
     -v $(pwd)/source:/source:ro \
     -v $(pwd):/output \
     -p 8080:8080 \
     -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
     kantra-ai:latest \
     plan \
       --analysis /data/output.yaml \
       --input /source \
       --output /output/.kantra-ai-plan.yaml \
       --interactive-web
   ```

   **Volume mounts explained:**
   - `-v $(pwd)/output.yaml:/data/output.yaml:ro` - Mount analysis file (read-only)
   - `-v $(pwd)/source:/source:ro` - Mount source code (read-only)
   - `-v $(pwd):/output` - Mount output directory for plan file
   - `-p 8080:8080` - Expose web interface port
   - `-e ANTHROPIC_API_KEY` - Pass AI provider credentials for execution

3. **Access the web UI:**
   - Open your browser to `http://localhost:8080`
   - Review and approve/defer phases
   - Execute directly from the browser
   - Click "Save" to write changes

4. **Stop the container:**
   - Press `Ctrl+C` in the terminal
   - The plan file is saved to your current directory

### Option 2: Interactive Container Session

Run an interactive shell in the container:

```bash
podman run -it --rm \
  -v $(pwd):/workspace \
  -p 8080:8080 \
  -w /workspace \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  kantra-ai:latest \
  /bin/sh
```

Then run kantra-ai commands inside the container:

```bash
kantra-ai plan \
  --analysis output.yaml \
  --input ./source \
  --interactive-web
```

### Rootless Podman

Podman runs rootless by default, which is great for security. If you encounter permission issues:

1. **Check file permissions:**
   ```bash
   ls -la output.yaml source/
   ```

2. **Ensure the kantra user (UID 1000) can read the mounted files:**
   ```bash
   # If needed, adjust permissions
   chmod -R a+r source/
   chmod a+r output.yaml
   ```

### Building for Different Architectures

Build for ARM64 (e.g., Apple Silicon):
```bash
podman build --platform linux/arm64 -t kantra-ai:arm64 .
```

Build for AMD64:
```bash
podman build --platform linux/amd64 -t kantra-ai:amd64 .
```

Build multi-arch:
```bash
podman build --platform linux/amd64,linux/arm64 -t kantra-ai:latest .
```

## Web UI Features

### Dashboard
- **Total Phases:** Number of phases in the plan
- **Total Violations:** Count of unique violations
- **Total Incidents:** Count of incidents across all violations
- **Estimated Cost:** Total estimated cost for all phases

### Charts & Visualizations
- **Complexity Distribution:** Pie chart showing violation complexity breakdown (trivial/low/medium/high/expert)
- **Violations by Category:** Bar chart of violations grouped by category (mandatory/optional/potential)
- **Risk Distribution:** Pie chart of phases by risk level (low/medium/high)

### Progress Tracking
- Visual progress bar showing approval status
- Percentage and count of approved phases
- Updates in real-time as you approve/defer
- Color-coded risk indicators

### Phase Cards
Each phase displays:
- **Name and Order:** Phase number and description
- **Risk Level:** Color-coded badge (🟢 Low / 🟡 Medium / 🔴 High)
- **Category:** Violation category (mandatory/optional/potential)
- **Effort Range:** Estimated effort level (1-10 scale)
- **Statistics:** Violation count, incident count, cost, estimated duration
- **High-Risk Warnings:** Detailed warnings for high-risk phases with recommendations
- **Actions:** Approve, Defer, and View Details buttons

### Phase Details (Expandable)
Click "Details" to view:
- **Violation Filtering:** Select/deselect specific violations to include/exclude
- **Bulk Actions:** Select all, deselect all, filter by effort level
- **Selection Summary:** Count of selected violations and excluded violations
- **Code Diffs:** Before/after code snippets with syntax highlighting
- **Incident Navigation:** Browse through multiple incidents within each violation
- **Line Numbers:** Exact file paths and line numbers for each incident

### Drag-and-Drop Phase Reordering
- Click and hold phase header to drag
- Visual feedback during drag (shadow, opacity changes)
- Drop to reorder execution priority
- Changes saved automatically when you click "Save"

### Actions

#### Save
- Persists all changes (approvals, deferrals, reordering) to `.kantra-ai-plan.yaml`
- **Keyboard shortcut:** `Ctrl/Cmd + S`

#### Execute
- Starts execution of approved phases directly from the web UI
- Shows confirmation dialog with estimates before starting
- **Keyboard shortcut:** `Ctrl/Cmd + E`
- Displays:
  - Number of phases to execute
  - Total violations and incidents
  - Estimated cost and duration
  - Warning to create backup before proceeding

#### Export
- Downloads plan as JSON file for backup or sharing
- Preserves all customizations (approvals, reordering, selections)

#### Import
- Uploads previously exported plan JSON
- Restores all settings and decisions

#### Settings
- Configure execution options:
  - **Confidence Filtering:** Set minimum confidence threshold
  - **Verification:** Run builds after fixes, fail-fast options
  - **Git Integration:** Create commits per phase/violation, auto-create PRs
  - **Batch Processing:** Configure batch size and parallelism

### Live Execution View

When you click "Execute", the UI switches to execution mode showing:

- **Progress Bar:** Visual indicator of overall completion
- **Execution Timer:** Real-time elapsed time counter (MM:SS format)
- **Current Phase:** Name of phase currently being processed
- **Activity Log:** Scrolling feed of recent events:
  - Phase start/end notifications
  - Info messages from executor
  - Error messages if any occur
  - Success confirmations
- **Statistics:** Running counts of completed phases
- **Cancel Button:** Stop execution at any time

### Execution Summary

After execution completes, view:
- Total phases executed
- Successful fixes count
- Failed fixes count (if any)
- Total cost and tokens used
- Final duration
- Links to detailed reports
- Option to return to plan view or close

### Keyboard Shortcuts

- **`Ctrl/Cmd + S`** - Save plan
- **`Ctrl/Cmd + E`** - Execute approved phases
- **`?`** - Show keyboard shortcuts help
- **`Esc`** - Close modals/dialogs

### Toast Notifications

Non-intrusive slide-in notifications for:
- Save confirmations
- Execution status updates
- Warnings (no phases approved, etc.)
- Error messages
- Auto-dismiss after 5 seconds
- Color-coded by type (success/warning/error/info)

## Common Issues

### Port Already in Use
If port 8080 is already in use:

1. **Find what's using the port:**
   ```bash
   lsof -i :8080
   ```

2. **Kill the process or use a different port** (requires code modification for now)

### Browser Doesn't Open Automatically
Navigate manually to `http://localhost:8080`

### Container Can't Access Files
Ensure volume mounts are correct and files are readable:
```bash
# Test with simple ls
podman run -it --rm \
  -v $(pwd)/output.yaml:/data/output.yaml:ro \
  kantra-ai:latest \
  ls -la /data/output.yaml
```

### WebSocket Connection Fails
Check that:
- The web server is running
- Port 8080 is exposed and not blocked by firewall
- You're accessing via `http://localhost:8080`, not a different hostname
- No proxy is interfering with WebSocket connections

### Execution Fails Immediately
Ensure you have:
- AI provider API key set as environment variable
- Source files accessible to the container (if using Podman)
- At least one phase approved for execution

## Next Steps

After reviewing and saving your plan:

1. **Execute approved phases** (from web UI or CLI):
   ```bash
   kantra-ai execute --plan .kantra-ai-plan.yaml --input /path/to/source
   ```

2. **Review execution results** in:
   - The execution summary view (if executed from web)
   - Generated state file (`.kantra-ai-state.yaml`)
   - HTML report (if configured)

3. **For deferred phases,** consider:
   - Using the kai VS Code extension for manual review
   - Re-evaluating with adjusted confidence thresholds
   - Breaking down into smaller, lower-risk phases

## Development

### Running Locally for Development
```bash
# Build
go build -o kantra-ai ./cmd/kantra-ai

# Run
./kantra-ai plan \
  --analysis examples/output.yaml \
  --input examples/source \
  --interactive-web
```

### Rebuilding After Changes
```bash
# Rebuild Go binary
go build -o kantra-ai ./cmd/kantra-ai

# Rebuild container
podman build -t kantra-ai:latest .
```

### Testing the Web UI
1. Generate a test plan
2. Start the web server
3. Open browser dev tools (F12)
4. Check console for JavaScript errors
5. Monitor Network tab for API calls and WebSocket messages
6. Test all features:
   - Approve/defer phases
   - Drag-and-drop reordering
   - Expand/collapse details
   - Filter violations
   - Execute and monitor progress
   - Cancel execution

### Running Tests
```bash
# Run web server tests
go test ./pkg/web/...

# Run all tests
go test ./...
```

## Advanced Usage

### Customizing Execution Settings

Before executing, click the Settings button to configure:

1. **Confidence Filtering:**
   - Enable/disable confidence threshold
   - Set minimum confidence (0-100%)
   - Choose action for low confidence fixes (skip/prompt/attempt)

2. **Build Verification:**
   - Run build after fixes
   - Build strategy (at-end/per-phase/per-violation)
   - Fail fast on build errors

3. **Git Integration:**
   - Auto-create commits
   - Commit strategy (single/per-phase/per-violation)
   - Auto-create pull request after execution

4. **Batch Processing:**
   - Enable batch mode for cost savings
   - Configure batch size (1-50 incidents)
   - Set parallelism level (1-10 concurrent batches)

### Selecting Specific Violations

1. Click "Details" on any phase
2. Use checkboxes to select/deselect specific violations
3. Bulk actions available:
   - "Select All" - Include all violations
   - "Deselect All" - Exclude all violations
   - "Select Low/Medium Effort" - Include only lower complexity items
4. Selection summary shows:
   - Number of violations selected
   - Number excluded
   - Updated cost estimate for selections

### Reviewing Code Diffs

1. Expand phase details
2. Browse violations list
3. For each incident:
   - View before/after code with syntax highlighting
   - See file path and line number
   - Navigate between multiple incidents
   - Review AI explanation and confidence

### Exporting and Sharing Plans

1. **Export Plan:**
   - Click "Export" button
   - Downloads JSON file with all decisions
   - Share with team members

2. **Import Plan:**
   - Click "Import" button
   - Select previously exported JSON
   - All approvals and customizations restored

## Future Enhancements

Planned features for future releases:
- AI sample preview before approving phases
- Collaborative features (comments, multi-user approval voting)
- Advanced analytics and cost tracking
- Integration with VS Code kai extension
- PDF report generation
- Mobile app version

## Feedback

Report issues or request features at:
https://github.com/tsanders/kantra-ai/issues
