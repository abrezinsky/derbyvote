package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/services"
)

// mockSettingsService implements services.SettingsServicer for testing
type mockSettingsService struct {
	mu            sync.Mutex
	votingOpen    bool
	settings      map[string]string
	setVotingCall int
}

func newMockSettingsService() *mockSettingsService {
	return &mockSettingsService{
		votingOpen: true,
		settings:   make(map[string]string),
	}
}

func (m *mockSettingsService) IsVotingOpen(ctx context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.votingOpen, nil
}

func (m *mockSettingsService) SetVotingOpen(ctx context.Context, open bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.votingOpen = open
	m.setVotingCall++
	return nil
}

func (m *mockSettingsService) GetSetting(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings[key], nil
}

func (m *mockSettingsService) SetSetting(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[key] = value
	return nil
}

// Unused interface methods
func (m *mockSettingsService) GetDerbyNetURL(ctx context.Context) (string, error)    { return "", nil }
func (m *mockSettingsService) SetDerbyNetURL(ctx context.Context, url string) error  { return nil }
func (m *mockSettingsService) GetBaseURL(ctx context.Context) (string, error)        { return "", nil }
func (m *mockSettingsService) SetBaseURL(ctx context.Context, url string) error      { return nil }
func (m *mockSettingsService) GetTimerEndTime(ctx context.Context) (int64, error)    { return 0, nil }
func (m *mockSettingsService) SetTimerEndTime(ctx context.Context, t int64) error    { return nil }
func (m *mockSettingsService) ClearTimer(ctx context.Context) error                  { return nil }
func (m *mockSettingsService) AllSettings(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockSettingsService) OpenVoting(ctx context.Context) error                        { return nil }
func (m *mockSettingsService) CloseVoting(ctx context.Context) error                       { return nil }
func (m *mockSettingsService) StartVotingTimer(ctx context.Context, min int) (string, error) {
	return "", nil
}
func (m *mockSettingsService) UpdateSettings(ctx context.Context, s services.Settings) error {
	return nil
}
func (m *mockSettingsService) ResetTables(ctx context.Context, t []string) (*services.ResetTablesResult, error) {
	return nil, nil
}
func (m *mockSettingsService) SetBroadcaster(b services.Broadcaster) {}
func (m *mockSettingsService) RequireRegisteredQR(ctx context.Context) (bool, error) {
	return false, nil
}
func (m *mockSettingsService) GetVoterTypes(ctx context.Context) ([]string, error) {
	return []string{"general", "racer"}, nil
}
func (m *mockSettingsService) SetVoterTypes(ctx context.Context, types []string) error {
	return nil
}

func TestNew_CreatesHubWithDependencies(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()

	hub := New(log, settings)

	if hub == nil {
		t.Fatal("expected hub to be created")
	}
	if hub.log == nil {
		t.Error("expected logger to be set")
	}
	if hub.settings == nil {
		t.Error("expected settings to be set")
	}
	if hub.clients == nil {
		t.Error("expected clients map to be initialized")
	}
	if hub.broadcast == nil {
		t.Error("expected broadcast channel to be initialized")
	}
	if hub.register == nil {
		t.Error("expected register channel to be initialized")
	}
	if hub.unregister == nil {
		t.Error("expected unregister channel to be initialized")
	}
}

func TestHub_BroadcastMessage(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// BroadcastMessage should not block even with no clients
	done := make(chan bool)
	go func() {
		hub.BroadcastMessage("test", map[string]string{"key": "value"})
		done <- true
	}()

	select {
	case <-done:
		// Success - didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("BroadcastMessage blocked with no clients")
	}
}

func TestHub_BroadcastVotingStatus_ImplementsBroadcaster(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	time.Sleep(10 * time.Millisecond)

	// Test that Hub implements Broadcaster interface by calling it
	done := make(chan bool)
	go func() {
		hub.BroadcastVotingStatus(true, "2024-01-01T12:00:00Z")
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("BroadcastVotingStatus blocked")
	}
}

