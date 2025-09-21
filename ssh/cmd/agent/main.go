package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"ssh-tunnel/internal/common"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

// SSHLogRequest represents a request to log SSH command
type SSHLogRequest struct {
	SessionID string `json:"session_id"`
	ClientID  string `json:"client_id"`
	AgentID   string `json:"agent_id"`
	Direction string `json:"direction"`
	User      string `json:"user"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	Command   string `json:"command"`
	Data      string `json:"data"`
}

type Agent struct {
	id        string
	token     string
	relayURL  string
	conn      *websocket.Conn
	sessions  map[string]net.Conn
	dbLoggers map[string]*common.DatabaseQueryLogger
	targets   map[string]string // sessionID -> target
	clients   map[string]string // sessionID -> clientID
	mutex     sync.RWMutex
	logger    *common.Logger
	running   bool
	heartbeat *time.Ticker
}

func NewAgent(id, token, relayURL string) *Agent {
	return &Agent{
		id:        id,
		token:     token,
		relayURL:  relayURL,
		sessions:  make(map[string]net.Conn),
		dbLoggers: make(map[string]*common.DatabaseQueryLogger),
		targets:   make(map[string]string),
		clients:   make(map[string]string),
		logger:    common.NewLogger(fmt.Sprintf("AGENT-%s", id)),
	}
}

func (a *Agent) connect() error {
	_, err := url.Parse(a.relayURL)
	if err != nil {
		return fmt.Errorf("invalid relay URL: %v", err)
	}

	a.logger.Info("Connecting to relay server: %s", a.relayURL)

	conn, _, err := websocket.DefaultDialer.Dial(a.relayURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %v", err)
	}

	a.conn = conn
	a.running = true

	// Register with relay
	registerMsg := common.NewMessage(common.MsgTypeRegister)
	registerMsg.AgentID = a.id
	registerMsg.Token = a.token
	if err := a.sendMessage(registerMsg); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	a.logger.Info("Successfully connected and registered with relay")
	return nil
}

func (a *Agent) start() error {
	if err := a.connect(); err != nil {
		return err
	}

	// Start heartbeat
	a.heartbeat = time.NewTicker(30 * time.Second)
	go a.heartbeatLoop()

	// Start message handler
	go a.messageLoop()

	a.logger.Info("Agent started successfully")
	return nil
}

func (a *Agent) stop() {
	a.running = false

	if a.heartbeat != nil {
		a.heartbeat.Stop()
	}

	if a.conn != nil {
		a.conn.Close()
	}

	// Close all sessions
	a.mutex.Lock()
	for sessionID, conn := range a.sessions {
		conn.Close()
		delete(a.sessions, sessionID)
	}
	a.mutex.Unlock()

	a.logger.Info("Agent stopped")
}

func (a *Agent) heartbeatLoop() {
	for a.running {
		select {
		case <-a.heartbeat.C:
			heartbeatMsg := common.NewMessage(common.MsgTypeHeartbeat)
			heartbeatMsg.AgentID = a.id
			if err := a.sendMessage(heartbeatMsg); err != nil {
				a.logger.Error("Failed to send heartbeat: %v", err)
			}
		}
	}
}

func (a *Agent) messageLoop() {
	defer a.stop()

	for a.running {
		messageType, messageData, err := a.conn.ReadMessage()
		if err != nil {
			if a.running {
				a.logger.Error("Failed to read message: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			// Handle JSON messages
			message, err := common.FromJSON(messageData)
			if err != nil {
				a.logger.Error("Failed to parse JSON message: %v", err)
				continue
			}

			a.handleMessage(message)
		} else if messageType == websocket.BinaryMessage {
			// For now, ignore binary messages to avoid corruption
			a.logger.Debug("Ignoring binary message (%d bytes) - using JSON only mode", len(messageData))
		}
	}
}

func (a *Agent) handleBinarySSHData(messageData []byte) {
	// Parse binary SSH data frame
	// Format: [TYPE:4][CLIENT_ID_LEN:1][CLIENT_ID][AGENT_ID_LEN:1][AGENT_ID][SESSION_ID_LEN:1][SESSION_ID][DATA]

	if len(messageData) < 7 {
		a.logger.Error("Binary message too short: %d bytes", len(messageData))
		return
	}

	offset := 0

	// Check type (4 bytes)
	if string(messageData[0:4]) != "DATA" {
		a.logger.Error("Invalid binary message type: %s", string(messageData[0:4]))
		return
	}
	offset += 4

	// Read client ID
	clientIDLen := int(messageData[offset])
	offset++
	if offset+clientIDLen > len(messageData) {
		a.logger.Error("Invalid client ID length")
		return
	}
	clientID := string(messageData[offset : offset+clientIDLen])
	offset += clientIDLen

	// Read agent ID
	if offset >= len(messageData) {
		a.logger.Error("Missing agent ID")
		return
	}
	agentIDLen := int(messageData[offset])
	offset++
	if offset+agentIDLen > len(messageData) {
		a.logger.Error("Invalid agent ID length")
		return
	}
	agentID := string(messageData[offset : offset+agentIDLen])
	offset += agentIDLen

	// Read session ID
	if offset >= len(messageData) {
		a.logger.Error("Missing session ID")
		return
	}
	sessionIDLen := int(messageData[offset])
	offset++
	if offset+sessionIDLen > len(messageData) {
		a.logger.Error("Invalid session ID length")
		return
	}
	sessionID := string(messageData[offset : offset+sessionIDLen])
	offset += sessionIDLen

	// Extract SSH data
	sshData := messageData[offset:]

	a.logger.Debug("Binary SSH data: ClientID=%s, AgentID=%s, SessionID=%s, DataLen=%d",
		clientID, agentID, sessionID, len(sshData))

	// Create message structure for handling
	msg := &common.Message{
		Type:      common.MsgTypeData,
		ClientID:  clientID,
		AgentID:   agentID,
		SessionID: sessionID,
		Data:      sshData,
	}

	// Handle as regular data message
	a.handleData(msg)
}

func (a *Agent) handleMessage(msg *common.Message) {
	a.logger.Debug("Received message: %s", msg.String())

	switch msg.Type {
	case common.MsgTypeRegister:
		a.logger.Debug("Registration confirmation received")
	case common.MsgTypeConnect:
		a.handleConnect(msg)
	case common.MsgTypeData:
		a.handleData(msg)
	case common.MsgTypeClose:
		a.handleClose(msg)
	case common.MsgTypeError:
		a.logger.Error("Received error from relay: %s", msg.Error)
	case common.MsgTypeHeartbeat:
		// Heartbeat response received
		a.logger.Debug("Heartbeat response received")
	case "shell_command":
		a.handleShellCommand(msg)
	default:
		a.logger.Error("Unknown message type: %s", msg.Type)
	}
}

func (a *Agent) handleConnect(msg *common.Message) {
	a.logger.Info("New connection request for session %s to target %s", msg.SessionID, msg.Target)

	// Connect to target SSH server
	a.logger.Debug("Attempting to connect to target: %s", msg.Target)
	conn, err := net.DialTimeout("tcp", msg.Target, 10*time.Second)
	if err != nil {
		a.logger.Error("Failed to connect to target %s: %v", msg.Target, err)

		errorMsg := common.NewMessage(common.MsgTypeError)
		errorMsg.SessionID = msg.SessionID
		errorMsg.AgentID = a.id
		errorMsg.Error = fmt.Sprintf("Failed to connect to target: %v", err)
		a.sendMessage(errorMsg)
		return
	}

	a.mutex.Lock()
	a.sessions[msg.SessionID] = conn
	// Create database query logger for this session
	a.dbLoggers[msg.SessionID] = common.NewDatabaseQueryLogger(a.logger, msg.Target)
	// Store target for this session
	a.targets[msg.SessionID] = msg.Target
	// Store client ID for this session
	a.clients[msg.SessionID] = msg.ClientID
	a.mutex.Unlock()

	a.logger.Info("Successfully connected to target %s for session %s", msg.Target, msg.SessionID)

	// Start forwarding data from target to relay
	go a.forwardFromTarget(msg.SessionID, conn)
}

func (a *Agent) handleData(msg *common.Message) {
	a.mutex.RLock()
	conn, exists := a.sessions[msg.SessionID]
	dbLogger, dbExists := a.dbLoggers[msg.SessionID]
	a.mutex.RUnlock()

	if !exists {
		a.logger.Error("Session not found: %s", msg.SessionID)
		return
	}

	a.logger.Debug("Received data for session %s: %d bytes", msg.SessionID, len(msg.Data))

	// Log database query if logger exists
	if dbExists {
		dbLogger.LogData(msg.Data, "CLIENT->TARGET", msg.SessionID)

		// Check for MySQL connection handshake to extract database name
		if databaseName := a.extractDatabaseFromHandshake(msg.Data); databaseName != "" {
			a.mutex.RLock()
			target := a.targets[msg.SessionID]
			clientID := a.clients[msg.SessionID]
			a.mutex.RUnlock()
			protocol := a.detectProtocol(target)
			a.sendDatabaseQuery(msg.SessionID, clientID, fmt.Sprintf("USE %s", databaseName), "CONNECT", "", databaseName, protocol)
		}

		// Always try to log data as potential SQL command, even if it's binary
		a.logPotentialDatabaseCommand(msg.SessionID, msg.Data, "CLIENT->TARGET")

		// Also extract and send query to relay if it looks like SQL
		queryText := string(msg.Data)
		if operation, tableName, databaseName := a.extractSQLInfo(queryText); operation != "" {
			a.mutex.RLock()
			target := a.targets[msg.SessionID]
			clientID := a.clients[msg.SessionID]
			a.mutex.RUnlock()
			protocol := a.detectProtocol(target)
			
			a.logger.Info("=== EXTRACTED SQL INFO ===")
			a.logger.Info("Operation: %s", operation)
			a.logger.Info("Table: %s", tableName)
			a.logger.Info("Database: %s", databaseName)
			a.logger.Info("Protocol: %s", protocol)
			a.logger.Info("Client: %s", clientID)
			
			a.sendDatabaseQuery(msg.SessionID, clientID, queryText, operation, tableName, databaseName, protocol)
		}
	}

	// Forward data to target
	if _, err := conn.Write(msg.Data); err != nil {
		a.logger.Error("Failed to write to target for session %s: %v", msg.SessionID, err)
		a.closeSession(msg.SessionID)
	} else {
		a.logger.Debug("Successfully forwarded %d bytes to target", len(msg.Data))
	}
}

func (a *Agent) handleClose(msg *common.Message) {
	a.logger.Info("Closing session: %s", msg.SessionID)
	a.closeSession(msg.SessionID)
}

func (a *Agent) forwardFromTarget(sessionID string, conn net.Conn) {
	a.logger.Debug("Starting data forwarding from target for session %s", sessionID)
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				a.logger.Error("Failed to read from target for session %s: %v", sessionID, err)
			} else {
				a.logger.Debug("Target connection closed for session %s", sessionID)
			}
			break
		}

		if n > 0 {
			a.logger.Debug("Read %d bytes from target for session %s", n, sessionID)

			// Log database response if logger exists
			a.mutex.RLock()
			dbLogger, dbExists := a.dbLoggers[sessionID]
			a.mutex.RUnlock()

			if dbExists {
				dbLogger.LogData(buffer[:n], "TARGET->CLIENT", sessionID)
				// Also log potential database responses
				a.logPotentialDatabaseCommand(sessionID, buffer[:n], "TARGET->CLIENT")
			}

			dataMsg := common.NewMessage(common.MsgTypeData)
			dataMsg.SessionID = sessionID
			dataMsg.AgentID = a.id
			dataMsg.Data = make([]byte, n)
			copy(dataMsg.Data, buffer[:n])

			// Validation before sending
			if dataMsg.AgentID == "" {
				a.logger.Error("❌ CRITICAL: AgentID is empty before sending!")
				dataMsg.AgentID = a.id
			}
			if dataMsg.SessionID == "" {
				a.logger.Error("❌ CRITICAL: SessionID is empty before sending!")
				continue
			}

			a.logger.Debug("Sending data message - AgentID: '%s', SessionID: '%s', DataLen: %d",
				dataMsg.AgentID, dataMsg.SessionID, len(dataMsg.Data))

			if err := a.sendMessage(dataMsg); err != nil {
				a.logger.Error("Failed to forward data to relay for session %s: %v", sessionID, err)
				break
			}
			a.logger.Debug("Successfully forwarded %d bytes to relay", n)
		}
	}

	// Connection closed, notify relay
	a.logger.Info("Target connection closed for session %s, notifying relay", sessionID)
	closeMsg := common.NewMessage(common.MsgTypeClose)
	closeMsg.SessionID = sessionID
	closeMsg.AgentID = a.id
	a.sendMessage(closeMsg)

	a.closeSession(sessionID)
}

func (a *Agent) closeSession(sessionID string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if conn, exists := a.sessions[sessionID]; exists {
		conn.Close()
		delete(a.sessions, sessionID)
		a.logger.Info("Session closed: %s", sessionID)
	}

	// Clean up database logger
	if dbLogger, exists := a.dbLoggers[sessionID]; exists {
		_ = dbLogger // Use the variable to avoid unused warning
		delete(a.dbLoggers, sessionID)
		a.logger.Debug("Database logger cleaned up for session: %s", sessionID)
	}

	// Clean up target mapping
	if _, exists := a.targets[sessionID]; exists {
		delete(a.targets, sessionID)
		a.logger.Debug("Target mapping cleaned up for session: %s", sessionID)
	}

	// Clean up client mapping
	if _, exists := a.clients[sessionID]; exists {
		delete(a.clients, sessionID)
		a.logger.Debug("Client mapping cleaned up for session: %s", sessionID)
	}
}

func (a *Agent) sendMessage(msg *common.Message) error {
	// Debug validation
	a.logger.Debug("=== SEND MESSAGE DEBUG ===")
	a.logger.Debug("Type: %s", msg.Type)
	a.logger.Debug("AgentID: '%s'", msg.AgentID)
	a.logger.Debug("SessionID: '%s'", msg.SessionID)
	a.logger.Debug("ClientID: '%s'", msg.ClientID)

	if msg.AgentID == "" {
		a.logger.Error("❌ CRITICAL: Attempting to send message with empty AgentID!")
	}

	// Always use JSON for now to avoid binary corruption
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %v", err)
	}

	return a.conn.WriteMessage(websocket.TextMessage, data)
}

func (a *Agent) isSSHSession(sessionID string) bool {
	a.mutex.RLock()
	target, exists := a.targets[sessionID]
	a.mutex.RUnlock()

	if !exists {
		return false
	}

	// Check if target is SSH port
	return strings.Contains(target, ":22") || strings.Contains(target, ":2222")
}

func (a *Agent) sendBinarySSHData(msg *common.Message) error {
	// Create binary frame for SSH data
	// Format: [TYPE:4][CLIENT_ID_LEN:1][CLIENT_ID][AGENT_ID_LEN:1][AGENT_ID][SESSION_ID_LEN:1][SESSION_ID][DATA]

	clientID := msg.ClientID
	if clientID == "" {
		clientID = ""
	}
	agentID := msg.AgentID
	if agentID == "" {
		agentID = a.id // Ensure AgentID is always set
	}
	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = ""
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

	a.logger.Debug("Sending binary SSH data: ClientID=%s, AgentID=%s, SessionID=%s, DataLen=%d",
		clientID, agentID, sessionID, len(msg.Data))

	return a.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (a *Agent) sendDatabaseQuery(sessionID, clientID, query, operation, tableName, databaseName, protocol string) {
	a.logger.Info("=== SENDING DATABASE QUERY TO RELAY ===")
	a.logger.Info("SessionID: %s", sessionID)
	a.logger.Info("ClientID: %s", clientID)
	a.logger.Info("Operation: %s", operation)
	a.logger.Info("Table: %s", tableName)
	a.logger.Info("Database: %s", databaseName)
	a.logger.Info("Protocol: %s", protocol)
	a.logger.Info("Query: %s", query)
	
	msg := common.NewMessage(common.MsgTypeDBQuery)
	msg.AgentID = a.id
	msg.ClientID = clientID
	msg.SessionID = sessionID
	msg.DBQuery = query
	msg.DBOperation = operation
	msg.DBTable = tableName
	msg.DBDatabase = databaseName
	msg.DBProtocol = protocol

	if err := a.sendMessage(msg); err != nil {
		a.logger.Error("Failed to send database query log: %v", err)
	} else {
		a.logger.Info("Successfully sent database query to relay: %s %s.%s", operation, databaseName, tableName)
	}
}

func (a *Agent) extractSQLInfo(queryText string) (operation, tableName, databaseName string) {
	// Clean and extract SQL from potentially binary data
	cleanQuery := a.extractCleanSQL(queryText)
	if cleanQuery == "" {
		return "", "", ""
	}

	// Simple SQL parser - extract operation, table name, and database name
	query := strings.TrimSpace(strings.ToUpper(cleanQuery))

	words := strings.Fields(query)
	if len(words) < 2 {
		return "", "", ""
	}

	operation = words[0]

	// Extract database name from table references like "database.table"
	extractDBFromTable := func(tableRef string) (string, string) {
		if strings.Contains(tableRef, ".") {
			parts := strings.Split(tableRef, ".")
			if len(parts) >= 2 {
				return strings.Trim(parts[0], "`"), strings.Trim(parts[1], "`")
			}
		}
		return "", strings.Trim(tableRef, "`")
	}

	switch operation {
	case "SELECT":
		// Find FROM keyword
		for i, word := range words {
			if word == "FROM" && i+1 < len(words) {
				databaseName, tableName = extractDBFromTable(strings.Trim(words[i+1], ",();"))
				break
			}
		}
	case "INSERT":
		// Find INTO keyword
		for i, word := range words {
			if word == "INTO" && i+1 < len(words) {
				databaseName, tableName = extractDBFromTable(strings.Trim(words[i+1], ",();"))
				break
			}
		}
	case "UPDATE", "DELETE":
		// Table name usually follows UPDATE or DELETE FROM
		if len(words) > 1 {
			if operation == "DELETE" && words[1] == "FROM" && len(words) > 2 {
				databaseName, tableName = extractDBFromTable(strings.Trim(words[2], ",();"))
			} else if operation == "UPDATE" {
				databaseName, tableName = extractDBFromTable(strings.Trim(words[1], ",();"))
			}
		}
	case "TRUNCATE":
		// TRUNCATE TABLE table_name
		for i, word := range words {
			if word == "TABLE" && i+1 < len(words) {
				databaseName, tableName = extractDBFromTable(strings.Trim(words[i+1], ",();"))
				break
			}
		}
	case "DROP":
		// DROP TABLE/DATABASE/INDEX etc
		if len(words) > 2 && words[1] == "TABLE" {
			databaseName, tableName = extractDBFromTable(strings.Trim(words[2], ",();"))
		} else if len(words) > 2 && words[1] == "DATABASE" {
			databaseName = strings.Trim(words[2], ",();`")
		}
	case "CREATE":
		// CREATE TABLE/DATABASE etc
		if len(words) > 2 && words[1] == "TABLE" {
			databaseName, tableName = extractDBFromTable(strings.Trim(words[2], ",();"))
		} else if len(words) > 2 && words[1] == "DATABASE" {
			databaseName = strings.Trim(words[2], ",();`")
		}
	case "ALTER":
		// ALTER TABLE table_name
		if len(words) > 2 && words[1] == "TABLE" {
			databaseName, tableName = extractDBFromTable(strings.Trim(words[2], ",();"))
		}
	case "SHOW":
		// SHOW TABLES, SHOW DATABASES etc
		if len(words) > 1 && words[1] == "TABLES" {
			operation = "SHOW_TABLES"
		} else if len(words) > 1 && words[1] == "DATABASES" {
			operation = "SHOW_DATABASES"
		}
	case "USE":
		// USE database_name
		if len(words) > 1 {
			databaseName = strings.Trim(words[1], ",();`")
		}
	}

	return operation, tableName, databaseName
}

