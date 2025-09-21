package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"ssh-tunnel/internal/common"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

type RelayServer struct {
	agents      map[string]*Agent
	clients     map[string]*Client
	sessions    map[string]*Session
	mutex       sync.RWMutex
	logger      *common.Logger
	db          *sql.DB
	webSessions map[string]*WebSession // Enhanced session storage with user info

	// Performance optimization: fast connection lookup
	connToAgent  map[*websocket.Conn]string
	connToClient map[*websocket.Conn]string
	connMutex    sync.RWMutex

	// Batch logging for performance
	logBuffer *LogBuffer
}

// LogBuffer for batch logging to improve performance
type LogBuffer struct {
	sshLogs   []SSHLogEntry
	queryLogs []QueryLogEntry
	mutex     sync.Mutex
	lastFlush time.Time
}

type SSHLogEntry struct {
	SessionID string
	AgentID   string
	ClientID  string
	Direction string
	Command   string
	User      string
	Host      string
	Port      string
	Data      string
	IsBase64  bool
	DataSize  int
	Timestamp time.Time
}

type QueryLogEntry struct {
	SessionID string
	AgentID   string
	ClientID  string
	Direction string
	Protocol  string
	Operation string
	TableName string
	QueryText string
	Timestamp time.Time
}

type WebSession struct {
	Username  string
	Role      string
	LoginTime time.Time
}

type Agent struct {
	ID          string          `json:"id"`
	Connection  *websocket.Conn `json:"-"`
	ConnectedAt time.Time       `json:"connected_at"`
	LastPing    time.Time       `json:"last_ping"`
	Status      string          `json:"status"`
}

type Client struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Connection  *websocket.Conn `json:"-"`
	ConnectedAt time.Time       `json:"connected_at"`
	LastPing    time.Time       `json:"last_ping"`
	AgentID     string          `json:"agent_id"`
	LocalPort   string          `json:"local_port"`
	TargetAddr  string          `json:"target_addr"`
	Status      string          `json:"status"`
}

type Session struct {
	ID       string
	AgentID  string
	ClientID string
	Target   string
	Protocol string // "mysql", "postgresql", "ssh", "tcp"
	Created  time.Time
}

type SSHTunnelLogRequest struct {
	SessionID string `json:"session_id"`
	ClientID  string `json:"client_id"`
	AgentID   string `json:"agent_id"`
	Direction string `json:"direction"`
	Command   string `json:"command"`
	User      string `json:"user"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	Data      string `json:"data"`
	IsBase64  bool   `json:"is_base64"`
}

type QueryLogRequest struct {
	SessionID    string `json:"session_id"`
	ClientID     string `json:"client_id"`
	AgentID      string `json:"agent_id"`
	Direction    string `json:"direction"`
	Protocol     string `json:"protocol"`
	Operation    string `json:"operation"`
	TableName    string `json:"table_name"`
	DatabaseName string `json:"database_name"`
	QueryText    string `json:"query_text"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

func NewRelayServer() *RelayServer {
	// Get log level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO" // Default to INFO instead of DEBUG
	}

	rs := &RelayServer{
		agents:       make(map[string]*Agent),
		clients:      make(map[string]*Client),
		sessions:     make(map[string]*Session),
		logger:       common.NewLogger("RELAY"),
		webSessions:  make(map[string]*WebSession),
		connToAgent:  make(map[*websocket.Conn]string),
		connToClient: make(map[*websocket.Conn]string),
		logBuffer:    &LogBuffer{lastFlush: time.Now()},
	}

	// Get database configuration from environment variables
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "root"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "tunnel"
	}

	// Debug: Log environment variables values
	// rs.logger.Info("=== DATABASE CONFIG DEBUG ===")
	// rs.logger.Info("DB_HOST env value: '%s' (using: %s)", os.Getenv("DB_HOST"), dbHost)
	// rs.logger.Info("DB_PORT env value: '%s' (using: %s)", os.Getenv("DB_PORT"), dbPort)
	// rs.logger.Info("DB_USER env value: '%s' (using: %s)", os.Getenv("DB_USER"), dbUser)
	// rs.logger.Info("DB_NAME env value: '%s' (using: %s)", os.Getenv("DB_NAME"), dbName)
	// rs.logger.Info("DB_PASSWORD env set: %t", os.Getenv("DB_PASSWORD") != "")

	// Construct DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	rs.logger.Info("Connecting to MySQL: %s:***@tcp(%s:%s)/%s", dbUser, dbHost, dbPort, dbName)

	// Initialize database
	var err error
	rs.db, err = sql.Open("mysql", dsn)
	if err != nil {
		rs.logger.Error("Failed to open database: %v", err)
		return rs
	}

	// Test connection
	err = rs.db.Ping()
	if err != nil {
		rs.logger.Error("Failed to connect to database: %v", err)
		return rs
	}

	rs.logger.Info("Connected to MySQL database '%s'", dbName)
	rs.initDatabase()

	// Start periodic log flusher for performance
	go rs.periodicFlush()

	return rs
}

