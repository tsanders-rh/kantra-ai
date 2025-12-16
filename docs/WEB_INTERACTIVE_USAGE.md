# Web Interactive Planner Usage Guide

This guide shows how to use the web-based interactive planner for kantra-ai, both locally and in a Podman container.

## Overview

The web interactive planner provides a modern web UI for reviewing and approving migration phases, replacing the CLI-based interactive mode with a visual interface featuring:

- Dashboard with metrics and progress tracking
- Visual phase cards with risk indicators
- Approve/defer phases with one click
- Real-time updates via WebSocket
- Save plan changes directly from the browser

## Local Usage (Without Container)

### Prerequisites
- Go 1.22 or later
- Access to Konveyor analysis output

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
   - View the dashboard with plan metrics
   - Click "Approve" or "Defer" for each phase
   - The progress bar updates in real-time
   - Click "Save" to save your decisions to `.kantra-ai-plan.yaml`

5. **Stop the server:**
   - Press `Ctrl+C` in the terminal
   - Your changes are saved to the plan file

6. **Execute the plan:**
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

3. **Access the web UI:**
   - Open your browser to `http://localhost:8080`
   - Review and approve/defer phases
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

## Web UI Features (MVP)

### Dashboard
- **Total Phases:** Number of phases in the plan
- **Total Violations:** Count of unique violations
- **Total Incidents:** Count of incidents across all violations
- **Estimated Cost:** Total estimated cost for all phases

### Progress Tracking
- Visual progress bar showing approval status
- Percentage and count of approved phases
- Updates in real-time as you approve/defer

### Phase Cards
Each phase displays:
- **Name and Order:** Phase number and description
- **Risk Level:** Color-coded (Low/Medium/High)
- **Category:** Violation category (mandatory/optional)
- **Effort Range:** Estimated effort level
- **Statistics:** Violation count, incident count, cost, duration
- **Actions:** Approve, Defer, View Details buttons

### Actions
- **Approve:** Mark phase for execution
- **Defer:** Skip this phase (won't be executed)
- **Save:** Persist changes to plan file
- **Execute:** (Coming soon) Start execution from web UI

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

## Next Steps

After reviewing and saving your plan:

1. **Execute approved phases:**
   ```bash
   kantra-ai execute --plan .kantra-ai-plan.yaml --input /path/to/source
   ```

2. **Review execution results** in the generated state file and HTML report

3. **For deferred phases,** consider using the kai VS Code extension for manual review

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
5. Monitor Network tab for API calls

## Future Enhancements

The following features are planned for future releases:
- Execute phases directly from web UI
- Live execution progress with incident updates
- Code diff viewer with syntax highlighting
- AI sample preview before approving
- Drag-and-drop phase reordering
- Enhanced risk warnings and analysis
- Collaborative features (comments, voting)

## Feedback

Report issues or request features at:
https://github.com/tsanders/kantra-ai/issues
