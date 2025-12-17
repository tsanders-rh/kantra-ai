# Web UI Quick Reference

A quick reference guide for the kantra-ai web-based interactive planner.

## Getting Started

```bash
# Start the web interface
kantra-ai plan --analysis output.yaml --input /path/to/source --interactive-web

# Access the UI
# Browser opens automatically to http://localhost:8080
```

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl/Cmd + S` | Save plan |
| `Ctrl/Cmd + E` | Execute approved phases |
| `?` | Show keyboard shortcuts help |
| `Esc` | Close modals/dialogs |

## Common Workflows

### Review and Approve Phases

1. Review dashboard metrics and charts
2. Expand phases to see detailed violations
3. Click **Approve** or **Defer** for each phase
4. Press `Ctrl/Cmd + S` or click **Save**

### Reorder Phases

1. Click and hold phase header
2. Drag to desired position
3. Drop to reorder
4. Click **Save** to persist

### Filter Violations

1. Click **Details** to expand phase
2. Use checkboxes to select/deselect violations
3. Or use bulk actions:
   - **Select All** - Include all
   - **Deselect All** - Exclude all
   - **Select Low/Medium Effort** - Filter by complexity
4. Review selection summary
5. Click **Save**

### Execute Plan

1. Ensure at least one phase is approved
2. Click **Execute** or press `Ctrl/Cmd + E`
3. Review confirmation dialog (shows estimates)
4. Click **Execute** to start
5. Monitor real-time progress
6. Click **Cancel** to stop if needed
7. Review execution summary when complete

### View Code Diffs

1. Expand phase **Details**
2. Scroll to violations list
3. Each incident shows:
   - File path and line number
   - Before/After code with syntax highlighting
   - AI confidence score
4. Use **Previous/Next** to navigate incidents

## Dashboard Overview

| Metric | Description |
|--------|-------------|
| **Total Phases** | Number of migration phases |
| **Total Violations** | Unique violation types |
| **Total Incidents** | Individual code locations to fix |
| **Estimated Cost** | Total AI API cost estimate |

## Phase Status Badges

| Badge | Meaning |
|-------|---------|
| ✓ **Approved** | Will be executed |
| ↷ **Deferred** | Skipped for now |
| ⏸ **Pending** | Not yet reviewed |

## Risk Levels

| Icon | Risk | Description |
|------|------|-------------|
| 🟢 | **Low** | Simple, mechanical changes |
| 🟡 | **Medium** | Moderate complexity, review recommended |
| 🔴 | **High** | Complex changes, careful review required |

## Execution View

During execution, monitor:

- **Progress Bar** - Overall completion percentage
- **Timer** - Elapsed time (MM:SS)
- **Current Phase** - Name of active phase
- **Activity Log** - Real-time event stream
- **Statistics** - Success/failure counts

## Settings Panel

Configure before execution:

### Confidence Filtering
- Enable threshold filtering
- Set minimum confidence (0-100%)
- Choose low-confidence action (skip/prompt/attempt)

### Build Verification
- Run build after fixes
- Build strategy (at-end/per-phase/per-violation)
- Fail fast on errors

### Git Integration
- Auto-create commits
- Commit strategy (single/per-phase/per-violation)
- Auto-create pull request

### Batch Processing
- Enable batching for cost savings
- Configure batch size (1-50)
- Set parallelism (1-10)

## Export/Import

### Export Plan
```
1. Click "Export" button
2. Downloads kantra-ai-plan-export.json
3. Share or backup
```

### Import Plan
```
1. Click "Import" button
2. Select JSON file
3. All settings restored
```

## API Endpoints (for developers)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/plan` | Get full plan data |
| POST | `/api/phase/approve` | Approve a phase |
| POST | `/api/phase/defer` | Defer a phase |
| POST | `/api/plan/save` | Save to YAML |
| POST | `/api/execute/start` | Start execution |
| POST | `/api/execute/cancel` | Cancel execution |
| WS | `/ws` | Live updates |

## Troubleshooting

### Port 8080 in use
```bash
# Find what's using the port
lsof -i :8080

# Kill the process
kill -9 <PID>
```

### WebSocket fails to connect
- Check firewall settings
- Verify accessing via `localhost:8080`
- Check browser console for errors

### Execution fails immediately
- Verify AI provider API key is set
- Check at least one phase is approved
- Review browser console for errors

## Tips & Tricks

1. **Quick Approve All**:
   - Approve phases in sequence from top to bottom
   - Press `Ctrl/Cmd + S` once at the end

2. **Safe Testing**:
   - Defer high-risk phases initially
   - Execute low/medium risk first
   - Review results before tackling harder phases

3. **Cost Management**:
   - Use violation filtering to exclude high-cost items
   - Review selection summary before executing
   - Monitor running cost in execution view

4. **Code Review**:
   - Expand all high-risk phases
   - Review code diffs before approving
   - Look for AI confidence < 80%

5. **Workflow Efficiency**:
   - Use keyboard shortcuts for speed
   - Drag-and-drop to reorder quickly
   - Save frequently during review

## File Locations

| File | Purpose |
|------|---------|
| `.kantra-ai-plan/` | Directory containing plan files |
| `.kantra-ai-plan/plan.yaml` | Plan with approval decisions |
| `.kantra-ai-plan/plan.html` | Visual HTML report |
| `.kantra-ai-state.yaml` | Execution state and history |
| `kantra-ai-plan-export.json` | Exported plan backup |

## Next Steps After Execution

1. **Review Results**:
   - Check execution summary
   - Review `.kantra-ai-state.yaml`
   - Check for any failed fixes

2. **Test Changes**:
   - Run application tests
   - Build and verify functionality
   - Manual QA of affected areas

3. **Handle Deferred Phases**:
   - Use kai VS Code extension for manual review
   - Re-evaluate after testing low-risk changes
   - Consider breaking into smaller phases

4. **Create PR** (if configured):
   - Review auto-generated pull request
   - Add any manual notes
   - Request team review

## Resources

- **Full Documentation**: [docs/WEB_INTERACTIVE_USAGE.md](WEB_INTERACTIVE_USAGE.md)
- **Design Document**: [docs/design/WEB_INTERACTIVE_PLAN.md](design/WEB_INTERACTIVE_PLAN.md)
- **Report Issues**: https://github.com/tsanders/kantra-ai/issues
- **Test Suite**: `go test ./pkg/web/...`