func (rs *RelayServer) initDatabase() {
	// Create tables
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id INT AUTO_INCREMENT PRIMARY KEY,
            username VARCHAR(50) UNIQUE NOT NULL,
            password VARCHAR(255) NOT NULL,
            role VARCHAR(20) DEFAULT 'user',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS clients (
            id INT AUTO_INCREMENT PRIMARY KEY,
            client_id VARCHAR(100) UNIQUE NOT NULL,
            client_name VARCHAR(255),
            token VARCHAR(255),
            status VARCHAR(20) DEFAULT 'connected',
            connected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            last_ping TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS client_groups (
            id INT AUTO_INCREMENT PRIMARY KEY,
            group_name VARCHAR(100) UNIQUE NOT NULL,
            description TEXT,
            created_by VARCHAR(100),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS projects (
            id INT AUTO_INCREMENT PRIMARY KEY,
            project_name VARCHAR(100) UNIQUE NOT NULL,
            description TEXT,
            created_by VARCHAR(100),
            status ENUM('active', 'inactive') DEFAULT 'active',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS user_project_assignments (
            id INT AUTO_INCREMENT PRIMARY KEY,
            user_id INT NOT NULL,
            project_id INT NOT NULL,
            role ENUM('viewer', 'operator', 'admin') DEFAULT 'viewer',
            assigned_by VARCHAR(100),
            assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            status ENUM('active', 'inactive') DEFAULT 'active',
            INDEX idx_user_id (user_id),
            INDEX idx_project_id (project_id),
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
            UNIQUE KEY unique_user_project (user_id, project_id)
        )`,
		`CREATE TABLE IF NOT EXISTS client_assignments (
            id INT AUTO_INCREMENT PRIMARY KEY,
            client_id VARCHAR(100) NOT NULL,
            agent_id VARCHAR(100),
            group_id INT,
            assignment_type ENUM('individual', 'group') DEFAULT 'individual',
            assigned_by VARCHAR(100),
            assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            status VARCHAR(20) DEFAULT 'active',
            INDEX idx_client_id (client_id),
            INDEX idx_agent_id (agent_id),
            INDEX idx_group_id (group_id),
            FOREIGN KEY (group_id) REFERENCES client_groups(id) ON DELETE CASCADE,
            UNIQUE KEY unique_individual_assignment (client_id, agent_id),
            UNIQUE KEY unique_group_assignment (client_id, group_id)
        )`,
		`CREATE TABLE IF NOT EXISTS agents (
            id INT AUTO_INCREMENT PRIMARY KEY,
            agent_id VARCHAR(100) UNIQUE NOT NULL,
            token VARCHAR(255),
            status VARCHAR(20) DEFAULT 'connected',
            connected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            last_ping TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS connection_logs (
            id INT AUTO_INCREMENT PRIMARY KEY,
            type VARCHAR(20) NOT NULL,
            agent_id VARCHAR(100),
            client_id VARCHAR(100),
            event VARCHAR(50) NOT NULL,
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            details TEXT
        )`,
		`CREATE TABLE IF NOT EXISTS tunnel_logs (
            id INT AUTO_INCREMENT PRIMARY KEY,
            session_id VARCHAR(100),
            agent_id VARCHAR(100),
            client_id VARCHAR(100),
            direction VARCHAR(20),
            protocol VARCHAR(20),
            operation VARCHAR(100),
            table_name VARCHAR(100),
            query_text LONGTEXT,
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS ssh_logs (
            id INT AUTO_INCREMENT PRIMARY KEY,
            session_id VARCHAR(100),
            agent_id VARCHAR(100),
            client_id VARCHAR(100),
            direction VARCHAR(20),
            ssh_user VARCHAR(100),
            ssh_host VARCHAR(100),
            ssh_port VARCHAR(10),
            command TEXT,
            data LONGTEXT,
            is_base64 BOOLEAN DEFAULT FALSE,
            data_size INT,
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS server_settings (
            id INT AUTO_INCREMENT PRIMARY KEY,
            setting_key VARCHAR(100) UNIQUE NOT NULL,
            setting_value TEXT,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        )`,
	}

	for _, query := range queries {
		_, err := rs.db.Exec(query)
		if err != nil {
			rs.logger.Error("Failed to create table: %v", err)
		}
	}

	// Add new columns to existing tables if they don't exist
	alterQueries := []string{
		`ALTER TABLE tunnel_logs ADD COLUMN agent_id VARCHAR(100) AFTER session_id`,
		`ALTER TABLE tunnel_logs ADD COLUMN client_id VARCHAR(100) AFTER agent_id`,
		`ALTER TABLE tunnel_logs ADD COLUMN database_name VARCHAR(100) AFTER table_name`,
		`ALTER TABLE agents ADD COLUMN token VARCHAR(255) AFTER agent_id`,
		`ALTER TABLE agents ADD COLUMN agent_name VARCHAR(255) AFTER agent_id`,
		`ALTER TABLE agents ADD COLUMN project_id INT AFTER agent_name`,
		`ALTER TABLE clients ADD COLUMN token VARCHAR(255) AFTER agent_id`,
		`ALTER TABLE users ADD COLUMN id INT AUTO_INCREMENT PRIMARY KEY FIRST`,
	}

	for _, query := range alterQueries {
		_, err := rs.db.Exec(query)
		if err != nil {
			// Check if error is about column already existing
			if !strings.Contains(err.Error(), "Duplicate column name") {
				rs.logger.Debug("ALTER TABLE query result: %v", err)
			}
		}
	}

	// Insert default users if not exists
	var count int
	rs.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		// Create admin user
		adminUsername := os.Getenv("ADMIN_USERNAME")
		if adminUsername == "" {
			adminUsername = "admin"
		}

		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			adminPassword = "admin123"
		}

		_, err := rs.db.Exec("INSERT INTO users (username, password, role) VALUES (?, ?, ?)", adminUsername, adminPassword, "admin")
		if err != nil {
			rs.logger.Error("Failed to create default admin user: %v", err)
		} else {
			rs.logger.Info("Created default admin user (%s/***)", adminUsername)
		}

		// Create regular user
		userUsername := os.Getenv("USER_USERNAME")
		if userUsername == "" {
			userUsername = "user"
		}

		userPassword := os.Getenv("USER_PASSWORD")
		if userPassword == "" {
			userPassword = "user123"
		}

		_, err = rs.db.Exec("INSERT INTO users (username, password, role) VALUES (?, ?, ?)", userUsername, userPassword, "user")
		if err != nil {
			rs.logger.Error("Failed to create default regular user: %v", err)
		} else {
			rs.logger.Info("Created default regular user (%s/***)", userUsername)
		}
	}

	// Insert default projects if not exists
	var projectCount int
	rs.db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectCount)
	if projectCount == 0 {
		// Create default project
		_, err := rs.db.Exec("INSERT INTO projects (project_name, description, created_by, status) VALUES (?, ?, ?, ?)",
			"Default Project", "Default project for all agents", "admin", "active")
		if err != nil {
			rs.logger.Error("Failed to create default project: %v", err)
		} else {
			rs.logger.Info("Created default project")
		}

		// Create development project
		_, err = rs.db.Exec("INSERT INTO projects (project_name, description, created_by, status) VALUES (?, ?, ?, ?)",
			"Development", "Development environment agents", "admin", "active")
		if err != nil {
			rs.logger.Error("Failed to create development project: %v", err)
		} else {
			rs.logger.Info("Created development project")
		}

		// Create production project
		_, err = rs.db.Exec("INSERT INTO projects (project_name, description, created_by, status) VALUES (?, ?, ?, ?)",
			"Production", "Production environment agents", "admin", "active")
		if err != nil {
			rs.logger.Error("Failed to create production project: %v", err)
		} else {
			rs.logger.Info("Created production project")
		}
	}

	// Assign admin user to all projects
	var adminUserId int
	err := rs.db.QueryRow("SELECT id FROM users WHERE username = 'admin' LIMIT 1").Scan(&adminUserId)
	if err == nil {
		// Get all projects and assign admin to them
		rows, err := rs.db.Query("SELECT id FROM projects")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var projectId int
				if rows.Scan(&projectId) == nil {
					// Check if assignment already exists
					var assignmentCount int
					rs.db.QueryRow("SELECT COUNT(*) FROM user_project_assignments WHERE user_id = ? AND project_id = ?",
						adminUserId, projectId).Scan(&assignmentCount)

					if assignmentCount == 0 {
						_, err = rs.db.Exec("INSERT INTO user_project_assignments (user_id, project_id, role, assigned_by, status) VALUES (?, ?, ?, ?, ?)",
							adminUserId, projectId, "admin", "system", "active")
						if err != nil {
							rs.logger.Error("Failed to assign admin to project %d: %v", projectId, err)
						} else {
							rs.logger.Info("Assigned admin user to project %d", projectId)
						}
					}
				}
			}
		}
	}

	// Initialize default server settings if not exists
	var settingsCount int
	rs.db.QueryRow("SELECT COUNT(*) FROM server_settings").Scan(&settingsCount)
	if settingsCount == 0 {
		// Detect server IP automatically
		defaultIP := rs.detectServerIP()

		// Insert default settings
		defaultSettings := map[string]string{
			"server_ip":   defaultIP,
			"server_port": "8080",
		}

		for key, value := range defaultSettings {
			_, err := rs.db.Exec("INSERT INTO server_settings (setting_key, setting_value) VALUES (?, ?)", key, value)
			if err != nil {
				rs.logger.Error("Failed to create default setting %s: %v", key, err)
			} else {
				rs.logger.Info("Created default setting: %s = %s", key, value)
			}
		}
	}
}

// saveClientToDatabase saves or updates client information in the database
func (rs *RelayServer) saveClientToDatabase(clientID, clientName, agentID, status, token string) {
	if rs.db == nil {
		return
	}

	// Clean all parameters
	clientID = rs.cleanString(clientID)
	clientName = rs.cleanString(clientName)
	agentID = rs.cleanString(agentID)
	status = rs.cleanString(status)
	token = rs.cleanString(token)

	// Get username from token
	var username string
	if token != "" {
		err := rs.db.QueryRow("SELECT username FROM users WHERE token = ?", token).Scan(&username)
		if err != nil {
			username = "unknown"
		}
	}

	// Check if client exists
	var existingID sql.NullString
	err := rs.db.QueryRow("SELECT client_id FROM clients WHERE client_id = ?", clientID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Client doesn't exist, insert new client
		_, err = rs.db.Exec(`
			INSERT INTO clients (client_id, client_name, agent_id, token, status, username, connected_at, last_ping) 
			VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			clientID, clientName, agentID, token, status, username,
		)
		if err != nil {
			rs.logger.Error("Failed to insert new client: %v", err)
		} else {
			rs.logger.Info("New client inserted: %s (name: %s, user: %s, agent: %s)", clientID, clientName, username, agentID)
		}
	} else if err != nil {
		rs.logger.Error("Failed to check existing client: %v", err)
	} else {
		// Client exists, update information
		_, err = rs.db.Exec(`
			UPDATE clients 
			SET client_name = ?, agent_id = ?, token = ?, status = ?, username = ?, last_ping = NOW() 
			WHERE client_id = ?`,
			clientName, agentID, token, status, username, clientID,
		)

		if err != nil {
			rs.logger.Error("Failed to update client: %v", err)
		} else {
			rs.logger.Info("Client updated: %s (name: %s, user: %s, agent: %s)", clientID, clientName, username, agentID)
		}
	}
}

// updateClientStatus updates only the client status without changing other data
func (rs *RelayServer) updateClientStatus(clientID, status string) {
	if rs.db == nil {
		return
	}

	// Clean parameters
	clientID = rs.cleanString(clientID)
	status = rs.cleanString(status)

	// Update only status and last_ping, preserve other data including username
	_, err := rs.db.Exec(`
		UPDATE clients 
		SET status = ?, last_ping = NOW() 
		WHERE client_id = ?`,
		status, clientID,
	)

	if err != nil {
		rs.logger.Error("Failed to update client status: %v", err)
	} else {
		rs.logger.Info("Client status updated: %s -> %s", clientID, status)
	}
}

// saveAgentToDatabase saves or updates agent information in the database
func (rs *RelayServer) saveAgentToDatabase(agentID, status string) {
	if rs.db == nil {
		return
	}

	// Clean all parameters
	agentID = rs.cleanString(agentID)
	status = rs.cleanString(status)

	// Use UPDATE to preserve existing token, or INSERT if agent doesn't exist
	// First try to update existing agent (preserving token)
	result, err := rs.db.Exec(`
		UPDATE agents 
		SET status = ?, last_ping = NOW(), updated_at = NOW() 
		WHERE agent_id = ?`,
		status, agentID,
	)

	if err != nil {
		rs.logger.Error("Failed to update agent in database: %v", err)
		return
	}

	// Check if any rows were affected (agent exists)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		rs.logger.Error("Failed to check affected rows: %v", err)
		return
	}

	// If no rows affected, agent doesn't exist - insert new agent (without token for now)
	if rowsAffected == 0 {
		_, err = rs.db.Exec(`
			INSERT INTO agents (agent_id, status, connected_at, last_ping) 
			VALUES (?, ?, NOW(), NOW())`,
			agentID, status,
		)
		if err != nil {
			rs.logger.Error("Failed to insert agent to database: %v", err)
		} else {
			rs.logger.Debug("New agent inserted to database: %s (status: %s)", agentID, status)
		}
	} else {
		rs.logger.Debug("Agent updated in database: %s (status: %s)", agentID, status)
	}
}

// Add missing logConnection method
func (rs *RelayServer) logConnection(connType, agentID, clientID, event, details string) {
	// Clean all string parameters before inserting
	connType = rs.cleanString(connType)
	agentID = rs.cleanString(agentID)
	clientID = rs.cleanString(clientID)
	event = rs.cleanOperation(event) // Use cleanOperation for event field as it might contain operations
	details = rs.cleanString(details)

	_, err := rs.db.Exec(
		"INSERT INTO connection_logs (type, agent_id, client_id, event, details) VALUES (?, ?, ?, ?, ?)",
		connType, agentID, clientID, event, details,
	)
	if err != nil {
		rs.logger.Error("Failed to log connection: %v", err)
	}
}

// cleanString removes unwanted characters and trims whitespace
func (rs *RelayServer) cleanString(s string) string {
	if s == "" {
		return s
	}

	// Trim whitespace
	s = strings.TrimSpace(s)

	// Remove specific unwanted characters
	unwantedChars := []string{"&", "?", "#", "�", "<", ">", "\"", "'", "`", "|", "\\", "/", "*", "%", "$", "!", "@", "^", "~"}
	for _, char := range unwantedChars {
		s = strings.ReplaceAll(s, char, "")
	}

	// Remove HTML entities manually
	htmlEntities := map[string]string{
		"&amp;":  "",
		"&lt;":   "",
		"&gt;":   "",
		"&quot;": "",
		"&apos;": "",
		"&#39;":  "",
		"&#34;":  "",
	}
	for entity, replacement := range htmlEntities {
		s = strings.ReplaceAll(s, entity, replacement)
	}

	// Remove control characters (except newlines and tabs which might be useful in queries)
	cleaned := strings.Map(func(r rune) rune {
		// Remove control characters
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return -1
		}
		// Remove non-printable Unicode characters
		if r > 126 && r < 160 {
			return -1
		}
		// Remove Unicode replacement character and other problematic characters
		if r == 0xFFFD || r == 0x00A0 || r == 0x200B || r == 0x200C || r == 0x200D {
			return -1
		}
		return r
	}, s)

	// Trim again after cleaning
	cleaned = strings.TrimSpace(cleaned)

	// Remove multiple spaces and replace with single space
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}

	return cleaned
}

// cleanOperation specifically cleans database operation strings
func (rs *RelayServer) cleanOperation(operation string) string {
	if operation == "" {
		return operation
	}

	// First apply general cleaning
	operation = rs.cleanString(operation)

	// Remove common prefixes that appear before SQL operations
	prefixesToRemove := []string{"=", "1", "?", "#", "&", "$", "@", "!", "*", "+", "-", "~", "`", "'", "\"", "(", ")", "[", "]", "{", "}", "<", ">", "|", "\\", "/", ":", ";", ",", "."}

	// Keep removing prefixes until we find a clean operation
	for {
		originalOperation := operation

		for _, prefix := range prefixesToRemove {
			if strings.HasPrefix(operation, prefix) {
				operation = strings.TrimPrefix(operation, prefix)
				operation = strings.TrimSpace(operation)
			}
		}

		// If no change was made, break the loop
		if operation == originalOperation {
			break
		}
	}

	// Ensure the operation starts with a valid SQL keyword
	validOperations := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "EXPLAIN", "DESCRIBE", "SHOW", "SET", "USE", "GRANT", "REVOKE", "TRUNCATE", "REPLACE"}

	operationUpper := strings.ToUpper(operation)
	for _, validOp := range validOperations {
		if strings.HasPrefix(operationUpper, validOp) {
			// Return the operation starting with the valid keyword
			return validOp + operation[len(validOp):]
		}
	}

	// If no valid operation found, return cleaned string
	return operation
}

// isAllowedOperation checks if an operation should be saved to database
func (rs *RelayServer) isAllowedOperation(operation string) bool {
	if operation == "" {
		return false
	}

	// Convert to uppercase for comparison
	op := strings.ToUpper(strings.TrimSpace(operation))

	// List of allowed operations to save to database (as requested by user)
	allowedOps := []string{
		"DROP_TABLE", "DROP", "CREATE", "ALTER", "DESCRIBE", "EXPLAIN",
		"UPDATE", "PREPARE", "BEGIN_TRANSACTION", "DELETE", "CREATE_TABLE",
		"ALTER_TABLE", "TRUNCATE",
		// Additional common SQL operations that are useful to track
		"SELECT", "INSERT", "COMMIT", "ROLLBACK", "SHOW", "USE",
		"CREATE_INDEX", "DROP_INDEX",
	}

	// Check if operation starts with any allowed operation
	for _, allowedOp := range allowedOps {
		if strings.HasPrefix(op, allowedOp) {
			return true
		}
	}

	rs.logger.Debug("Operation '%s' not in allowed list, skipping", operation)
	return false
}

