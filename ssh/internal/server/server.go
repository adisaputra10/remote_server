package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"ssh-terminal/internal/proto"
	"ssh-terminal/internal/transport"
)

// stringAddr implements net.Addr for string addresses
type stringAddr struct {
	addr string
}

func (s stringAddr) Network() string { return "tcp" }
func (s stringAddr) String() string  { return s.addr }

// Config represents server configuration
type Config struct {
	Port        int    `json:"port"`
	Host        string `json:"host"`
	LogFile     string `json:"log_file"`
	Token       string `json:"token"`
	DatabaseURL string `json:"database_url"`
	EnableDB    bool   `json:"enable_database"`
	TLSCert     string `json:"tls_cert"`
	TLSKey      string `json:"tls_key"`
}

// Server represents the tunnel server
type Server struct {
	config   *Config
	logger   *log.Logger
	agents   map[string]*AgentConnection
	clients  map[string]*ClientConnection
	tunnels  map[string]*TunnelConnection
	upgrader websocket.Upgrader
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// AgentConnection represents a connected agent
type AgentConnection struct {
	ID       string
	Info     *proto.AgentInfo
	Session  *transport.MuxSession
	LastSeen time.Time
	mu       sync.RWMutex
}

// ClientConnection represents a connected client
type ClientConnection struct {
	ID       string
	Info     *proto.ClientInfo
	Session  *transport.MuxSession
	LastSeen time.Time
	mu       sync.RWMutex
}

// TunnelConnection represents an active tunnel
type TunnelConnection struct {
	ID           string
	AgentID      string
	ClientID     string
	Type         string
	LocalPort    int
	RemoteHost   string
	RemotePort   int
	Active       bool
	Info         *proto.TunnelInfo
	AgentStream  net.Conn
	ClientStream net.Conn
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
}

// NewServer creates a new server
func NewServer(config *Config, logger *log.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:  config,
		logger:  logger,
		agents:  make(map[string]*AgentConnection),
		clients: make(map[string]*ClientConnection),
		tunnels: make(map[string]*TunnelConnection),
		upgrader: websocket.Upgrader{
			CheckOrigin:       func(r *http.Request) bool { return true },
			ReadBufferSize:    32768,
			WriteBufferSize:   32768,
			HandshakeTimeout:  30 * time.Second,
			EnableCompression: false,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Printf("🚀 Starting server on %s:%d", s.config.Host, s.config.Port)

	// Setup HTTP routes
	http.HandleFunc("/ws/agent", s.handleAgentWebSocket)
	http.HandleFunc("/ws/client", s.handleClientWebSocket)
	http.HandleFunc("/ws/tunnel", s.handleTunnelWebSocket)
	http.HandleFunc("/api/agents", s.handleAgentsAPI)
	http.HandleFunc("/api/clients", s.handleClientsAPI)
	http.HandleFunc("/api/tunnels", s.handleTunnelsAPI)

	// Start cleanup routine
	go s.startCleanupRoutine()

	// Start server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Printf("✅ Server listening on %s", addr)

	if s.config.TLSCert != "" && s.config.TLSKey != "" {
		return http.ListenAndServeTLS(addr, s.config.TLSCert, s.config.TLSKey, nil)
	}

	return http.ListenAndServe(addr, nil)
}

// handleAgentWebSocket handles agent WebSocket connections
func (s *Server) handleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("📡 Agent connection attempt from: %s", r.RemoteAddr)

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("❌ Failed to upgrade agent connection: %v", err)
		return
	}
	defer conn.Close()

	// Create multiplexed session
	session, err := transport.NewMuxSession(conn, false) // Server side
	if err != nil {
		s.logger.Printf("❌ Failed to create agent mux session: %v", err)
		return
	}
	defer session.Close()

	// Handle agent registration and communication
	s.handleAgentSession(session, &stringAddr{r.RemoteAddr})
}

// handleClientWebSocket handles client WebSocket connections
func (s *Server) handleClientWebSocket(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("📡 Client connection attempt from: %s", r.RemoteAddr)

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("❌ Failed to upgrade client connection: %v", err)
		return
	}
	defer conn.Close()

	// Create multiplexed session
	session, err := transport.NewMuxSession(conn, false) // Server side
	if err != nil {
		s.logger.Printf("❌ Failed to create client mux session: %v", err)
		return
	}
	defer session.Close()

	// Handle client registration and communication
	s.handleClientSession(session, &stringAddr{r.RemoteAddr})
}

// handleTunnelWebSocket handles tunnel WebSocket connections (for existing client compatibility)
func (s *Server) handleTunnelWebSocket(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("📡 Tunnel connection attempt from: %s", r.RemoteAddr)

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("❌ Failed to upgrade tunnel connection: %v", err)
		return
	}
	defer conn.Close()

	// Handle tunnel request (legacy mode for existing client)
	s.handleLegacyTunnel(conn, r.RemoteAddr)
}