func TestHub_StartVotingCountdown_ContextCancellation(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan bool)
	stopped := make(chan bool)

	go func() {
		started <- true
		hub.StartVotingCountdown(ctx)
		stopped <- true
	}()

	// Wait for countdown to start
	<-started
	time.Sleep(50 * time.Millisecond)

	// Cancel should stop the countdown
	cancel()

	select {
	case <-stopped:
		// Success - countdown stopped when context cancelled
	case <-time.After(500 * time.Millisecond):
		t.Error("countdown did not stop when context was cancelled")
	}
}

func TestHub_CountdownClosesVotingWhenTimerExpires(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	settings.votingOpen = true
	// Set close time to 100ms in the past
	settings.settings["voting_close_time"] = time.Now().Add(-100 * time.Millisecond).Format(time.RFC3339)

	hub := New(log, settings)
	hub.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.StartVotingCountdown(ctx)

	// Wait for countdown to process
	time.Sleep(1500 * time.Millisecond)

	settings.mu.Lock()
	closed := !settings.votingOpen
	settings.mu.Unlock()

	if !closed {
		t.Error("expected voting to be closed after timer expired")
	}
}

func TestHub_MultipleInstances_NoGlobalState(t *testing.T) {
	log := logger.New()
	settings1 := newMockSettingsService()
	settings2 := newMockSettingsService()

	hub1 := New(log, settings1)
	hub2 := New(log, settings2)

	// Verify they are independent instances
	if hub1 == hub2 {
		t.Error("expected different hub instances")
	}
	if hub1.settings == hub2.settings {
		t.Error("expected different settings instances")
	}

	// Modify one and verify the other is unaffected
	hub1.Start()
	hub2.Start()

	settings1.votingOpen = false
	settings2.votingOpen = true

	open1, _ := settings1.IsVotingOpen(context.Background())
	open2, _ := settings2.IsVotingOpen(context.Background())

	if open1 != false {
		t.Error("expected settings1 voting to be closed")
	}
	if open2 != true {
		t.Error("expected settings2 voting to be open")
	}
}

func TestHub_Start_RunsInBackground(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)

	// Start should return immediately (runs in goroutine)
	done := make(chan bool)
	go func() {
		hub.Start()
		done <- true
	}()

	select {
	case <-done:
		// Success - Start returned immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("Start() blocked instead of running in background")
	}
}

func TestHub_ClientRegistration(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	time.Sleep(10 * time.Millisecond)

	// Create a mock client
	client := &Client{
		hub:  hub,
		send: make(chan models.WSMessage, 256),
	}

	// Register client
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	// Verify client was registered
	hub.mutex.RLock()
	_, exists := hub.clients[client]
	hub.mutex.RUnlock()

	if !exists {
		t.Error("expected client to be registered")
	}
}

func TestHub_ClientUnregistration(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	time.Sleep(10 * time.Millisecond)

	// Create and register a mock client
	client := &Client{
		hub:  hub,
		send: make(chan models.WSMessage, 256),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	// Unregister client
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	// Verify client was unregistered
	hub.mutex.RLock()
	_, exists := hub.clients[client]
	hub.mutex.RUnlock()

	if exists {
		t.Error("expected client to be unregistered")
	}
}

// ==================== WebSocket Integration Tests ====================

func TestServeWs_ClientConnection(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	// Convert http://... to ws://...
	url := "ws" + server.URL[4:]

	// Connect WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	// Give server time to register client
	time.Sleep(100 * time.Millisecond)

	// Verify client was registered
	hub.mutex.RLock()
	clientCount := len(hub.clients)
	hub.mutex.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 client, got %d", clientCount)
	}
}

func TestServeWs_BroadcastToClient(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	// Give server time to register client
	time.Sleep(100 * time.Millisecond)

	// Read and discard the initial voting_status message
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read initial voting_status: %v", err)
	}

	// Broadcast a message
	hub.BroadcastMessage("test_event", map[string]string{
		"key": "value",
	})

	// Read the broadcasted message from WebSocket
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	// Verify message content
	var msg models.WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.Type != "test_event" {
		t.Errorf("expected type 'test_event', got %s", msg.Type)
	}
}