func (rs *RelayServer) logTunnelQuery(sessionID, agentID, clientID, direction, protocol, operation, tableName, databaseName, queryText string) {
	// Clean all string parameters before processing
	sessionID = rs.cleanString(sessionID)
	agentID = rs.cleanString(agentID)
	clientID = rs.cleanString(clientID)
	direction = rs.cleanString(direction)
	protocol = rs.cleanString(protocol)
	operation = rs.cleanOperation(operation) // Use special cleaning for operation
	tableName = rs.cleanString(tableName)
	databaseName = rs.cleanString(databaseName)
	queryText = rs.cleanString(queryText)

	// Check if this operation should be saved to database
	if !rs.isAllowedOperation(operation) {
		rs.logger.Debug("Skipping operation '%s' - not in allowed list", operation)
		return
	}

	_, err := rs.db.Exec(
		"INSERT INTO tunnel_logs (session_id, agent_id, client_id, direction, protocol, operation, table_name, database_name, query_text) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		sessionID, agentID, clientID, direction, protocol, operation, tableName, databaseName, queryText,
	)
	if err != nil {
		rs.logger.Error("Failed to log tunnel query: %v", err)
	} else {
		rs.logger.Debug("Database query logged successfully")
	}
}

// logSSHCommand logs SSH commands and activities
func (rs *RelayServer) logSSHCommand(sessionID, agentID, clientID, direction, sshUser, sshHost, sshPort, command string, dataSize int) {
	// Clean all string parameters before processing
	sessionID = rs.cleanString(sessionID)
	agentID = rs.cleanString(agentID)
	clientID = rs.cleanString(clientID)
	direction = rs.cleanString(direction)
	sshUser = rs.cleanString(sshUser)
	sshHost = rs.cleanString(sshHost)
	sshPort = rs.cleanString(sshPort)
	command = rs.cleanString(command)

	_, err := rs.db.Exec(
		"INSERT INTO ssh_logs (session_id, agent_id, client_id, direction, ssh_user, ssh_host, ssh_port, command, data_size) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		sessionID, agentID, clientID, direction, sshUser, sshHost, sshPort, command, dataSize,
	)
	if err != nil {
		rs.logger.Error("Failed to log SSH command: %v", err)
	} else {
		rs.logger.Debug("SSH command logged successfully: %s", command)
	}
}

func (rs *RelayServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		rs.logger.Error("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	rs.logger.Info("New WebSocket connection from %s", r.RemoteAddr)
	rs.logger.Debug("Connection headers: %+v", r.Header)

	for {
		messageType, messageData, err := conn.ReadMessage()
		if err != nil {
			rs.logger.Error("Failed to read message from %s: %v", r.RemoteAddr, err)
			break
		}

		if messageType == websocket.TextMessage {
			// Handle JSON messages
			rs.logger.Debug("Received raw JSON message: %s", string(messageData))

			message, err := common.FromJSON(messageData)
			if err != nil {
				rs.logger.Error("Failed to parse JSON message from %s: %v", r.RemoteAddr, err)
				continue
			}

			rs.handleMessage(conn, message)
		} else if messageType == websocket.BinaryMessage {
			// For now, ignore binary messages to avoid corruption
			rs.logger.Debug("Ignoring binary message (%d bytes) - using JSON only mode", len(messageData))
		}
	}

	// Clean up connection
	rs.logger.Info("Cleaning up connection from %s", r.RemoteAddr)
	rs.cleanupConnection(conn)
}

func (rs *RelayServer) handleBinarySSHData(conn *websocket.Conn, messageData []byte) {
	// Parse binary SSH data frame
	// Format: [TYPE:4][CLIENT_ID_LEN:1][CLIENT_ID][AGENT_ID_LEN:1][AGENT_ID][SESSION_ID_LEN:1][SESSION_ID][DATA]

	if len(messageData) < 7 {
		rs.logger.Error("Binary message too short: %d bytes", len(messageData))
		return
	}

	offset := 0

	// Check type (4 bytes)
	if string(messageData[0:4]) != "DATA" {
		rs.logger.Error("Invalid binary message type: %s", string(messageData[0:4]))
		return
	}
	offset += 4

	// Read client ID
	clientIDLen := int(messageData[offset])
	offset++
	if offset+clientIDLen > len(messageData) {
		rs.logger.Error("Invalid client ID length")
		return
	}
	clientID := string(messageData[offset : offset+clientIDLen])
	offset += clientIDLen

	// Read agent ID
	if offset >= len(messageData) {
		rs.logger.Error("Missing agent ID")
		return
	}
	agentIDLen := int(messageData[offset])
	offset++
	if offset+agentIDLen > len(messageData) {
		rs.logger.Error("Invalid agent ID length")
		return
	}
	agentID := string(messageData[offset : offset+agentIDLen])
	offset += agentIDLen

	// Read session ID
	if offset >= len(messageData) {
		rs.logger.Error("Missing session ID")
		return
	}
	sessionIDLen := int(messageData[offset])
	offset++
	if offset+sessionIDLen > len(messageData) {
		rs.logger.Error("Invalid session ID length")
		return
	}
	sessionID := string(messageData[offset : offset+sessionIDLen])
	offset += sessionIDLen

	// Extract SSH data
	sshData := messageData[offset:]

	rs.logger.Debug("Binary SSH data: ClientID=%s, AgentID=%s, SessionID=%s, DataLen=%d",
		clientID, agentID, sessionID, len(sshData))

	// Create message structure for forwarding
	msg := &common.Message{
		Type:      common.MsgTypeData,
		ClientID:  clientID,
		AgentID:   agentID,
		SessionID: sessionID,
		Data:      sshData,
	}

	// Forward to appropriate target
	rs.handleData(conn, msg)
}

// validateAgentToken validates agent token against database
func (rs *RelayServer) validateAgentToken(agentID, token string) bool {
	if agentID == "" || token == "" {
		rs.logger.Error("Agent ID or token is empty")
		return false
	}

	// Clean inputs
	agentID = rs.cleanString(agentID)
	token = rs.cleanString(token)

	// Check token in database
	var dbToken sql.NullString
	err := rs.db.QueryRow("SELECT token FROM agents WHERE agent_id = ?", agentID).Scan(&dbToken)
	if err != nil {
		if err == sql.ErrNoRows {
			rs.logger.Error("Agent not found in database: %s", agentID)
		} else {
			rs.logger.Error("Database error validating token for agent %s: %v", agentID, err)
		}
		return false
	}

	// Compare tokens
	if !dbToken.Valid || dbToken.String != token {
		rs.logger.Error("Token mismatch for agent %s", agentID)
		return false
	}

	rs.logger.Info("Token validation successful for agent: %s", agentID)
	return true
}

// validateClientToken validates client token against database
func (rs *RelayServer) validateClientToken(clientID, token string) bool {
	if clientID == "" || token == "" {
		rs.logger.Error("Client ID or token is empty")
		return false
	}

	// Clean inputs
	clientID = rs.cleanString(clientID)
	token = rs.cleanString(token)

	// Check token in database
	var dbToken sql.NullString
	err := rs.db.QueryRow("SELECT token FROM clients WHERE client_id = ?", clientID).Scan(&dbToken)
	if err != nil {
		if err == sql.ErrNoRows {
			rs.logger.Error("Client not found in database: %s", clientID)
		} else {
			rs.logger.Error("Database error validating token for client %s: %v", clientID, err)
		}
		return false
	}

	// Compare tokens
	if !dbToken.Valid || dbToken.String != token {
		rs.logger.Error("Token mismatch for client %s", clientID)
		return false
	}

	rs.logger.Info("Token validation successful for client: %s", clientID)
	return true
}

// validateUserToken validates user token against database users table
func (rs *RelayServer) validateUserToken(token string) (string, bool) {
	if token == "" {
		rs.logger.Error("User token is empty")
		return "", false
	}

	// Clean inputs
	token = rs.cleanString(token)

	// Check token in users database
	var username string
	var role string
	err := rs.db.QueryRow("SELECT username, role FROM users WHERE token = ?", token).Scan(&username, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			rs.logger.Error("User token not found in database: %s", token)
		} else {
			rs.logger.Error("Database error validating user token: %v", err)
		}
		return "", false
	}

	rs.logger.Info("Token validation successful for user: %s (role: %s)", username, role)
	return username, true
}

func (rs *RelayServer) handleMessage(conn *websocket.Conn, msg *common.Message) {
	rs.logger.Debug("Received message: %s", msg.String())

	switch msg.Type {
	case common.MsgTypeRegister:
		rs.handleRegister(conn, msg)
	case common.MsgTypeConnect:
		rs.handleConnect(conn, msg)
	case common.MsgTypeData:
		rs.handleData(conn, msg)
	case common.MsgTypeClose:
		rs.handleClose(conn, msg)
	case common.MsgTypeHeartbeat:
		rs.handleHeartbeat(conn, msg)
	case common.MsgTypeDBQuery:
		rs.handleDBQuery(conn, msg)
	case common.MsgTypeSSHLog:
		rs.handleSSHLog(conn, msg)
	case "shell_command":
		rs.handleShellCommand(conn, msg)
	case "shell_response":
		rs.handleShellResponse(conn, msg)
	case "shell_error":
		rs.handleShellError(conn, msg)
	default:
		rs.logger.Error("Unknown message type: %s", msg.Type)
	}
}

