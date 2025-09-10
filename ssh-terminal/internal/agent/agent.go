package agent

import (
	"context"
	"crypto/tls"
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
	serverURL, err := url.Parse(a.config.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	
	// Convert HTTP(S) to WS(S)
	switch serverURL.Scheme {
	case "http":
		serverURL.Scheme = "ws"
	case "https":
		serverURL.Scheme = "wss"
	}
	
	// Add agent endpoint
	serverURL.Path = "/ws/agent"
	
	// Setup WebSocket dialer
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second
	
	if a.config.Insecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	
	// Add auth header
	headers := http.Header{}
	if a.config.Token != "" {
		headers.Set("Authorization", "Bearer "+a.config.Token)
	}
	
	a.logger.Printf("🔌 Connecting to server: %s", serverURL.String())
	
	// Connect
	conn, _, err := dialer.Dial(serverURL.String(), headers)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	
	// Create multiplexed session
	session, err := transport.NewMuxSession(conn, true)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create mux session: %w", err)
	}
	
	a.mu.Lock()
	a.session = session
	a.mu.Unlock()
	
	// Send registration
	if err := a.sendRegistration(); err != nil {
		session.Close()
		return fmt.Errorf("failed to send registration: %w", err)
	}
	
	a.logger.Printf("✅ Connected to server successfully")
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
	defer a.logger.Printf("🔚 Message handler stopped")
	
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
		}
		
		a.mu.RLock()
		session := a.session
		a.mu.RUnlock()
		
		if session == nil {
			time.Sleep(time.Second)
			continue
		}
		
		// Accept incoming streams
		stream, err := session.AcceptStream()
		if err != nil {
			a.logger.Printf("❌ Failed to accept stream: %v", err)
			time.Sleep(time.Second)
			continue
		}
		
		// Handle stream in goroutine
		go a.handleStream(stream)
	}
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
			msg := proto.NewMessage(proto.MessageTypeAgentHeartbeat)
			msg.AgentID = a.config.ID
			
			if err := a.sendMessage(msg); err != nil {
				a.logger.Printf("❌ Failed to send heartbeat: %v", err)
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