func TestServeWs_ClientDisconnect(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Give server time to register client
	time.Sleep(100 * time.Millisecond)

	// Close connection
	ws.Close()

	// Give server time to unregister client
	time.Sleep(200 * time.Millisecond)

	// Verify client was unregistered
	hub.mutex.RLock()
	clientCount := len(hub.clients)
	hub.mutex.RUnlock()

	if clientCount != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", clientCount)
	}
}

func TestServeWs_MultipleClients(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]

	// Connect 3 clients
	ws1, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect client 1: %v", err)
	}
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect client 2: %v", err)
	}
	defer ws2.Close()

	ws3, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect client 3: %v", err)
	}
	defer ws3.Close()

	// Give server time to register all clients
	time.Sleep(200 * time.Millisecond)

	// Discard initial voting_status messages from all clients
	for i, ws := range []*websocket.Conn{ws1, ws2, ws3} {
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("client %d failed to read initial voting_status: %v", i+1, err)
		}
	}

	// Verify 3 clients registered
	hub.mutex.RLock()
	clientCount := len(hub.clients)
	hub.mutex.RUnlock()

	if clientCount != 3 {
		t.Errorf("expected 3 clients, got %d", clientCount)
	}

	// Broadcast message
	hub.BroadcastMessage("broadcast_test", map[string]int{"count": 123})

	// All clients should receive the message
	for i, ws := range []*websocket.Conn{ws1, ws2, ws3} {
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("client %d failed to read message: %v", i+1, err)
			continue
		}

		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			t.Errorf("client %d failed to unmarshal: %v", i+1, err)
			continue
		}

		if msg.Type != "broadcast_test" {
			t.Errorf("client %d got wrong type: %s", i+1, msg.Type)
		}
	}
}

func TestReadPump_IncomingMessage(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Send a message from client
	testMsg := models.WSMessage{
		Type:    "client_message",
		Payload: map[string]string{"data": "test"},
	}
	msgBytes, _ := json.Marshal(testMsg)

	if err := ws.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// Give server time to process
	time.Sleep(100 * time.Millisecond)

	// readPump should have logged the message (we can't directly verify but exercise the code)
}

func TestWritePump_PingPong(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	// Set up pong handler to track pings
	pongReceived := make(chan bool, 1)
	ws.SetPongHandler(func(string) error {
		select {
		case pongReceived <- true:
		default:
		}
		return nil
	})

	// Start reading to process pongs
	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Wait for a ping (writePump sends ping every 54 seconds, but we can't wait that long)
	// Just verify the connection stays alive for a reasonable time
	time.Sleep(200 * time.Millisecond)

	// Connection should still be alive
	if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
		t.Errorf("connection closed unexpectedly: %v", err)
	}
}

func TestCheckAndUpdateCountdown_SendsCountdown(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	// Set close time to 5 seconds in the future
	settings.settings["voting_close_time"] = time.Now().Add(5 * time.Second).Format(time.RFC3339)

	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Read and discard the initial voting_status message
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read initial voting_status: %v", err)
	}

	// Manually trigger countdown check
	hub.checkAndUpdateCountdown()

	// Read the countdown message from WebSocket
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	// Verify it's a countdown message
	var msg models.WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.Type != "countdown" {
		t.Errorf("expected type 'countdown', got %s", msg.Type)
	}
}

func TestCheckAndUpdateCountdown_ClosesExpiredVoting(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	settings.votingOpen = true
	// Set close time to the past
	settings.settings["voting_close_time"] = time.Now().Add(-1 * time.Second).Format(time.RFC3339)

	hub := New(log, settings)
	hub.Start()

	time.Sleep(50 * time.Millisecond)

	// Trigger countdown check
	hub.checkAndUpdateCountdown()

	time.Sleep(100 * time.Millisecond)

	// Verify voting was closed
	settings.mu.Lock()
	votingOpen := settings.votingOpen
	settings.mu.Unlock()

	if votingOpen {
		t.Error("expected voting to be closed after countdown expired")
	}
}

