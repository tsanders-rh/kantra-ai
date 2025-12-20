package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsanders/kantra-ai/pkg/confidence"
	"github.com/tsanders/kantra-ai/pkg/executor"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/gitutil"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/verifier"
)

//go:embed static/*
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

// ExecutionSettings holds user-configurable execution settings from the web UI.
type ExecutionSettings struct {
	// Confidence filtering
	ConfidenceEnabled    bool    `json:"confidenceEnabled"`
	ConfidenceThreshold  int     `json:"confidenceThreshold"` // 0-100 percentage
	LowConfidenceAction  string  `json:"lowConfidenceAction"` // "skip", "prompt", "attempt"

	// Verification
	RunVerification       bool   `json:"runVerification"`
	VerificationType      string `json:"verificationType"`      // "build", "test"
	VerificationStrategy  string `json:"verificationStrategy"`  // "at-end", "per-phase", "per-violation"
	FailFast              bool   `json:"failFast"`

	// Git integration
	CreateCommits      bool    `json:"createCommits"`
	CommitStrategy     string  `json:"commitStrategy"`
	CreatePR           bool    `json:"createPR"`
	PRStrategy         string  `json:"prStrategy"`
	PRCommentThreshold float64 `json:"prCommentThreshold"`

	// Batch processing
	BatchEnabled       bool    `json:"batchEnabled"`
	BatchSize          int     `json:"batchSize"`
	Parallelism        int     `json:"parallelism"`
}

// ExecutionStatus tracks the current state of plan execution
type ExecutionStatus struct {
	State            string      `json:"state"` // "idle", "running", "completed", "failed", "cancelled"
	Message          string      `json:"message"`
	StartTime        time.Time   `json:"start_time,omitempty"`
	EndTime          time.Time   `json:"end_time,omitempty"`
	CurrentPhase     int         `json:"current_phase"`
	TotalPhases      int         `json:"total_phases"`
	SuccessfulFixes  int         `json:"successful_fixes"`
	FailedFixes      int         `json:"failed_fixes"`
	TotalCost        float64     `json:"total_cost"`
	Error            string      `json:"error,omitempty"`
}

// PlanServer serves the web-based interactive plan approval UI.
type PlanServer struct {
	plan             *planfile.Plan
	planPath         string
	inputPath        string
	provider         provider.Provider
	addr             string
	clients          map[*websocket.Conn]bool
	clientsMutex     sync.RWMutex
	server           *http.Server
	executing        bool
	executionMutex   sync.Mutex
	executionCtx     context.Context
	executionCancel  context.CancelFunc
	executionSettings *ExecutionSettings
	executionStatus  ExecutionStatus
}

// NewPlanServer creates a new web server for interactive plan approval.
func NewPlanServer(plan *planfile.Plan, planPath string, inputPath string, prov provider.Provider) *PlanServer {
	return &PlanServer{
		plan:      plan,
		planPath:  planPath,
		inputPath: inputPath,
		provider:  prov,
		addr:      "localhost:8080",
		clients:   make(map[*websocket.Conn]bool),
		executionStatus: ExecutionStatus{
			State:   "idle",
			Message: "No execution in progress",
		},
	}
}

