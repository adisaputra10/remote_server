package main

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

type GoTeleportServerDB struct {
	config   *ServerConfig
	agents   map[string]*Agent
	clients  map[string]*Client
	sessions map[string]*Session
	mutex    sync.RWMutex
	logger   *log.Logger
	db       *sql.DB
	upgrader websocket.Upgrader
}

type ServerConfig struct {
	Port        int    `json:"port"`
	LogFile     string `json:"log_file"`
	AuthToken   string `json:"auth_token"`
	DatabaseURL string `json:"database_url"`
	EnableDB    bool   `json:"enable_database"`
}

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Role      string    `json:"role"` // "admin" or "user"
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

type UserAgentAssignment struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	AgentID    string    `json:"agent_id"`
	AssignedBy int       `json:"assigned_by"`
	AssignedAt time.Time `json:"assigned_at"`
	Active     bool      `json:"active"`
}

type Agent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Address     string                 `json:"address"`
	Platform    string                 `json:"platform"`
	Status      string                 `json:"status"`
	LastSeen    time.Time              `json:"last_seen"`
	ConnectedAt time.Time              `json:"connected_at"`
	Connection  *websocket.Conn        `json:"-"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type Client struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Username      string            `json:"username"`
	Role          string            `json:"role"`
	Status        string            `json:"status"`
	LastSeen      time.Time         `json:"last_seen"`
	Connection    *websocket.Conn   `json:"-"`
	Address       string            `json:"address"`
	Metadata      map[string]string `json:"metadata"`
	Authenticated bool              `json:"authenticated"`
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

type DatabaseCommand struct {
	ID        int                    `json:"id"`
	SessionID string                 `json:"session_id"`
	AgentID   string                 `json:"agent_id"`
	Command   string                 `json:"command"`
	Protocol  string                 `json:"protocol"`
	ClientIP  string                 `json:"client_ip"`
	ProxyName string                 `json:"proxy_name"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	CreatedAt time.Time              `json:"created_at"`
}

type QueryLog struct {
	ID        int                    `json:"id"`
	AgentID   string                 `json:"agent_id"`
	AgentName string                 `json:"agent_name"`
	Username  string                 `json:"username"`
	SessionID string                 `json:"session_id"`
	EventType string                 `json:"event_type"` // CMD_EXEC, DB_COMMAND, AGENT_START, etc.
	Command   string                 `json:"command"`
	Protocol  string                 `json:"protocol"`
	ClientIP  string                 `json:"client_ip"`
	ProxyName string                 `json:"proxy_name"`
	Details   string                 `json:"details"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	CreatedAt time.Time              `json:"created_at"`
}

type CommandLog struct {
	ID         int       `json:"id"`
	SessionID  string    `json:"session_id"`
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	AgentID    string    `json:"agent_id"`
	AgentName  string    `json:"agent_name"`
	Username   string    `json:"username"`
	Command    string    `json:"command"`
	Output     string    `json:"output"`
	Status     string    `json:"status"`
	Duration   int64     `json:"duration_ms"`
	Timestamp  time.Time `json:"timestamp"`
}

type AccessLog struct {
	ID         int       `json:"id"`
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	Username   string    `json:"username"`
	AgentID    string    `json:"agent_id"`
	AgentName  string    `json:"agent_name"`
	SessionID  string    `json:"session_id"`
	Action     string    `json:"action"` // connect, disconnect, command, login, logout
	Details    string    `json:"details"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Timestamp  time.Time `json:"timestamp"`
}

// CORS middleware
func (s *GoTeleportServerDB) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
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

	// Setup routes with CORS middleware
	http.HandleFunc("/ws/agent", server.corsMiddleware(server.handleAgentConnection))
	http.HandleFunc("/ws/client", server.corsMiddleware(server.handleClientConnection))
	http.HandleFunc("/api/auth/login", server.corsMiddleware(server.handleAPILogin))
	http.HandleFunc("/api/auth/logout", server.corsMiddleware(server.handleAPILogout))
	http.HandleFunc("/api/users", server.corsMiddleware(server.handleUsersAPI))
	http.HandleFunc("/api/agents", server.corsMiddleware(server.handleAgentsAPI))
	http.HandleFunc("/api/user-assignments", server.corsMiddleware(server.handleUserAssignmentsAPI))
	http.HandleFunc("/api/logs", server.corsMiddleware(server.handleLogsAPI))
	http.HandleFunc("/api/sessions", server.corsMiddleware(server.handleSessionsAPI))
	http.HandleFunc("/api/access-logs", server.corsMiddleware(server.handleAccessLogsAPI))
	http.HandleFunc("/api/query-logs", server.corsMiddleware(server.handleQueryLogsAPI))
	http.HandleFunc("/api/database-commands", server.corsMiddleware(server.handleGetDatabaseCommands))
	http.HandleFunc("/api/stats", server.corsMiddleware(server.handleStatsAPI))
	http.HandleFunc("/login", server.corsMiddleware(server.handleLogin))
	http.HandleFunc("/admin", server.handleAdmin)
	http.HandleFunc("/access-logs", server.handleAccessLogsPage)
	http.HandleFunc("/logs", server.handleLogsPage)
	http.HandleFunc("/dashboard", server.handleDashboard)
	http.HandleFunc("/", server.handleIndex)

	// Start processing agent logs
	server.processAgentLogs()

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

	// Setup logging - write to both console and file
	var writers []io.Writer
	writers = append(writers, os.Stdout) // Always log to console

	if config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}
		writers = append(writers, logFile)
	}

	multiWriter := io.MultiWriter(writers...)
	logger := log.New(multiWriter, "", log.LstdFlags)

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

	// Initialize default users
	if err := s.initDefaultUsers(); err != nil {
		return fmt.Errorf("failed to initialize default users: %v", err)
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
			client_name VARCHAR(255) DEFAULT '',
			agent_id VARCHAR(255) NOT NULL,
			agent_name VARCHAR(255) DEFAULT '',
			username VARCHAR(255) DEFAULT '',
			command TEXT NOT NULL,
			output TEXT,
			status VARCHAR(50) DEFAULT 'executed',
			duration_ms BIGINT DEFAULT 0,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_session (session_id),
			INDEX idx_client (client_id),
			INDEX idx_agent (agent_id),
			INDEX idx_username (username),
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
			username VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'online',
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			metadata JSON,
			INDEX idx_status (status),
			INDEX idx_username (username),
			INDEX idx_last_seen (last_seen)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			role ENUM('admin', 'user') DEFAULT 'user',
			active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_username (username),
			INDEX idx_role (role),
			INDEX idx_active (active)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS access_logs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			client_id VARCHAR(255) NOT NULL,
			client_name VARCHAR(255),
			username VARCHAR(255),
			agent_id VARCHAR(255),
			agent_name VARCHAR(255),
			session_id VARCHAR(255),
			action ENUM('connect', 'disconnect', 'command', 'login', 'logout') NOT NULL,
			details TEXT,
			ip_address VARCHAR(45),
			user_agent TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_client_id (client_id),
			INDEX idx_agent_id (agent_id),
			INDEX idx_username (username),
			INDEX idx_action (action),
			INDEX idx_timestamp (timestamp),
			INDEX idx_session_id (session_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS user_agent_assignments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			agent_id VARCHAR(255) NOT NULL,
			assigned_by INT NOT NULL,
			assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			active BOOLEAN DEFAULT TRUE,
			UNIQUE KEY unique_user_agent (user_id, agent_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (assigned_by) REFERENCES users(id),
			INDEX idx_user_id (user_id),
			INDEX idx_agent_id (agent_id),
			INDEX idx_active (active)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		
		`CREATE TABLE IF NOT EXISTS database_commands (
			id INT AUTO_INCREMENT PRIMARY KEY,
			session_id VARCHAR(255) NOT NULL,
			agent_id VARCHAR(255) NOT NULL,
			command TEXT NOT NULL,
			protocol VARCHAR(50) DEFAULT 'mysql',
			client_ip VARCHAR(45),
			proxy_name VARCHAR(255),
			metadata JSON,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_session_id (session_id),
			INDEX idx_agent_id (agent_id),
			INDEX idx_protocol (protocol),
			INDEX idx_proxy_name (proxy_name),
			INDEX idx_timestamp (timestamp)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		
		`CREATE TABLE IF NOT EXISTS query_logs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			agent_id VARCHAR(255) NOT NULL,
			agent_name VARCHAR(255) DEFAULT '',
			username VARCHAR(255) DEFAULT '',
			session_id VARCHAR(255) DEFAULT '',
			event_type VARCHAR(50) NOT NULL,
			command TEXT,
			protocol VARCHAR(50) DEFAULT '',
			client_ip VARCHAR(45) DEFAULT '',
			proxy_name VARCHAR(255) DEFAULT '',
			details TEXT,
			metadata JSON,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_agent_id (agent_id),
			INDEX idx_event_type (event_type),
			INDEX idx_session_id (session_id),
			INDEX idx_timestamp (timestamp)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, query := range tables {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	// Add missing columns to existing tables (for backward compatibility)
	alterQueries := []string{
		`ALTER TABLE command_logs ADD COLUMN client_name VARCHAR(255) DEFAULT ''`,
		`ALTER TABLE command_logs ADD COLUMN agent_name VARCHAR(255) DEFAULT ''`,
		`ALTER TABLE command_logs ADD COLUMN username VARCHAR(255) DEFAULT ''`,
		`ALTER TABLE command_logs ADD INDEX idx_username (username)`,
		`ALTER TABLE clients ADD COLUMN username VARCHAR(255) DEFAULT ''`,
		`ALTER TABLE clients ADD INDEX idx_username (username)`,
		`ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT TRUE`,
		`ALTER TABLE users ADD INDEX idx_active (active)`,
	}

	for _, query := range alterQueries {
		_, err := s.db.Exec(query)
		if err != nil {
			// Ignore errors for ALTER TABLE operations - columns might already exist
			s.logEvent("DB_ALTER", "ALTER TABLE query failed (may be expected if column exists)",
				fmt.Sprintf("Query: %s, Error: %v", query, err))
		} else {
			s.logEvent("DB_ALTER", "ALTER TABLE query succeeded", query)
		}
	}

	return nil
}

func (s *GoTeleportServerDB) processAgentLogs() {
	logFile := "agent-db.log"
	
	s.logEvent("LOG_PROCESSOR", "Starting agent log processor", fmt.Sprintf("File: %s", logFile))
	
	// Start goroutine to periodically read and process agent log
	go func() {
		lastPosition := int64(0)
		ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
		defer ticker.Stop()
		
		for range ticker.C {
			lastPosition = s.readAgentLogFromPosition(logFile, lastPosition)
		}
	}()
}

func (s *GoTeleportServerDB) readAgentLogFromPosition(filename string, lastPos int64) int64 {
	file, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			s.logEvent("LOG_PROCESSOR", "Failed to open agent log file", fmt.Sprintf("File: %s, Error: %v", filename, err))
		}
		return lastPos
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		s.logEvent("LOG_PROCESSOR", "Failed to get file info", fmt.Sprintf("File: %s, Error: %v", filename, err))
		return lastPos
	}

	currentSize := fileInfo.Size()
	s.logEvent("LOG_PROCESSOR", "Processing agent log", fmt.Sprintf("File: %s, Size: %d, LastPos: %d", filename, currentSize, lastPos))

	// If file is smaller than last position, file was rotated
	if currentSize < lastPos {
		lastPos = 0
		s.logEvent("LOG_PROCESSOR", "File rotated, resetting position", fmt.Sprintf("File: %s", filename))
	}

	// Seek to last position
	_, err = file.Seek(lastPos, 0)
	if err != nil {
		s.logEvent("LOG_PROCESSOR", "Failed to seek in file", fmt.Sprintf("File: %s, Position: %d, Error: %v", filename, lastPos, err))
		return lastPos
	}

	// Read new lines
	scanner := bufio.NewScanner(file)
	newPos := lastPos
	lineCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		newPos += int64(len(line)) + 1 // +1 for newline
		lineCount++
		
		s.logEvent("LOG_PROCESSOR", "Processing log line", fmt.Sprintf("Line %d: %s", lineCount, line))
		
		// Parse and store log entry
		s.parseAndStoreLogEntry(line)
	}

	if lineCount > 0 {
		s.logEvent("LOG_PROCESSOR", "Log processing completed", fmt.Sprintf("File: %s, Lines: %d, NewPos: %d", filename, lineCount, newPos))
	}

	return newPos
}

