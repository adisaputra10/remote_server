package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GoTeleportServer struct {
	agents      map[string]*Agent
	clients     map[string]*Client
	sessions    map[string]*Session
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
	config      *ServerConfig
	logger      *log.Logger
}

type ServerConfig struct {
	Port       int    `json:"port"`
	LogFile    string `json:"log_file"`
	MaxAgents  int    `json:"max_agents"`
	MaxClients int    `json:"max_clients"`
	AuthToken  string `json:"auth_token"`
}

type Agent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Address     string                 `json:"address"`
	Platform    string                 `json:"platform"`
	Status      string                 `json:"status"`
	LastSeen    time.Time              `json:"last_seen"`
	Connection  *websocket.Conn        `json:"-"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type Client struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Status     string              `json:"status"`
	LastSeen   time.Time           `json:"last_seen"`
	Connection *websocket.Conn     `json:"-"`
}

type Session struct {
	ID        string              `json:"id"`
	ClientID  string              `json:"client_id"`
	AgentID   string              `json:"agent_id"`
	Status    string              `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	LastUsed  time.Time           `json:"last_used"`
}

type Message struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	Command   string                 `json:"command,omitempty"`
	Data      string                 `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: goteleport-server.exe <config-file>")
	}

	server, err := NewGoTeleportServer(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	server.Start()
}

func NewGoTeleportServer(configFile string) (*GoTeleportServer, error) {
	// Read config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Setup logger
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	logger := log.New(logFile, "", log.LstdFlags)

	server := &GoTeleportServer{
		agents:   make(map[string]*Agent),
		clients:  make(map[string]*Client),
		sessions: make(map[string]*Session),
		config:   &config,
		logger:   logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}

	return server, nil
}

func (s *GoTeleportServer) Start() {
	s.logEvent("SERVER_START", "GoTeleport Server starting", fmt.Sprintf("Port: %d", s.config.Port))

	// Setup HTTP routes
	http.HandleFunc("/", s.handleWebUI)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/agents", s.handleAgents)
	http.HandleFunc("/api/clients", s.handleClients)
	http.HandleFunc("/api/sessions", s.handleSessions)
	
	// WebSocket endpoints
	http.HandleFunc("/ws/agent", s.handleAgentWebSocket)
	http.HandleFunc("/ws/client", s.handleClientWebSocket)

	// Start cleanup routine
	go s.cleanupRoutine()

	addr := fmt.Sprintf(":%d", s.config.Port)
	fmt.Printf("🚀 GoTeleport Server starting on %s\n", addr)
	fmt.Printf("🌐 Web UI: http://localhost%s\n", addr)
	fmt.Printf("📡 Agent WS: ws://localhost%s/ws/agent\n", addr)
	fmt.Printf("💻 Client WS: ws://localhost%s/ws/client\n", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *GoTeleportServer) handleWebUI(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>GoTeleport Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .card { background: white; padding: 20px; margin: 20px 0; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .status { display: inline-block; padding: 4px 8px; border-radius: 4px; color: white; font-size: 12px; }
        .online { background: #28a745; }
        .offline { background: #dc3545; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #007bff; color: white; }
        .refresh-btn { background: #007bff; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <h1>🚀 GoTeleport Server Dashboard</h1>
            <p>Layer 7 Remote Access Server</p>
            <button class="refresh-btn" onclick="location.reload()">🔄 Refresh</button>
        </div>
        
        <div class="card">
            <h2>📊 Server Status</h2>
            <div id="status">Loading...</div>
        </div>
        
        <div class="card">
            <h2>🖥️ Connected Agents</h2>
            <div id="agents">Loading...</div>
        </div>
        
        <div class="card">
            <h2>💻 Connected Clients</h2>
            <div id="clients">Loading...</div>
        </div>
        
        <div class="card">
            <h2>🔗 Active Sessions</h2>
            <div id="sessions">Loading...</div>
        </div>
    </div>

    <script>
        function loadData() {
            // Load status
            fetch('/api/status')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('status').innerHTML = 
                        '<p><strong>Uptime:</strong> ' + data.uptime + '</p>' +
                        '<p><strong>Agents:</strong> ' + data.agent_count + '</p>' +
                        '<p><strong>Clients:</strong> ' + data.client_count + '</p>' +
                        '<p><strong>Sessions:</strong> ' + data.session_count + '</p>';
                });
            
            // Load agents
            fetch('/api/agents')
                .then(r => r.json())
                .then(agents => {
                    if (agents.length === 0) {
                        document.getElementById('agents').innerHTML = '<p>No agents connected</p>';
                        return;
                    }
                    
                    let html = '<table><tr><th>Name</th><th>Platform</th><th>Status</th><th>Last Seen</th></tr>';
                    agents.forEach(agent => {
                        html += '<tr>' +
                            '<td>' + agent.name + '</td>' +
                            '<td>' + agent.platform + '</td>' +
                            '<td><span class="status ' + (agent.status === 'online' ? 'online' : 'offline') + '">' + agent.status + '</span></td>' +
                            '<td>' + new Date(agent.last_seen).toLocaleString() + '</td>' +
                            '</tr>';
                    });
                    html += '</table>';
                    document.getElementById('agents').innerHTML = html;
                });
        }
        
        loadData();
        setInterval(loadData, 5000); // Refresh every 5 seconds
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *GoTeleportServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	status := map[string]interface{}{
		"agent_count":   len(s.agents),
		"client_count":  len(s.clients),
		"session_count": len(s.sessions),
		"uptime":       time.Since(time.Now()).String(),
	}
	s.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *GoTeleportServer) handleAgents(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	agents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	s.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (s *GoTeleportServer) handleClients(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	clients := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func (s *GoTeleportServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	s.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *GoTeleportServer) handleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logEvent("ERROR", "Failed to upgrade agent connection", err.Error())
		return
	}

	s.handleAgentConnection(conn)
}

func (s *GoTeleportServer) handleClientWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logEvent("ERROR", "Failed to upgrade client connection", err.Error())
		return
	}

	s.handleClientConnection(conn)
}

func (s *GoTeleportServer) handleAgentConnection(conn *websocket.Conn) {
	defer conn.Close()

	// Generate agent ID
	agentID := s.generateID()
	
	agent := &Agent{
		ID:         agentID,
		Status:     "online",
		LastSeen:   time.Now(),
		Connection: conn,
		Metadata:   make(map[string]interface{}),
	}

	// Wait for agent registration
	var regMsg Message
	if err := conn.ReadJSON(&regMsg); err != nil {
		s.logEvent("ERROR", "Failed to read agent registration", err.Error())
		return
	}

	if regMsg.Type != "register" {
		s.logEvent("ERROR", "Invalid agent registration message", regMsg.Type)
		return
	}

	// Update agent info
	if name, ok := regMsg.Metadata["name"].(string); ok {
		agent.Name = name
	}
	if platform, ok := regMsg.Metadata["platform"].(string); ok {
		agent.Platform = platform
	}
	agent.Metadata = regMsg.Metadata

	// Register agent
	s.mutex.Lock()
	s.agents[agentID] = agent
	s.mutex.Unlock()

	s.logEvent("AGENT_CONNECT", "Agent registered", fmt.Sprintf("ID: %s, Name: %s", agentID, agent.Name))

	// Send registration response
	response := Message{
		Type:      "registered",
		AgentID:   agentID,
		Timestamp: time.Now(),
	}
	conn.WriteJSON(response)

	// Handle agent messages
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			s.logEvent("AGENT_DISCONNECT", "Agent disconnected", fmt.Sprintf("ID: %s, Error: %v", agentID, err))
			break
		}

		agent.LastSeen = time.Now()
		s.handleAgentMessage(agent, &msg)
	}

	// Cleanup
	s.mutex.Lock()
	delete(s.agents, agentID)
	s.mutex.Unlock()
}

func (s *GoTeleportServer) handleClientConnection(conn *websocket.Conn) {
	defer conn.Close()

	// Generate client ID
	clientID := s.generateID()
	
	client := &Client{
		ID:         clientID,
		Status:     "online",
		LastSeen:   time.Now(),
		Connection: conn,
	}

	// Wait for client registration
	var regMsg Message
	if err := conn.ReadJSON(&regMsg); err != nil {
		s.logEvent("ERROR", "Failed to read client registration", err.Error())
		return
	}

	if regMsg.Type != "register" {
		s.logEvent("ERROR", "Invalid client registration message", regMsg.Type)
		return
	}

	if name, ok := regMsg.Metadata["name"].(string); ok {
		client.Name = name
	}

	// Register client
	s.mutex.Lock()
	s.clients[clientID] = client
	s.mutex.Unlock()

	s.logEvent("CLIENT_CONNECT", "Client registered", fmt.Sprintf("ID: %s, Name: %s", clientID, client.Name))

	// Send registration response
	response := Message{
		Type:      "registered",
		ClientID:  clientID,
		Timestamp: time.Now(),
	}
	conn.WriteJSON(response)

	// Handle client messages
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			s.logEvent("CLIENT_DISCONNECT", "Client disconnected", fmt.Sprintf("ID: %s, Error: %v", clientID, err))
			break
		}

		client.LastSeen = time.Now()
		s.handleClientMessage(client, &msg)
	}

	// Cleanup
	s.mutex.Lock()
	delete(s.clients, clientID)
	s.mutex.Unlock()
}

func (s *GoTeleportServer) handleAgentMessage(agent *Agent, msg *Message) {
	switch msg.Type {
	case "command_result":
		// Forward command result to client
		s.forwardToClient(msg.SessionID, msg)
	case "heartbeat":
		// Update last seen
		agent.LastSeen = time.Now()
	default:
		s.logEvent("AGENT_MSG", "Unknown message type", msg.Type)
	}
}

func (s *GoTeleportServer) handleClientMessage(client *Client, msg *Message) {
	switch msg.Type {
	case "list_agents":
		s.sendAgentList(client)
	case "connect_agent":
		s.createSession(client, msg.AgentID)
	case "command":
		s.forwardToAgent(msg.SessionID, msg)
	case "disconnect":
		s.closeSession(msg.SessionID)
	default:
		s.logEvent("CLIENT_MSG", "Unknown message type", msg.Type)
	}
}

func (s *GoTeleportServer) sendAgentList(client *Client) {
	s.mutex.RLock()
	agents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	s.mutex.RUnlock()

	response := Message{
		Type:      "agent_list",
		Data:      fmt.Sprintf("%d", len(agents)),
		Metadata:  map[string]interface{}{"agents": agents},
		Timestamp: time.Now(),
	}

	client.Connection.WriteJSON(response)
}

func (s *GoTeleportServer) createSession(client *Client, agentID string) {
	sessionID := s.generateID()

	session := &Session{
		ID:        sessionID,
		ClientID:  client.ID,
		AgentID:   agentID,
		Status:    "active",
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	s.mutex.Lock()
	s.sessions[sessionID] = session
	s.mutex.Unlock()

	s.logEvent("SESSION_CREATE", "Session created", fmt.Sprintf("ID: %s, Client: %s, Agent: %s", sessionID, client.ID, agentID))

	// Notify client
	response := Message{
		Type:      "session_created",
		SessionID: sessionID,
		AgentID:   agentID,
		ClientID:  client.ID,
		Timestamp: time.Now(),
	}

	client.Connection.WriteJSON(response)
}

func (s *GoTeleportServer) forwardToAgent(sessionID string, msg *Message) {
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	if !exists {
		s.mutex.RUnlock()
		return
	}

	agent, exists := s.agents[session.AgentID]
	if !exists {
		s.mutex.RUnlock()
		return
	}
	s.mutex.RUnlock()

	// Update session
	session.LastUsed = time.Now()

	// Forward to agent
	agent.Connection.WriteJSON(msg)
	
	s.logEvent("FORWARD_AGENT", "Message forwarded to agent", fmt.Sprintf("Session: %s, Command: %s", sessionID, msg.Command))
}

func (s *GoTeleportServer) forwardToClient(sessionID string, msg *Message) {
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	if !exists {
		s.mutex.RUnlock()
		return
	}

	client, exists := s.clients[session.ClientID]
	if !exists {
		s.mutex.RUnlock()
		return
	}
	s.mutex.RUnlock()

	// Update session
	session.LastUsed = time.Now()

	// Forward to client
	client.Connection.WriteJSON(msg)
	
	s.logEvent("FORWARD_CLIENT", "Message forwarded to client", fmt.Sprintf("Session: %s", sessionID))
}

func (s *GoTeleportServer) closeSession(sessionID string) {
	s.mutex.Lock()
	delete(s.sessions, sessionID)
	s.mutex.Unlock()

	s.logEvent("SESSION_CLOSE", "Session closed", sessionID)
}

func (s *GoTeleportServer) generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])[:16]
}

func (s *GoTeleportServer) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		
		s.mutex.Lock()
		// Cleanup old sessions (inactive for 10 minutes)
		for id, session := range s.sessions {
			if now.Sub(session.LastUsed) > 10*time.Minute {
				delete(s.sessions, id)
				s.logEvent("SESSION_CLEANUP", "Session cleaned up due to inactivity", id)
			}
		}
		s.mutex.Unlock()
	}
}

func (s *GoTeleportServer) logEvent(eventType, description, details string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] [%s] %s | %s", timestamp, eventType, description, details)
	
	if s.logger != nil {
		s.logger.Println(logEntry)
	}
	
	// Also print to stdout
	fmt.Printf("📝 %s\n", logEntry)
}