func (rs *RelayServer) handleRegister(conn *websocket.Conn, msg *common.Message) {
	rs.mutex.Lock()
	rs.connMutex.Lock()
	defer rs.mutex.Unlock()
	defer rs.connMutex.Unlock()

	if msg.ClientID != "" {
		// CLIENT REGISTRATION - validate user token
		username, valid := rs.validateUserToken(msg.Token)
		if !valid {
			rs.logger.Error("Client registration failed: invalid user token for client %s", msg.ClientID)

			// Send error response
			errorResponse := common.NewMessage(common.MsgTypeError)
			errorResponse.ClientID = msg.ClientID
			errorResponse.Error = "Invalid user token"
			rs.sendMessage(conn, errorResponse)

			// Close connection
			conn.Close()
			return
		}

		rs.logger.Info("Client %s authenticated as user: %s", msg.ClientID, username)

		client := &Client{
			ID:          msg.ClientID,
			Name:        msg.ClientName,
			Connection:  conn,
			ConnectedAt: time.Now(),
			LastPing:    time.Now(),
			AgentID:     msg.AgentID, // Target agent ID from -a parameter
			Status:      "connected",
		}
		rs.clients[msg.ClientID] = client

		// Add to fast lookup map
		rs.connToClient[conn] = msg.ClientID

		rs.logger.Info("Client registered: %s (name: %s, target agent: %s)", msg.ClientID, msg.ClientName, msg.AgentID)

		// Save client to database (permanent storage)
		go rs.saveClientToDatabase(msg.ClientID, msg.ClientName, msg.AgentID, "connected", msg.Token)

		// Log to database asynchronously
		go rs.logConnection("client", "", msg.ClientID, "connected", fmt.Sprintf("target_agent: %s, user: %s", msg.AgentID, username))
	} else if msg.AgentID != "" {
		// AGENT REGISTRATION - validate agent token
		if !rs.validateAgentToken(msg.AgentID, msg.Token) {
			rs.logger.Error("Agent registration failed: invalid token for agent %s", msg.AgentID)

			// Send error response
			errorResponse := common.NewMessage(common.MsgTypeError)
			errorResponse.AgentID = msg.AgentID
			errorResponse.Error = "Invalid agent token"
			rs.sendMessage(conn, errorResponse)

			// Close connection
			conn.Close()
			return
		}

		agent := &Agent{
			ID:          msg.AgentID,
			Connection:  conn,
			ConnectedAt: time.Now(),
			LastPing:    time.Now(),
			Status:      "connected",
		}
		rs.agents[msg.AgentID] = agent

		// Add to fast lookup map
		rs.connToAgent[conn] = msg.AgentID

		rs.logger.Info("Agent registered successfully: %s", msg.AgentID)

		// Log to database asynchronously
		go rs.logConnection("agent", msg.AgentID, "", "connected", "")
		// Save agent to database with "connected" status and ensure token is preserved
		go rs.saveAgentToDatabase(msg.AgentID, "connected")
	}

	// Send confirmation
	response := common.NewMessage(common.MsgTypeRegister)
	response.AgentID = msg.AgentID
	response.ClientID = msg.ClientID
	rs.sendMessage(conn, response)
}

func (rs *RelayServer) handleConnect(conn *websocket.Conn, msg *common.Message) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.logger.Info("Processing connect request - Client: %s, Agent: %s, Target: %s, SessionID: %s",
		msg.ClientID, msg.AgentID, msg.Target, msg.SessionID)

	// Use session ID from client if provided, otherwise generate new one
	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = common.GenerateID()
	}

	// Create new session
	session := &Session{
		ID:       sessionID,
		AgentID:  msg.AgentID,
		ClientID: msg.ClientID,
		Target:   msg.Target,
		Created:  time.Now(),
	}
	rs.sessions[session.ID] = session

	rs.logger.Info("New session created: %s (Agent: %s, Client: %s, Target: %s)",
		session.ID, session.AgentID, session.ClientID, session.Target)

	// Debug: List all available agents
	// rs.logger.Info("=== AVAILABLE AGENTS ===")
	if len(rs.agents) == 0 {
		// rs.logger.Info("❌ No agents registered!")
	} else {
		for _, _ = range rs.agents {
			// rs.logger.Info("✅ Agent: %s (Status: %s)", agentID, agent.Status)
		}
	}
	rs.logger.Info("Looking for agent: %s", msg.AgentID)

	// Forward connect message to agent
	if agentConn, exists := rs.agents[msg.AgentID]; exists {
		connectMsg := common.NewMessage(common.MsgTypeConnect)
		connectMsg.SessionID = session.ID
		connectMsg.ClientID = msg.ClientID
		connectMsg.Target = msg.Target
		rs.logger.Info("✅ Forwarding connect message to agent %s", msg.AgentID)
		rs.sendMessage(agentConn.Connection, connectMsg)
	} else {
		rs.logger.Error("❌ Agent not found: %s", msg.AgentID)
		errorMsg := common.NewMessage(common.MsgTypeError)
		errorMsg.SessionID = session.ID
		errorMsg.Error = "Agent not available"
		rs.sendMessage(conn, errorMsg)
	}
}

func (rs *RelayServer) handleData(conn *websocket.Conn, msg *common.Message) {
	// Check for special tunnel listening log message
	if len(msg.Data) > 0 && strings.HasPrefix(string(msg.Data), "tunnel_listening:") {
		rs.handleTunnelListeningLog(conn, msg)
		return
	}

	rs.mutex.RLock()
	session, exists := rs.sessions[msg.SessionID]
	rs.mutex.RUnlock()

	if !exists {
		rs.logger.Error("Session not found: %s", msg.SessionID)
		return
	}

	// Fast connection lookup using O(1) maps
	rs.connMutex.RLock()
	var senderType, senderID string
	if agentID, isAgent := rs.connToAgent[conn]; isAgent {
		senderType = "AGENT"
		senderID = agentID
	} else if clientID, isClient := rs.connToClient[conn]; isClient {
		senderType = "CLIENT"
		senderID = clientID
	}
	rs.connMutex.RUnlock()

	// Minimal logging - only for debug mode or errors
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		rs.logger.Debug("Data forwarding: Session=%s, Sender=%s(%s), Size=%d",
			msg.SessionID, senderType, senderID, len(msg.Data))
	}

	var targetConn *websocket.Conn
	rs.mutex.RLock()
	if senderType == "CLIENT" {
		// Data from client to agent
		if agent, exists := rs.agents[session.AgentID]; exists {
			targetConn = agent.Connection
		}
	} else if senderType == "AGENT" {
		// Data from agent to client
		if client, exists := rs.clients[session.ClientID]; exists {
			targetConn = client.Connection
		}
	}
	rs.mutex.RUnlock()

	if targetConn != nil {
		// Set appropriate message IDs for routing
		if senderType == "CLIENT" {
			msg.ClientID = senderID
		} else if senderType == "AGENT" {
			msg.AgentID = senderID
		}

		// Forward data immediately without blocking
		rs.sendMessage(targetConn, msg)

		// Only log SSH commands, not raw data (performance optimization)
		if rs.isSSHCommand(msg.Data) && session.Protocol == "ssh" {
			command := rs.extractSSHCommand(msg.Data)
			if command != "" {
				direction := "client_to_agent"
				if senderType == "AGENT" {
					direction = "agent_to_client"
				}

				rs.batchLogSSH(SSHLogEntry{
					SessionID: msg.SessionID,
					AgentID:   session.AgentID,
					ClientID:  session.ClientID,
					Direction: direction,
					Command:   command,
					DataSize:  len(msg.Data),
					Timestamp: time.Now(),
				})
			}
		}
	} else {
		rs.logger.Error("Target connection not found for session: %s", msg.SessionID)
	}
}

func (rs *RelayServer) handleTunnelListeningLog(conn *websocket.Conn, msg *common.Message) {
	// Extract tunnel listening information
	logData := string(msg.Data)
	if strings.HasPrefix(logData, "tunnel_listening:") {
		tunnelInfo := strings.TrimPrefix(logData, "tunnel_listening:")

		// Get client ID from connection
		rs.connMutex.RLock()
		_, isClient := rs.connToClient[conn]
		rs.connMutex.RUnlock()

		if isClient && msg.ClientID != "" {
			// Log tunnel listening event to database
			details := fmt.Sprintf("tunnel_listening: %s", tunnelInfo)
			go rs.logConnection("client", msg.AgentID, msg.ClientID, "tunnel_listening", details)

			rs.logger.Info("Client %s tunnel listening: %s", msg.ClientID, tunnelInfo)
		}
	}
}

func (rs *RelayServer) handleClose(conn *websocket.Conn, msg *common.Message) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	if session, exists := rs.sessions[msg.SessionID]; exists {
		delete(rs.sessions, msg.SessionID)
		rs.logger.Info("Session closed: %s", msg.SessionID)

		// Forward close message to the other end
		var targetConn *websocket.Conn
		if msg.ClientID != "" {
			if agent, exists := rs.agents[session.AgentID]; exists {
				targetConn = agent.Connection
			}
		} else if msg.AgentID != "" {
			if client, exists := rs.clients[session.ClientID]; exists {
				targetConn = client.Connection
			}
		}

		if targetConn != nil {
			rs.sendMessage(targetConn, msg)
		}
	}
}

func (rs *RelayServer) handleHeartbeat(conn *websocket.Conn, msg *common.Message) {
	// Send heartbeat response
	response := common.NewMessage(common.MsgTypeHeartbeat)
	response.AgentID = msg.AgentID
	response.ClientID = msg.ClientID
	rs.sendMessage(conn, response)
}

func (rs *RelayServer) handleDBQuery(conn *websocket.Conn, msg *common.Message) {
	// Debug logging
	// rs.logger.Info("=== RECEIVED DATABASE QUERY ===")
	// rs.logger.Info("SessionID: %s", msg.SessionID)
	// rs.logger.Info("AgentID: %s", msg.AgentID)
	// rs.logger.Info("ClientID: %s", msg.ClientID)
	// rs.logger.Info("Operation: %s", msg.DBOperation)
	// rs.logger.Info("Table: %s", msg.DBTable)
	// rs.logger.Info("Database: %s", msg.DBDatabase)
	// rs.logger.Info("Protocol: %s", msg.DBProtocol)
	rs.logger.Info("Query: %s", msg.DBQuery[:min(100, len(msg.DBQuery))])

	// Log database query to tunnel_logs table only
	rs.logTunnelQuery(msg.SessionID, msg.AgentID, msg.ClientID, "inbound", msg.DBProtocol, msg.DBOperation, msg.DBTable, msg.DBDatabase, msg.DBQuery)

	rs.logger.Info("Database query logged from client %s: %s %s.%s",
		msg.ClientID, msg.DBOperation, msg.DBDatabase, msg.DBTable)
}

// handleShellCommand forwards shell commands from client to agent
func (rs *RelayServer) handleShellCommand(conn *websocket.Conn, msg *common.Message) {
	// rs.logger.Info("=== SHELL COMMAND ROUTING ===")
	// rs.logger.Info("From Client: %s", msg.ClientID)
	// rs.logger.Info("To Agent: %s", msg.AgentID)
	// rs.logger.Info("Command: %s", msg.DBQuery)
	// rs.logger.Info("Session: %s", msg.SessionID)

	// List all available agents
	rs.mutex.RLock()
	// rs.logger.Info("Available agents:")
	for agentID, agent := range rs.agents {
		// rs.logger.Info("  - Agent ID: %s, Status: %s", agentID, agent.Status)
		_ = agentID
		_ = agent
	}
	agent, exists := rs.agents[msg.AgentID]
	rs.mutex.RUnlock()

	if !exists {
		rs.logger.Error("❌ Agent %s not found for shell command", msg.AgentID)
		// Send error back to client
		errorMsg := common.NewMessage("shell_error")
		errorMsg.SessionID = msg.SessionID
		errorMsg.ClientID = msg.ClientID
		errorMsg.AgentID = msg.AgentID
		errorMsg.Data = []byte(fmt.Sprintf("Agent %s not found", msg.AgentID))
		rs.sendMessage(conn, errorMsg)
		return
	}

	rs.logger.Info("✅ Found agent %s, forwarding command...", msg.AgentID)

	// Forward the command to agent
	rs.sendMessage(agent.Connection, msg)

	rs.logger.Info("✅ Command forwarded successfully to agent %s", msg.AgentID)

	// Log the shell command
	rs.logSSHCommand(msg.SessionID, msg.AgentID, msg.ClientID, "outbound", "root", "remote", "22", msg.DBQuery, len(msg.DBQuery))
}