func (s *GoTeleportServerDB) parseAndStoreLogEntry(logLine string) {
	// Skip empty lines
	if strings.TrimSpace(logLine) == "" {
		return
	}
	
	s.logEvent("LOG_PARSER", "Parsing log entry", fmt.Sprintf("Line: %s", logLine))
	
	// Only process DB_COMMAND logs, skip everything else
	if strings.Contains(logLine, "[DB_COMMAND]") {
		s.logEvent("LOG_PARSER", "Parsing DB_COMMAND log", "")
		s.parseDbCommandLog(logLine)
	} else {
		s.logEvent("LOG_PARSER", "Skipping non-DB_COMMAND log", fmt.Sprintf("Line: %s", logLine))
	}
}

func (s *GoTeleportServerDB) parseCmdExecLog(logLine string) {
	// Example: 2025/09/04 19:14:08 [2025-09-04 19:14:08] [CMD_EXEC] User: ThinkPad | Agent: database-agent-1 | Event: Command execution | Details: Session: 69199774f538775d, Command: ls
	
	parts := strings.Split(logLine, "] [CMD_EXEC] ")
	if len(parts) != 2 {
		return
	}
	
	// Parse timestamp
	timestampStr := strings.TrimPrefix(parts[0], "2025/09/04 19:14:08 [")
	timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
	if err != nil {
		timestamp = time.Now()
	}
	
	// Parse details
	details := parts[1]
	username := s.extractValue(details, "User: ", " |")
	agentName := s.extractValue(details, "Agent: ", " |")
	sessionID := s.extractValue(details, "Session: ", ",")
	command := s.extractValue(details, "Command: ", "")
	
	// Store to database
	s.storeQueryLog("", agentName, username, sessionID, "CMD_EXEC", command, "", "", "", details, timestamp)
}

func (s *GoTeleportServerDB) parseDbCommandLog(logLine string) {
	// Example: 2025/09/04 19:18:20 [2025-09-04 19:18:20] [DB_COMMAND] Agent: database-agent-1 | Proxy: mysql-main | Session: db_mysql-main_1756988300 | Client: [::1]:65270 | Username: john | Protocol: mysql | Command: SELECT VERSION()
	
	parts := strings.Split(logLine, "] [DB_COMMAND] ")
	if len(parts) != 2 {
		s.logEvent("LOG_PARSER", "Failed to parse DB_COMMAND format", fmt.Sprintf("Line: %s", logLine))
		return
	}
	
	// Parse timestamp
	timestampStr := strings.TrimPrefix(parts[0], "2025/09/04 19:18:20 [")
	timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
	if err != nil {
		s.logEvent("LOG_PARSER", "Failed to parse timestamp", fmt.Sprintf("Timestamp: %s", timestampStr))
		timestamp = time.Now()
	}
	
	// Parse details
	details := parts[1]
	agentName := s.extractValue(details, "Agent: ", " |")
	proxyName := s.extractValue(details, "Proxy: ", " |")
	sessionID := s.extractValue(details, "Session: ", " |")
	clientIP := s.extractValue(details, "Client: ", " |")
	logUsername := s.extractValue(details, "Username: ", " |") // Username from agent log
	protocol := s.extractValue(details, "Protocol: ", " |")
	command := s.extractValue(details, "Command: ", "")
	
	// If agent name not found, fallback to proxy name
	if agentName == "" {
		agentName = proxyName
		if agentName == "" {
			agentName = "mysql-main" // Default agent name
		}
	}
	
	// Always try to get username from active client first, fallback to log username
	username := ""
	agentID := ""
	activeClient := s.getActiveClient()
	if activeClient != nil {
		username = activeClient.Username // Use client config username (e.g., "demi")
		agentID = activeClient.Name      // Use client name as agent ID (e.g., "DemoClient-01")
	}
	
	// If no active client, fallback to log username or default
	if username == "" {
		if logUsername != "" {
			username = logUsername
		} else {
			username = "mysql-user"
		}
	}
	
	s.logEvent("LOG_PARSER", "Parsed DB_COMMAND", fmt.Sprintf("Agent: %s, User: %s, Session: %s, Command: %s", agentName, username, sessionID, command))
	
	// Store to database
	s.storeQueryLog(agentID, agentName, username, sessionID, "DB_COMMAND", command, protocol, clientIP, proxyName, details, timestamp)
}

func (s *GoTeleportServerDB) parseAgentEventLog(logLine string) {
	// Example: 2025/09/04 19:10:17 [2025-09-04 19:10:17] [AGENT_START] User: ThinkPad | Agent: database-agent-1 | Event: GoTeleport Agent starting | Details: database-agent-1
	
	var eventType string
	if strings.Contains(logLine, "[AGENT_START]") {
		eventType = "AGENT_START"
	} else if strings.Contains(logLine, "[AGENT_CONNECT]") {
		eventType = "AGENT_CONNECT"
	} else if strings.Contains(logLine, "[AGENT_DISCONNECT]") {
		eventType = "AGENT_DISCONNECT"
	}
	
	parts := strings.Split(logLine, fmt.Sprintf("] [%s] ", eventType))
	if len(parts) != 2 {
		return
	}
	
	// Parse timestamp
	timestampStr := strings.TrimPrefix(parts[0], "2025/09/04 19:10:17 [")
	timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
	if err != nil {
		timestamp = time.Now()
	}
	
	// Parse details
	details := parts[1]
	username := s.extractValue(details, "User: ", " |")
	agentName := s.extractValue(details, "Agent: ", " |")
	event := s.extractValue(details, "Event: ", " |")
	
	// Store to database
	s.storeQueryLog("", agentName, username, "", eventType, event, "", "", "", details, timestamp)
}

