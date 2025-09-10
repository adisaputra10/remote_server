package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Config represents agent configuration
type Config struct {
	ID        string `json:"id"`
	ServerURL string `json:"server_url"`
}

// Agent represents the tunnel agent
type Agent struct {
	id       string
	config   *Config
	wsConn   *websocket.Conn
	logger   *log.Logger
	mu       sync.RWMutex
	done     chan bool
	tunnels  map[string]*ActiveTunnel
}

// ActiveTunnel represents an active tunnel connection
type ActiveTunnel struct {
	ID       string
	Type     string // mysql, postgresql
	LocalAddr string
	RemoteAddr string
	Listener net.Listener
	Done     chan bool
}

// handleMessageData processes incoming message data
func (a *Agent) handleMessageData(data []byte) {
	a.logger.Printf("🔍 Processing message: %s", string(data))
	
	// Parse message
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		a.logger.Printf("❌ Failed to parse message: %v", err)
		return
	}
	
	msgType, ok := msg["type"].(string)
	if !ok {
		a.logger.Printf("❌ Invalid message type")
		return
	}
	
	// Handle different message types
	switch msgType {
	case "ping":
		a.handlePing(msg)
	case "test_message":
		a.handleTestMessage(msg)
	case "tunnel_request":
		a.handleTunnelRequest(msg)
	case "tunnel_close":
		a.handleTunnelClose(msg)
	default:
		a.logger.Printf("⚠️ Unknown message type: %s", msgType)
		// Echo back unknown messages
		a.mu.RLock()
		conn := a.wsConn
		a.mu.RUnlock()
		
		if conn != nil {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				a.logger.Printf("❌ Failed to echo message: %v", err)
			}
		}
	}
}

// handlePing handles ping messages
func (a *Agent) handlePing(msg map[string]interface{}) {
	a.logger.Printf("🏓 Handling ping message")
	
	response := map[string]interface{}{
		"type": "pong",
		"agent_id": a.id,
		"timestamp": time.Now().Unix(),
	}
	
	a.sendMessage(response)
}

// handleTestMessage handles test messages
func (a *Agent) handleTestMessage(msg map[string]interface{}) {
	a.logger.Printf("🧪 Handling test message")
	
	// Echo back the test message
	a.mu.RLock()
	conn := a.wsConn
	a.mu.RUnlock()
	
	if conn != nil {
		data, _ := json.Marshal(msg)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			a.logger.Printf("❌ Failed to echo test message: %v", err)
		} else {
			a.logger.Printf("✅ Test message echoed successfully")
		}
	}
}

// handleTunnelRequest handles tunnel creation requests
func (a *Agent) handleTunnelRequest(msg map[string]interface{}) {
	a.logger.Printf("🚇 Handling tunnel request")
	
	// Extract tunnel parameters
	tunnelID, _ := msg["tunnel_id"].(string)
	tunnelType, _ := msg["tunnel_type"].(string)
	remoteHost, _ := msg["remote_host"].(string)
	remotePort, _ := msg["remote_port"].(float64)
	localPort, _ := msg["local_port"].(float64)
	
	if tunnelID == "" || tunnelType == "" || remoteHost == "" {
		a.logger.Printf("❌ Invalid tunnel request parameters")
		a.sendTunnelResponse(tunnelID, "error", "Invalid parameters")
		return
	}
	
	a.logger.Printf("📋 Tunnel details: ID=%s, Type=%s, Remote=%s:%d, Local=:%d", 
		tunnelID, tunnelType, remoteHost, int(remotePort), int(localPort))
	
	// Create tunnel
	if err := a.createTunnel(tunnelID, tunnelType, remoteHost, int(remotePort), int(localPort)); err != nil {
		a.logger.Printf("❌ Failed to create tunnel: %v", err)
		a.sendTunnelResponse(tunnelID, "error", err.Error())
		return
	}
	
	a.logger.Printf("✅ Tunnel created successfully: %s", tunnelID)
	a.sendTunnelResponse(tunnelID, "success", "Tunnel created")
}

// handleTunnelClose handles tunnel close requests
func (a *Agent) handleTunnelClose(msg map[string]interface{}) {
	a.logger.Printf("🚪 Handling tunnel close request")
	
	tunnelID, _ := msg["tunnel_id"].(string)
	if tunnelID == "" {
		a.logger.Printf("❌ Invalid tunnel close request - no tunnel ID")
		return
	}
	
	a.closeTunnel(tunnelID)
	a.logger.Printf("✅ Tunnel closed: %s", tunnelID)
}