// handleSSHLog processes SSH log messages from universal client
func (rs *RelayServer) handleSSHLog(conn *websocket.Conn, msg *common.Message) {
	rs.logger.Debug("Received SSH log message from client: %s", msg.ClientID)

	// Parse the SSH log request from Data field
	var logRequest map[string]interface{}
	if err := json.Unmarshal(msg.Data, &logRequest); err != nil {
		rs.logger.Error("Failed to parse SSH log data: %v", err)
		return
	}

	// Extract fields from the log request
	sessionID := getString(logRequest, "session_id")
	agentID := getString(logRequest, "agent_id")
	clientID := getString(logRequest, "client_id")
	direction := getString(logRequest, "direction")
	user := getString(logRequest, "user")
	host := getString(logRequest, "host")
	port := getString(logRequest, "port")
	command := getString(logRequest, "command")
	data := getString(logRequest, "data")
	isBase64 := getBool(logRequest, "is_base64")

	// Decode base64 data if needed
	actualData := data
	if isBase64 {
		if decodedBytes, err := base64.StdEncoding.DecodeString(data); err == nil {
			actualData = string(decodedBytes)
		} else {
			rs.logger.Error("Failed to decode base64 data: %v", err)
		}
	}

	// Create SSH log entry
	rs.batchLogSSH(SSHLogEntry{
		SessionID: sessionID,
		AgentID:   agentID,
		ClientID:  clientID,
		Direction: direction,
		Command:   command,
		User:      user,
		Host:      host,
		Port:      port,
		Data:      actualData,
		IsBase64:  isBase64,
		DataSize:  len(actualData),
		Timestamp: time.Now(),
	})

	rs.logger.Debug("SSH log processed: %s@%s - %s", user, host, direction)
}

// Helper function to safely get string from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Helper function to safely get bool from map
func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// handleShellResponse forwards shell command responses from agent to client
func (rs *RelayServer) handleShellResponse(conn *websocket.Conn, msg *common.Message) {
	// rs.logger.Info("=== SHELL RESPONSE ROUTING ===")
	// rs.logger.Info("From Agent: %s", msg.AgentID)
	// rs.logger.Info("To Client: %s", msg.ClientID)
	// rs.logger.Info("Response Data: %s", string(msg.Data))
	// rs.logger.Info("Session: %s", msg.SessionID)

	// Find the target client
	rs.mutex.RLock()
	client, exists := rs.clients[msg.ClientID]
	rs.mutex.RUnlock()

	if !exists {
		rs.logger.Error("❌ Client %s not found for shell response", msg.ClientID)
		return
	}

	rs.logger.Info("✅ Found client %s, forwarding response...", msg.ClientID)

	// Forward the response to client
	rs.sendMessage(client.Connection, msg)

	rs.logger.Info("✅ Response forwarded successfully to client %s", msg.ClientID)

	// Log the shell response
	rs.logSSHCommand(msg.SessionID, msg.AgentID, msg.ClientID, "inbound", "root", "remote", "22", msg.DBQuery, len(msg.Data))
}

// handleShellError forwards shell command errors from agent to client
func (rs *RelayServer) handleShellError(conn *websocket.Conn, msg *common.Message) {
	rs.logger.Debug("Forwarding shell error from agent %s to client %s", msg.AgentID, msg.ClientID)

	// Find the target client
	rs.mutex.RLock()
	client, exists := rs.clients[msg.ClientID]
	rs.mutex.RUnlock()

	if !exists {
		rs.logger.Error("Client %s not found for shell error", msg.ClientID)
		return
	}

	// Forward the error to client
	rs.sendMessage(client.Connection, msg)
}

func (rs *RelayServer) sendMessage(conn *websocket.Conn, msg *common.Message) {
	// For now, always use JSON to avoid binary frame corruption issues
	// TODO: Re-enable binary optimization after debugging

	data, err := msg.ToJSON()
	if err != nil {
		rs.logger.Error("Failed to serialize message: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		rs.logger.Error("Failed to send message: %v", err)
	}
}

func (rs *RelayServer) isSSHData(msg *common.Message) bool {
	// Check if this message is part of an SSH session
	rs.mutex.RLock()
	session, exists := rs.sessions[msg.SessionID]
	rs.mutex.RUnlock()

	if !exists {
		return false
	}

	// If target contains port 22 or session was created with SSH protocol, treat as SSH
	return strings.Contains(session.Target, ":22") ||
		strings.Contains(session.Target, ":2222")
}

func (rs *RelayServer) sendBinarySSHData(conn *websocket.Conn, msg *common.Message) {
	// Create binary frame for SSH data
	// Format: [TYPE:4][CLIENT_ID_LEN:1][CLIENT_ID][AGENT_ID_LEN:1][AGENT_ID][SESSION_ID_LEN:1][SESSION_ID][DATA]

	var clientID, agentID, sessionID string
	if msg.ClientID != "" {
		clientID = msg.ClientID
	}
	if msg.AgentID != "" {
		agentID = msg.AgentID
	}
	if msg.SessionID != "" {
		sessionID = msg.SessionID
	}

	// Build binary frame
	frame := make([]byte, 0, 4+1+len(clientID)+1+len(agentID)+1+len(sessionID)+len(msg.Data))

	// Type (4 bytes)
	frame = append(frame, []byte("DATA")...)

	// ClientID length and data
	frame = append(frame, byte(len(clientID)))
	frame = append(frame, []byte(clientID)...)

	// AgentID length and data
	frame = append(frame, byte(len(agentID)))
	frame = append(frame, []byte(agentID)...)

	// SessionID length and data
	frame = append(frame, byte(len(sessionID)))
	frame = append(frame, []byte(sessionID)...)

	// SSH data
	frame = append(frame, msg.Data...)

	rs.logger.Debug("Sending binary SSH data: ClientID=%s, AgentID=%s, SessionID=%s, DataLen=%d",
		clientID, agentID, sessionID, len(msg.Data))

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		rs.logger.Error("Failed to send binary SSH data: %v", err)
	}
}

func (rs *RelayServer) cleanupConnection(conn *websocket.Conn) {
	rs.mutex.Lock()
	rs.connMutex.Lock()
	defer rs.mutex.Unlock()
	defer rs.connMutex.Unlock()

	var disconnectedAgentID, disconnectedClientID string

	// Remove from agents using fast lookup
	if agentID, exists := rs.connToAgent[conn]; exists {
		if _, agentExists := rs.agents[agentID]; agentExists {
			delete(rs.agents, agentID)
			delete(rs.connToAgent, conn)
			disconnectedAgentID = agentID
			rs.logger.Info("Agent disconnected: %s", agentID)
			// Log asynchronously for performance
			go rs.logConnection("agent", agentID, "", "disconnected", "")
			// Update agent status in database to "disconnected"
			go rs.saveAgentToDatabase(agentID, "disconnected")
		}
	}

	// Remove from clients using fast lookup
	if clientID, exists := rs.connToClient[conn]; exists {
		if _, clientExists := rs.clients[clientID]; clientExists {
			delete(rs.clients, clientID)
			delete(rs.connToClient, conn)
			disconnectedClientID = clientID
			rs.logger.Info("Client disconnected: %s", clientID)
			// Log asynchronously for performance
			go rs.logConnection("client", "", clientID, "disconnected", "")
			// Update only client status in database to "disconnected", preserve username
			go rs.updateClientStatus(clientID, "disconnected")
		}
	}

	// Clean up sessions efficiently
	if disconnectedAgentID != "" || disconnectedClientID != "" {
		for sessionID, session := range rs.sessions {
			if session.AgentID == disconnectedAgentID || session.ClientID == disconnectedClientID {
				delete(rs.sessions, sessionID)
				rs.logger.Info("Session cleaned up: %s", sessionID)
			}
		}
	}
}

func (rs *RelayServer) start(port int) {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		rs.logger.Info("Received shutdown signal, cleaning up...")
		if rs.db != nil {
			rs.db.Close()
		}
		rs.logger.Close()
		os.Exit(0)
	}()

	// Setup routes
	rs.setupRoutes()

	// Start HTTP server
	rs.logger.Info("Starting relay server on port %d", port)
	rs.logger.Info("WebSocket endpoint: ws://localhost:%d/ws/agent or ws://localhost:%d/ws/client", port, port)
	rs.logger.Info("Web dashboard: http://localhost:%d", port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func (rs *RelayServer) setupRoutes() {
	// WebSocket endpoints
	http.HandleFunc("/ws/agent", rs.corsMiddleware(rs.handleWebSocket))
	http.HandleFunc("/ws/client", rs.corsMiddleware(rs.handleWebSocket))

	// Web dashboard routes
	http.HandleFunc("/", rs.corsMiddleware(rs.requireAuth(rs.handleDashboard)))
	http.HandleFunc("/login", rs.corsMiddleware(rs.handleLogin))
	http.HandleFunc("/logout", rs.corsMiddleware(rs.handleLogout))

	// API endpoints
	http.HandleFunc("/api/agents", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPIAgents)))
	http.HandleFunc("/api/agents/", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPIAgents))) // Handle /api/agents/{id}
	http.HandleFunc("/api/clients", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPIClients)))
	http.HandleFunc("/api/clients/", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPIClients))) // Handle /api/clients/{id}
	http.HandleFunc("/api/projects", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPIProjects)))
	http.HandleFunc("/api/projects/", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPIProjects))) // Handle /api/projects/{id}
	http.HandleFunc("/api/logs", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPILogs)))
	http.HandleFunc("/api/tunnel-logs", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPITunnelLogs)))
	http.HandleFunc("/api/ssh-logs", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPISSHLogs)))
	http.HandleFunc("/api/settings", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPISettings)))
	http.HandleFunc("/api/log-query", rs.corsMiddleware(rs.handleAPILogQuery))
	http.HandleFunc("/api/log-ssh", rs.corsMiddleware(rs.handleAPILogSSH))

	// Health endpoint
	http.HandleFunc("/health", rs.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"agents":    len(rs.agents),
			"clients":   len(rs.clients),
			"sessions":  len(rs.sessions),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

// API Authentication Middleware (supports Basic Auth)
func (rs *RelayServer) requireAPIAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// rs.logger.Info("=== API AUTH CHECK ===")
		// rs.logger.Info("Method: %s, URL: %s", r.Method, r.URL.Path)
		// rs.logger.Info("Authorization header present: %t", r.Header.Get("Authorization") != "")

		// Check for Basic Auth header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Basic ") {
			// Parse Basic Auth
			payload := strings.TrimPrefix(authHeader, "Basic ")
			data, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				http.Error(w, "Invalid authentication", http.StatusUnauthorized)
				return
			}

			credentials := strings.SplitN(string(data), ":", 2)
			if len(credentials) != 2 {
				http.Error(w, "Invalid authentication", http.StatusUnauthorized)
				return
			}

			username, password := credentials[0], credentials[1]

			// Validate credentials against database
			var storedPassword, role string
			err = rs.db.QueryRow("SELECT password, role FROM users WHERE username = ?", username).Scan(&storedPassword, &role)
			if err != nil {
				http.Error(w, "Invalid authentication", http.StatusUnauthorized)
				return
			}

			if storedPassword != password {
				http.Error(w, "Invalid authentication", http.StatusUnauthorized)
				return
			}

			// Set user info in headers for handler
			r.Header.Set("X-User-Role", role)
			r.Header.Set("X-Username", username)

			handler(w, r)
			return
		}

		// Fallback to cookie-based auth (for web interface)
		cookie, err := r.Cookie("tunnel-session")
		if err != nil || cookie.Value == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		session, exists := rs.webSessions[cookie.Value]
		if !exists {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		// Store user info in request context for later use
		r.Header.Set("X-User-Role", session.Role)
		r.Header.Set("X-Username", session.Username)

		handler(w, r)
	}
}