// handleAgentSession handles agent session communication
func (s *Server) handleAgentSession(session *transport.MuxSession, remoteAddr net.Addr) {
	var agentConn *AgentConnection

	defer func() {
		if agentConn != nil {
			s.logger.Printf("📝 Agent disconnected: %s", agentConn.ID)
			s.mu.Lock()
			delete(s.agents, agentConn.ID)
			s.mu.Unlock()
		}
	}()

	// Wait for registration
	stream, err := session.AcceptStream()
	if err != nil {
		s.logger.Printf("❌ Failed to accept agent stream: %v", err)
		return
	}

	// Read registration message
	buffer := make([]byte, 64*1024)
	n, err := stream.Read(buffer)
	if err != nil {
		s.logger.Printf("❌ Failed to read agent registration: %v", err)
		stream.Close()
		return
	}

	msg, err := proto.FromJSON(buffer[:n])
	if err != nil {
		s.logger.Printf("❌ Failed to parse agent registration: %v", err)
		stream.Close()
		return
	}

	if msg.Type != proto.MessageTypeAgentRegister {
		s.logger.Printf("❌ Expected agent registration, got: %s", msg.Type)
		stream.Close()
		return
	}

	agentInfo, ok := msg.Data.(*proto.AgentInfo)
	if !ok {
		s.logger.Printf("❌ Invalid agent registration data")
		stream.Close()
		return
	}

	// Create agent connection
	agentConn = &AgentConnection{
		ID:       agentInfo.ID,
		Info:     agentInfo,
		Session:  session,
		LastSeen: time.Now(),
	}

	s.mu.Lock()
	s.agents[agentConn.ID] = agentConn
	s.mu.Unlock()

	s.logger.Printf("✅ Agent registered: %s (%s)", agentInfo.Name, agentInfo.ID)
	stream.Close()

	// Handle incoming streams
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		_, err := session.AcceptStream()
		if err != nil {
			s.logger.Printf("❌ Failed to accept agent stream: %v", err)
			return
		}

		go s.handleAgentMessages(agentConn)
	}
}

// handleClientSession handles client session communication
func (s *Server) handleClientSession(session *transport.MuxSession, remoteAddr net.Addr) {
	var clientConn *ClientConnection

	defer func() {
		if clientConn != nil {
			s.logger.Printf("📝 Client disconnected: %s", clientConn.ID)
			s.mu.Lock()
			delete(s.clients, clientConn.ID)
			s.mu.Unlock()
		}
	}()

	// Wait for registration
	stream, err := session.AcceptStream()
	if err != nil {
		s.logger.Printf("❌ Failed to accept client stream: %v", err)
		return
	}

	// Read registration message
	buffer := make([]byte, 64*1024)
	n, err := stream.Read(buffer)
	if err != nil {
		s.logger.Printf("❌ Failed to read client registration: %v", err)
		stream.Close()
		return
	}

	msg, err := proto.FromJSON(buffer[:n])
	if err != nil {
		s.logger.Printf("❌ Failed to parse client registration: %v", err)
		stream.Close()
		return
	}

	if msg.Type != proto.MessageTypeClientConnect {
		s.logger.Printf("❌ Expected client registration, got: %s", msg.Type)
		stream.Close()
		return
	}

	clientInfo, ok := msg.Data.(*proto.ClientInfo)
	if !ok {
		s.logger.Printf("❌ Invalid client registration data")
		stream.Close()
		return
	}

	// Create client connection
	clientConn = &ClientConnection{
		ID:       clientInfo.ID,
		Info:     clientInfo,
		Session:  session,
		LastSeen: time.Now(),
	}

	s.mu.Lock()
	s.clients[clientConn.ID] = clientConn
	s.mu.Unlock()

	s.logger.Printf("✅ Client registered: %s (%s)", clientInfo.Name, clientInfo.ID)
	stream.Close()

	// Handle incoming streams
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		_, err := session.AcceptStream()
		if err != nil {
			s.logger.Printf("❌ Failed to accept client stream: %v", err)
			return
		}

		go s.handleClientMessages(clientConn)
	}
}

// handleDatabaseCommand logs database commands
func (s *Server) handleDatabaseCommand(msg *proto.Message) {
	// This method can be implemented based on the old proto structure
	s.logger.Printf("📝 DB_COMMAND received")
}

// startCleanupRoutine starts the cleanup routine for stale connections
func (s *Server) startCleanupRoutine() {
	ticker := time.NewTicker(60 * time.Second) // Cleanup every minute
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupStaleConnections()
		}
	}
}

// cleanupStaleConnections removes stale agent and client connections
func (s *Server) cleanupStaleConnections() {
	now := time.Now()
	staleThreshold := 5 * time.Minute

	s.mu.Lock()
	defer s.mu.Unlock()

	// Cleanup stale agents
	for id, agent := range s.agents {
		agent.mu.RLock()
		lastSeen := agent.LastSeen
		agent.mu.RUnlock()

		if now.Sub(lastSeen) > staleThreshold {
			s.logger.Printf("🧹 Removing stale agent: %s", id)
			agent.Session.Close()
			delete(s.agents, id)
		}
	}

	// Cleanup stale clients
	for id, client := range s.clients {
		client.mu.RLock()
		lastSeen := client.LastSeen
		client.mu.RUnlock()

		if now.Sub(lastSeen) > staleThreshold {
			s.logger.Printf("🧹 Removing stale client: %s", id)
			client.Session.Close()
			delete(s.clients, id)
		}
	}
}

// Stop stops the server
func (s *Server) Stop() {
	s.logger.Printf("🛑 Stopping server")
	s.cancel()

	// Close all connections
	s.mu.Lock()
	for _, agent := range s.agents {
		agent.Session.Close()
	}
	for _, client := range s.clients {
		client.Session.Close()
	}
	for _, tunnel := range s.tunnels {
		tunnel.cancel()
	}
	s.mu.Unlock()
}