// createTunnel creates a new database tunnel
func (a *Agent) createTunnel(tunnelID, tunnelType, remoteHost string, remotePort, localPort int) error {
	a.logger.Printf("🔧 Creating tunnel: %s -> %s:%d", tunnelType, remoteHost, remotePort)
	
	// Determine local address
	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	if localPort == 0 {
		// Auto-assign port
		localAddr = "127.0.0.1:0"
	}
	
	// Create listener
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}
	
	actualAddr := listener.Addr().String()
	a.logger.Printf("📡 Listener created on: %s", actualAddr)
	
	// Create tunnel object
	tunnel := &ActiveTunnel{
		ID:        tunnelID,
		Type:      tunnelType,
		LocalAddr: actualAddr,
		RemoteAddr: fmt.Sprintf("%s:%d", remoteHost, remotePort),
		Listener:  listener,
		Done:      make(chan bool),
	}
	
	// Store tunnel
	a.mu.Lock()
	a.tunnels[tunnelID] = tunnel
	a.mu.Unlock()
	
	// Start accepting connections
	go a.handleTunnelConnections(tunnel, remoteHost, remotePort)
	
	return nil
}

// handleTunnelConnections handles incoming connections for a tunnel
func (a *Agent) handleTunnelConnections(tunnel *ActiveTunnel, remoteHost string, remotePort int) {
	a.logger.Printf("👂 Starting to accept connections for tunnel: %s", tunnel.ID)
	
	for {
		select {
		case <-tunnel.Done:
			a.logger.Printf("🛑 Stopping tunnel connections: %s", tunnel.ID)
			return
		default:
			// Accept connection
			conn, err := tunnel.Listener.Accept()
			if err != nil {
				a.logger.Printf("❌ Failed to accept connection for tunnel %s: %v", tunnel.ID, err)
				continue
			}
			
			a.logger.Printf("🔗 New connection accepted for tunnel: %s", tunnel.ID)
			
			// Handle connection in goroutine
			go a.handleTunnelConnection(tunnel, conn, remoteHost, remotePort)
		}
	}
}

// handleTunnelConnection handles a single tunnel connection
func (a *Agent) handleTunnelConnection(tunnel *ActiveTunnel, clientConn net.Conn, remoteHost string, remotePort int) {
	defer clientConn.Close()
	
	a.logger.Printf("🌉 Handling connection for tunnel %s to %s:%d", tunnel.ID, remoteHost, remotePort)
	
	// Connect to remote database
	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
	remoteConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		a.logger.Printf("❌ Failed to connect to remote database %s: %v", remoteAddr, err)
		return
	}
	defer remoteConn.Close()
	
	a.logger.Printf("✅ Connected to remote database: %s", remoteAddr)
	
	// Start bidirectional copying
	done := make(chan bool, 2)
	
	// Copy client -> remote
	go func() {
		defer func() { done <- true }()
		written, err := io.Copy(remoteConn, clientConn)
		if err != nil {
			a.logger.Printf("❌ Error copying client->remote for tunnel %s: %v", tunnel.ID, err)
		} else {
			a.logger.Printf("📤 Copied %d bytes client->remote for tunnel %s", written, tunnel.ID)
		}
	}()
	
	// Copy remote -> client
	go func() {
		defer func() { done <- true }()
		written, err := io.Copy(clientConn, remoteConn)
		if err != nil {
			a.logger.Printf("❌ Error copying remote->client for tunnel %s: %v", tunnel.ID, err)
		} else {
			a.logger.Printf("📥 Copied %d bytes remote->client for tunnel %s", written, tunnel.ID)
		}
	}()
	
	// Wait for either direction to finish
	<-done
	a.logger.Printf("🏁 Connection finished for tunnel: %s", tunnel.ID)
}

// closeTunnel closes a tunnel
func (a *Agent) closeTunnel(tunnelID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	tunnel, exists := a.tunnels[tunnelID]
	if !exists {
		a.logger.Printf("⚠️ Tunnel not found: %s", tunnelID)
		return
	}
	
	a.logger.Printf("🚪 Closing tunnel: %s", tunnelID)
	
	// Close listener
	if tunnel.Listener != nil {
		tunnel.Listener.Close()
	}
	
	// Signal done
	close(tunnel.Done)
	
	// Remove from map
	delete(a.tunnels, tunnelID)
	
	a.logger.Printf("✅ Tunnel closed: %s", tunnelID)
}