// extractCleanSQL extracts clean SQL commands from potentially binary data
func (a *Agent) extractCleanSQL(data string) string {
	// Look for SQL keywords in the data
	sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "SHOW", "DESCRIBE", "EXPLAIN"}
	
	// Try to find SQL patterns
	dataUpper := strings.ToUpper(data)
	
	for _, keyword := range sqlKeywords {
		if idx := strings.Index(dataUpper, keyword); idx >= 0 {
			// Found SQL keyword, extract from this position
			remaining := data[idx:]
			
			// Clean the SQL by removing non-printable characters but keep SQL-valid chars
			var cleaned strings.Builder
			inQuote := false
			quoteChar := byte(0)
			
			for i, b := range []byte(remaining) {
				// Handle quotes
				if (b == '\'' || b == '"' || b == '`') && (i == 0 || remaining[i-1] != '\\') {
					if !inQuote {
						inQuote = true
						quoteChar = b
					} else if b == quoteChar {
						inQuote = false
						quoteChar = 0
					}
					cleaned.WriteByte(b)
					continue
				}
				
				// If in quote, keep everything
				if inQuote {
					cleaned.WriteByte(b)
					continue
				}
				
				// Keep printable ASCII chars and some special SQL chars
				if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
					cleaned.WriteByte(b)
				} else if b < 32 || b > 126 {
					// Replace non-printable with space
					cleaned.WriteByte(' ')
				}
			}
			
			result := strings.TrimSpace(cleaned.String())
			// Remove multiple spaces
			result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
			
			// Validate that result looks like SQL
			if len(result) >= 6 && strings.Contains(strings.ToUpper(result), keyword) {
				return result
			}
		}
	}
	
	return ""
}