// Web Authentication Middleware
func (rs *RelayServer) requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple session check using cookies
		cookie, err := r.Cookie("tunnel-session")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		session, exists := rs.webSessions[cookie.Value]
		if !exists {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Store user info in request context for later use
		r.Header.Set("X-User-Role", session.Role)
		r.Header.Set("X-Username", session.Username)

		handler(w, r)
	}
}

// Require admin role for certain operations
func (rs *RelayServer) requireAdmin(handler http.HandlerFunc) http.HandlerFunc {
	return rs.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}
		handler(w, r)
	})
}

// Check if user has access to a project
func (rs *RelayServer) hasProjectAccess(username string, projectId int) bool {
	// Admin users have access to all projects
	var userRole string
	err := rs.db.QueryRow("SELECT role FROM users WHERE username = ?", username).Scan(&userRole)
	if err == nil && userRole == "admin" {
		return true
	}

	// Check user project assignments
	var count int
	err = rs.db.QueryRow(`
		SELECT COUNT(*) FROM user_project_assignments upa 
		JOIN users u ON upa.user_id = u.id 
		WHERE u.username = ? AND upa.project_id = ? AND upa.status = 'active'`,
		username, projectId).Scan(&count)

	return err == nil && count > 0
}

// Get user's accessible projects
func (rs *RelayServer) getUserProjects(username string) ([]int, error) {
	// Admin users can access all projects
	var userRole string
	err := rs.db.QueryRow("SELECT role FROM users WHERE username = ?", username).Scan(&userRole)
	if err == nil && userRole == "admin" {
		var projects []int
		rows, err := rs.db.Query("SELECT id FROM projects WHERE status = 'active'")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var projectId int
			if rows.Scan(&projectId) == nil {
				projects = append(projects, projectId)
			}
		}
		return projects, nil
	}

	// Regular users - get assigned projects only
	var projects []int
	rows, err := rs.db.Query(`
		SELECT DISTINCT upa.project_id FROM user_project_assignments upa 
		JOIN users u ON upa.user_id = u.id 
		WHERE u.username = ? AND upa.status = 'active'`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectId int
		if rows.Scan(&projectId) == nil {
			projects = append(projects, projectId)
		}
	}
	return projects, nil
}

// Filter agents based on user's project access
func (rs *RelayServer) filterAgentsByProject(username string, agents []map[string]interface{}) []map[string]interface{} {
	userProjects, err := rs.getUserProjects(username)
	if err != nil {
		rs.logger.Error("Failed to get user projects: %v", err)
		return []map[string]interface{}{}
	}

	// Convert to map for quick lookup
	projectMap := make(map[int]bool)
	for _, pid := range userProjects {
		projectMap[pid] = true
	}

	var filteredAgents []map[string]interface{}
	for _, agent := range agents {
		// If agent has no project_id or user has access to the project
		if projectId, ok := agent["project_id"]; !ok || projectId == nil || projectMap[projectId.(int)] {
			filteredAgents = append(filteredAgents, agent)
		}
	}

	return filteredAgents
}