// Start starts the web server and optionally opens the browser.
func (s *PlanServer) Start(ctx context.Context, openBrowser bool) error {
	// Create router
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	// API endpoints
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/plan", s.handleGetPlan)
	mux.HandleFunc("/api/phase/approve", s.handleApprovePhase)
	mux.HandleFunc("/api/phase/defer", s.handleDeferPhase)
	mux.HandleFunc("/api/plan/save", s.handleSavePlan)
	mux.HandleFunc("/api/execute/start", s.handleExecuteStart)
	mux.HandleFunc("/api/execute/cancel", s.handleExecuteCancel)
	mux.HandleFunc("/api/execute/status", s.handleExecuteStatus)
	mux.HandleFunc("/ws", s.handleWebSocket)

	// Create server
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Check if port is available
	if !s.isPortAvailable() {
		return fmt.Errorf("port %s is already in use", s.addr)
	}

	fmt.Printf("\n🌐 Starting web interface at http://%s\n", s.addr)

	if openBrowser {
		go s.openBrowserDelayed("http://" + s.addr)
	}

	// Start server
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the web server.
func (s *PlanServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// isPortAvailable checks if the configured port is available.
func (s *PlanServer) isPortAvailable() bool {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// openBrowserDelayed opens the browser after a short delay.
func (s *PlanServer) openBrowserDelayed(url string) {
	time.Sleep(500 * time.Millisecond)
	if err := openBrowser(url); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

// openBrowser opens the default browser to the given URL.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// handleIndex serves the main HTML page.
func (s *PlanServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(data); err != nil {
		// Log error but can't send response as headers are already written
		fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
	}
}

// handleGetPlan returns the current plan as JSON.
func (s *PlanServer) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.plan); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding plan: %v\n", err)
	}
}

// handleApprovePhase approves a phase by ID.
func (s *PlanServer) handleApprovePhase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PhaseID string `json:"phase_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find and approve phase
	found := false
	for i := range s.plan.Phases {
		if s.plan.Phases[i].ID == req.PhaseID {
			s.plan.Phases[i].Deferred = false
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Phase not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "approved"}); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
	}
}

// handleDeferPhase defers a phase by ID.
func (s *PlanServer) handleDeferPhase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PhaseID string `json:"phase_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find and defer phase
	found := false
	for i := range s.plan.Phases {
		if s.plan.Phases[i].ID == req.PhaseID {
			s.plan.Phases[i].Deferred = true
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Phase not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deferred"}); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
	}
}

// handleSavePlan saves the current plan to disk.
func (s *PlanServer) handleSavePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := planfile.SavePlan(s.plan, s.planPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "saved", "path": s.planPath}); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
	}
}