// extractDatabaseFromHandshake extracts database name from MySQL connection handshake
func (a *Agent) extractDatabaseFromHandshake(data []byte) string {
	// MySQL client connection packet structure:
	// - Protocol version (1 byte)
	// - Capability flags (4 bytes)
	// - Max packet size (4 bytes)
	// - Character set (1 byte)
	// - Reserved (23 bytes)
	// - Username (null-terminated string)
	// - Auth response length + auth response
	// - Database name (null-terminated string) - if CLIENT_CONNECT_WITH_DB flag is set
	
	if len(data) < 36 { // Minimum packet size
		return ""
	}
	
	// Check if this looks like a MySQL handshake response (client login packet)
	// Look for capability flags that include CLIENT_CONNECT_WITH_DB (0x00000008)
	if len(data) >= 8 {
		capabilityFlags := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
		hasConnectWithDB := (capabilityFlags & 0x00000008) != 0
		
		if hasConnectWithDB {
			// Skip fixed fields (36 bytes)
			offset := 36
			
			// Skip username (null-terminated)
			for offset < len(data) && data[offset] != 0 {
				offset++
			}
			if offset >= len(data) {
				return ""
			}
			offset++ // Skip null terminator
			
			// Skip auth response length and auth response
			if offset < len(data) {
				authLength := int(data[offset])
				offset += 1 + authLength
			}
			
			// Extract database name (null-terminated)
			if offset < len(data) {
				dbStart := offset
				for offset < len(data) && data[offset] != 0 {
					offset++
				}
				if offset > dbStart {
					return string(data[dbStart:offset])
				}
			}
		}
	}
	
	return ""
}

