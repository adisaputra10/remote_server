package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	
	"ssh-terminal/internal/proto"
	"ssh-terminal/internal/transport"
)

// Config represents agent configuration
type Config struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	ServerURL       string                    `json:"server_url"`
	Token           string                    `json:"token"`
	Platform        string                    `json:"platform"`
	Version         string                    `json:"version"`
	LogFile         string                    `json:"log_file"`
	DatabaseProxies []DatabaseProxyConfig     `json:"database_proxies"`
	Metadata        map[string]string         `json:"metadata"`
	Insecure        bool                      `json:"insecure"`
}

type DatabaseProxyConfig struct {
	Name       string `json:"name"`
	LocalPort  int    `json:"local_port"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port"`
	Protocol   string `json:"protocol"`
	Enabled    bool   `json:"enabled"`
}

// Agent represents the tunnel agent
type Agent struct {
	config    *Config
	logger    *log.Logger
	session   *transport.MuxSession
	wsConn    *websocket.Conn  // Direct WebSocket connection for simple mode
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	proxies   map[string]*DatabaseProxy
	tunnels   map[string]*Tunnel
}

// DatabaseProxy handles database connections
type DatabaseProxy struct {
	Name       string
	LocalPort  int
	TargetHost string
	TargetPort int
	Protocol   string
	listener   net.Listener
	agent      *Agent
	ctx        context.Context
	cancel     context.CancelFunc
}

// Tunnel handles individual tunnel connections
type Tunnel struct {
	ID         string
	AgentConn  net.Conn
	TargetConn net.Conn
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAgent creates a new agent
func NewAgent(config *Config, logger *log.Logger) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Generate persistent ID if not provided
	if config.ID == "" {
		config.ID = generatePersistentID(config.Name, runtime.GOOS)
	}
	
	return &Agent{
		config:  config,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		proxies: make(map[string]*DatabaseProxy),
		tunnels: make(map[string]*Tunnel),
	}
}

// generatePersistentID creates a consistent ID based on name and platform
func generatePersistentID(name, platform string) string {
	data := fmt.Sprintf("%s-%s", name, platform)
	hash := fmt.Sprintf("%x", []byte(data))
	if len(hash) > 16 {
		hash = hash[:16]
	}
	return hash
}

// Start starts the agent
func (a *Agent) Start() error {
	a.logger.Printf("🚀 Starting agent: %s (ID: %s)", a.config.Name, a.config.ID)
	
	// Start database proxies
	if err := a.startDatabaseProxies(); err != nil {
		return fmt.Errorf("failed to start database proxies: %w", err)
	}
	
	// Connect to server
	if err := a.connectToServer(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	
	// Start message handler
	go a.handleMessages()
	
	// Start heartbeat
	go a.startHeartbeat()
	
	return nil
}

// connectToServer establishes connection to the server
func (a *Agent) connectToServer() error {
	a.logger.Printf("🔄 Parsing server URL: %s", a.config.ServerURL)
	
	serverURL, err := url.Parse(a.config.ServerURL)
	if err != nil {
		a.logger.Printf("❌ Invalid server URL: %v", err)
		return fmt.Errorf("invalid server URL: %w", err)
	}
	
	// Convert HTTP(S) to WS(S)
	switch serverURL.Scheme {
	case "http":
		serverURL.Scheme = "ws"
		a.logger.Printf("🔄 Converting HTTP to WebSocket")
	case "https":
		serverURL.Scheme = "wss"
		a.logger.Printf("🔄 Converting HTTPS to WebSocket Secure")
	}
	
	// Add agent endpoint
	serverURL.Path = "/agent"
	a.logger.Printf("🔄 Final WebSocket URL: %s", serverURL.String())
	
	// Setup WebSocket dialer
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second
	
	if a.config.Insecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		a.logger.Printf("⚠️  TLS verification disabled")
	}
	
	// Add auth header
	headers := http.Header{}
	if a.config.Token != "" {
		headers.Set("Authorization", "Bearer "+a.config.Token)
		a.logger.Printf("🔐 Adding authorization header")
	}
	
	a.logger.Printf("🔌 Attempting WebSocket connection...")
	
	// Connect directly with WebSocket (no yamux for now)
	conn, resp, err := dialer.Dial(serverURL.String(), headers)
	if err != nil {
		a.logger.Printf("❌ WebSocket dial failed: %v", err)
		if resp != nil {
			a.logger.Printf("❌ HTTP Response: %d %s", resp.StatusCode, resp.Status)
		}
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	
	a.logger.Printf("✅ WebSocket connection established")
	
	// Store connection directly for simple mode
	a.mu.Lock()
	// Create a simple session wrapper
	a.session = &transport.MuxSession{} // We'll use direct websocket for now
	a.mu.Unlock()
	
	a.logger.Printf("🔄 Sending registration to server...")
	
	// Send registration via direct websocket
	if err := a.sendRegistrationDirect(conn); err != nil {
		conn.Close()
		a.logger.Printf("❌ Registration failed: %v", err)
		return fmt.Errorf("failed to send registration: %w", err)
	}
	
	// Store the connection for message handling
	a.wsConn = conn
	
	a.logger.Printf("✅ Connected and registered to server successfully")
	return nil
}

// sendRegistrationDirect sends registration via direct websocket
func (a *Agent) sendRegistrationDirect(conn *websocket.Conn) error {
	a.logger.Printf("📝 Preparing registration data...")
	
	registration := map[string]interface{}{
		"type": "agent_register",
		"data": map[string]interface{}{
			"id":       a.config.ID,
			"name":     a.config.Name,
			"platform": a.config.Platform,
			"version":  a.config.Version,
			"token":    a.config.Token,
		},
	}
	
	a.logger.Printf("📋 Registration info: ID=%s, Name=%s, Platform=%s", 
		a.config.ID, a.config.Name, a.config.Platform)
	
	data, err := json.Marshal(registration)
	if err != nil {
		a.logger.Printf("❌ Failed to marshal registration: %v", err)
		return fmt.Errorf("failed to marshal registration: %w", err)
	}
	
	a.logger.Printf("📤 Sending registration message [%d bytes]...", len(data))
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		a.logger.Printf("❌ Failed to send registration: %v", err)
		return fmt.Errorf("failed to send registration: %w", err)
	}
	
	a.logger.Printf("✅ Registration sent successfully")
	return nil
}

