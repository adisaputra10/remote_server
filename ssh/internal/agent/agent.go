package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Config represents agent configuration
type Config struct {
	ID               string                 `json:"id"`
	ServerURL        string                 `json:"server_url"`
	AgentName        string                 `json:"agent_name"`
	Platform         string                 `json:"platform"`
	AuthToken        string                 `json:"auth_token"`
	Metadata         map[string]interface{} `json:"metadata"`
	WorkingDir       string                 `json:"working_dir"`
	AllowedUsers     []string               `json:"allowed_users"`
	DatabaseProxies  []DatabaseProxy        `json:"database_proxies"`
}

// DatabaseProxy represents a database proxy configuration
type DatabaseProxy struct {
	Name       string `json:"name"`
	LocalPort  int    `json:"local_port"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port"`
	Protocol   string `json:"protocol"`
	Enabled    bool   `json:"enabled"`
	listener   net.Listener
	agent      *Agent
	ctx        context.Context
	cancel     context.CancelFunc
}

// Tunnel represents a database tunnel
type Tunnel struct {
	ID         string
	Type       string
	LocalAddr  string
	RemoteAddr string
	Active     bool
}

// Agent represents the tunnel agent
type Agent struct {
	id       string
	config   *Config
	wsConn   *websocket.Conn
	logger   *log.Logger
	mu       sync.RWMutex
	writeMu  sync.Mutex // Separate mutex for websocket writes
	done     chan bool
	tunnels  map[string]*ActiveTunnel
}

// ActiveTunnel represents an active tunnel connection
type ActiveTunnel struct {
	ID         string
	Type       string // mysql, postgresql
	LocalAddr  string
	RemoteAddr string
	Listener   net.Listener
	Done       chan bool
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

	a.logger.Printf("🔍 Message type: %s", msgType)
	
	// Log tunnel requests specifically
	if msgType == "tunnel_request" {
		a.logger.Printf("🚇 TUNNEL REQUEST RECEIVED!")
		if tunnelID, ok := msg["tunnel_id"].(string); ok {
			a.logger.Printf("🔍 Tunnel ID: %s", tunnelID)
		}
		if agentID, ok := msg["agent_id"].(string); ok {
			a.logger.Printf("🔍 Agent ID: %s", agentID)
		}
		if tunnelType, ok := msg["tunnel_type"].(string); ok {
			a.logger.Printf("🔍 Tunnel Type: %s", tunnelType)
		}
	}

	// Handle different message types
	switch msgType {
	case "ping":
		a.handlePing(msg)
	case "pong":
		a.handlePong(msg)
	case "heartbeat":
		a.handleHeartbeat(msg)
	case "register":
		a.handleRegisterResponse(msg)
	case "registered":
		a.handleRegistered(msg)
	case "test_message":
		a.handleTestMessage(msg)
	case "tunnel_request":
		a.handleTunnelRequest(msg)
	case "tunnel_data":
		a.handleTunnelData(msg)
	case "tunnel_close":
		a.handleTunnelClose(msg)
	default:
		a.logger.Printf("⚠️ Unknown message type: %s", msgType)
		// Don't echo back unknown messages to prevent loops
	}
}