func (a *Agent) detectProtocol(target string) string {
	if strings.Contains(target, ":3306") {
		return "mysql"
	}
	if strings.Contains(target, ":5432") {
		return "postgresql"
	}
	if strings.Contains(target, ":27017") {
		return "mongodb"
	}
	if strings.Contains(target, ":6379") {
		return "redis"
	}
	return "unknown"
}

func (a *Agent) logPotentialDatabaseCommand(sessionID string, data []byte, direction string) {
	// Try to extract meaningful information from any database protocol data
	a.mutex.RLock()
	target := a.targets[sessionID]
	clientID := a.clients[sessionID]
	a.mutex.RUnlock()

	if target == "" || clientID == "" {
		return
	}

	protocol := a.detectProtocol(target)
	
	// Convert data to string and check if it contains readable SQL
	dataStr := string(data)
	
	// Check for common SQL keywords even in binary data
	sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "SHOW", "DESCRIBE", "EXPLAIN", "USE"}
	
	hasSQL := false
	for _, keyword := range sqlKeywords {
		if strings.Contains(strings.ToUpper(dataStr), keyword) {
			hasSQL = true
			break
		}
	}
	
	// Log if it contains SQL keywords or if it's a significant data packet
	if hasSQL || len(data) > 20 {
		// Try to extract SQL info
		if operation, tableName, databaseName := a.extractSQLInfo(dataStr); operation != "" {
			a.sendDatabaseQuery(sessionID, clientID, dataStr, operation, tableName, databaseName, protocol)
		} else if hasSQL {
			// Even if we can't parse it completely, log it as unknown operation
			queryText := a.cleanDataForLogging(data)
			if queryText != "" {
				a.sendDatabaseQuery(sessionID, clientID, queryText, "UNKNOWN", "", "", protocol)
			}
		}
	}
}