// sendRegistration sends agent registration to server
func (a *Agent) sendRegistration() error {
	msg := proto.NewMessage(proto.MessageTypeAgentRegister)
	msg.AgentID = a.config.ID
	msg.Data = &proto.AgentInfo{
		ID:       a.config.ID,
		Name:     a.config.Name,
		Platform: a.config.Platform,
		Version:  a.config.Version,
		Metadata: a.config.Metadata,
		LastSeen: time.Now(),
	}
	
	return a.sendMessage(msg)
}

// sendMessage sends a message to the server
func (a *Agent) sendMessage(msg *proto.Message) error {
	a.mu.RLock()
	session := a.session
	a.mu.RUnlock()
	
	if session == nil {
		return fmt.Errorf("no active session")
	}
	
	// Open control stream for messages
	stream, err := session.OpenStream()
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()
	
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	_, err = stream.Write(data)
	return err
}

// handleMessages handles incoming messages
func (a *Agent) handleMessages() {
	a.logger.Printf("🎧 Starting message handler...")
	defer a.logger.Printf("🔚 Message handler stopped")
	
	for {
		select {
		case <-a.ctx.Done():
			a.logger.Printf("🛑 Context cancelled, stopping message handler")
			return
		default:
		}
		
		a.mu.RLock()
		conn := a.wsConn
		a.mu.RUnlock()
		
		if conn == nil {
			a.logger.Printf("⏳ No connection available, waiting...")
			time.Sleep(time.Second)
			continue
		}
		
		// Read message from WebSocket directly
		_, data, err := conn.ReadMessage()
		if err != nil {
			a.logger.Printf("❌ Failed to read message: %v", err)
			time.Sleep(time.Second)
			continue
		}
		
		a.logger.Printf("📨 Received message [%d bytes]", len(data))
		
		// Parse and handle message
		go a.handleMessageData(data)
	}
}

// handleMessageData processes incoming message data
func (a *Agent) handleMessageData(data []byte) {
	a.logger.Printf("� Processing message: %s", string(data))
	
	// Simple echo back for now (to test basic communication)
	a.mu.RLock()
	conn := a.wsConn
	a.mu.RUnlock()
	
	if conn != nil {
		a.logger.Printf("📤 Echoing message back to server...")
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			a.logger.Printf("❌ Failed to echo message: %v", err)
		} else {
			a.logger.Printf("✅ Message echoed successfully")
		}
	}
}

// handlePing responds to ping messages
func (a *Agent) handlePing(msg map[string]interface{}) {
	response := map[string]interface{}{
		"type": "pong",
		"data": "Agent is alive",
	}
	a.sendMessageDirect(response)
}

// handleTestMessage handles test messages
func (a *Agent) handleTestMessage(msg map[string]interface{}) {
	a.logger.Printf("🧪 Test message received: %v", msg["data"])
	
	response := map[string]interface{}{
		"type": "test_response",
		"data": fmt.Sprintf("Agent processed: %v", msg["data"]),
	}
	a.sendMessageDirect(response)
}