// Login Handler
func (rs *RelayServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var username, password string

		// Check if request is JSON or form data
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			// Handle JSON request from Vue.js frontend
			var loginReq struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}

			if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			username = loginReq.Username
			password = loginReq.Password
		} else {
			// Handle form data from HTML form
			username = r.FormValue("username")
			password = r.FormValue("password")
		}

		var dbPassword, dbRole string
		err := rs.db.QueryRow("SELECT password, role FROM users WHERE username = ?", username).Scan(&dbPassword, &dbRole)
		if err != nil || dbPassword != password {
			if strings.Contains(contentType, "application/json") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			} else {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			}
			return
		}

		// Create session
		sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
		rs.webSessions[sessionID] = &WebSession{
			Username:  username,
			Role:      dbRole,
			LoginTime: time.Now(),
		}

		if strings.Contains(contentType, "application/json") {
			// Return JSON response for API
			token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"token":   token,
				"user": map[string]string{
					"username": username,
					"role":     dbRole,
				},
				"session_id": sessionID,
			})
		} else {
			// Set cookie and redirect for HTML form
			cookie := &http.Cookie{
				Name:     "tunnel-session",
				Value:    sessionID,
				Path:     "/",
				MaxAge:   86400, // 24 hours
				HttpOnly: true,
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
		return
	}

	// Serve login page
	loginHTML := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>SSH Tunnel - Login</title>
        <style>
            body { font-family: Arial, sans-serif; max-width: 400px; margin: 100px auto; padding: 20px; }
            .form-group { margin-bottom: 15px; }
            label { display: block; margin-bottom: 5px; }
            input[type="text"], input[type="password"] { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
            button { width: 100%; padding: 10px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; }
            button:hover { background: #0056b3; }
        </style>
    </head>
    <body>
        <h2>SSH Tunnel Dashboard</h2>
        <form method="post">
            <div class="form-group">
                <label>Username:</label>
                <input type="text" name="username" required>
            </div>
            <div class="form-group">
                <label>Password:</label>
                <input type="password" name="password" required>
            </div>
            <button type="submit">Login</button>
        </form>
        <div style="margin-top: 20px; padding: 15px; background: #e9ecef; border-radius: 4px;">
            <h4>Available Accounts:</h4>
            <p><strong>Admin:</strong> admin / admin123 (full access)</p>
            <p><strong>User:</strong> user / user123 (read-only access)</p>
        </div>
    </body>
    </html>
    `
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(loginHTML))
}

// Logout Handler
func (rs *RelayServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("tunnel-session")
	if err == nil {
		delete(rs.webSessions, cookie.Value)
		// Clear cookie
		cookie := &http.Cookie{
			Name:     "tunnel-session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Dashboard Handler
func (rs *RelayServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Redirect to Vue.js frontend
	http.Redirect(w, r, "http://localhost:3000", http.StatusTemporaryRedirect)
}

// API Handlers
func (rs *RelayServer) handleAPIAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rs.handleGetAgents(w, r)
	case "POST":
		rs.handleAddAgent(w, r)
	case "DELETE":
		rs.handleDeleteAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rs *RelayServer) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	// Get username from request headers (set by auth middleware)
	username := r.Header.Get("X-Username")

	// Get agents from database with project info
	rows, err := rs.db.Query(`
		SELECT a.agent_id, a.agent_name, a.project_id, a.token, a.status, a.connected_at, a.last_ping,
		       p.project_name
		FROM agents a 
		LEFT JOIN projects p ON a.project_id = p.id
		ORDER BY a.connected_at DESC
	`)
	if err != nil {
		rs.logger.Error("Failed to query agents: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var agents []map[string]interface{}
	for rows.Next() {
		var agentID, agentName, token, status sql.NullString
		var projectID sql.NullInt64
		var projectName sql.NullString
		var connectedAt, lastPing sql.NullTime

		err := rows.Scan(&agentID, &agentName, &projectID, &token, &status, &connectedAt, &lastPing, &projectName)
		if err != nil {
			rs.logger.Error("Failed to scan agent row: %v", err)
			continue
		}

		agent := map[string]interface{}{
			"id":           agentID.String,
			"agent_id":     agentID.String,
			"agent_name":   agentName.String,
			"token":        token.String,
			"status":       status.String,
			"connected_at": connectedAt.Time,
			"last_ping":    lastPing.Time,
		}

		// Add project info if exists
		if projectID.Valid {
			agent["project_id"] = int(projectID.Int64)
			agent["project_name"] = projectName.String
		}

		agents = append(agents, agent)
	}

	// Filter agents based on user's project access
	filteredAgents := rs.filterAgentsByProject(username, agents)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredAgents)
}

func (rs *RelayServer) handleAddAgent(w http.ResponseWriter, r *http.Request) {
	var agentData struct {
		AgentID string `json:"agent_id"`
		Token   string `json:"token"`
		Status  string `json:"status"`
	}

	// Parse JSON body
	err := json.NewDecoder(r.Body).Decode(&agentData)
	if err != nil {
		rs.logger.Error("Failed to parse agent data: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if agentData.AgentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}
	if agentData.Token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}

	// Set default status if not provided
	if agentData.Status == "" {
		agentData.Status = "disconnected"
	}

	// Clean data
	agentID := rs.cleanString(agentData.AgentID)
	token := rs.cleanString(agentData.Token)
	status := rs.cleanString(agentData.Status)

	rs.logger.Info("Adding new agent: %s", agentID)

	// Check if agent already exists
	var existingCount int
	err = rs.db.QueryRow("SELECT COUNT(*) FROM agents WHERE agent_id = ?", agentID).Scan(&existingCount)
	if err != nil {
		rs.logger.Error("Failed to check existing agent: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if existingCount > 0 {
		http.Error(w, "Agent already exists", http.StatusConflict)
		return
	}

	// Insert new agent
	_, err = rs.db.Exec(`
		INSERT INTO agents (agent_id, token, status, connected_at, last_ping) 
		VALUES (?, ?, ?, NOW(), NOW())`,
		agentID, token, status,
	)
	if err != nil {
		rs.logger.Error("Failed to insert agent: %v", err)
		http.Error(w, "Failed to add agent", http.StatusInternalServerError)
		return
	}

	rs.logger.Info("Agent added successfully: %s", agentID)

	// Return success response
	response := map[string]interface{}{
		"success":  true,
		"message":  "Agent added successfully",
		"agent_id": agentID,
		"status":   status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rs *RelayServer) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	// rs.logger.Info("=== DELETE AGENT REQUEST ===")
	// rs.logger.Info("Method: %s", r.Method)
	// rs.logger.Info("URL Path: %s", r.URL.Path)

	// Extract agent ID from URL path
	urlPath := r.URL.Path
	parts := strings.Split(urlPath, "/")
	rs.logger.Info("URL parts: %v", parts)

	if len(parts) < 4 {
		rs.logger.Error("Not enough URL parts. Expected: /api/agents/{agentID}")
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	agentID := rs.cleanString(parts[3]) // /api/agents/{agentID}
	rs.logger.Info("Extracted Agent ID: '%s'", agentID)

	if agentID == "" {
		rs.logger.Error("Empty agent ID after cleaning")
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	rs.logger.Info("Deleting agent: %s", agentID)

	// Check if agent exists
	var existingCount int
	err := rs.db.QueryRow("SELECT COUNT(*) FROM agents WHERE agent_id = ?", agentID).Scan(&existingCount)
	if err != nil {
		rs.logger.Error("Failed to check existing agent: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if existingCount == 0 {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Delete agent from database
	result, err := rs.db.Exec("DELETE FROM agents WHERE agent_id = ?", agentID)
	if err != nil {
		rs.logger.Error("Failed to delete agent: %v", err)
		http.Error(w, "Failed to delete agent", http.StatusInternalServerError)
		return
	}

	// Check how many rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		rs.logger.Error("Failed to get rows affected: %v", err)
	} else {
		rs.logger.Info("Rows affected by delete: %d", rowsAffected)
		if rowsAffected == 0 {
			rs.logger.Info("WARNING: No rows were deleted for agent: %s", agentID)
		}
	}

	// Also remove from memory if exists
	rs.mutex.Lock()
	delete(rs.agents, agentID)
	rs.mutex.Unlock()

	rs.logger.Info("Agent deleted successfully: %s", agentID)

	// Return success response
	response := map[string]interface{}{
		"success":  true,
		"message":  "Agent deleted successfully",
		"agent_id": agentID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rs *RelayServer) handleAPIClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rs.handleGetClients(w, r)
	case "POST":
		rs.handleAddClient(w, r)
	case "DELETE":
		rs.handleDeleteClient(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rs *RelayServer) handleGetClients(w http.ResponseWriter, r *http.Request) {
	// Get clients from database including agent_id
	rows, err := rs.db.Query(`
		SELECT client_id, client_name, agent_id, username, token, status, connected_at, last_ping 
		FROM clients 
		ORDER BY connected_at DESC
	`)
	if err != nil {
		rs.logger.Error("Failed to query clients: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []map[string]interface{}
	for rows.Next() {
		var clientID, clientName, agentID, username, token, status sql.NullString
		var connectedAt, lastPing sql.NullTime

		err := rows.Scan(&clientID, &clientName, &agentID, &username, &token, &status, &connectedAt, &lastPing)
		if err != nil {
			rs.logger.Error("Failed to scan client row: %v", err)
			continue
		}

		// Use username as name if available, otherwise fallback to client_name
		displayName := username.String
		if displayName == "" {
			displayName = clientName.String
		}

		client := map[string]interface{}{
			"id":           clientID.String,
			"name":         displayName,
			"agent_id":     agentID.String,
			"username":     username.String,
			"token":        token.String,
			"status":       status.String,
			"connected_at": connectedAt.Time,
			"last_ping":    lastPing.Time,
		}
		clients = append(clients, client)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func (rs *RelayServer) handleAddClient(w http.ResponseWriter, r *http.Request) {
	var clientData struct {
		ClientID   string `json:"client_id"`
		ClientName string `json:"client_name"`
		Token      string `json:"token"`
		Status     string `json:"status"`
	}

	// Parse JSON body
	err := json.NewDecoder(r.Body).Decode(&clientData)
	if err != nil {
		rs.logger.Error("Failed to parse client data: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if clientData.ClientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}
	if clientData.ClientName == "" {
		http.Error(w, "client_name is required", http.StatusBadRequest)
		return
	}
	if clientData.Token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}

	// Set default status if not provided
	if clientData.Status == "" {
		clientData.Status = "disconnected"
	}

	// Clean data
	clientID := rs.cleanString(clientData.ClientID)
	clientName := rs.cleanString(clientData.ClientName)
	token := rs.cleanString(clientData.Token)
	status := rs.cleanString(clientData.Status)

	rs.logger.Info("Adding new client: %s", clientID)

	// Check if client already exists
	var existingCount int
	err = rs.db.QueryRow("SELECT COUNT(*) FROM clients WHERE client_id = ?", clientID).Scan(&existingCount)
	if err != nil {
		rs.logger.Error("Failed to check existing client: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if existingCount > 0 {
		http.Error(w, "Client already exists", http.StatusConflict)
		return
	}

	// Insert new client (without agent_id - will be handled by assignments)
	_, err = rs.db.Exec(`
		INSERT INTO clients (client_id, client_name, token, status, connected_at, last_ping) 
		VALUES (?, ?, ?, ?, NOW(), NOW())`,
		clientID, clientName, token, status,
	)
	if err != nil {
		rs.logger.Error("Failed to insert client: %v", err)
		http.Error(w, "Failed to add client", http.StatusInternalServerError)
		return
	}

	rs.logger.Info("Successfully added client: %s", clientID)

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Client added successfully",
		"client": map[string]string{
			"client_id":   clientID,
			"client_name": clientName,
			"status":      status,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rs *RelayServer) handleDeleteClient(w http.ResponseWriter, r *http.Request) {
	// rs.logger.Info("=== DELETE CLIENT REQUEST ===")
	// rs.logger.Info("Method: %s", r.Method)
	// rs.logger.Info("URL Path: %s", r.URL.Path)

	// Extract client ID from URL path
	urlPath := r.URL.Path
	parts := strings.Split(urlPath, "/")
	rs.logger.Info("URL parts: %v", parts)

	if len(parts) < 4 {
		rs.logger.Error("Not enough URL parts. Expected: /api/clients/{clientID}")
		http.Error(w, "Client ID is required", http.StatusBadRequest)
		return
	}

	clientID := rs.cleanString(parts[3]) // /api/clients/{clientID}
	rs.logger.Info("Extracted Client ID: '%s'", clientID)

	if clientID == "" {
		rs.logger.Error("Empty client ID after cleaning")
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	rs.logger.Info("Deleting client: %s", clientID)

	// Delete from database
	result, err := rs.db.Exec("DELETE FROM clients WHERE client_id = ?", clientID)
	if err != nil {
		rs.logger.Error("Failed to delete client: %v", err)
		http.Error(w, "Failed to delete client", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		rs.logger.Error("Failed to get rows affected: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	rs.logger.Info("Successfully deleted client: %s", clientID)

	response := map[string]interface{}{
		"success": true,
		"message": "Client deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Project Management Handlers
func (rs *RelayServer) handleAPIProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rs.handleGetProjects(w, r)
	case "POST":
		rs.handleAddProject(w, r)
	case "PUT":
		rs.handleUpdateProject(w, r)
	case "DELETE":
		rs.handleDeleteProject(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rs *RelayServer) handleGetProjects(w http.ResponseWriter, r *http.Request) {
	// Get username from request headers (set by auth middleware)
	username := r.Header.Get("X-Username")

	// Get user's accessible projects
	userProjects, err := rs.getUserProjects(username)
	if err != nil {
		rs.logger.Error("Failed to get user projects: %v", err)
		http.Error(w, "Failed to get projects", http.StatusInternalServerError)
		return
	}

	if len(userProjects) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
		return
	}

	// Convert project IDs to comma-separated string for SQL IN clause
	projectIDStrs := make([]string, len(userProjects))
	for i, id := range userProjects {
		projectIDStrs[i] = fmt.Sprintf("%d", id)
	}
	projectIDsStr := strings.Join(projectIDStrs, ",")

	// Get project details
	query := fmt.Sprintf(`
		SELECT id, project_name, description, created_by, status, created_at, updated_at
		FROM projects 
		WHERE id IN (%s)
		ORDER BY created_at DESC
	`, projectIDsStr)

	rows, err := rs.db.Query(query)
	if err != nil {
		rs.logger.Error("Failed to query projects: %v", err)
		http.Error(w, "Failed to get projects", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projects []map[string]interface{}
	for rows.Next() {
		var id int
		var projectName, description, createdBy, status sql.NullString
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(&id, &projectName, &description, &createdBy, &status, &createdAt, &updatedAt)
		if err != nil {
			rs.logger.Error("Failed to scan project row: %v", err)
			continue
		}

		project := map[string]interface{}{
			"id":           id,
			"project_name": projectName.String,
			"description":  description.String,
			"created_by":   createdBy.String,
			"status":       status.String,
			"created_at":   createdAt.Time,
			"updated_at":   updatedAt.Time,
		}

		projects = append(projects, project)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func (rs *RelayServer) handleAddProject(w http.ResponseWriter, r *http.Request) {
	// Only admin can create projects
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	var projectData struct {
		ProjectName string `json:"project_name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	// Parse JSON body
	err := json.NewDecoder(r.Body).Decode(&projectData)
	if err != nil {
		rs.logger.Error("Failed to parse project data: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if projectData.ProjectName == "" {
		http.Error(w, "project_name is required", http.StatusBadRequest)
		return
	}

	// Set default status if not provided
	if projectData.Status == "" {
		projectData.Status = "active"
	}

	// Get username
	username := r.Header.Get("X-Username")

	// Insert new project
	_, err = rs.db.Exec(`
		INSERT INTO projects (project_name, description, created_by, status) 
		VALUES (?, ?, ?, ?)`,
		projectData.ProjectName, projectData.Description, username, projectData.Status,
	)
	if err != nil {
		rs.logger.Error("Failed to insert project: %v", err)
		http.Error(w, "Failed to add project", http.StatusInternalServerError)
		return
	}

	rs.logger.Info("Successfully added project: %s", projectData.ProjectName)

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Project added successfully",
		"project": map[string]string{
			"project_name": projectData.ProjectName,
			"description":  projectData.Description,
			"status":       projectData.Status,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rs *RelayServer) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	// Only admin can update projects
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	// TODO: Implementation for updating projects
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (rs *RelayServer) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	// Only admin can delete projects
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	// TODO: Implementation for deleting projects
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (rs *RelayServer) handleAPILogs(w http.ResponseWriter, r *http.Request) {
	rows, err := rs.db.Query("SELECT type, agent_id, client_id, event, timestamp, details FROM connection_logs ORDER BY timestamp DESC LIMIT 50")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var logType, agentID, clientID, event, details sql.NullString
		var timestamp time.Time

		err := rows.Scan(&logType, &agentID, &clientID, &event, &timestamp, &details)
		if err != nil {
			continue
		}

		log := map[string]interface{}{
			"type":      logType.String,
			"agent_id":  agentID.String,
			"client_id": clientID.String,
			"event":     event.String,
			"timestamp": timestamp,
			"details":   details.String,
		}
		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// Clean HTML entities from text
func cleanHTMLEntities(text string) string {
	return html.UnescapeString(text)
}

func (rs *RelayServer) handleAPITunnelLogs(w http.ResponseWriter, r *http.Request) {
	rows, err := rs.db.Query("SELECT session_id, agent_id, client_id, direction, protocol, operation, table_name, database_name, query_text, timestamp FROM tunnel_logs ORDER BY timestamp DESC LIMIT 100")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var sessionID, agentID, clientID, direction, protocol, operation, tableName, databaseName, queryText sql.NullString
		var timestamp time.Time

		err := rows.Scan(&sessionID, &agentID, &clientID, &direction, &protocol, &operation, &tableName, &databaseName, &queryText, &timestamp)
		if err != nil {
			continue
		}

		// Clean HTML entities and trim whitespace from query text
		cleanedQueryText := rs.cleanString(cleanHTMLEntities(queryText.String))

		log := map[string]interface{}{
			"session_id":    rs.cleanString(sessionID.String),
			"agent_id":      rs.cleanString(agentID.String),
			"client_id":     rs.cleanString(clientID.String),
			"direction":     rs.cleanString(direction.String),
			"protocol":      rs.cleanString(protocol.String),
			"operation":     rs.cleanOperation(operation.String),
			"table_name":    rs.cleanString(tableName.String),
			"database_name": rs.cleanString(databaseName.String),
			"query_text":    cleanedQueryText,
			"timestamp":     timestamp,
		}
		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// Handle API for logging database queries from clients
func (rs *RelayServer) handleAPILogQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SessionID == "" || req.ClientID == "" {
		http.Error(w, "Missing required fields: session_id, client_id", http.StatusBadRequest)
		return
	}

	// Log the query
	rs.logTunnelQuery(req.SessionID, req.AgentID, req.ClientID, req.Direction, req.Protocol, req.Operation, req.TableName, req.DatabaseName, req.QueryText)

	// Return success response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Query logged successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handle API for SSH logs retrieval
func (rs *RelayServer) handleAPISSHLogs(w http.ResponseWriter, r *http.Request) {
	rows, err := rs.db.Query("SELECT session_id, agent_id, client_id, direction, ssh_user, ssh_host, ssh_port, command, data, is_base64, data_size, timestamp FROM ssh_logs ORDER BY timestamp DESC LIMIT 100")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var sessionID, agentID, clientID, direction, sshUser, sshHost, sshPort, command, data sql.NullString
		var isBase64 sql.NullBool
		var dataSize sql.NullInt64
		var timestamp time.Time

		err := rows.Scan(&sessionID, &agentID, &clientID, &direction, &sshUser, &sshHost, &sshPort, &command, &data, &isBase64, &dataSize, &timestamp)
		if err != nil {
			continue
		}

		// Clean and format SSH command
		cleanedCommand := rs.cleanString(cleanHTMLEntities(command.String))
		
		// Handle data decoding if needed for display
		actualData := data.String
		if isBase64.Bool && actualData != "" {
			if decodedBytes, err := base64.StdEncoding.DecodeString(actualData); err == nil {
				actualData = string(decodedBytes)
			}
		}

		log := map[string]interface{}{
			"session_id": rs.cleanString(sessionID.String),
			"agent_id":   rs.cleanString(agentID.String),
			"client_id":  rs.cleanString(clientID.String),
			"direction":  rs.cleanString(direction.String),
			"ssh_user":   rs.cleanString(sshUser.String),
			"ssh_host":   rs.cleanString(sshHost.String),
			"ssh_port":   rs.cleanString(sshPort.String),
			"command":    cleanedCommand,
			"data":       actualData,
			"data_size":  dataSize.Int64,
			"timestamp":  timestamp,
		}
		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// Handle API for logging SSH commands from clients
func (rs *RelayServer) handleAPILogSSH(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SSHTunnelLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SessionID == "" || req.ClientID == "" {
		http.Error(w, "Missing required fields: session_id, client_id", http.StatusBadRequest)
		return
	}

	// Calculate data size
	dataSize := len(req.Data)

	// Add to batch logging buffer instead of direct logging
	entry := SSHLogEntry{
		SessionID: req.SessionID,
		AgentID:   req.AgentID,
		ClientID:  req.ClientID,
		Direction: req.Direction,
		User:      req.User,
		Host:      req.Host,
		Port:      req.Port,
		Command:   req.Command,
		Data:      req.Data,
		IsBase64:  req.IsBase64,
		DataSize:  dataSize,
		Timestamp: time.Now(),
	}

	rs.logBuffer.mutex.Lock()
	rs.logBuffer.sshLogs = append(rs.logBuffer.sshLogs, entry)
	rs.logBuffer.mutex.Unlock()

	// Return success response
	response := map[string]interface{}{
		"status":  "success",
		"message": "SSH command logged successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CORS Middleware
func (rs *RelayServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow multiple origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8081",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8081",
			"http://192.168.1.115:8081",
			"http://192.168.1.115:3000",
			"http://168.231.119.242:3000",
		}

		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// If no specific origin matched, allow all for development
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// Batch logging functions for performance optimization
func (rs *RelayServer) periodicFlush() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		rs.flushSSHLogs()
		rs.flushQueryLogs()
	}
}

func (rs *RelayServer) batchLogSSH(entry SSHLogEntry) {
	rs.logBuffer.mutex.Lock()
	rs.logBuffer.sshLogs = append(rs.logBuffer.sshLogs, entry)

	// Flush every 100 entries or 5 seconds
	if len(rs.logBuffer.sshLogs) >= 100 ||
		time.Since(rs.logBuffer.lastFlush) > 5*time.Second {
		go rs.flushSSHLogs()
	}
	rs.logBuffer.mutex.Unlock()
}

func (rs *RelayServer) batchLogQuery(entry QueryLogEntry) {
	rs.logBuffer.mutex.Lock()
	rs.logBuffer.queryLogs = append(rs.logBuffer.queryLogs, entry)

	// Flush every 100 entries or 5 seconds
	if len(rs.logBuffer.queryLogs) >= 100 ||
		time.Since(rs.logBuffer.lastFlush) > 5*time.Second {
		go rs.flushQueryLogs()
	}
	rs.logBuffer.mutex.Unlock()
}

func (rs *RelayServer) flushSSHLogs() {
	rs.logBuffer.mutex.Lock()
	if len(rs.logBuffer.sshLogs) == 0 {
		rs.logBuffer.mutex.Unlock()
		return
	}

	logs := make([]SSHLogEntry, len(rs.logBuffer.sshLogs))
	copy(logs, rs.logBuffer.sshLogs)
	rs.logBuffer.sshLogs = rs.logBuffer.sshLogs[:0] // Clear slice but keep capacity
	rs.logBuffer.lastFlush = time.Now()
	rs.logBuffer.mutex.Unlock()

	// Batch insert
	if rs.db != nil {
		tx, err := rs.db.Begin()
		if err != nil {
			rs.logger.Error("Failed to begin transaction for SSH logs: %v", err)
			return
		}

		stmt, err := tx.Prepare("INSERT INTO ssh_logs (session_id, agent_id, client_id, direction, ssh_user, ssh_host, ssh_port, command, data, is_base64, data_size, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			tx.Rollback()
			rs.logger.Error("Failed to prepare SSH log statement: %v", err)
			return
		}

		for _, log := range logs {
			_, err := stmt.Exec(log.SessionID, log.AgentID, log.ClientID,
				log.Direction, log.User, log.Host, log.Port,
				log.Command, log.Data, log.IsBase64, log.DataSize, log.Timestamp)
			if err != nil {
				rs.logger.Error("Failed to insert SSH log: %v", err)
			}
		}

		stmt.Close()
		if err := tx.Commit(); err != nil {
			rs.logger.Error("Failed to commit SSH logs: %v", err)
		}
	}
}

func (rs *RelayServer) flushQueryLogs() {
	rs.logBuffer.mutex.Lock()
	if len(rs.logBuffer.queryLogs) == 0 {
		rs.logBuffer.mutex.Unlock()
		return
	}

	logs := make([]QueryLogEntry, len(rs.logBuffer.queryLogs))
	copy(logs, rs.logBuffer.queryLogs)
	rs.logBuffer.queryLogs = rs.logBuffer.queryLogs[:0] // Clear slice but keep capacity
	rs.logBuffer.lastFlush = time.Now()
	rs.logBuffer.mutex.Unlock()

	// Batch insert
	if rs.db != nil {
		tx, err := rs.db.Begin()
		if err != nil {
			rs.logger.Error("Failed to begin transaction for query logs: %v", err)
			return
		}

		stmt, err := tx.Prepare("INSERT INTO tunnel_logs (session_id, agent_id, client_id, direction, protocol, operation, table_name, query_text, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			tx.Rollback()
			rs.logger.Error("Failed to prepare query log statement: %v", err)
			return
		}

		for _, log := range logs {
			_, err := stmt.Exec(log.SessionID, log.AgentID, log.ClientID,
				log.Direction, log.Protocol, log.Operation,
				log.TableName, log.QueryText, log.Timestamp)
			if err != nil {
				rs.logger.Error("Failed to insert query log: %v", err)
			}
		}

		stmt.Close()
		if err := tx.Commit(); err != nil {
			rs.logger.Error("Failed to commit query logs: %v", err)
		}
	}
}

// Helper functions for SSH command detection
func (rs *RelayServer) isSSHCommand(data []byte) bool {
	// Quick check for SSH commands vs raw data
	if len(data) < 3 || len(data) > 1024 {
		return false
	}

	// Simple heuristic: contains printable command characters
	printable := 0
	for _, b := range data {
		if b >= 32 && b <= 126 {
			printable++
		}
	}

	return printable > len(data)/2
}

func (rs *RelayServer) extractSSHCommand(data []byte) string {
	// Extract meaningful commands from SSH data
	if len(data) > 512 {
		return string(data[:512]) // Truncate long commands
	}
	return string(data)
}

// detectServerIP tries to detect the server IP address
func (rs *RelayServer) detectServerIP() string {
	// Try to get IP from environment variable first
	if serverIP := os.Getenv("SERVER_IP"); serverIP != "" {
		return serverIP
	}

	// Try to get from DB_HOST if it's not localhost
	dbHost := os.Getenv("DB_HOST")
	if dbHost != "" && dbHost != "localhost" && dbHost != "127.0.0.1" {
		return dbHost
	}

	// Fallback to a reasonable default
	return "168.231.119.242"
}

// handleAPISettings handles GET and PUT requests for server settings
func (rs *RelayServer) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rs.handleGetSettings(w, r)
	case "PUT":
		rs.handleUpdateSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetSettings returns current server settings
func (rs *RelayServer) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	// rs.logger.Info("=== GET SETTINGS REQUEST ===")

	settings := make(map[string]interface{})

	// Get all settings from database
	rows, err := rs.db.Query("SELECT setting_key, setting_value FROM server_settings")
	if err != nil {
		rs.logger.Error("Failed to query settings: %v", err)
		http.Error(w, "Failed to retrieve settings", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	settingsCount := 0
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}

		settingsCount++
		rs.logger.Info("Found setting: %s = %s", key, value)

		// Convert port to number
		if key == "server_port" {
			if port, err := strconv.Atoi(value); err == nil {
				settings[key] = port
			} else {
				settings[key] = 8080 // default
			}
		} else {
			settings[key] = value
		}
	}

	rs.logger.Info("Total settings found: %d", settingsCount)

	// Add some metadata
	settings["lastUpdated"] = time.Now().Format("2006-01-02 15:04:05")

	// If no settings found, return defaults
	if settingsCount == 0 {
		rs.logger.Info("No settings found in database, using auto-detected defaults")
		settings["server_ip"] = rs.detectServerIP()
		settings["server_port"] = 8080
	}

	rs.logger.Info("Returning settings: %+v", settings)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    settings,
	})
}

// handleUpdateSettings updates server settings
func (rs *RelayServer) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var updateData struct {
		ServerIP   string `json:"serverIP"`
		ServerPort int    `json:"serverPort"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate data
	if updateData.ServerIP == "" {
		http.Error(w, "Server IP is required", http.StatusBadRequest)
		return
	}

	if updateData.ServerPort < 1 || updateData.ServerPort > 65535 {
		updateData.ServerPort = 8080 // default
	}

	// Update settings in database
	settings := map[string]string{
		"server_ip":   updateData.ServerIP,
		"server_port": strconv.Itoa(updateData.ServerPort),
	}

	for key, value := range settings {
		_, err := rs.db.Exec(`
			INSERT INTO server_settings (setting_key, setting_value) 
			VALUES (?, ?) 
			ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)
		`, key, value)

		if err != nil {
			rs.logger.Error("Failed to update setting %s: %v", key, err)
			http.Error(w, "Failed to update settings", http.StatusInternalServerError)
			return
		}
	}

	rs.logger.Info("Server settings updated: IP=%s, Port=%d", updateData.ServerIP, updateData.ServerPort)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Settings updated successfully",
		"data": map[string]interface{}{
			"serverIP":   updateData.ServerIP,
			"serverPort": updateData.ServerPort,
		},
	})
}

func main() {
	var port int

	var rootCmd = &cobra.Command{
		Use:   "tunnel-relay",
		Short: "SSH Tunnel Relay Server",
		Long:  "A relay server that facilitates secure SSH tunneling between agents and clients",
		Run: func(cmd *cobra.Command, args []string) {
			server := NewRelayServer()
			server.start(port)
		},
	}

	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
