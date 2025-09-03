package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/go-sql-driver/mysql"
)

type GoTeleportServerDB struct {
	config    *ServerConfig
	agents    map[string]*Agent
	clients   map[string]*Client
	sessions  map[string]*Session
	mutex     sync.RWMutex
	logger    *log.Logger
	db        *sql.DB
	upgrader  websocket.Upgrader
}

type ServerConfig struct {
	Port        int    `json:"port"`
	LogFile     string `json:"log_file"`
	AuthToken   string `json:"auth_token"`
	DatabaseURL string `json:"database_url"`
	EnableDB    bool   `json:"enable_database"`
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
	Address    string              `json:"address"`
	Metadata   map[string]string   `json:"metadata"`
}

type Session struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	ClientID  string    `json:"client_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
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

type CommandLog struct {
	ID        int       `json:"id"`
	SessionID string    `json:"session_id"`
	ClientID  string    `json:"client_id"`
	AgentID   string    `json:"agent_id"`
	Command   string    `json:"command"`
	Output    string    `json:"output"`
	Status    string    `json:"status"`
	Duration  int64     `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: goteleport-server.exe <config-file>")
	}

	server, err := NewGoTeleportServerDB(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	fmt.Printf("🚀 GoTeleport Server with MySQL Logging\n")
	fmt.Printf("📊 Port: %d\n", server.config.Port)
	fmt.Printf("📝 Log File: %s\n", server.config.LogFile)
	fmt.Printf("🗄️  Database: %s\n", server.config.DatabaseURL)

	server.logEvent("SERVER_START", "GoTeleport Server starting", fmt.Sprintf("Port: %d, DB: %v", server.config.Port, server.config.EnableDB))

	// Setup routes
	http.HandleFunc("/ws/agent", server.handleAgentConnection)
	http.HandleFunc("/ws/client", server.handleClientConnection)
	http.HandleFunc("/api/logs", server.handleLogsAPI)
	http.HandleFunc("/api/sessions", server.handleSessionsAPI)
	http.HandleFunc("/api/stats", server.handleStatsAPI)

	// Start server
	addr := fmt.Sprintf(":%d", server.config.Port)
	fmt.Printf("🌐 Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func NewGoTeleportServerDB(configFile string) (*GoTeleportServerDB, error) {
	// Read config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Setup logging
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	logger := log.New(logFile, "", log.LstdFlags)

	server := &GoTeleportServerDB{
		config:   &config,
		agents:   make(map[string]*Agent),
		clients:  make(map[string]*Client),
		sessions: make(map[string]*Session),
		logger:   logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	// Setup database if enabled
	if config.EnableDB {
		if err := server.initDatabase(); err != nil {
			return nil, fmt.Errorf("failed to init database: %v", err)
		}
	}

	return server, nil
}

func (s *GoTeleportServerDB) initDatabase() error {
	db, err := sql.Open("mysql", s.config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	s.db = db

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Create tables
	if err := s.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	fmt.Println("✅ Database connected and tables ready")
	return nil
}

func (s *GoTeleportServerDB) createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS command_logs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			session_id VARCHAR(255) NOT NULL,
			client_id VARCHAR(255) NOT NULL,
			agent_id VARCHAR(255) NOT NULL,
			command TEXT NOT NULL,
			output TEXT,
			status VARCHAR(50) DEFAULT 'executed',
			duration_ms BIGINT DEFAULT 0,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_session (session_id),
			INDEX idx_client (client_id),
			INDEX idx_agent (agent_id),
			INDEX idx_timestamp (timestamp)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR(255) PRIMARY KEY,
			agent_id VARCHAR(255) NOT NULL,
			client_id VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_agent (agent_id),
			INDEX idx_client (client_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS agents (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			platform VARCHAR(100),
			status VARCHAR(50) DEFAULT 'online',
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			metadata JSON,
			INDEX idx_status (status),
			INDEX idx_last_seen (last_seen)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS clients (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'online',
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			metadata JSON,
			INDEX idx_status (status),
			INDEX idx_last_seen (last_seen)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, query := range tables {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	return nil
}

func (s *GoTeleportServerDB) handleAgentConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logEvent("ERROR", "Failed to upgrade agent connection", err.Error())
		return
	}
	defer conn.Close()

	agentID := fmt.Sprintf("%x", time.Now().UnixNano())
	agent := &Agent{
		ID:         agentID,
		Status:     "online",
		LastSeen:   time.Now(),
		Connection: conn,
		Address:    r.RemoteAddr,
		Metadata:   make(map[string]interface{}),
	}

	// Read registration message
	var regMsg Message
	if err := conn.ReadJSON(&regMsg); err != nil {
		s.logEvent("ERROR", "Failed to read agent registration", err.Error())
		return
	}

	if regMsg.Type != "register" {
		s.logEvent("ERROR", "Invalid agent registration message", regMsg.Type)
		return
	}

	if name, ok := regMsg.Metadata["name"].(string); ok {
		agent.Name = name
	}
	if platform, ok := regMsg.Metadata["platform"].(string); ok {
		agent.Platform = platform
	}

	// Register agent
	s.mutex.Lock()
	s.agents[agentID] = agent
	s.mutex.Unlock()

	// Save to database
	if s.config.EnableDB {
		s.saveAgentToDB(agent)
	}

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

	if s.config.EnableDB {
		s.updateAgentStatus(agentID, "offline")
	}
}

func (s *GoTeleportServerDB) handleClientConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logEvent("ERROR", "Failed to upgrade client connection", err.Error())
		return
	}
	defer conn.Close()

	clientID := fmt.Sprintf("%x", time.Now().UnixNano())
	client := &Client{
		ID:         clientID,
		Status:     "online",
		LastSeen:   time.Now(),
		Connection: conn,
		Address:    r.RemoteAddr,
		Metadata:   make(map[string]string),
	}

	// Read registration message
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

	// Save to database
	if s.config.EnableDB {
		s.saveClientToDB(client)
	}

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

	if s.config.EnableDB {
		s.updateClientStatus(clientID, "offline")
	}
}

func (s *GoTeleportServerDB) handleClientMessage(client *Client, msg *Message) {
	switch msg.Type {
	case "list_agents":
		s.sendAgentList(client)
	case "connect_agent":
		s.createSession(client, msg.AgentID)
	case "command":
		// Log command to database before forwarding
		if s.config.EnableDB {
			s.logCommandToDB(msg.SessionID, client.ID, msg.AgentID, msg.Command, "", "sent", 0)
		}
		s.forwardToAgent(msg.SessionID, msg)
	case "disconnect":
		s.closeSession(msg.SessionID)
	default:
		s.logEvent("CLIENT_MSG", "Unknown message type", msg.Type)
	}
}

func (s *GoTeleportServerDB) handleAgentMessage(agent *Agent, msg *Message) {
	switch msg.Type {
	case "command_result":
		// Log command result to database
		if s.config.EnableDB {
			if sessionID := msg.SessionID; sessionID != "" {
				// Extract command and output from metadata if available
				command := ""
				output := msg.Data
				duration := int64(0)
				
				if metadata := msg.Metadata; metadata != nil {
					if cmd, ok := metadata["command"].(string); ok {
						command = cmd
					}
					if dur, ok := metadata["duration"].(float64); ok {
						duration = int64(dur)
					}
				}
				
				s.logCommandToDB(sessionID, "", agent.ID, command, output, "completed", duration)
			}
		}
		// Forward command result to client
		s.forwardToClient(msg.SessionID, msg)
	case "heartbeat":
		// Update last seen
		agent.LastSeen = time.Now()
	default:
		s.logEvent("AGENT_MSG", "Unknown agent message", msg.Type)
	}
}

func (s *GoTeleportServerDB) logCommandToDB(sessionID, clientID, agentID, command, output, status string, duration int64) {
	if s.db == nil {
		return
	}

	query := `INSERT INTO command_logs (session_id, client_id, agent_id, command, output, status, duration_ms) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	
	_, err := s.db.Exec(query, sessionID, clientID, agentID, command, output, status, duration)
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to log command", err.Error())
	}
}

func (s *GoTeleportServerDB) saveAgentToDB(agent *Agent) {
	if s.db == nil {
		return
	}

	metadata, _ := json.Marshal(agent.Metadata)
	query := `INSERT INTO agents (id, name, platform, status, metadata) 
			  VALUES (?, ?, ?, ?, ?) 
			  ON DUPLICATE KEY UPDATE 
			  name=VALUES(name), platform=VALUES(platform), status=VALUES(status), metadata=VALUES(metadata)`
	
	_, err := s.db.Exec(query, agent.ID, agent.Name, agent.Platform, agent.Status, string(metadata))
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to save agent", err.Error())
	}
}

func (s *GoTeleportServerDB) saveClientToDB(client *Client) {
	if s.db == nil {
		return
	}

	metadata, _ := json.Marshal(client.Metadata)
	query := `INSERT INTO clients (id, name, status, metadata) 
			  VALUES (?, ?, ?, ?) 
			  ON DUPLICATE KEY UPDATE 
			  name=VALUES(name), status=VALUES(status), metadata=VALUES(metadata)`
	
	_, err := s.db.Exec(query, client.ID, client.Name, client.Status, string(metadata))
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to save client", err.Error())
	}
}

func (s *GoTeleportServerDB) updateAgentStatus(agentID, status string) {
	if s.db == nil {
		return
	}

	query := `UPDATE agents SET status = ? WHERE id = ?`
	_, err := s.db.Exec(query, status, agentID)
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to update agent status", err.Error())
	}
}

func (s *GoTeleportServerDB) updateClientStatus(clientID, status string) {
	if s.db == nil {
		return
	}

	query := `UPDATE clients SET status = ? WHERE id = ?`
	_, err := s.db.Exec(query, status, clientID)
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to update client status", err.Error())
	}
}

// API Handlers
func (s *GoTeleportServerDB) handleLogsAPI(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get query parameters
	sessionID := r.URL.Query().Get("session_id")
	clientID := r.URL.Query().Get("client_id")
	agentID := r.URL.Query().Get("agent_id")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	// Build query
	query := `SELECT id, session_id, client_id, agent_id, command, output, status, duration_ms, timestamp 
			  FROM command_logs WHERE 1=1`
	args := []interface{}{}

	if sessionID != "" {
		query += " AND session_id = ?"
		args = append(args, sessionID)
	}
	if clientID != "" {
		query += " AND client_id = ?"
		args = append(args, clientID)
	}
	if agentID != "" {
		query += " AND agent_id = ?"
		args = append(args, agentID)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []CommandLog
	for rows.Next() {
		var log CommandLog
		err := rows.Scan(&log.ID, &log.SessionID, &log.ClientID, &log.AgentID, 
						&log.Command, &log.Output, &log.Status, &log.Duration, &log.Timestamp)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (s *GoTeleportServerDB) handleSessionsAPI(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *GoTeleportServerDB) handleStatsAPI(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := map[string]interface{}{
		"agents":   len(s.agents),
		"clients":  len(s.clients),
		"sessions": len(s.sessions),
		"uptime":   time.Since(time.Now()).String(),
	}

	if s.db != nil {
		var totalCommands int
		s.db.QueryRow("SELECT COUNT(*) FROM command_logs").Scan(&totalCommands)
		stats["total_commands"] = totalCommands
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Rest of the methods remain similar but with database logging added...
func (s *GoTeleportServerDB) sendAgentList(client *Client) {
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

func (s *GoTeleportServerDB) createSession(client *Client, agentID string) {
	sessionID := fmt.Sprintf("%x", time.Now().UnixNano())
	
	session := &Session{
		ID:        sessionID,
		AgentID:   agentID,
		ClientID:  client.ID,
		Status:    "active",
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	s.mutex.Lock()
	s.sessions[sessionID] = session
	s.mutex.Unlock()

	// Save session to database
	if s.config.EnableDB {
		query := `INSERT INTO sessions (id, agent_id, client_id, status) VALUES (?, ?, ?, ?)`
		s.db.Exec(query, sessionID, agentID, client.ID, "active")
	}

	response := Message{
		Type:      "session_created",
		SessionID: sessionID,
		AgentID:   agentID,
		ClientID:  client.ID,
		Timestamp: time.Now(),
	}

	client.Connection.WriteJSON(response)
	s.logEvent("SESSION_CREATE", "Session created", fmt.Sprintf("ID: %s, Agent: %s, Client: %s", sessionID, agentID, client.ID))
}

func (s *GoTeleportServerDB) forwardToAgent(sessionID string, msg *Message) {
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

	session.LastUsed = time.Now()
	agent.Connection.WriteJSON(msg)
	
	s.logEvent("FORWARD_AGENT", "Message forwarded to agent", fmt.Sprintf("Session: %s", sessionID))
}

func (s *GoTeleportServerDB) forwardToClient(sessionID string, msg *Message) {
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

	session.LastUsed = time.Now()
	client.Connection.WriteJSON(msg)
	
	s.logEvent("FORWARD_CLIENT", "Message forwarded to client", fmt.Sprintf("Session: %s", sessionID))
}

func (s *GoTeleportServerDB) closeSession(sessionID string) {
	s.mutex.Lock()
	delete(s.sessions, sessionID)
	s.mutex.Unlock()

	if s.config.EnableDB {
		query := `UPDATE sessions SET status = 'closed' WHERE id = ?`
		s.db.Exec(query, sessionID)
	}

	s.logEvent("SESSION_CLOSE", "Session closed", sessionID)
}

func (s *GoTeleportServerDB) logEvent(eventType, description, data string) {
	if s.logger != nil {
		logEntry := fmt.Sprintf("[%s] %s: %s | %s",
			time.Now().Format("2006-01-02 15:04:05"),
			eventType, description, data)
		s.logger.Println(logEntry)
	}
}

func (s *GoTeleportServerDB) Close() {
	if s.db != nil {
		s.db.Close()
	}
}