// sendMessageDirect sends message via direct websocket
func (a *Agent) sendMessageDirect(msg map[string]interface{}) error {
	a.mu.RLock()
	conn := a.wsConn
	a.mu.RUnlock()
	
	if conn == nil {
		return fmt.Errorf("no websocket connection")
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	return nil
}

// handleStream handles an individual stream
func (a *Agent) handleStream(stream net.Conn) {
	defer stream.Close()
	
	// Read message
	buffer := make([]byte, 64*1024) // 64KB buffer
	n, err := stream.Read(buffer)
	if err != nil {
		a.logger.Printf("❌ Failed to read from stream: %v", err)
		return
	}
	
	// Parse message
	msg, err := proto.FromJSON(buffer[:n])
	if err != nil {
		a.logger.Printf("❌ Failed to parse message: %v", err)
		return
	}
	
	// Handle message based on type
	switch msg.Type {
	case proto.MessageTypeTunnelStart:
		a.handleTunnelStart(msg, stream)
	case proto.MessageTypeClientCommand:
		a.handleCommand(msg, stream)
	default:
		a.logger.Printf("⚠️ Unknown message type: %s", msg.Type)
	}
}

// handleTunnelStart handles tunnel start requests
func (a *Agent) handleTunnelStart(msg *proto.Message, agentStream net.Conn) {
	tunnelReq, ok := msg.Data.(*proto.TunnelRequest)
	if !ok {
		a.logger.Printf("❌ Invalid tunnel request data")
		return
	}
	
	a.logger.Printf("🔄 Starting tunnel to %s:%d", tunnelReq.TargetHost, tunnelReq.TargetPort)
	
	// Connect to target
	targetAddr := fmt.Sprintf("%s:%d", tunnelReq.TargetHost, tunnelReq.TargetPort)
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		a.logger.Printf("❌ Failed to connect to target %s: %v", targetAddr, err)
		
		// Send error response
		errorMsg := proto.NewMessage(proto.MessageTypeTunnelError)
		errorMsg.SessionID = msg.SessionID
		errorMsg.Error = err.Error()
		a.sendMessage(errorMsg)
		return
	}
	
	// Send ready response
	readyMsg := proto.NewMessage(proto.MessageTypeTunnelReady)
	readyMsg.SessionID = msg.SessionID
	a.sendMessage(readyMsg)
	
	// Create tunnel
	tunnel := &Tunnel{
		ID:         msg.SessionID,
		AgentConn:  agentStream,
		TargetConn: targetConn,
	}
	tunnel.ctx, tunnel.cancel = context.WithCancel(a.ctx)
	
	a.mu.Lock()
	a.tunnels[tunnel.ID] = tunnel
	a.mu.Unlock()
	
	// Start bidirectional copy
	go tunnel.start()
}

// handleCommand handles command execution requests
func (a *Agent) handleCommand(msg *proto.Message, stream net.Conn) {
	command, ok := msg.Data.(string)
	if !ok {
		a.logger.Printf("❌ Invalid command data")
		return
	}
	
	a.logger.Printf("📋 Executing command: %s", command)
	
	// Execute command (implement as needed)
	result := fmt.Sprintf("Command executed: %s", command)
	
	// Send response
	response := proto.NewMessage(proto.MessageTypeResponse)
	response.SessionID = msg.SessionID
	response.Data = result
	
	responseData, _ := response.ToJSON()
	stream.Write(responseData)
}

// startHeartbeat sends periodic heartbeats
func (a *Agent) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			heartbeat := map[string]interface{}{
				"type": "heartbeat",
				"data": map[string]interface{}{
					"status":    "alive",
					"timestamp": time.Now().Unix(),
					"agent_id":  a.config.ID,
				},
			}
			
			if err := a.sendMessageDirect(heartbeat); err != nil {
				a.logger.Printf("❌ Failed to send heartbeat: %v", err)
			} else {
				a.logger.Printf("💓 Heartbeat sent")
			}
		}
	}
}

// startDatabaseProxies starts all configured database proxies
func (a *Agent) startDatabaseProxies() error {
	for _, proxyConfig := range a.config.DatabaseProxies {
		if !proxyConfig.Enabled {
			continue
		}
		
		proxy := &DatabaseProxy{
			Name:       proxyConfig.Name,
			LocalPort:  proxyConfig.LocalPort,
			TargetHost: proxyConfig.TargetHost,
			TargetPort: proxyConfig.TargetPort,
			Protocol:   proxyConfig.Protocol,
			agent:      a,
		}
		proxy.ctx, proxy.cancel = context.WithCancel(a.ctx)
		
		if err := proxy.start(); err != nil {
			return fmt.Errorf("failed to start proxy %s: %w", proxy.Name, err)
		}
		
		a.proxies[proxy.Name] = proxy
		a.logger.Printf("✅ Database proxy %s started on port %d", proxy.Name, proxy.LocalPort)
	}
	
	return nil
}

// Stop stops the agent
func (a *Agent) Stop() {
	a.logger.Printf("🛑 Stopping agent")
	a.cancel()
	
	// Close session
	a.mu.Lock()
	if a.session != nil {
		a.session.Close()
		a.session = nil
	}
	a.mu.Unlock()
	
	// Stop proxies
	for _, proxy := range a.proxies {
		proxy.stop()
	}
	
	// Stop tunnels
	for _, tunnel := range a.tunnels {
		tunnel.stop()
	}
}