// handlePing handles ping messages
func (a *Agent) handlePing(msg map[string]interface{}) {
	a.logger.Printf("🏓 Handling ping message")

	response := map[string]interface{}{
		"type":      "pong",
		"agent_id":  a.id,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	a.sendMessage(response)
}

// handlePong handles pong messages
func (a *Agent) handlePong(msg map[string]interface{}) {
	a.logger.Printf("🏓 Received pong message")
}

// handleHeartbeat handles heartbeat messages from server
func (a *Agent) handleHeartbeat(msg map[string]interface{}) {
	a.logger.Printf("💓 Received heartbeat from server")
}

// handleRegisterResponse handles registration response from server (should not happen)
func (a *Agent) handleRegisterResponse(msg map[string]interface{}) {
	a.logger.Printf("📝 Received register message (ignoring to prevent loop)")
}

// handleRegistered handles successful registration confirmation
func (a *Agent) handleRegistered(msg map[string]interface{}) {
	a.logger.Printf("✅ Successfully registered with server")
	if agentID, ok := msg["agent_id"].(string); ok {
		a.logger.Printf("🆔 Server assigned ID: %s", agentID)
	}
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
	a.logger.Printf("🚇 HANDLING TUNNEL REQUEST START")
	a.logger.Printf("🔍 Full message content: %+v", msg)

	// Extract tunnel parameters
	tunnelID, _ := msg["tunnel_id"].(string)
	tunnelType, _ := msg["tunnel_type"].(string)
	remoteHost, _ := msg["remote_host"].(string)
	remotePort, _ := msg["remote_port"].(float64)
	localPort, _ := msg["local_port"].(float64)

	a.logger.Printf("🔍 Extracted parameters:")
	a.logger.Printf("  - tunnelID: '%s'", tunnelID)
	a.logger.Printf("  - tunnelType: '%s'", tunnelType)
	a.logger.Printf("  - remoteHost: '%s'", remoteHost)
	a.logger.Printf("  - remotePort: %f", remotePort)
	a.logger.Printf("  - localPort: %f", localPort)

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

// handleTunnelData handles tunnel data forwarding through WebSocket
func (a *Agent) handleTunnelData(msg map[string]interface{}) {
	a.logger.Printf("📡 Handling tunnel data")

	tunnelID, _ := msg["tunnel_id"].(string)
	if tunnelID == "" {
		a.logger.Printf("❌ Invalid tunnel data - no tunnel ID")
		return
	}

	// Get tunnel
	a.mu.RLock()
	tunnel, exists := a.tunnels[tunnelID]
	a.mu.RUnlock()

	if !exists {
		a.logger.Printf("❌ Tunnel not found: %s", tunnelID)
		return
	}

	a.logger.Printf("🌉 Processing tunnel data for tunnel: %s -> %s", tunnelID, tunnel.RemoteAddr)
	
	// For now, we'll implement a simple version that connects directly to MySQL
	// In a full implementation, this would handle bidirectional data streaming
	// through the WebSocket connection
}

// createTunnel creates a new database tunnel
func (a *Agent) createTunnel(tunnelID, tunnelType, remoteHost string, remotePort, localPort int) error {
	a.logger.Printf("🔧 Creating tunnel: %s -> %s:%d", tunnelType, remoteHost, remotePort)
	a.logger.Printf("🔍 Tunnel configuration:")
	a.logger.Printf("  - Tunnel ID: %s", tunnelID)
	a.logger.Printf("  - Tunnel Type: %s", tunnelType)
	a.logger.Printf("  - Target: %s:%d", remoteHost, remotePort)

	// Try to find available port starting from 3307
	var listener net.Listener
	var err error
	var actualPort int

	// Try ports 3307, 3308, 3309, etc. until we find available one
	for port := 3307; port <= 3320; port++ {
		localAddr := fmt.Sprintf("127.0.0.1:%d", port)
		a.logger.Printf("🔌 Trying to create listener on: %s", localAddr)
		
		listener, err = net.Listen("tcp", localAddr)
		if err == nil {
			actualPort = port
			a.logger.Printf("✅ Successfully created listener on: %s", localAddr)
			break
		} else {
			a.logger.Printf("⚠️  Port %d busy, trying next port...", port)
		}
	}

	if listener == nil {
		a.logger.Printf("❌ Failed to find available port after trying 3307-3320")
		return fmt.Errorf("failed to find available port: %v", err)
	}

	actualAddr := listener.Addr().String()
	a.logger.Printf("📡 Agent listener created successfully on: %s", actualAddr)
	a.logger.Printf("🎯 Tunnel will forward: %s -> %s:%d", actualAddr, remoteHost, remotePort)

	// Create tunnel object
	tunnel := &ActiveTunnel{
		ID:         tunnelID,
		Type:       tunnelType,
		LocalAddr:  actualAddr,
		RemoteAddr: fmt.Sprintf("%s:%d", remoteHost, remotePort),
		Listener:   listener,
		Done:       make(chan bool),
	}

	// Store tunnel
	a.mu.Lock()
	a.tunnels[tunnelID] = tunnel
	a.mu.Unlock()

	// Start accepting connections
	go a.handleTunnelConnections(tunnel, remoteHost, remotePort)

	a.logger.Printf("✅ Tunnel created successfully: %s (listening on %s)", tunnelID, actualAddr)
	a.logger.Printf("🎯 Ready to forward connections from port %d to %s:%d", actualPort, remoteHost, remotePort)

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

	clientAddr := clientConn.RemoteAddr().String()
	a.logger.Printf("🌉 Handling connection for tunnel %s: %s -> %s:%d", tunnel.ID, clientAddr, remoteHost, remotePort)

	// Connect to remote database
	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
	a.logger.Printf("🔌 Connecting to remote database: %s", remoteAddr)
	
	remoteConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		a.logger.Printf("❌ Failed to connect to remote database %s: %v", remoteAddr, err)
		return
	}
	defer remoteConn.Close()

	a.logger.Printf("✅ Connected to remote database: %s", remoteAddr)
	a.logger.Printf("📡 Starting data relay: %s <-> %s", clientAddr, remoteAddr)

	// Start bidirectional copying with query logging
	done := make(chan bool, 2)

	// Copy client -> remote (intercept queries)
	go func() {
		defer func() { done <- true }()
		written, err := a.copyWithQueryLogging(remoteConn, clientConn, tunnel.ID, "client->remote")
		if err != nil {
			a.logger.Printf("❌ Error copying client->remote for tunnel %s: %v", tunnel.ID, err)
		} else {
			a.logger.Printf("📤 Copied %d bytes client->remote for tunnel %s", written, tunnel.ID)
		}
	}()

	// Copy remote -> client (intercept responses)
	go func() {
		defer func() { done <- true }()
		written, err := a.copyWithQueryLogging(clientConn, remoteConn, tunnel.ID, "remote->client")
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

	// Use write mutex to prevent concurrent writes
	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// sendTunnelResponse sends a tunnel response message
func (a *Agent) sendTunnelResponse(tunnelID, status, message string) {
	response := map[string]interface{}{
		"type":      "tunnel_response",
		"tunnel_id": tunnelID,
		"status":    status,
		"message":   message,
		"agent_id":  a.id,
		"timestamp": time.Now().Format(time.RFC3339),
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

	// The modular server expects direct connection to the path in ServerURL
	// Don't modify the path, use it as is

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
		"timestamp": time.Now().Format(time.RFC3339),
		"metadata": map[string]interface{}{
			"name":     a.config.AgentName,
			"platform": a.config.Platform,
		},
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
	a.logger.Printf("💓 Starting heartbeat loop (15 second interval)...")

	ticker := time.NewTicker(15 * time.Second) // Changed to 15 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.logger.Printf("💓 Sending heartbeat...")

			heartbeat := map[string]interface{}{
				"type":      "heartbeat",
				"agent_id":  a.id,
				"timestamp": time.Now().Format(time.RFC3339),
			}

			if err := a.sendMessage(heartbeat); err != nil {
				a.logger.Printf("❌ Failed to send heartbeat: %v", err)
				// If heartbeat fails, we might be disconnected
				return
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

// copyWithQueryLogging copies data between connections while logging database queries
func (a *Agent) copyWithQueryLogging(dst, src net.Conn, tunnelID, direction string) (int64, error) {
	buffer := make([]byte, 32*1024) // 32KB buffer
	var totalWritten int64
	packetCount := 0

	a.logger.Printf("📡 Starting data copy: %s for tunnel %s", direction, tunnelID)

	for {
		// Read from source
		n, err := src.Read(buffer)
		if err != nil {
			if err == io.EOF {
				a.logger.Printf("📡 Data copy finished: %s for tunnel %s (EOF)", direction, tunnelID)
				break
			}
			a.logger.Printf("❌ Read error in %s for tunnel %s: %v", direction, tunnelID, err)
			return totalWritten, err
		}

		packetCount++
		a.logger.Printf("📦 [%s] Packet #%d: %d bytes for tunnel %s", direction, packetCount, n, tunnelID)

		// Log data if it looks like a query (client to server)
		if direction == "client->remote" && n > 5 {
			a.logDatabaseQuery(buffer[:n], tunnelID, direction)
		}

		// Log hex dump for small packets (debugging)
		if n <= 100 {
			a.logger.Printf("📋 [%s] Hex dump: %x", direction, buffer[:n])
		}

		// Write to destination
		written, err := dst.Write(buffer[:n])
		if err != nil {
			a.logger.Printf("❌ Write error in %s for tunnel %s: %v", direction, tunnelID, err)
			return totalWritten, err
		}

		totalWritten += int64(written)
		a.logger.Printf("✅ [%s] Successfully relayed %d bytes (total: %d) for tunnel %s", direction, written, totalWritten, tunnelID)
	}

	return totalWritten, nil
}

// logDatabaseQuery attempts to extract and log database queries
func (a *Agent) logDatabaseQuery(data []byte, tunnelID, direction string) {
	// Simple MySQL query detection (COM_QUERY = 0x03)
	if len(data) > 5 && data[4] == 0x03 {
		// Extract query text (skip packet header + command byte)
		query := string(data[5:])
		// Clean up the query
		query = strings.TrimSpace(query)
		query = strings.ReplaceAll(query, "\n", " ")
		query = strings.ReplaceAll(query, "\r", " ")
		
		// Only log if it's not empty and not just whitespace
		if len(query) > 0 && strings.TrimSpace(query) != "" {
			a.logger.Printf("🗂️  [QUERY] Tunnel %s: %s", tunnelID, query)
		}
	} else if len(data) > 20 {
		// Check for common SQL keywords at the beginning
		queryText := string(data)
		upperQuery := strings.ToUpper(strings.TrimSpace(queryText))
		
		if strings.HasPrefix(upperQuery, "SELECT") || 
		   strings.HasPrefix(upperQuery, "INSERT") || 
		   strings.HasPrefix(upperQuery, "UPDATE") || 
		   strings.HasPrefix(upperQuery, "DELETE") || 
		   strings.HasPrefix(upperQuery, "SHOW") || 
		   strings.HasPrefix(upperQuery, "DESC") ||
		   strings.HasPrefix(upperQuery, "EXPLAIN") {
			// Clean up the query
			query := strings.TrimSpace(queryText)
			query = strings.ReplaceAll(query, "\n", " ")
			query = strings.ReplaceAll(query, "\r", " ")
			
			if len(query) > 0 {
				a.logger.Printf("🗂️  [QUERY] Tunnel %s: %s", tunnelID, query)
			}
		}
	}
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