func (s *GoTeleportServerDB) extractValue(text, prefix, suffix string) string {
	start := strings.Index(text, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	
	if suffix == "" {
		return strings.TrimSpace(text[start:])
	}
	
	end := strings.Index(text[start:], suffix)
	if end == -1 {
		return strings.TrimSpace(text[start:])
	}
	
	return strings.TrimSpace(text[start : start+end])
}

func (s *GoTeleportServerDB) storeQueryLog(agentID, agentName, username, sessionID, eventType, command, protocol, clientIP, proxyName, details string, timestamp time.Time) {
	if !s.config.EnableDB {
		return
	}
	
	// Sanitize command to remove binary data and invalid UTF-8
	command = s.sanitizeString(command)
	details = s.sanitizeString(details)
	
	_, err := s.db.Exec(`
		INSERT INTO query_logs 
		(agent_id, agent_name, username, session_id, event_type, command, protocol, client_ip, proxy_name, details, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
		agentID, agentName, username, sessionID, eventType, command, protocol, clientIP, proxyName, details, timestamp)
	
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to store query log", fmt.Sprintf("Error: %v", err))
	}
}

func (s *GoTeleportServerDB) sanitizeString(str string) string {
	// Remove non-printable characters and ensure valid UTF-8
	var result []rune
	for _, r := range str {
		if r >= 32 && r <= 126 || r >= 160 { // Keep printable ASCII and extended chars
			result = append(result, r)
		} else if r == '\n' || r == '\r' || r == '\t' { // Keep basic whitespace
			result = append(result, r)
		}
	}
	return string(result)
}

func (s *GoTeleportServerDB) handleAgentConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logEvent("ERROR", "Failed to upgrade agent connection", err.Error())
		return
	}
	defer conn.Close()

	agentID := fmt.Sprintf("%x", time.Now().UnixNano())
	now := time.Now()
	agent := &Agent{
		ID:          agentID,
		Status:      "online",
		LastSeen:    now,
		ConnectedAt: now,
		Connection:  conn,
		Address:     r.RemoteAddr,
		Metadata:    make(map[string]interface{}),
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

	// Validate auth token
	if regMsg.Data != s.config.AuthToken {
		s.logEvent("AUTH_FAILED", "Invalid auth token", fmt.Sprintf("Client: %s, Token: %s", clientID, regMsg.Data))
		conn.WriteJSON(Message{
			Type:      "auth_failed",
			Data:      "Invalid authentication token",
			Timestamp: time.Now(),
		})
		return
	}

	// Extract client information from metadata
	var username, password string
	if name, ok := regMsg.Metadata["name"].(string); ok {
		client.Name = name
	}

	if user, ok := regMsg.Metadata["username"].(string); ok {
		username = user
		client.Username = user
	}

	if pass, ok := regMsg.Metadata["password"].(string); ok {
		password = pass
	}

	// Validate required fields
	if username == "" {
		s.logEvent("AUTH_FAILED", "Missing username", fmt.Sprintf("Client: %s", clientID))
		conn.WriteJSON(Message{
			Type:      "auth_failed",
			Data:      "Username is required",
			Timestamp: time.Now(),
		})
		return
	}

	if password == "" {
		s.logEvent("AUTH_FAILED", "Missing password", fmt.Sprintf("Client: %s, User: %s", clientID, username))
		conn.WriteJSON(Message{
			Type:      "auth_failed",
			Data:      "Password is required",
			Timestamp: time.Now(),
		})
		return
	}

	// Validate user credentials
	user, authenticated := s.authenticateUser(username, password)
	if !authenticated {
		s.logEvent("AUTH_FAILED", "Invalid credentials", fmt.Sprintf("Client: %s, User: %s", clientID, username))
		conn.WriteJSON(Message{
			Type:      "auth_failed",
			Data:      "Invalid username or password",
			Timestamp: time.Now(),
		})
		return
	}

	if !user.Active {
		s.logEvent("AUTH_FAILED", "User account disabled", fmt.Sprintf("Client: %s, User: %s", clientID, username))
		conn.WriteJSON(Message{
			Type:      "auth_failed",
			Data:      "Account is disabled",
			Timestamp: time.Now(),
		})
		return
	}

	// Set user role
	client.Role = user.Role

	// Register client
	s.mutex.Lock()
	s.clients[clientID] = client
	s.mutex.Unlock()

	// Save to database
	if s.config.EnableDB {
		s.saveClientToDB(client)
	}

	s.logEvent("CLIENT_CONNECT", "Client registered", fmt.Sprintf("ID: %s, Name: %s, User: %s, Role: %s", clientID, client.Name, client.Username, client.Role))

	// Log access
	s.logAccess(clientID, client.Name, client.Username, "", "", "", "login",
		fmt.Sprintf("Client connected with role %s", client.Role),
		r.RemoteAddr, r.UserAgent())

	// Send registration response
	response := Message{
		Type:      "registered",
		ClientID:  clientID,
		Timestamp: time.Now(),
	}
	conn.WriteJSON(response)

	// Broadcast client info to all agents
	s.broadcastClientInfoToAgents(client)

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
	case "login":
		s.handleClientAuth(client, msg)
	case "get_agents":
		s.sendAgentList(client)
	case "list_agents":
		s.sendAgentList(client)
	case "connect_agent":
		s.createSession(client, msg.AgentID)
	case "command":
		// Log command to database before forwarding
		if s.config.EnableDB {
			s.logCommandToDB(msg.SessionID, client.ID, client.Name, msg.AgentID, "", client.Username, msg.Command, "", "sent", 0)
		}
		s.forwardToAgent(msg.SessionID, msg)
	case "port_forward_request":
		s.handlePortForwardRequest(client, msg)
	case "disconnect":
		s.closeSession(msg.SessionID)
	default:
		s.logEvent("CLIENT_MSG", "Unknown message type", msg.Type)
	}
}

func (s *GoTeleportServerDB) handleClientAuth(client *Client, msg *Message) {
	if metadata := msg.Metadata; metadata != nil {
		username, _ := metadata["username"].(string)
		password, _ := metadata["password"].(string)
		
		// Validate credentials (simple auth for demo)
		if username == "admin" && password == "admin123" {
			client.Username = username
			client.Authenticated = true
			
			response := Message{
				Type:      "authenticated",
				ClientID:  client.ID,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"username": username,
					"success":  true,
				},
			}
			client.Connection.WriteJSON(response)
			s.logger.Printf("Client %s authenticated as %s", client.ID, username)
		} else {
			response := Message{
				Type:      "auth_error", 
				ClientID:  client.ID,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"error": "Invalid credentials",
				},
			}
			client.Connection.WriteJSON(response)
		}
	}
}

func (s *GoTeleportServerDB) handlePortForwardRequest(client *Client, msg *Message) {
	if !client.Authenticated {
		response := Message{
			Type:      "port_forward_error",
			ClientID:  client.ID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error": "Not authenticated",
			},
		}
		client.Connection.WriteJSON(response)
		return
	}

	// Extract port forward parameters
	if metadata := msg.Metadata; metadata != nil {
		agentID, _ := metadata["agent_id"].(string)
		localPort, _ := metadata["local_port"].(float64)
		targetHost, _ := metadata["target_host"].(string)
		targetPort, _ := metadata["target_port"].(float64)

		s.logger.Printf("Port forward request: client=%s, agent=%s, local=%d, target=%s:%d",
			client.ID, agentID, int(localPort), targetHost, int(targetPort))

		// Check if agent exists
		s.mutex.RLock()
		agent, exists := s.agents[agentID]
		s.mutex.RUnlock()

		if !exists {
			response := Message{
				Type:      "port_forward_error",
				ClientID:  client.ID,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"error": "Agent not found",
				},
			}
			client.Connection.WriteJSON(response)
			return
		}

		// Forward request to agent
		forwardMsg := Message{
			Type:      "start_port_forward",
			ClientID:  client.ID,
			AgentID:   agentID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"local_port":  localPort,
				"target_host": targetHost,
				"target_port": targetPort,
			},
		}

		if err := agent.Connection.WriteJSON(forwardMsg); err != nil {
			s.logger.Printf("Failed to forward port request to agent %s: %v", agentID, err)
			response := Message{
				Type:      "port_forward_error",
				ClientID:  client.ID,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"error": "Failed to contact agent",
				},
			}
			client.Connection.WriteJSON(response)
		} else {
			// Send success response
			response := Message{
				Type:      "port_forward_started",
				ClientID:  client.ID,
				AgentID:   agentID,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"local_port":  localPort,
					"target_host": targetHost,
					"target_port": targetPort,
				},
			}
			client.Connection.WriteJSON(response)
		}
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

				// Get client info from session
				s.mutex.RLock()
				session := s.sessions[sessionID]
				var clientID, clientName, username string
				if session != nil {
					clientID = session.ClientID
					if client := s.clients[clientID]; client != nil {
						clientName = client.Name
						username = client.Username
					}
				}
				s.mutex.RUnlock()

				s.logCommandToDB(sessionID, clientID, clientName, agent.ID, agent.Name, username, command, output, "completed", duration)
			}
		}
		// Forward command result to client
		s.forwardToClient(msg.SessionID, msg)
	case "database_command":
		// Log database command
		if s.config.EnableDB {
			s.logDatabaseCommandToDB(agent.ID, msg)
		}
		s.logEvent("DB_COMMAND", "Database command executed", fmt.Sprintf("Agent: %s, Command: %s", agent.Name, msg.Command))
	case "heartbeat":
		// Update last seen
		agent.LastSeen = time.Now()
	default:
		s.logEvent("AGENT_MSG", "Unknown agent message", msg.Type)
	}
}

func (s *GoTeleportServerDB) logCommandToDB(sessionID, clientID, clientName, agentID, agentName, username, command, output, status string, duration int64) {
	if s.db == nil {
		return
	}

	query := `INSERT INTO command_logs (session_id, client_id, client_name, agent_id, agent_name, username, command, output, status, duration_ms) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, sessionID, clientID, clientName, agentID, agentName, username, command, output, status, duration)
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to log command", err.Error())
		return
	}

	// Also log access for command execution
	details := fmt.Sprintf("Command executed: %s [%s]", command, status)
	s.logAccess(clientID, clientName, username, agentID, agentName, sessionID, "command",
		details, "", "")
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
	query := `INSERT INTO clients (id, name, username, status, metadata) 
			  VALUES (?, ?, ?, ?, ?) 
			  ON DUPLICATE KEY UPDATE 
			  name=VALUES(name), username=VALUES(username), status=VALUES(status), metadata=VALUES(metadata)`

	_, err := s.db.Exec(query, client.ID, client.Name, client.Username, client.Status, string(metadata))
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
	query := `SELECT id, session_id, client_id, client_name, agent_id, agent_name, username, command, output, status, duration_ms, timestamp 
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
		err := rows.Scan(&log.ID, &log.SessionID, &log.ClientID, &log.ClientName, &log.AgentID, &log.AgentName, &log.Username,
			&log.Command, &log.Output, &log.Status, &log.Duration, &log.Timestamp)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")

	// Format response as expected by frontend
	response := map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	}

	json.NewEncoder(w).Encode(response)
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
	activeConnections := len(s.clients)
	activeAgents := len(s.agents)
	s.mutex.RUnlock()

	stats := map[string]interface{}{
		"activeConnections": activeConnections,
		"activeAgents":      activeAgents,
		"activeUsers":       s.getActiveUsersCount(),
		"totalCommands":     0,
	}

	if s.db != nil {
		var totalCommands int
		s.db.QueryRow("SELECT COUNT(*) FROM command_logs").Scan(&totalCommands)
		stats["totalCommands"] = totalCommands
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Helper method to get active users count
func (s *GoTeleportServerDB) getActiveUsersCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	activeUsers := make(map[string]bool)
	for _, client := range s.clients {
		if client.Username != "" {
			activeUsers[client.Username] = true
		}
	}
	return len(activeUsers)
}

// API Authentication handlers
func (s *GoTeleportServerDB) handleAPILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, authenticated := s.authenticateUser(loginReq.Username, loginReq.Password)
	if !authenticated {
		s.logEvent("API_LOGIN_FAILED", "API login failed", fmt.Sprintf("Username: %s, IP: %s", loginReq.Username, r.RemoteAddr))
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate simple token (in production, use JWT or proper session)
	token := s.generateSessionToken(user.Username)

	// Log successful login
	s.logEvent("API_LOGIN_SUCCESS", "API login successful", fmt.Sprintf("Username: %s, Role: %s, IP: %s", user.Username, user.Role, r.RemoteAddr))

	// Log access
	s.logAccess("", "Web API", user.Username, "", "", "", "login", "API login successful", r.RemoteAddr, r.UserAgent())

	response := map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
		"message": "Login successful",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleAPILogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get username from token (simplified - in production, validate JWT)
	token := r.Header.Get("Authorization")
	username := s.getUsernameFromToken(token)

	s.logEvent("API_LOGOUT", "API logout", fmt.Sprintf("Username: %s, IP: %s", username, r.RemoteAddr))

	// Log access
	s.logAccess("", "Web API", username, "", "", "", "logout", "API logout", r.RemoteAddr, r.UserAgent())

	response := map[string]interface{}{
		"message": "Logout successful",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper methods for token management
func (s *GoTeleportServerDB) generateSessionToken(username string) string {
	// Simple token generation with username:timestamp encoded in base64
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s:%d", username, timestamp)

	// Add a simple signature to prevent tampering
	hash := sha256.Sum256([]byte(data + "SECRET_KEY"))
	signature := hex.EncodeToString(hash[:8]) // Use first 8 bytes as signature

	// Combine data and signature
	tokenData := fmt.Sprintf("%s:%s", data, signature)

	// Encode in base64
	return hex.EncodeToString([]byte(tokenData))
}

func (s *GoTeleportServerDB) getUsernameFromToken(authHeader string) string {
	// Simplified token parsing
	if authHeader == "" {
		return "unknown"
	}

	// Remove "Bearer " prefix if present
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "unknown"
	}

	// Decode from hex
	tokenBytes, err := hex.DecodeString(token)
	if err != nil {
		return "unknown"
	}

	// Parse token data
	tokenData := string(tokenBytes)
	parts := strings.Split(tokenData, ":")
	if len(parts) < 3 {
		return "unknown"
	}

	username := parts[0]
	timestampStr := parts[1]
	signature := parts[2]

	// Verify signature
	data := fmt.Sprintf("%s:%s", username, timestampStr)
	hash := sha256.Sum256([]byte(data + "SECRET_KEY"))
	expectedSignature := hex.EncodeToString(hash[:8])

	if signature != expectedSignature {
		return "unknown"
	}

	// Check if token is not too old (optional: 24 hours)
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return "unknown"
	}

	// Token expires after 24 hours
	if time.Now().Unix()-timestamp > 86400 {
		return "unknown"
	}

	return username
}

func (s *GoTeleportServerDB) getUserFromToken(authHeader string) (*User, error) {
	if authHeader == "" {
		return nil, fmt.Errorf("no authorization header")
	}

	// Remove "Bearer " prefix if present
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, fmt.Errorf("invalid token format")
	}

	// Simple token validation - in production use proper JWT validation
	// For now, extract username from the token (this is a simplified approach)
	username := s.getUsernameFromToken(authHeader)
	if username == "unknown" || username == "api_user" {
		return nil, fmt.Errorf("invalid token")
	}

	// Get user from database
	var user User
	query := `SELECT id, username, role, active, created_at FROM users WHERE username = ? AND active = 1`
	var createdAt string
	err := s.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Role, &user.Active, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
		user.CreatedAt = parsedTime
	}

	return &user, nil
}

// Users API handler - handles CRUD operations for users
func (s *GoTeleportServerDB) handleUsersAPI(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not enabled", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetUsers(w, r)
	case "POST":
		s.handleCreateUser(w, r)
	case "PUT":
		s.handleUpdateUser(w, r)
	case "DELETE":
		s.handleDeleteUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *GoTeleportServerDB) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, username, role, status, email, full_name, created_at, updated_at 
			  FROM users ORDER BY id`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var username, role, status, email, fullName string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &username, &role, &status, &email, &fullName, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		user := map[string]interface{}{
			"id":         id,
			"username":   username,
			"role":       role,
			"status":     status,
			"email":      email,
			"full_name":  fullName,
			"created_at": createdAt,
			"updated_at": updatedAt,
		}
		users = append(users, user)
	}

	response := map[string]interface{}{
		"users": users,
		"total": len(users),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Email == "" {
		req.Email = req.Username + "@goteleport.local"
	}
	if req.FullName == "" {
		req.FullName = req.Username
	}

	// Hash password
	hashedPassword := hashPassword(req.Password)

	// Check if username already exists
	var existingCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&existingCount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	if existingCount > 0 {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Insert new user
	query := `INSERT INTO users (username, password_hash, role, status, email, full_name, created_at, updated_at) 
			  VALUES (?, ?, ?, 'active', ?, ?, NOW(), NOW())`

	result, err := s.db.Exec(query, req.Username, hashedPassword, req.Role, req.Email, req.FullName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	userID, _ := result.LastInsertId()

	// Log the action
	s.logEvent("USER_CREATE", "User created via API", fmt.Sprintf("Username: %s, Role: %s", req.Username, req.Role))

	response := map[string]interface{}{
		"id":      userID,
		"message": "User created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Status   string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}

	if req.Username != "" {
		setParts = append(setParts, "username = ?")
		args = append(args, req.Username)
	}
	if req.Password != "" {
		setParts = append(setParts, "password_hash = ?")
		args = append(args, hashPassword(req.Password))
	}
	if req.Role != "" {
		setParts = append(setParts, "role = ?")
		args = append(args, req.Role)
	}
	if req.Email != "" {
		setParts = append(setParts, "email = ?")
		args = append(args, req.Email)
	}
	if req.FullName != "" {
		setParts = append(setParts, "full_name = ?")
		args = append(args, req.FullName)
	}
	if req.Status != "" {
		setParts = append(setParts, "status = ?")
		args = append(args, req.Status)
	}

	if len(setParts) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, userID)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(setParts, ", "))

	result, err := s.db.Exec(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Log the action
	s.logEvent("USER_UPDATE", "User updated via API", fmt.Sprintf("UserID: %s", userID))

	response := map[string]interface{}{
		"message": "User updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Don't allow deleting the last admin user
	var adminCount int
	s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'").Scan(&adminCount)

	var userRole string
	err := s.db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if userRole == "admin" && adminCount <= 1 {
		http.Error(w, "Cannot delete the last admin user", http.StatusBadRequest)
		return
	}

	// Soft delete - set status to 'deleted'
	query := `UPDATE users SET status = 'deleted', updated_at = NOW() WHERE id = ?`

	result, err := s.db.Exec(query, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete user: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Log the action
	s.logEvent("USER_DELETE", "User deleted via API", fmt.Sprintf("UserID: %s", userID))

	response := map[string]interface{}{
		"message": "User deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleAgentsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleGetAgents(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *GoTeleportServerDB) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// For now, allow access to agents for all authenticated users
	// TODO: Implement proper user-agent assignment filtering
	var currentUser *User = &User{Role: "admin"} // Temporary: treat all as admin

	// Get agents based on user role and assignments
	var agents []map[string]interface{}
	var allowedAgentIDs map[string]bool

	// If user is not admin, get their assigned agents
	if currentUser.Role != "admin" && s.db != nil {
		allowedAgentIDs = make(map[string]bool)
		query := `SELECT agent_id FROM user_agent_assignments WHERE user_id = ? AND active = 1`
		rows, err := s.db.Query(query, currentUser.ID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var agentID string
				if err := rows.Scan(&agentID); err == nil {
					allowedAgentIDs[agentID] = true
				}
			}
		}
	}

	// Get online agents from memory
	for id, agent := range s.agents {
		// Check if user has access to this agent
		if currentUser.Role != "admin" && !allowedAgentIDs[id] {
			continue
		}

		agentInfo := map[string]interface{}{
			"id":           id,
			"name":         agent.Name,
			"status":       "online",
			"last_seen":    agent.LastSeen,
			"connected_at": agent.ConnectedAt,
		}

		// Get additional info from database if available
		if s.db != nil {
			var dbName, dbStatus, metadata string
			var lastSeen time.Time
			query := `SELECT name, status, last_seen, metadata FROM agents WHERE id = ?`
			err := s.db.QueryRow(query, id).Scan(&dbName, &dbStatus, &lastSeen, &metadata)
			if err == nil {
				agentInfo["db_name"] = dbName
				agentInfo["db_status"] = dbStatus
				agentInfo["db_last_seen"] = lastSeen
				agentInfo["metadata"] = metadata
			}
		}

		agents = append(agents, agentInfo)
	}

	// Also get offline agents from database
	if s.db != nil {
		query := `SELECT id, name, status, last_seen, metadata FROM agents WHERE status = 'offline' OR id NOT IN (SELECT id FROM agents WHERE status = 'online')`
		rows, err := s.db.Query(query)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, name, status, metadata string
				var lastSeen time.Time
				err := rows.Scan(&id, &name, &status, &lastSeen, &metadata)
				if err != nil {
					continue
				}

				// Check if user has access to this agent
				if currentUser.Role != "admin" && !allowedAgentIDs[id] {
					continue
				}

				// Check if agent is not already in online list
				found := false
				for _, agent := range agents {
					if agent["id"] == id {
						found = true
						break
					}
				}

				if !found {
					agentInfo := map[string]interface{}{
						"id":        id,
						"name":      name,
						"status":    status,
						"last_seen": lastSeen,
						"metadata":  metadata,
					}
					agents = append(agents, agentInfo)
				}
			}
		}
	}

	response := map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// User-Agent Assignments API
func (s *GoTeleportServerDB) handleUserAssignmentsAPI(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not enabled", http.StatusServiceUnavailable)
		return
	}

	// For now, allow access for all users with proper auth header
	// TODO: Implement proper admin-only access
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Temporary: assume user is admin if they have any auth header
	// In real implementation, properly validate token and check role

	switch r.Method {
	case "GET":
		s.handleGetUserAssignments(w, r)
	case "POST":
		s.handleCreateUserAssignment(w, r)
	case "DELETE":
		s.handleDeleteUserAssignment(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *GoTeleportServerDB) handleGetUserAssignments(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	agentID := r.URL.Query().Get("agent_id")

	var query string
	var args []interface{}

	if userID != "" && agentID != "" {
		query = `SELECT ua.id, ua.user_id, ua.agent_id, ua.assigned_by, ua.assigned_at, ua.active,
				        u.username, admin.username as assigned_by_username
				 FROM user_agent_assignments ua
				 JOIN users u ON ua.user_id = u.id
				 JOIN users admin ON ua.assigned_by = admin.id
				 WHERE ua.user_id = ? AND ua.agent_id = ? AND ua.active = 1`
		args = append(args, userID, agentID)
	} else if userID != "" {
		query = `SELECT ua.id, ua.user_id, ua.agent_id, ua.assigned_by, ua.assigned_at, ua.active,
				        u.username, admin.username as assigned_by_username
				 FROM user_agent_assignments ua
				 JOIN users u ON ua.user_id = u.id
				 JOIN users admin ON ua.assigned_by = admin.id
				 WHERE ua.user_id = ? AND ua.active = 1`
		args = append(args, userID)
	} else if agentID != "" {
		query = `SELECT ua.id, ua.user_id, ua.agent_id, ua.assigned_by, ua.assigned_at, ua.active,
				        u.username, admin.username as assigned_by_username
				 FROM user_agent_assignments ua
				 JOIN users u ON ua.user_id = u.id
				 JOIN users admin ON ua.assigned_by = admin.id
				 WHERE ua.agent_id = ? AND ua.active = 1`
		args = append(args, agentID)
	} else {
		query = `SELECT ua.id, ua.user_id, ua.agent_id, ua.assigned_by, ua.assigned_at, ua.active,
				        u.username, admin.username as assigned_by_username
				 FROM user_agent_assignments ua
				 JOIN users u ON ua.user_id = u.id
				 JOIN users admin ON ua.assigned_by = admin.id
				 WHERE ua.active = 1`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var assignments []map[string]interface{}
	for rows.Next() {
		var assignment UserAgentAssignment
		var username, assignedByUsername string
		var assignedAtStr string

		err := rows.Scan(&assignment.ID, &assignment.UserID, &assignment.AgentID,
			&assignment.AssignedBy, &assignedAtStr, &assignment.Active,
			&username, &assignedByUsername)
		if err != nil {
			continue
		}

		if parsedTime, err := time.Parse("2006-01-02 15:04:05", assignedAtStr); err == nil {
			assignment.AssignedAt = parsedTime
		}

		assignmentInfo := map[string]interface{}{
			"id":                   assignment.ID,
			"user_id":              assignment.UserID,
			"agent_id":             assignment.AgentID,
			"assigned_by":          assignment.AssignedBy,
			"assigned_at":          assignment.AssignedAt,
			"active":               assignment.Active,
			"username":             username,
			"assigned_by_username": assignedByUsername,
		}
		assignments = append(assignments, assignmentInfo)
	}

	response := map[string]interface{}{
		"assignments": assignments,
		"total":       len(assignments),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleCreateUserAssignment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  int    `json:"user_id"`
		AgentID string `json:"agent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logEvent("ASSIGNMENT_ERROR", "Invalid JSON in create assignment", fmt.Sprintf("Error: %v", err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the request for debugging
	s.logEvent("ASSIGNMENT_REQUEST", "Create assignment request",
		fmt.Sprintf("UserID: %d, AgentID: %s", req.UserID, req.AgentID))

	// Get admin user from token
	authHeader := r.Header.Get("Authorization")
	currentUser, err := s.getUserFromToken(authHeader)
	if err != nil {
		s.logEvent("ASSIGNMENT_ERROR", "Token validation failed", fmt.Sprintf("Error: %v", err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if assignment already exists
	var existingID int
	query := `SELECT id FROM user_agent_assignments WHERE user_id = ? AND agent_id = ? AND active = 1`
	err = s.db.QueryRow(query, req.UserID, req.AgentID).Scan(&existingID)
	if err == nil {
		s.logEvent("ASSIGNMENT_ERROR", "Assignment already exists",
			fmt.Sprintf("UserID: %d, AgentID: %s", req.UserID, req.AgentID))
		http.Error(w, "Assignment already exists", http.StatusConflict)
		return
	}

	// Create new assignment
	query = `INSERT INTO user_agent_assignments (user_id, agent_id, assigned_by, assigned_at, active) VALUES (?, ?, ?, NOW(), 1)`
	result, err := s.db.Exec(query, req.UserID, req.AgentID, currentUser.ID)
	if err != nil {
		s.logEvent("ASSIGNMENT_ERROR", "Database error in create assignment",
			fmt.Sprintf("UserID: %d, AgentID: %s, Error: %v", req.UserID, req.AgentID, err))
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	assignmentID, _ := result.LastInsertId()

	// Log the assignment
	s.logEvent("USER_AGENT_ASSIGNED", "User assigned to agent",
		fmt.Sprintf("UserID: %d, AgentID: %s, AssignedBy: %s", req.UserID, req.AgentID, currentUser.Username))

	response := map[string]interface{}{
		"id":          assignmentID,
		"user_id":     req.UserID,
		"agent_id":    req.AgentID,
		"assigned_by": currentUser.ID,
		"message":     "Assignment created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleDeleteUserAssignment(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	agentID := r.URL.Query().Get("agent_id")

	if userID == "" || agentID == "" {
		http.Error(w, "user_id and agent_id are required", http.StatusBadRequest)
		return
	}

	// Get admin user from token
	authHeader := r.Header.Get("Authorization")
	currentUser, err := s.getUserFromToken(authHeader)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Soft delete assignment by setting active = 0
	query := `UPDATE user_agent_assignments SET active = 0 WHERE user_id = ? AND agent_id = ? AND active = 1`
	result, err := s.db.Exec(query, userID, agentID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	// Log the unassignment
	s.logEvent("USER_AGENT_UNASSIGNED", "User unassigned from agent",
		fmt.Sprintf("UserID: %s, AgentID: %s, UnassignedBy: %s", userID, agentID, currentUser.Username))

	response := map[string]interface{}{
		"message": "Assignment removed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *GoTeleportServerDB) handleAccessLogsAPI(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	clientID := r.URL.Query().Get("client_id")
	agentID := r.URL.Query().Get("agent_id")
	username := r.URL.Query().Get("username")
	limitStr := r.URL.Query().Get("limit")

	limit := 100 // default limit
	if limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || parsedLimit != 1 {
			limit = 100
		}
	}

	logs, err := s.getAccessLogs(limit, clientID, agentID, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get access logs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	})
}

func (s *GoTeleportServerDB) handleQueryLogsAPI(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	agentID := r.URL.Query().Get("agent_id")
	eventType := r.URL.Query().Get("event_type")
	username := r.URL.Query().Get("username")
	limitStr := r.URL.Query().Get("limit")

	limit := 100 // default limit
	if limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || parsedLimit != 1 {
			limit = 100
		}
	}

	logs, err := s.getQueryLogs(limit, agentID, eventType, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get query logs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	})
}

func (s *GoTeleportServerDB) getQueryLogs(limit int, agentID, eventType, username string) ([]QueryLog, error) {
	if !s.config.EnableDB {
		return []QueryLog{}, nil
	}

	query := `SELECT id, agent_id, agent_name, username, session_id, event_type, command, protocol, client_ip, proxy_name, details, timestamp, created_at 
			 FROM query_logs WHERE 1=1`
	args := []interface{}{}

	if agentID != "" {
		query += " AND agent_id = ?"
		args = append(args, agentID)
	}

	if eventType != "" {
		query += " AND event_type = ?"
		args = append(args, eventType)
	}

	if username != "" {
		query += " AND username = ?"
		args = append(args, username)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []QueryLog
	for rows.Next() {
		var log QueryLog
		var details string
		err := rows.Scan(&log.ID, &log.AgentID, &log.AgentName, &log.Username, &log.SessionID, 
			&log.EventType, &log.Command, &log.Protocol, &log.ClientIP, &log.ProxyName, 
			&details, &log.Timestamp, &log.CreatedAt)
		if err != nil {
			continue
		}
		
		log.Details = details
		logs = append(logs, log)
	}

	return logs, nil
}

// Rest of the methods remain similar but with database logging added...
func (s *GoTeleportServerDB) sendAgentList(client *Client) {
	s.mutex.RLock()

	// Get user information
	var userRole string = "user" // Default to user role
	var userID int = 0

	// Try to get user info from database based on username
	if s.db != nil && client.Username != "" {
		query := `SELECT id, role FROM users WHERE username = ? AND active = 1`
		err := s.db.QueryRow(query, client.Username).Scan(&userID, &userRole)
		if err != nil {
			// User not found in database, default to user role
			userRole = "user"
			userID = 0
		}
	}

	// Get all agents
	allAgents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		allAgents = append(allAgents, agent)
	}
	s.mutex.RUnlock()

	// Filter agents based on user role and assignments
	var filteredAgents []*Agent

	if userRole == "admin" {
		// Admin can see all agents
		filteredAgents = allAgents
	} else if s.db != nil && userID > 0 {
		// Regular user - get assigned agents only
		var allowedAgentIDs map[string]bool = make(map[string]bool)

		query := `SELECT agent_id FROM user_agent_assignments WHERE user_id = ? AND active = 1`
		rows, err := s.db.Query(query, userID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var agentID string
				if err := rows.Scan(&agentID); err == nil {
					allowedAgentIDs[agentID] = true
				}
			}
		}

		// Filter agents based on assignments
		for _, agent := range allAgents {
			if allowedAgentIDs[agent.ID] {
				filteredAgents = append(filteredAgents, agent)
			}
		}
	} else {
		// User not found in database or no database - show all agents for development/testing
		// In production, you may want to restrict this
		filteredAgents = allAgents
	}

	response := Message{
		Type:      "agent_list",
		Data:      fmt.Sprintf("%d", len(filteredAgents)),
		Metadata:  map[string]interface{}{"agents": filteredAgents},
		Timestamp: time.Now(),
	}

	client.Connection.WriteJSON(response)
}

func (s *GoTeleportServerDB) createSession(client *Client, agentID string) {
	// Validate user access to agent first
	var userRole string = "user" // Default to user role
	var userID int = 0
	hasAccess := false

	// Try to get user info from database based on username
	if s.db != nil && client.Username != "" {
		query := `SELECT id, role FROM users WHERE username = ? AND active = 1`
		err := s.db.QueryRow(query, client.Username).Scan(&userID, &userRole)
		if err != nil {
			// User not found in database, default to user role
			userRole = "user"
			userID = 0
		}
	}

	// Check access permissions
	if userRole == "admin" {
		// Admin can access all agents
		hasAccess = true
	} else if s.db != nil && userID > 0 {
		// Regular user - check assignments
		query := `SELECT COUNT(*) FROM user_agent_assignments WHERE user_id = ? AND agent_id = ? AND active = 1`
		var count int
		err := s.db.QueryRow(query, userID, agentID).Scan(&count)
		if err == nil && count > 0 {
			hasAccess = true
		}
	}

	if !hasAccess {
		// Send access denied response
		response := Message{
			Type:      "access_denied",
			Data:      "You don't have permission to access this agent",
			AgentID:   agentID,
			Timestamp: time.Now(),
		}
		client.Connection.WriteJSON(response)
		s.logEvent("ACCESS_DENIED", "Client access denied to agent", fmt.Sprintf("Client: %s, Agent: %s, User: %s", client.ID, agentID, client.Username))
		return
	}

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

	// Log access for session creation
	s.mutex.RLock()
	agent := s.agents[agentID]
	s.mutex.RUnlock()

	if agent != nil {
		s.logAccess(client.ID, client.Name, client.Username, agentID, agent.Name, sessionID, "connect",
			"Session created between client and agent", "", "")
	}
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
		logEntry := fmt.Sprintf("📝 [%s] %s: %s | %s",
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

// Web interface handlers
func (s *GoTeleportServerDB) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>GoTeleport Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 400px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .btn { background: #007bff; color: white; padding: 12px 24px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; display: inline-block; margin: 5px; }
        .btn:hover { background: #0056b3; }
        .info { background: #e3f2fd; padding: 15px; border-radius: 4px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 GoTeleport Server</h1>
        <div class="info">
            <p><strong>Server Status:</strong> Running</p>
            <p><strong>Port:</strong> %d</p>
            <p><strong>WebSocket Endpoint:</strong> /ws/client</p>
        </div>
        <div style="text-align: center;">
            <a href="/login" class="btn">🔐 Admin Login</a>
            <a href="/dashboard" class="btn">📊 Dashboard</a>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, html, s.config.Port)
}

func (s *GoTeleportServerDB) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if user, authenticated := s.authenticateUser(username, password); authenticated && user.Active {
			s.logEvent("WEB_LOGIN", "User logged in", fmt.Sprintf("User: %s, Role: %s", username, user.Role))

			if user.Role == "admin" {
				http.Redirect(w, r, "/admin", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			}
			return
		} else {
			s.logEvent("WEB_AUTH_FAILED", "Invalid login attempt", fmt.Sprintf("User: %s", username))
		}
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>GoTeleport - Login</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 400px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .form-group { margin-bottom: 20px; }
        label { display: block; margin-bottom: 5px; color: #555; }
        input[type="text"], input[type="password"] { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
        .btn { background: #007bff; color: white; padding: 12px 24px; border: none; border-radius: 4px; cursor: pointer; width: 100%; }
        .btn:hover { background: #0056b3; }
        .error { color: red; margin-top: 10px; }
        .users { background: #fff3cd; padding: 15px; border-radius: 4px; margin: 20px 0; border: 1px solid #ffeaa7; }
        .users h3 { margin-top: 0; color: #856404; }
        .user-item { margin: 5px 0; font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 GoTeleport Login</h1>
        
        <div class="users">
            <h3>Available Test Users:</h3>
            <div class="user-item"><strong>Admin:</strong> admin1 / admin123</div>
            <div class="user-item"><strong>User:</strong> user1 / user123</div>
        </div>
        
        <form method="POST">
            <div class="form-group">
                <label for="username">Username:</label>
                <input type="text" id="username" name="username" required>
            </div>
            <div class="form-group">
                <label for="password">Password:</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn">Login</button>
        </form>
        
        <div style="text-align: center; margin-top: 20px;">
            <a href="/">← Back to Home</a>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *GoTeleportServerDB) handleAdmin(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	agentCount := len(s.agents)
	clientCount := len(s.clients)
	sessionCount := len(s.sessions)

	// Get detailed client info
	clientsInfo := make([]Client, 0, len(s.clients))
	for _, client := range s.clients {
		clientsInfo = append(clientsInfo, *client)
	}
	s.mutex.RUnlock()

	html := `<!DOCTYPE html>
<html>
<head>
    <title>GoTeleport - Admin Panel</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: #007bff; color: white; padding: 20px; border-radius: 8px; text-align: center; }
        .stat-number { font-size: 2em; font-weight: bold; margin-bottom: 5px; }
        .stat-label { font-size: 0.9em; opacity: 0.9; }
        .section { margin: 30px 0; }
        .section h2 { color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; color: #333; font-weight: bold; }
        .role-admin { background: #dc3545; color: white; padding: 4px 8px; border-radius: 4px; font-size: 0.8em; }
        .role-user { background: #28a745; color: white; padding: 4px 8px; border-radius: 4px; font-size: 0.8em; }
        .status-online { color: #28a745; font-weight: bold; }
        .nav { text-align: center; margin-top: 30px; }
        .btn { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; margin: 0 5px; }
        .btn:hover { background: #0056b3; }
    </style>
</head>
<body>
    <div class="container">
        <h1>👑 Admin Panel</h1>
        
        <div class="stats">
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Active Agents</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Connected Clients</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Active Sessions</div>
            </div>
        </div>
        
        <div class="section">
            <h2>Connected Clients</h2>
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Name</th>
                        <th>Username</th>
                        <th>Role</th>
                        <th>Status</th>
                        <th>Last Seen</th>
                    </tr>
                </thead>
                <tbody>`

	for _, client := range clientsInfo {
		roleClass := "role-user"
		if client.Role == "admin" {
			roleClass = "role-admin"
		}

		html += fmt.Sprintf(`
                    <tr>
                        <td>%s</td>
                        <td>%s</td>
                        <td>%s</td>
                        <td><span class="%s">%s</span></td>
                        <td class="status-online">%s</td>
                        <td>%s</td>
                    </tr>`,
			client.ID[:8]+"...", client.Name, client.Username, roleClass, client.Role, client.Status, client.LastSeen.Format("2006-01-02 15:04:05"))
	}

	html += `
                </tbody>
            </table>
        </div>
        
        <div class="nav">
            <a href="/logs" class="btn">📝 Command Logs</a>
            <a href="/api/sessions" class="btn">📊 View Sessions</a>
            <a href="/access-logs" class="btn">📋 Access Logs</a>
            <a href="/dashboard" class="btn">📈 Dashboard</a>
            <a href="/" class="btn">🏠 Home</a>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, html, agentCount, clientCount, sessionCount)
}

func (s *GoTeleportServerDB) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	agentCount := len(s.agents)
	clientCount := len(s.clients)
	sessionCount := len(s.sessions)
	s.mutex.RUnlock()

	html := `<!DOCTYPE html>
<html>
<head>
    <title>GoTeleport - Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: #28a745; color: white; padding: 20px; border-radius: 8px; text-align: center; }
        .stat-number { font-size: 2em; font-weight: bold; margin-bottom: 5px; }
        .stat-label { font-size: 0.9em; opacity: 0.9; }
        .info { background: #e3f2fd; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .nav { text-align: center; margin-top: 30px; }
        .btn { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; margin: 0 5px; }
        .btn:hover { background: #0056b3; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 GoTeleport Dashboard</h1>
        
        <div class="stats">
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Active Agents</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Connected Clients</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Active Sessions</div>
            </div>
        </div>
        
        <div class="info">
            <h3>🔗 Connection Information</h3>
            <p><strong>WebSocket Endpoint:</strong> ws://localhost:%d/ws/client</p>
            <p><strong>API Endpoints:</strong></p>
            <ul>
                <li>/api/logs - View server logs</li>
                <li>/api/sessions - View active sessions</li>
                <li>/api/stats - Server statistics</li>
            </ul>
        </div>
        
        <div class="nav">
            <a href="/login" class="btn">🔐 Admin Login</a>
            <a href="/" class="btn">🏠 Home</a>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, html, agentCount, clientCount, sessionCount, s.config.Port)
}

func (s *GoTeleportServerDB) handleAccessLogsPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>GoTeleport - Access Logs</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .filters { margin-bottom: 20px; display: flex; gap: 10px; align-items: center; }
        .filters input, .filters select { padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        .table-container { overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #f2f2f2; font-weight: bold; }
        .action-login { background: #28a745; color: white; padding: 2px 8px; border-radius: 3px; }
        .action-connect { background: #007bff; color: white; padding: 2px 8px; border-radius: 3px; }
        .action-command { background: #ffc107; color: black; padding: 2px 8px; border-radius: 3px; }
        .action-disconnect { background: #dc3545; color: white; padding: 2px 8px; border-radius: 3px; }
        .nav { text-align: center; margin-top: 30px; }
        .btn { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; margin: 0 5px; }
        .btn:hover { background: #0056b3; }
    </style>
    <script>
        function loadAccessLogs() {
            const username = document.getElementById('username').value;
            const clientId = document.getElementById('clientId').value;
            const agentId = document.getElementById('agentId').value;
            
            let url = '/api/access-logs?';
            if (username) url += 'username=' + encodeURIComponent(username) + '&';
            if (clientId) url += 'client_id=' + encodeURIComponent(clientId) + '&';
            if (agentId) url += 'agent_id=' + encodeURIComponent(agentId) + '&';
            
            fetch(url)
                .then(response => response.json())
                .then(data => {
                    const tbody = document.getElementById('logs-tbody');
                    tbody.innerHTML = '';
                    
                    data.logs.forEach(log => {
                        const row = document.createElement('tr');
                        row.innerHTML = ` + "`" + `
                            <td>${log.timestamp}</td>
                            <td>${log.username || '-'}</td>
                            <td>${log.client_name || '-'}</td>
                            <td>${log.agent_name || '-'}</td>
                            <td><span class="action-${log.action}">${log.action}</span></td>
                            <td>${log.details || '-'}</td>
                            <td>${log.ip_address || '-'}</td>
                        ` + "`" + `;
                        tbody.appendChild(row);
                    });
                    
                    document.getElementById('total-count').textContent = data.total;
                })
                .catch(error => console.error('Error:', error));
        }
        
        // Load logs when page loads
        window.onload = function() { loadAccessLogs(); };
    </script>
</head>
<body>
    <div class="container">
        <h1>📋 Access Logs</h1>
        
        <div class="filters">
            <input type="text" id="username" placeholder="Username" onchange="loadAccessLogs()">
            <input type="text" id="clientId" placeholder="Client ID" onchange="loadAccessLogs()">
            <input type="text" id="agentId" placeholder="Agent ID" onchange="loadAccessLogs()">
            <button onclick="loadAccessLogs()" class="btn">🔍 Filter</button>
            <span>Total: <strong id="total-count">0</strong> records</span>
        </div>
        
        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>Timestamp</th>
                        <th>Username</th>
                        <th>Client</th>
                        <th>Agent</th>
                        <th>Action</th>
                        <th>Details</th>
                        <th>IP Address</th>
                    </tr>
                </thead>
                <tbody id="logs-tbody">
                    <tr><td colspan="7">Loading...</td></tr>
                </tbody>
            </table>
        </div>
        
        <div class="nav">
            <a href="/admin" class="btn">📊 Admin Panel</a>
            <a href="/logs" class="btn">📝 Command Logs</a>
            <a href="/access-logs" class="btn">📋 Access Logs</a>
            <a href="/dashboard" class="btn">🎛️ Dashboard</a>
            <a href="/" class="btn">🏠 Home</a>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *GoTeleportServerDB) handleLogsPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>GoTeleport - Command Logs</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .filters { margin-bottom: 20px; display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
        .filters input, .filters select { padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        .stats { display: flex; gap: 20px; margin-bottom: 20px; }
        .stat-card { background: #e3f2fd; padding: 15px; border-radius: 8px; text-align: center; min-width: 120px; }
        .stat-number { font-size: 1.5em; font-weight: bold; color: #1976d2; }
        .stat-label { font-size: 0.9em; color: #666; }
        .table-container { overflow-x: auto; max-height: 600px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { border: 1px solid #ddd; padding: 10px; text-align: left; font-size: 0.9em; }
        th { background-color: #f2f2f2; font-weight: bold; position: sticky; top: 0; }
        .status-executed { background: #28a745; color: white; padding: 2px 6px; border-radius: 3px; font-size: 0.8em; }
        .status-failed { background: #dc3545; color: white; padding: 2px 6px; border-radius: 3px; font-size: 0.8em; }
        .status-running { background: #ffc107; color: black; padding: 2px 6px; border-radius: 3px; font-size: 0.8em; }
        .command-cell { max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .output-cell { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: monospace; font-size: 0.8em; }
        .duration { text-align: right; font-family: monospace; }
        .nav { text-align: center; margin-top: 30px; }
        .btn { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; margin: 0 5px; }
        .btn:hover { background: #0056b3; }
        .refresh-btn { background: #28a745; }
        .refresh-btn:hover { background: #218838; }
        .export-btn { background: #6f42c1; }
        .export-btn:hover { background: #5a2d91; }
        .search-container { display: flex; align-items: center; gap: 10px; }
        .pagination { text-align: center; margin: 20px 0; }
        .pagination button { margin: 0 5px; padding: 5px 10px; }
    </style>
    <script>
        let currentPage = 1;
        const pageSize = 50;
        let allLogs = [];
        
        function loadCommandLogs() {
            const sessionId = document.getElementById('sessionId').value;
            const clientId = document.getElementById('clientId').value;
            const agentId = document.getElementById('agentId').value;
            const status = document.getElementById('status').value;
            const command = document.getElementById('command').value;
            
            let url = '/api/logs?';
            if (sessionId) url += 'session_id=' + encodeURIComponent(sessionId) + '&';
            if (clientId) url += 'client_id=' + encodeURIComponent(clientId) + '&';
            if (agentId) url += 'agent_id=' + encodeURIComponent(agentId) + '&';
            if (status) url += 'status=' + encodeURIComponent(status) + '&';
            if (command) url += 'command=' + encodeURIComponent(command) + '&';
            
            document.getElementById('loading').style.display = 'block';
            
            fetch(url)
                .then(response => response.json())
                .then(data => {
                    console.log('API Response:', data);
                    allLogs = data.logs || [];
                    console.log('Loaded logs:', allLogs.length);
                    updateStats(data);
                    displayLogs();
                    document.getElementById('loading').style.display = 'none';
                })
                .catch(error => {
                    console.error('Error:', error);
                    document.getElementById('loading').style.display = 'none';
                    alert('Error loading logs: ' + error.message);
                });
        }
        
        function updateStats(data) {
            const totalLogs = data.logs ? data.logs.length : 0;
            const executedCount = data.logs ? data.logs.filter(log => log.status === 'executed').length : 0;
            const failedCount = data.logs ? data.logs.filter(log => log.status === 'failed').length : 0;
            
            document.getElementById('total-logs').textContent = totalLogs;
            document.getElementById('executed-count').textContent = executedCount;
            document.getElementById('failed-count').textContent = failedCount;
        }
        
        function displayLogs() {
            const startIndex = (currentPage - 1) * pageSize;
            const endIndex = startIndex + pageSize;
            const logsToShow = allLogs.slice(startIndex, endIndex);
            
            const tbody = document.getElementById('logs-tbody');
            tbody.innerHTML = '';
            
            if (logsToShow.length === 0) {
                tbody.innerHTML = '<tr><td colspan="9" style="text-align: center; padding: 20px;">No logs found</td></tr>';
                return;
            }
            
            logsToShow.forEach(log => {
                const row = document.createElement('tr');
                row.innerHTML = ` + "`" + `
                    <td>${formatTimestamp(log.timestamp)}</td>
                    <td>${log.session_id ? log.session_id.substring(0, 8) + '...' : '-'}</td>
                    <td>${log.client_name || '-'}</td>
                    <td>${log.agent_name || '-'}</td>
                    <td>${log.username || '-'}</td>
                    <td class="command-cell" title="${log.command || ''}">${log.command || '-'}</td>
                    <td class="output-cell" title="${log.output || ''}">${log.output || '-'}</td>
                    <td><span class="status-${log.status}">${log.status}</span></td>
                    <td class="duration">${log.duration_ms}ms</td>
                ` + "`" + `;
                tbody.appendChild(row);
            });
            
            updatePagination();
        }
        
        function formatTimestamp(timestamp) {
            const date = new Date(timestamp);
            return date.toLocaleString();
        }
        
        function updatePagination() {
            const totalPages = Math.ceil(allLogs.length / pageSize);
            const pagination = document.getElementById('pagination');
            pagination.innerHTML = '';
            
            if (totalPages <= 1) return;
            
            // Previous button
            if (currentPage > 1) {
                const prevBtn = document.createElement('button');
                prevBtn.textContent = '« Previous';
                prevBtn.onclick = () => { currentPage--; displayLogs(); };
                pagination.appendChild(prevBtn);
            }
            
            // Page numbers
            for (let i = Math.max(1, currentPage - 2); i <= Math.min(totalPages, currentPage + 2); i++) {
                const pageBtn = document.createElement('button');
                pageBtn.textContent = i;
                pageBtn.onclick = () => { currentPage = i; displayLogs(); };
                if (i === currentPage) {
                    pageBtn.style.background = '#007bff';
                    pageBtn.style.color = 'white';
                }
                pagination.appendChild(pageBtn);
            }
            
            // Next button
            if (currentPage < totalPages) {
                const nextBtn = document.createElement('button');
                nextBtn.textContent = 'Next »';
                nextBtn.onclick = () => { currentPage++; displayLogs(); };
                pagination.appendChild(nextBtn);
            }
            
            // Page info
            const pageInfo = document.createElement('span');
            pageInfo.textContent = ` + "`" + ` Page ${currentPage} of ${totalPages} (${allLogs.length} total records)` + "`" + `;
            pageInfo.style.marginLeft = '20px';
            pagination.appendChild(pageInfo);
        }
        
        function exportLogs() {
            if (allLogs.length === 0) {
                alert('No logs to export');
                return;
            }
            
            let csv = 'Timestamp,Session ID,Client Name,Agent Name,Username,Command,Output,Status,Duration (ms)\\n';
            allLogs.forEach(log => {
                csv += ` + "`" + `"${log.timestamp}","${log.session_id || ''}","${log.client_name || ''}","${log.agent_name || ''}","${log.username || ''}","${(log.command || '').replace(/"/g, '""')}","${(log.output || '').replace(/"/g, '""')}","${log.status}","${log.duration_ms}"\\n` + "`" + `;
            });
            
            const blob = new Blob([csv], { type: 'text/csv' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'command_logs_' + new Date().toISOString().split('T')[0] + '.csv';
            a.click();
            window.URL.revokeObjectURL(url);
        }
        
        function clearFilters() {
            document.getElementById('sessionId').value = '';
            document.getElementById('clientId').value = '';
            document.getElementById('agentId').value = '';
            document.getElementById('status').value = '';
            document.getElementById('command').value = '';
            currentPage = 1;
            loadCommandLogs();
        }
        
        // Auto-refresh every 30 seconds
        setInterval(loadCommandLogs, 30000);
        
        // Load logs when page loads
        window.onload = function() { loadCommandLogs(); };
    </script>
</head>
<body>
    <div class="container">
        <h1>📋 Command Logs</h1>
        
        <div class="stats">
            <div class="stat-card">
                <div class="stat-number" id="total-logs">0</div>
                <div class="stat-label">Total Commands</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="executed-count">0</div>
                <div class="stat-label">Executed</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="failed-count">0</div>
                <div class="stat-label">Failed</div>
            </div>
        </div>
        
        <div class="filters">
            <input type="text" id="sessionId" placeholder="Session ID" style="width: 150px;">
            <input type="text" id="clientId" placeholder="Client ID" style="width: 150px;">
            <input type="text" id="agentId" placeholder="Agent ID" style="width: 150px;">
            <select id="status">
                <option value="">All Status</option>
                <option value="executed">Executed</option>
                <option value="failed">Failed</option>
                <option value="running">Running</option>
            </select>
            <input type="text" id="command" placeholder="Search command..." style="width: 200px;">
            <button onclick="loadCommandLogs()" class="btn">🔍 Filter</button>
            <button onclick="clearFilters()" class="btn">🗑️ Clear</button>
            <button onclick="loadCommandLogs()" class="btn refresh-btn">🔄 Refresh</button>
            <button onclick="exportLogs()" class="btn export-btn">📥 Export CSV</button>
        </div>
        
        <div id="loading" style="display: none; text-align: center; padding: 20px;">
            <strong>Loading logs...</strong>
        </div>
        
        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>Timestamp</th>
                        <th>Session ID</th>
                        <th>Client Name</th>
                        <th>Agent Name</th>
                        <th>Username</th>
                        <th>Command</th>
                        <th>Output</th>
                        <th>Status</th>
                        <th>Duration</th>
                    </tr>
                </thead>
                <tbody id="logs-tbody">
                    <tr><td colspan="9">Loading...</td></tr>
                </tbody>
            </table>
        </div>
        
        <div class="pagination" id="pagination"></div>
        
        <div class="nav">
            <a href="/admin" class="btn">📊 Admin Panel</a>
            <a href="/access-logs" class="btn">📋 Access Logs</a>
            <a href="/dashboard" class="btn">🎛️ Dashboard</a>
            <a href="/" class="btn">🏠 Home</a>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Authentication functions
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func (s *GoTeleportServerDB) authenticateUser(username, password string) (*User, bool) {
	// If database is not enabled, use simple hardcoded authentication
	if s.db == nil {
		if username == "admin" && password == "admin123" {
			return &User{
				ID:       1,
				Username: username,
				Role:     "admin",
				Active:   true,
			}, true
		}
		if username == "user" && password == "user123" {
			return &User{
				ID:       2,
				Username: username,
				Role:     "user",
				Active:   true,
			}, true
		}
		return nil, false
	}

	var user User
	var passwordHash string
	query := `SELECT id, username, password_hash, role 
			  FROM users 
			  WHERE username = ? AND status = 'active'`

	err := s.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &passwordHash, &user.Role)

	if err != nil {
		return nil, false
	}

	// For existing users with bcrypt, just check direct match for now
	// Later we can implement proper bcrypt checking
	if passwordHash == password {
		user.Active = true
		return &user, true
	}

	// Also try SHA256 for new users
	hashedPassword := hashPassword(password)
	if passwordHash == hashedPassword {
		user.Active = true
		return &user, true
	}

	return nil, false
}

func (s *GoTeleportServerDB) createUser(username, password, role string) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	hashedPassword := hashPassword(password)
	query := `INSERT INTO users (username, password_hash, role, status, email, full_name) 
			  VALUES (?, ?, ?, 'active', ?, ?)`

	email := username + "@goteleport.local"
	fullName := username
	if role == "admin" {
		fullName = "Administrator"
	} else {
		fullName = "User"
	}

	_, err := s.db.Exec(query, username, hashedPassword, role, email, fullName)
	return err
}

func (s *GoTeleportServerDB) initDefaultUsers() error {
	// Check if any users exist
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	s.logEvent("USER_CHECK", "Checking existing users", fmt.Sprintf("Found %d users in database", count))

	// Don't create default users if any already exist
	if count > 0 {
		s.logEvent("USER_INIT", "Users already exist, skipping default user creation", fmt.Sprintf("Found %d users", count))
		return nil
	}

	// Create default users if none exist
	defaultUsers := []struct {
		username, password, role string
	}{
		{"admin", "admin123", "admin"},
		{"user", "user123", "user"},
	}

	for _, u := range defaultUsers {
		if err := s.createUser(u.username, u.password, u.role); err != nil {
			s.logEvent("DB_ERROR", "Failed to create default user", fmt.Sprintf("User: %s, Error: %v", u.username, err))
		} else {
			s.logEvent("USER_CREATED", "Default user created", fmt.Sprintf("User: %s, Role: %s", u.username, u.role))
		}
	}

	return nil
}

// Access logging functions
func (s *GoTeleportServerDB) logAccess(clientID, clientName, username, agentID, agentName, sessionID, action, details, ipAddress, userAgent string) {
	if s.db == nil {
		return
	}

	query := `INSERT INTO access_logs 
			  (client_id, client_name, username, agent_id, agent_name, session_id, action, details, ip_address, user_agent) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, clientID, clientName, username, agentID, agentName, sessionID, action, details, ipAddress, userAgent)
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to log access", err.Error())
	}
}

func (s *GoTeleportServerDB) getAccessLogs(limit int, clientID, agentID, username string) ([]AccessLog, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `SELECT id, client_id, client_name, username, agent_id, agent_name, session_id, 
			  action, details, ip_address, user_agent, timestamp 
			  FROM access_logs WHERE 1=1`
	args := []interface{}{}

	if clientID != "" {
		query += " AND client_id = ?"
		args = append(args, clientID)
	}

	if agentID != "" {
		query += " AND agent_id = ?"
		args = append(args, agentID)
	}

	if username != "" {
		query += " AND username = ?"
		args = append(args, username)
	}

	query += " ORDER BY timestamp DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AccessLog
	for rows.Next() {
		var log AccessLog
		err := rows.Scan(&log.ID, &log.ClientID, &log.ClientName, &log.Username,
			&log.AgentID, &log.AgentName, &log.SessionID, &log.Action,
			&log.Details, &log.IPAddress, &log.UserAgent, &log.Timestamp)
		if err != nil {
			s.logEvent("DB_ERROR", "Failed to scan access log", err.Error())
			continue
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// Database Command Logging Functions
func (s *GoTeleportServerDB) logDatabaseCommandToDB(agentID string, msg *Message) {
	if s.db == nil {
		return
	}

	// Extract metadata
	var proxyName, protocol, clientIP string
	if metadata := msg.Metadata; metadata != nil {
		if pn, ok := metadata["proxy_name"].(string); ok {
			proxyName = pn
		}
		if pr, ok := metadata["protocol"].(string); ok {
			protocol = pr
		}
		if ci, ok := metadata["client_ip"].(string); ok {
			clientIP = ci
		}
	}

	query := `INSERT INTO database_commands (session_id, agent_id, command, protocol, client_ip, proxy_name, metadata, timestamp, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	metadataJSON, _ := json.Marshal(msg.Metadata)
	
	_, err := s.db.Exec(query, msg.SessionID, agentID, msg.Command, protocol, clientIP, proxyName, 
		string(metadataJSON), msg.Timestamp, time.Now())
	
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to log database command", err.Error())
	}
}

func (s *GoTeleportServerDB) getDatabaseCommands(limit int, offset int, agentID string) ([]DatabaseCommand, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not enabled")
	}

	query := `SELECT id, session_id, agent_id, command, protocol, client_ip, proxy_name, metadata, timestamp, created_at 
			  FROM database_commands`
	args := []interface{}{}
	
	if agentID != "" {
		query += " WHERE agent_id = ?"
		args = append(args, agentID)
	}
	
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []DatabaseCommand
	for rows.Next() {
		var cmd DatabaseCommand
		var metadataJSON string
		
		err := rows.Scan(&cmd.ID, &cmd.SessionID, &cmd.AgentID, &cmd.Command, 
			&cmd.Protocol, &cmd.ClientIP, &cmd.ProxyName, &metadataJSON, 
			&cmd.Timestamp, &cmd.CreatedAt)
		if err != nil {
			s.logEvent("DB_ERROR", "Failed to scan database command", err.Error())
			continue
		}
		
		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &cmd.Metadata)
		}
		
		commands = append(commands, cmd)
	}

	return commands, nil
}

func (s *GoTeleportServerDB) handleGetDatabaseCommands(w http.ResponseWriter, r *http.Request) {
	if !s.config.EnableDB {
		http.Error(w, "Database not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	agentID := r.URL.Query().Get("agent_id")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	commands, err := s.getDatabaseCommands(limit, offset, agentID)
	if err != nil {
		s.logEvent("DB_ERROR", "Failed to get database commands", err.Error())
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"commands": commands,
		"total":    len(commands),
		"limit":    limit,
		"offset":   offset,
	})
}

// broadcastClientInfoToAgents sends client info to all connected agents
func (s *GoTeleportServerDB) broadcastClientInfoToAgents(client *Client) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	clientInfo := Message{
		Type:      "client_info",
		ClientID:  client.ID,
		Data:      client.Name,
		Metadata: map[string]interface{}{
			"name":     client.Name,
			"username": client.Username,
			"role":     client.Role,
			"status":   client.Status,
		},
		Timestamp: time.Now(),
	}

	for _, agent := range s.agents {
		if agent.Connection != nil {
			if err := agent.Connection.WriteJSON(clientInfo); err != nil {
				s.logEvent("ERROR", "Failed to send client info to agent", fmt.Sprintf("Agent: %s, Error: %v", agent.ID, err))
			}
		}
	}
}

// getActiveClient returns the most recent active client
func (s *GoTeleportServerDB) getActiveClient() *Client {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var mostRecentClient *Client
	var latestTime time.Time

	for _, client := range s.clients {
		if client.Status == "online" && client.LastSeen.After(latestTime) {
			mostRecentClient = client
			latestTime = client.LastSeen
		}
	}

	return mostRecentClient
}
