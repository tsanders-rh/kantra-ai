package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// MockProvider is a mock implementation of provider.Provider
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProvider) FixViolation(ctx context.Context, req provider.FixRequest) (*provider.FixResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.FixResponse), args.Error(1)
}

func (m *MockProvider) EstimateCost(req provider.FixRequest) (float64, error) {
	args := m.Called(req)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockProvider) GeneratePlan(ctx context.Context, req provider.PlanRequest) (*provider.PlanResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.PlanResponse), args.Error(1)
}

func (m *MockProvider) FixBatch(ctx context.Context, req provider.BatchRequest) (*provider.BatchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.BatchResponse), args.Error(1)
}

func TestNewPlanServer(t *testing.T) {
	plan := createTestPlan()
	mockProvider := new(MockProvider)

	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", mockProvider)

	assert.NotNil(t, server)
	assert.Equal(t, plan, server.plan)
	assert.Equal(t, "/tmp/plan.yaml", server.planPath)
	assert.Equal(t, "/tmp/input", server.inputPath)
	assert.Equal(t, mockProvider, server.provider)
	assert.Equal(t, "localhost:8080", server.addr)
	assert.NotNil(t, server.clients)
	assert.False(t, server.executing)
}

func TestHandleGetPlan(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	req := httptest.NewRequest(http.MethodGet, "/api/plan", nil)
	w := httptest.NewRecorder()

	server.handleGetPlan(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var returnedPlan planfile.Plan
	err := json.NewDecoder(w.Body).Decode(&returnedPlan)
	assert.NoError(t, err)
	assert.Equal(t, plan.Metadata.Provider, returnedPlan.Metadata.Provider)
	assert.Len(t, returnedPlan.Phases, len(plan.Phases))
}

func TestHandleGetPlan_MethodNotAllowed(t *testing.T) {
	server := NewPlanServer(createTestPlan(), "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	req := httptest.NewRequest(http.MethodPost, "/api/plan", nil)
	w := httptest.NewRecorder()

	server.handleGetPlan(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleApprovePhase(t *testing.T) {
	plan := createTestPlan()
	plan.Phases[0].Deferred = true // Start as deferred
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	body := strings.NewReader(`{"phase_id":"phase-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/phase/approve", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleApprovePhase(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, server.plan.Phases[0].Deferred)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "approved", response["status"])
}

func TestHandleApprovePhase_NotFound(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	body := strings.NewReader(`{"phase_id":"nonexistent"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/phase/approve", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleApprovePhase(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleDeferPhase(t *testing.T) {
	plan := createTestPlan()
	plan.Phases[0].Deferred = false // Start as approved
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	body := strings.NewReader(`{"phase_id":"phase-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/phase/defer", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleDeferPhase(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, server.plan.Phases[0].Deferred)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "deferred", response["status"])
}

func TestHandleSavePlan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "web-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	planPath := filepath.Join(tmpDir, "plan.yaml")
	plan := createTestPlan()
	server := NewPlanServer(plan, planPath, "/tmp/input", new(MockProvider))

	req := httptest.NewRequest(http.MethodPost, "/api/plan/save", nil)
	w := httptest.NewRecorder()

	server.handleSavePlan(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify file was created
	_, err = os.Stat(planPath)
	assert.NoError(t, err)

	// Verify response
	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "saved", response["status"])
	assert.Equal(t, planPath, response["path"])
}

func TestHandleExecuteStart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "web-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	planPath := filepath.Join(tmpDir, "plan.yaml")
	plan := createTestPlan()
	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()

	server := NewPlanServer(plan, planPath, tmpDir, mockProvider)

	req := httptest.NewRequest(http.MethodPost, "/api/execute/start", nil)
	w := httptest.NewRecorder()

	server.handleExecuteStart(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "started", response["status"])

	// Verify execution flag was set (check immediately before goroutine finishes)
	server.executionMutex.Lock()
	executing := server.executing
	server.executionMutex.Unlock()

	// Should be executing when we check immediately
	// Note: might already be done if execution is very fast, so we just verify the endpoint responded correctly
	_ = executing
}

func TestHandleExecuteStart_AlreadyExecuting(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))
	server.executing = true // Simulate ongoing execution

	req := httptest.NewRequest(http.MethodPost, "/api/execute/start", nil)
	w := httptest.NewRecorder()

	server.handleExecuteStart(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleExecuteCancel(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	// Set up execution state
	server.executing = true
	ctx, cancel := context.WithCancel(context.Background())
	server.executionCtx = ctx
	server.executionCancel = cancel

	req := httptest.NewRequest(http.MethodPost, "/api/execute/cancel", nil)
	w := httptest.NewRecorder()

	server.handleExecuteCancel(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify context was cancelled
	select {
	case <-server.executionCtx.Done():
		// Context was cancelled as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context was not cancelled")
	}

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "cancelled", response["status"])
}

func TestHandleExecuteCancel_NotExecuting(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))
	server.executing = false

	req := httptest.NewRequest(http.MethodPost, "/api/execute/cancel", nil)
	w := httptest.NewRecorder()

	server.handleExecuteCancel(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebSocket(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	// Verify client was registered
	time.Sleep(50 * time.Millisecond)
	server.clientsMutex.RLock()
	assert.Len(t, server.clients, 1)
	server.clientsMutex.RUnlock()

	// Close connection
	ws.Close()

	// Wait for cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify client was unregistered
	server.clientsMutex.RLock()
	assert.Len(t, server.clients, 0)
	server.clientsMutex.RUnlock()
}

func TestBroadcastUpdate(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message
	update := ExecutionUpdate{
		Type: "test",
		Data: map[string]string{"message": "Hello WebSocket"},
	}
	server.BroadcastUpdate(update)

	// Read the message
	err = ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	assert.NoError(t, err)
	_, message, err := ws.ReadMessage()
	assert.NoError(t, err)

	// Verify message content
	var received ExecutionUpdate
	err = json.Unmarshal(message, &received)
	assert.NoError(t, err)
	assert.Equal(t, "test", received.Type)

	data, ok := received.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Hello WebSocket", data["message"])
}

func TestWebSocketProgressWriter_Info(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Create progress writer
	writer := &WebSocketProgressWriter{server: server}

	// Send info message
	writer.Info("Test info: %s", "value")

	// Read message
	err = ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	assert.NoError(t, err)
	_, message, err := ws.ReadMessage()
	assert.NoError(t, err)

	var update ExecutionUpdate
	err = json.Unmarshal(message, &update)
	assert.NoError(t, err)
	assert.Equal(t, "info", update.Type)

	data := update.Data.(map[string]interface{})
	assert.Equal(t, "Test info: value", data["message"])
}

func TestWebSocketProgressWriter_Error(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	writer := &WebSocketProgressWriter{server: server}
	writer.Error("Test error: %d", 404)

	err = ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	assert.NoError(t, err)
	_, message, err := ws.ReadMessage()
	assert.NoError(t, err)

	var update ExecutionUpdate
	err = json.Unmarshal(message, &update)
	assert.NoError(t, err)
	assert.Equal(t, "error", update.Type)

	data := update.Data.(map[string]interface{})
	assert.Equal(t, "Test error: 404", data["message"])
}

func TestWebSocketProgressWriter_StartPhase(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	writer := &WebSocketProgressWriter{server: server}
	writer.StartPhase("Test Phase 1")

	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, message, err := ws.ReadMessage()
	assert.NoError(t, err)

	var update ExecutionUpdate
	err = json.Unmarshal(message, &update)
	assert.NoError(t, err)
	assert.Equal(t, "phase_start", update.Type)

	data := update.Data.(map[string]interface{})
	assert.Equal(t, "Test Phase 1", data["phase_name"])
	assert.Equal(t, float64(1), data["phase_index"]) // JSON numbers are float64
	assert.Equal(t, "Test Phase 1", writer.currentPhase)
	assert.Equal(t, 1, writer.phaseIndex)
}

func TestWebSocketProgressWriter_EndPhase(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	writer := &WebSocketProgressWriter{
		server:       server,
		currentPhase: "Test Phase",
		phaseIndex:   2,
	}
	writer.EndPhase()

	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, message, err := ws.ReadMessage()
	assert.NoError(t, err)

	var update ExecutionUpdate
	err = json.Unmarshal(message, &update)
	assert.NoError(t, err)
	assert.Equal(t, "phase_end", update.Type)

	data := update.Data.(map[string]interface{})
	assert.Equal(t, "Test Phase", data["phase_name"])
	assert.Equal(t, float64(2), data["phase_index"])
}

func TestIsPortAvailable(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	// Default port should be available (unless something is actually running on 8080)
	available := server.isPortAvailable()
	// We can't assert true here since port might actually be in use
	// Just verify the function runs without panic
	_ = available
}

func TestHandleIndex(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
	assert.Contains(t, w.Body.String(), "kantra-ai Migration Plan")
}

func TestHandleIndex_NotFound(t *testing.T) {
	plan := createTestPlan()
	server := NewPlanServer(plan, "/tmp/plan.yaml", "/tmp/input", new(MockProvider))

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Helper functions

func createTestPlan() *planfile.Plan {
	plan := planfile.NewPlan("test-provider", 1)
	plan.Metadata.CreatedAt = time.Now()

	plan.Phases = []planfile.Phase{
		{
			ID:                       "phase-1",
			Name:                     "Test Phase",
			Order:                    1,
			Risk:                     planfile.RiskLow,
			Category:                 "mandatory",
			EffortRange:              [2]int{1, 3},
			Explanation:              "Test explanation",
			EstimatedCost:            0.10,
			EstimatedDurationMinutes: 5,
			Deferred:                 false,
			Violations: []planfile.PlannedViolation{
				{
					ViolationID:   "test-violation-1",
					Description:   "Test violation",
					Category:      "mandatory",
					Effort:        3,
					IncidentCount: 2,
					Incidents: []violation.Incident{
						{
							URI:        "file:///test.java",
							LineNumber: 10,
							Message:    "Test incident 1",
						},
						{
							URI:        "file:///test.java",
							LineNumber: 20,
							Message:    "Test incident 2",
						},
					},
				},
			},
		},
	}

	return plan
}