// sendMessage sends a message to the server
func (a *Agent) sendMessage(msg map[string]interface{}) error {
	a.mu.RLock()
	conn := a.wsConn
	a.mu.RUnlock()
	
	if conn == nil {
		return fmt.Errorf("no WebSocket connection")
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	
	return nil
}

// sendTunnelResponse sends a tunnel response message
func (a *Agent) sendTunnelResponse(tunnelID, status, message string) {
	response := map[string]interface{}{
		"type": "tunnel_response",
		"tunnel_id": tunnelID,
		"status": status,
		"message": message,
		"agent_id": a.id,
		"timestamp": time.Now().Unix(),
	}
	
	if err := a.sendMessage(response); err != nil {
		a.logger.Printf("❌ Failed to send tunnel response: %v", err)
	} else {
		a.logger.Printf("📤 Sent tunnel response: %s - %s", tunnelID, status)
	}
}

// readLoop reads messages from WebSocket connection
func (a *Agent) readLoop() {
	defer func() {
		a.logger.Printf("👋 WebSocket read loop ended")
		a.done <- true
	}()
	
	a.logger.Printf("👂 Starting WebSocket read loop...")
	
	for {
		a.mu.RLock()
		conn := a.wsConn
		a.mu.RUnlock()
		
		if conn == nil {
			a.logger.Printf("❌ No WebSocket connection in read loop")
			return
		}
		
		_, message, err := conn.ReadMessage()
		if err != nil {
			a.logger.Printf("❌ Failed to read WebSocket message: %v", err)
			return
		}
		
		a.logger.Printf("📥 Received message: %s", string(message))
		
		// Process message
		go a.handleMessageData(message)
	}
}

// Connect connects to the server
func (a *Agent) Connect() error {
	a.logger.Printf("🔗 Connecting to server: %s", a.config.ServerURL)
	
	// Parse server URL
	serverURL, err := url.Parse(a.config.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %v", err)
	}
	
	// Convert HTTP/HTTPS to WS/WSS
	switch serverURL.Scheme {
	case "http":
		serverURL.Scheme = "ws"
	case "https":
		serverURL.Scheme = "wss"
	}
	
	// Add WebSocket endpoint path
	serverURL.Path = "/ws/agent"
	
	wsURL := serverURL.String()
	a.logger.Printf("📡 WebSocket URL: %s", wsURL)
	
	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}
	
	a.mu.Lock()
	a.wsConn = conn
	a.mu.Unlock()
	
	a.logger.Printf("✅ WebSocket connected successfully")
	
	// Send registration message
	regMsg := map[string]interface{}{
		"type":      "register",
		"agent_id":  a.id,
		"timestamp": time.Now().Unix(),
	}
	
	if err := a.sendMessage(regMsg); err != nil {
		return fmt.Errorf("failed to send registration: %v", err)
	}
	
	a.logger.Printf("📤 Registration message sent")
	
	// Start read loop
	go a.readLoop()
	
	// Start heartbeat
	go a.heartbeatLoop()
	
	return nil
}

// heartbeatLoop sends periodic heartbeat messages
func (a *Agent) heartbeatLoop() {
	a.logger.Printf("💓 Starting heartbeat loop...")
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			a.logger.Printf("💓 Sending heartbeat...")
			
			heartbeat := map[string]interface{}{
				"type":      "heartbeat",
				"agent_id":  a.id,
				"timestamp": time.Now().Unix(),
			}
			
			if err := a.sendMessage(heartbeat); err != nil {
				a.logger.Printf("❌ Failed to send heartbeat: %v", err)
			} else {
				a.logger.Printf("✅ Heartbeat sent")
			}
			
		case <-a.done:
			a.logger.Printf("🛑 Stopping heartbeat loop")
			return
		}
	}
}

// Run starts the agent
func (a *Agent) Run() error {
	a.logger.Printf("🚀 Starting agent: %s", a.id)
	
	// Connect to server
	if err := a.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	
	a.logger.Printf("🎯 Agent ready to accept tunnel requests!")
	a.logger.Printf("🎯 Supported tunnel types: mysql, postgresql")
	
	// Wait for done signal
	<-a.done
	
	a.logger.Printf("🛑 Agent stopping...")
	
	// Close all tunnels
	a.mu.Lock()
	for tunnelID := range a.tunnels {
		a.closeTunnel(tunnelID)
	}
	a.mu.Unlock()
	
	// Close WebSocket connection
	a.mu.RLock()
	if a.wsConn != nil {
		a.wsConn.Close()
	}
	a.mu.RUnlock()
	
	a.logger.Printf("👋 Agent stopped")
	return nil
}

// Stop stops the agent
func (a *Agent) Stop() {
	a.logger.Printf("🛑 Stopping agent...")
	close(a.done)
}

// generatePersistentID generates a persistent agent ID
func generatePersistentID() string {
	// Use hostname + MAC address + timestamp for uniqueness
	hostname, _ := os.Hostname()
	
	// Simple hash-based ID generation
	data := fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// NewAgent creates a new agent
func NewAgent(config *Config, logger *log.Logger) *Agent {
	// Generate persistent ID if not provided
	if config.ID == "" {
		config.ID = generatePersistentID()
		logger.Printf("📝 Generated new agent ID: %s", config.ID)
	}
	
	return &Agent{
		id:      config.ID,
		config:  config,
		logger:  logger,
		done:    make(chan bool),
		tunnels: make(map[string]*ActiveTunnel),
	}
}
