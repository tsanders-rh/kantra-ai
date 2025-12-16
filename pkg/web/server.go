package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsanders/kantra-ai/pkg/executor"
	"github.com/tsanders/kantra-ai/pkg/planfile"
)

//go:embed static/*
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

// PlanServer serves the web-based interactive plan approval UI.
type PlanServer struct {
	plan         *planfile.Plan
	planPath     string
	addr         string
	executor     *executor.Executor
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	server       *http.Server
}

// NewPlanServer creates a new web server for interactive plan approval.
func NewPlanServer(plan *planfile.Plan, planPath string) *PlanServer {
	return &PlanServer{
		plan:     plan,
		planPath: planPath,
		addr:     "localhost:8080",
		clients:  make(map[*websocket.Conn]bool),
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
	w.Write(data)
}

// handleGetPlan returns the current plan as JSON.
func (s *PlanServer) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.plan)
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
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
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
	json.NewEncoder(w).Encode(map[string]string{"status": "deferred"})
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
	json.NewEncoder(w).Encode(map[string]string{"status": "saved", "path": s.planPath})
}

// handleExecuteStart starts plan execution (placeholder for MVP).
func (s *PlanServer) handleExecuteStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement execution with live updates
	// For MVP, just return success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "started",
		"message": "Execution feature coming soon - use CLI for now",
	})
}

// handleExecuteStatus returns execution status (placeholder for MVP).
func (s *PlanServer) handleExecuteStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "idle",
		"message": "Execution feature coming soon",
	})
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