func (a *Agent) cleanDataForLogging(data []byte) string {
	// Convert binary data to readable string, removing non-printable characters
	result := ""
	for _, b := range data {
		if b >= 32 && b <= 126 { // Printable ASCII characters
			result += string(b)
		} else if b == 10 || b == 13 { // Line breaks
			result += " "
		}
	}
	
	// Clean up multiple spaces and trim
	result = strings.TrimSpace(result)
	words := strings.Fields(result)
	if len(words) > 20 {
		words = words[:20] // Limit to first 20 words
	}
	
	return strings.Join(words, " ")
}

// handleShellCommand executes shell commands on the agent machine
func (a *Agent) handleShellCommand(msg *common.Message) {
	command := msg.DBQuery // Use DBQuery field for command
	a.logger.Info("=== EXECUTING SHELL COMMAND ===")
	a.logger.Info("Command: %s", command)
	a.logger.Info("Agent ID: %s", a.id)
	a.logger.Info("Session ID: %s", msg.SessionID)
	a.logger.Info("Client ID: %s", msg.ClientID)

	if command == "" {
		a.sendShellError(msg.SessionID, msg.ClientID, "Empty command")
		return
	}

	// Log SSH command to database
	a.logSSHCommand(msg.SessionID, msg.ClientID, command, "input")

	// Execute command using the system shell
	output, err := a.executeSystemCommand(command)
	a.logger.Info("Command output: %s", output)
	if err != nil {
		a.logger.Error("Command error: %v", err)
	}

	// Log command output
	if err == nil {
		a.logSSHCommand(msg.SessionID, msg.ClientID, output, "output")
	}

	// Send response back
	responseMsg := common.NewMessage("shell_response")
	responseMsg.SessionID = msg.SessionID
	responseMsg.ClientID = msg.ClientID
	responseMsg.AgentID = a.id
	responseMsg.DBQuery = command     // Store command in DBQuery field
	responseMsg.Data = []byte(output) // Convert string to []byte

	if err != nil {
		// Send error response
		a.sendShellError(msg.SessionID, msg.ClientID, fmt.Sprintf("Command failed: %v", err))
	} else {
		// Send successful response
		if err := a.sendMessage(responseMsg); err != nil {
			a.logger.Error("Failed to send shell response: %v", err)
		}
	}
}