func TestCheckAndUpdateCountdown_InvalidCloseTime(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	// Set invalid close time
	settings.settings["voting_close_time"] = "invalid-time"

	hub := New(log, settings)
	hub.Start()

	// This should not panic or error
	hub.checkAndUpdateCountdown()

	// Success if no panic
}

func TestCheckAndUpdateCountdown_EmptyCloseTime(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	// No close time set
	settings.settings["voting_close_time"] = ""

	hub := New(log, settings)
	hub.Start()

	// This should not panic or error
	hub.checkAndUpdateCountdown()

	// Success if no panic
}

func TestServeWs_UpgradeError(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	// Create a request without upgrade headers - should fail
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	hub.ServeWs(w, req)

	// Should have logged error and returned (we can't verify logging, but exercise the code)
	// The upgrade will fail because request doesn't have proper WS headers
}

func TestReadPump_MessageProcessing(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Read and discard the initial voting_status message
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	ws.ReadMessage()

	// Send a valid JSON message that will be unmarshaled successfully
	testMsg := models.WSMessage{
		Type:    "test_message",
		Payload: map[string]string{"key": "value"},
	}
	msgBytes, _ := json.Marshal(testMsg)

	if err := ws.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// Give server time to process the message
	time.Sleep(100 * time.Millisecond)

	// The readPump should have processed the message and logged it
	// (we can't verify the log, but this exercises the code path)
}

func TestWritePump_ChannelClosed(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Read initial message
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	ws.ReadMessage()

	// Set up close handler to detect when server sends close message
	closeReceived := make(chan bool, 1)
	ws.SetCloseHandler(func(code int, text string) error {
		closeReceived <- true
		return nil
	})

	// Start reading to process close message
	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Find the client and close its send channel by unregistering it
	hub.mutex.RLock()
	var client *Client
	for c := range hub.clients {
		client = c
		break
	}
	hub.mutex.RUnlock()

	if client == nil {
		t.Fatal("no client found")
	}

	// Unregister the client - this will close the send channel
	// which should trigger writePump to send a close message
	hub.unregister <- client

	// Wait for close message to be received
	select {
	case <-closeReceived:
		// Success - close message was sent
	case <-time.After(500 * time.Millisecond):
		t.Error("expected to receive close message from server")
	}
}

func TestReadPump_PongHandler(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Read initial message
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	ws.ReadMessage()

	// Send a pong message from client to server
	// This will trigger the server's SetPongHandler which updates the read deadline
	if err := ws.WriteControl(websocket.PongMessage, []byte("pong"), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("failed to send pong: %v", err)
	}

	// Give server time to process pong
	time.Sleep(100 * time.Millisecond)

	// Server's pong handler should have been triggered and updated the read deadline
	// We can't directly verify this, but the code path has been exercised
	if err := ws.WriteMessage(websocket.TextMessage, []byte("test")); err != nil {
		t.Errorf("connection should still be alive after pong: %v", err)
	}
}

func TestWritePump_WriteError(t *testing.T) {
	log := logger.New()
	settings := newMockSettingsService()
	hub := New(log, settings)
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWs))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Read and discard initial message
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	ws.ReadMessage()

	// Close connection from client side
	ws.Close()
	time.Sleep(50 * time.Millisecond)

	// Try to broadcast a message - server will attempt to write to closed connection
	// This should trigger the writer error paths
	hub.BroadcastMessage("test", map[string]string{"key": "value"})

	// Give server time to detect write error and clean up
	time.Sleep(200 * time.Millisecond)

	// Verify client was cleaned up after write error
	hub.mutex.RLock()
	clientCount := len(hub.clients)
	hub.mutex.RUnlock()

	if clientCount != 0 {
		t.Errorf("expected 0 clients after write error, got %d", clientCount)
	}
}

