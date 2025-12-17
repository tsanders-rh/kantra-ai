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
	"github.com/tsanders/kantra-ai/pkg/executor"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/provider"
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
	CreateCommits      bool    `json:"createCommits"`
	CommitStrategy     string  `json:"commitStrategy"`
	CreatePR           bool    `json:"createPR"`
	PRStrategy         string  `json:"prStrategy"`
	PRCommentThreshold float64 `json:"prCommentThreshold"`
	BatchEnabled       bool    `json:"batchEnabled"`
	BatchSize          int     `json:"batchSize"`
	Parallelism        int     `json:"parallelism"`
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
			BatchEnabled: true,
			BatchSize:    10,
			Parallelism:  4,
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

// handleExecuteStatus returns execution status (placeholder for MVP).
func (s *PlanServer) handleExecuteStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "idle",
		"message": "Execution feature coming soon",
	}); err != nil {
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

// executePhases runs the plan execution in the background.
func (s *PlanServer) executePhases() {
	defer func() {
		s.executionMutex.Lock()
		s.executing = false
		s.executionMutex.Unlock()
	}()

	// Create execution context
	s.executionCtx, s.executionCancel = context.WithCancel(context.Background())
	defer s.executionCancel()

	// Create progress writer that broadcasts to WebSocket clients
	progress := &WebSocketProgressWriter{server: s}

	// Get settings (use defaults if not set)
	settings := s.executionSettings
	if settings == nil {
		settings = &ExecutionSettings{
			BatchEnabled: true,
			BatchSize:    10,
			Parallelism:  4,
		}
	}

	// Build batch config from settings
	batchConfig := fixer.BatchConfig{
		Enabled:      settings.BatchEnabled,
		MaxBatchSize: settings.BatchSize,
		Parallelism:  settings.Parallelism,
		GroupByFile:  true, // Always enabled for optimal token usage
	}

	// Create executor config
	execConfig := executor.Config{
		PlanPath:           s.planPath,
		InputPath:          s.inputPath,
		Provider:           s.provider,
		Progress:           progress,
		DryRun:             false,
		GitCommit:          settings.CommitStrategy,
		CreatePR:           settings.CreatePR,
		PRStrategy:         settings.PRStrategy,
		PRCommentThreshold: settings.PRCommentThreshold,
		BatchConfig:        batchConfig,
	}

	exec, err := executor.New(execConfig)
	if err != nil {
		s.BroadcastUpdate(ExecutionUpdate{
			Type: "error",
			Data: map[string]string{
				"message": fmt.Sprintf("Failed to create executor: %v", err),
			},
		})
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
		s.BroadcastUpdate(ExecutionUpdate{
			Type: "error",
			Data: map[string]interface{}{
				"message": fmt.Sprintf("Execution failed: %v", err),
				"result":  result,
			},
		})
		return
	}

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