func (a *Agent) executeSystemCommand(command string) (string, error) {
	// For Windows
	if len(os.Getenv("COMSPEC")) > 0 {
		cmd := exec.Command("cmd", "/C", command)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// For Unix/Linux
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (a *Agent) sendShellError(sessionID, clientID, errorMsg string) {
	errMsg := common.NewMessage("shell_error")
	errMsg.SessionID = sessionID
	errMsg.ClientID = clientID
	errMsg.AgentID = a.id
	errMsg.Data = []byte(errorMsg) // Convert string to []byte

	if err := a.sendMessage(errMsg); err != nil {
		a.logger.Error("Failed to send shell error: %v", err)
	}
}

func (a *Agent) logSSHCommand(sessionID, clientID, command, direction string) {
	// Extract host from relay URL for logging
	relayHost := "localhost:8080" // Default
	if u, err := url.Parse(a.relayURL); err == nil {
		relayHost = u.Host
	}

	logReq := SSHLogRequest{
		SessionID: sessionID,
		ClientID:  clientID,
		AgentID:   a.id,
		Direction: direction,
		User:      "remote", // Could be enhanced to get actual user
		Host:      relayHost,
		Port:      "22",
		Command:   command,
		Data:      command,
	}

	// Send to relay API
	go func() {
		jsonData, err := json.Marshal(logReq)
		if err != nil {
			a.logger.Error("Failed to marshal SSH log: %v", err)
			return
		}

		// Extract relay server HTTP URL from WebSocket URL
		relayHTTP := strings.Replace(a.relayURL, "ws://", "http://", 1)
		relayHTTP = strings.Replace(relayHTTP, "/ws", "", 1)
		apiURL := relayHTTP + "/api/log-ssh"

		resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			a.logger.Error("Failed to send SSH command log: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			a.logger.Debug("SSH command logged: %s -> %s", direction, command)
		} else {
			a.logger.Error("Failed to log SSH command, status: %d", resp.StatusCode)
		}
	}()
}

func main() {
	var (
		agentID  string
		relayURL string
		token    string
	)

	var rootCmd = &cobra.Command{
		Use:   "tunnel-agent",
		Short: "SSH Tunnel Agent",
		Long:  "An agent that runs on the target server and forwards SSH connections through a relay",
		Run: func(cmd *cobra.Command, args []string) {
			if agentID == "" {
				agentID = common.GenerateID()
			}

			if token == "" {
				log.Fatal("Token is required. Use -t or --token flag to provide agent authentication token.")
			}

			agent := NewAgent(agentID, token, relayURL)

			// Setup signal handling for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				agent.logger.Info("Received shutdown signal, stopping agent...")
				agent.stop()
				agent.logger.Close()
				os.Exit(0)
			}()

			if err := agent.start(); err != nil {
				log.Fatalf("Failed to start agent: %v", err)
			}

			// Wait for interrupt signal
			select {}
		},
	}

	rootCmd.Flags().StringVarP(&agentID, "agent-id", "a", "", "Agent ID (auto-generated if not provided)")
	rootCmd.Flags().StringVarP(&relayURL, "relay-url", "r", "ws://localhost:8080/ws/agent", "Relay server WebSocket URL")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "Agent authentication token (required for agent verification)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