// handleExecuteStart starts plan execution.
func (s *PlanServer) handleExecuteStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse settings from request body
	var reqBody struct {
		Settings ExecutionSettings `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		// If parsing fails, use default settings (for backward compatibility)
		reqBody.Settings = ExecutionSettings{
			ConfidenceEnabled:    false,
			ConfidenceThreshold:  70,
			LowConfidenceAction:  "skip",
			RunVerification:      false,
			VerificationType:     "build",
			VerificationStrategy: "at-end",
			FailFast:             false,
			BatchEnabled:         true,
			BatchSize:            10,
			Parallelism:          4,
		}
	}

	// Check if already executing
	s.executionMutex.Lock()
	if s.executing {
		s.executionMutex.Unlock()
		http.Error(w, "Execution already in progress", http.StatusConflict)
		return
	}
	s.executing = true
	s.executionSettings = &reqBody.Settings
	s.executionMutex.Unlock()

	// Start execution in background
	go s.executePhases()

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "started",
		"message": "Execution started",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
	}
}

// handleExecuteCancel cancels the current execution.
func (s *PlanServer) handleExecuteCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.executionMutex.Lock()
	if !s.executing {
		s.executionMutex.Unlock()
		http.Error(w, "No execution in progress", http.StatusBadRequest)
		return
	}

	// Cancel the execution context
	if s.executionCancel != nil {
		s.executionCancel()
	}

	// Update execution status to cancelled
	s.executionStatus = ExecutionStatus{
		State:           "cancelled",
		Message:         "Execution cancelled by user",
		EndTime:         time.Now(),
		StartTime:       s.executionStatus.StartTime,
		TotalPhases:     s.executionStatus.TotalPhases,
		CurrentPhase:    s.executionStatus.CurrentPhase,
		SuccessfulFixes: s.executionStatus.SuccessfulFixes,
		FailedFixes:     s.executionStatus.FailedFixes,
		TotalCost:       s.executionStatus.TotalCost,
	}
	s.executionMutex.Unlock()

	// Broadcast cancellation message
	s.BroadcastUpdate(ExecutionUpdate{
		Type: "cancelled",
		Data: map[string]string{
			"message": "Execution cancelled by user",
		},
	})

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "cancelled",
		"message": "Execution cancelled",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
	}
}

// handleExecuteStatus returns the current execution status.
func (s *PlanServer) handleExecuteStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.executionMutex.Lock()
	status := s.executionStatus
	s.executionMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
	}
}

// handleWebSocket handles WebSocket connections for live updates.
func (s *PlanServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	s.clientsMutex.Lock()
	s.clients[conn] = true
	s.clientsMutex.Unlock()

	// Handle messages from client
	go func() {
		defer func() {
			s.clientsMutex.Lock()
			delete(s.clients, conn)
			s.clientsMutex.Unlock()
			conn.Close()
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// BroadcastUpdate sends an update to all connected WebSocket clients.
func (s *PlanServer) BroadcastUpdate(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal update: %v", err)
		return
	}

	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send update to client: %v", err)
		}
	}
}

// ExecutionUpdate represents a WebSocket update message.
type ExecutionUpdate struct {
	Type string      `json:"type"` // "progress", "incident", "complete", "error"
	Data interface{} `json:"data"`
}

// mapConfidenceAction converts web UI action string to confidence.Action.
// Web UI uses: "skip", "prompt", "attempt"
// Backend uses: "skip", "manual-review-file", "warn-and-apply"
func mapConfidenceAction(webAction string) confidence.Action {
	switch webAction {
	case "skip":
		return confidence.ActionSkip
	case "prompt":
		return confidence.ActionManualReviewFile
	case "attempt":
		return confidence.ActionWarnAndApply
	default:
		return confidence.ActionSkip // Default to safest option
	}
}

// mapVerificationStrategy converts web UI strategy string to verifier.VerificationStrategy.
// Web UI uses: "at-end", "per-phase", "per-violation"
// Backend uses: "at-end", "per-violation", "per-fix"
// Note: "per-phase" maps to "per-violation" as closest equivalent
func mapVerificationStrategy(webStrategy string) verifier.VerificationStrategy {
	switch webStrategy {
	case "at-end":
		return verifier.StrategyAtEnd
	case "per-phase", "per-violation":
		return verifier.StrategyPerViolation
	default:
		return verifier.StrategyAtEnd // Default to safest option
	}
}

// setExecutionError sets the execution status to failed and broadcasts an error.
func (s *PlanServer) setExecutionError(errMsg string) {
	s.executionMutex.Lock()
	s.executionStatus = ExecutionStatus{
		State:   "failed",
		Message: "Execution failed",
		Error:   errMsg,
		EndTime: time.Now(),
	}
	s.executionMutex.Unlock()

	s.BroadcastUpdate(ExecutionUpdate{
		Type: "error",
		Data: map[string]string{
			"message": errMsg,
		},
	})
}

// executePhases runs the plan execution in the background.
func (s *PlanServer) executePhases() {
	defer func() {
		s.executionMutex.Lock()
		s.executing = false
		s.executionMutex.Unlock()
	}()

	// Initialize execution status
	s.executionMutex.Lock()
	s.executionStatus = ExecutionStatus{
		State:       "running",
		Message:     "Execution started",
		StartTime:   time.Now(),
		TotalPhases: len(s.plan.Phases),
	}
	s.executionMutex.Unlock()

	// Create execution context
	s.executionCtx, s.executionCancel = context.WithCancel(context.Background())
	defer s.executionCancel()

	// Create progress writer that broadcasts to WebSocket clients
	progress := &WebSocketProgressWriter{server: s}

	// Get settings (use defaults if not set)
	settings := s.executionSettings
	if settings == nil {
		settings = &ExecutionSettings{
			ConfidenceEnabled:    false,
			ConfidenceThreshold:  70,
			LowConfidenceAction:  "skip",
			RunVerification:      false,
			VerificationType:     "build",
			VerificationStrategy: "at-end",
			FailFast:             false,
			BatchEnabled:         true,
			BatchSize:            10,
			Parallelism:          4,
		}
	}

	// Build batch config from settings
	batchConfig := fixer.BatchConfig{
		Enabled:      settings.BatchEnabled,
		MaxBatchSize: settings.BatchSize,
		Parallelism:  settings.Parallelism,
		GroupByFile:  true, // Always enabled for optimal token usage
	}

	// Build confidence config from settings
	confidenceConfig := confidence.DefaultConfig()
	confidenceConfig.Enabled = settings.ConfidenceEnabled
	if settings.ConfidenceThreshold > 0 {
		// Convert from percentage (0-100) to float (0.0-1.0)
		confidenceConfig.Default = float64(settings.ConfidenceThreshold) / 100.0
	}
	confidenceConfig.OnLowConfidence = mapConfidenceAction(settings.LowConfidenceAction)

	// Initialize git trackers if commits are enabled
	var commitTracker *gitutil.CommitTracker
	var verifiedTracker *gitutil.VerifiedCommitTracker
	var prTracker *gitutil.PRTracker

	if settings.CreateCommits && settings.CommitStrategy != "" {
		strategy, err := gitutil.ParseStrategy(settings.CommitStrategy)
		if err != nil {
			s.setExecutionError(fmt.Sprintf("Invalid commit strategy: %v", err))
			return
		}

		// If verification is enabled, create VerifiedCommitTracker
		if settings.RunVerification {
			verifyType, err := verifier.ParseVerificationType(settings.VerificationType)
			if err != nil {
				s.setExecutionError(fmt.Sprintf("Invalid verification type: %v", err))
				return
			}

			verifyStrat := mapVerificationStrategy(settings.VerificationStrategy)

			verifyConfig := verifier.Config{
				Type:         verifyType,
				Strategy:     verifyStrat,
				WorkingDir:   s.inputPath,
				FailFast:     settings.FailFast,
				SkipOnDryRun: false,
			}

			verifiedTracker, err = gitutil.NewVerifiedCommitTracker(strategy, s.inputPath, s.provider.Name(), verifyConfig)
			if err != nil {
				s.setExecutionError(fmt.Sprintf("Failed to initialize verification: %v", err))
				return
			}
			commitTracker = verifiedTracker.GetCommitTracker()
		} else {
			// No verification, just create regular CommitTracker
			commitTracker = gitutil.NewCommitTracker(strategy, s.inputPath, s.provider.Name())
		}
	}

	// Initialize PR tracker if enabled
	if settings.CreatePR && settings.PRStrategy != "" {
		parsedPRStrategy, err := gitutil.ParsePRStrategy(settings.PRStrategy)
		if err != nil {
			s.setExecutionError(fmt.Sprintf("Invalid PR strategy: %v", err))
			return
		}

		// Get GitHub token from environment
		githubToken := os.Getenv("GITHUB_TOKEN")

		// Generate branch name
		branchName := fmt.Sprintf("kantra-ai/remediation-%d", time.Now().Unix())

		// Create PR config
		prConfig := gitutil.PRConfig{
			Strategy:           parsedPRStrategy,
			BranchPrefix:       branchName,
			GitHubToken:        githubToken,
			CommentThreshold:   settings.PRCommentThreshold,
		}

		prTracker, err = gitutil.NewPRTracker(prConfig, s.inputPath, s.provider.Name(), progress)
		if err != nil {
			s.setExecutionError(fmt.Sprintf("Failed to initialize PR tracker: %v", err))
			return
		}
	}

	// Create executor config
	execConfig := executor.Config{
		PlanPath:            s.planPath,
		InputPath:           s.inputPath,
		Provider:            s.provider,
		Progress:            progress,
		DryRun:              false,
		GitCommit:           settings.CommitStrategy,
		CreatePR:            settings.CreatePR,
		PRStrategy:          settings.PRStrategy,
		PRCommentThreshold:  settings.PRCommentThreshold,
		BatchConfig:         batchConfig,
		ConfidenceConfig:    confidenceConfig,
		CommitTracker:       commitTracker,
		VerifiedTracker:     verifiedTracker,
		PRTracker:           prTracker,
	}

	exec, err := executor.New(execConfig)
	if err != nil {
		s.setExecutionError(fmt.Sprintf("Failed to create executor: %v", err))
		return
	}

	// Send initial progress update
	s.BroadcastUpdate(ExecutionUpdate{
		Type: "progress",
		Data: map[string]interface{}{
			"phase":      0,
			"total":      len(s.plan.Phases),
			"percentage": 0,
			"message":    "Starting execution...",
		},
	})

	// Execute plan
	result, err := exec.Execute(s.executionCtx)
	if err != nil {
		// Check if it was a cancellation
		if s.executionCtx.Err() == context.Canceled {
			// Status already set by handleExecuteCancel
		} else {
			s.executionMutex.Lock()
			s.executionStatus = ExecutionStatus{
				State:           "failed",
				Message:         "Execution failed",
				Error:           err.Error(),
				EndTime:         time.Now(),
				SuccessfulFixes: result.SuccessfulFixes,
				FailedFixes:     result.FailedFixes,
				TotalCost:       result.TotalCost,
			}
			s.executionMutex.Unlock()
		}

		s.BroadcastUpdate(ExecutionUpdate{
			Type: "error",
			Data: map[string]interface{}{
				"message": fmt.Sprintf("Execution failed: %v", err),
				"result":  result,
			},
		})
		return
	}

	// Update status to completed
	s.executionMutex.Lock()
	s.executionStatus = ExecutionStatus{
		State:           "completed",
		Message:         "Execution completed successfully",
		EndTime:         time.Now(),
		StartTime:       s.executionStatus.StartTime, // Preserve start time
		TotalPhases:     result.TotalPhases,
		CurrentPhase:    result.TotalPhases,
		SuccessfulFixes: result.SuccessfulFixes,
		FailedFixes:     result.FailedFixes,
		TotalCost:       result.TotalCost,
	}
	s.executionMutex.Unlock()

	// Send completion message
	s.BroadcastUpdate(ExecutionUpdate{
		Type: "complete",
		Data: map[string]interface{}{
			"total_phases":     result.TotalPhases,
			"executed_phases":  result.ExecutedPhases,
			"completed_phases": result.CompletedPhases,
			"failed_phases":    result.FailedPhases,
			"successful_fixes": result.SuccessfulFixes,
			"failed_fixes":     result.FailedFixes,
			"total_cost":       result.TotalCost,
			"total_tokens":     result.TotalTokens,
			"commits":          result.Commits,
			"prs":              result.PRs,
		},
	})
}

// WebSocketProgressWriter implements ux.ProgressWriter and broadcasts to WebSocket clients.
type WebSocketProgressWriter struct {
	server       *PlanServer
	currentPhase string
	phaseIndex   int
}

func (w *WebSocketProgressWriter) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	w.server.BroadcastUpdate(ExecutionUpdate{
		Type: "info",
		Data: map[string]string{
			"message": message,
		},
	})
}

func (w *WebSocketProgressWriter) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	w.server.BroadcastUpdate(ExecutionUpdate{
		Type: "error",
		Data: map[string]string{
			"message": message,
		},
	})
}

func (w *WebSocketProgressWriter) StartPhase(phaseName string) {
	w.currentPhase = phaseName
	w.phaseIndex++

	// Update execution status with current phase
	w.server.executionMutex.Lock()
	w.server.executionStatus.CurrentPhase = w.phaseIndex
	w.server.executionMutex.Unlock()

	w.server.BroadcastUpdate(ExecutionUpdate{
		Type: "phase_start",
		Data: map[string]interface{}{
			"phase_name":  phaseName,
			"phase_index": w.phaseIndex,
			"total":       len(w.server.plan.Phases),
		},
	})
}

func (w *WebSocketProgressWriter) EndPhase() {
	w.server.BroadcastUpdate(ExecutionUpdate{
		Type: "phase_end",
		Data: map[string]interface{}{
			"phase_name":  w.currentPhase,
			"phase_index": w.phaseIndex,
		},
	})
}

// Printf implements gitutil.ProgressWriter interface
func (w *WebSocketProgressWriter) Printf(format string, args ...interface{}) {
	w.Info(format, args...)
}
