package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GoTeleportAgent struct {
	config     *AgentConfig
	conn       *websocket.Conn
	logger     *log.Logger
	agentID    string
	sessions   map[string]*AgentSession
	dbProxies  map[string]*DatabaseProxy
	mutex      sync.RWMutex
}

type AgentConfig struct {
	ServerURL       string                    `json:"server_url"`
	AgentName       string                    `json:"agent_name"`
	Platform        string                    `json:"platform"`
	LogFile         string                    `json:"log_file"`
	AuthToken       string                    `json:"auth_token"`
	Metadata        map[string]string         `json:"metadata"`
	WorkingDir      string                    `json:"working_dir"`
	AllowedUsers    []string                  `json:"allowed_users"`
	DatabaseProxies []DatabaseProxyConfig     `json:"database_proxies"`
}

type DatabaseProxyConfig struct {
	Name       string `json:"name"`
	LocalPort  int    `json:"local_port"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port"`
	Protocol   string `json:"protocol"` // mysql, postgresql, etc
	Enabled    bool   `json:"enabled"`
}

type DatabaseProxy struct {
	Config    DatabaseProxyConfig
	Listener  net.Listener
	Agent     *GoTeleportAgent
	Logger    *log.Logger
	Active    bool
	mutex     sync.RWMutex
}

type DatabaseCommand struct {
	SessionID string    `json:"session_id"`
	Command   string    `json:"command"`
	Protocol  string    `json:"protocol"`
	ClientIP  string    `json:"client_ip"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
	ProxyName string    `json:"proxy_name"`
}

type AgentSession struct {
	ID          string
	ClientID    string
	WorkingDir  string
	Environment map[string]string
	CreatedAt   time.Time
	LastUsed    time.Time
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

type CommandResult struct {
	Command   string        `json:"command"`
	Output    string        `json:"output"`
	Error     string        `json:"error"`
	ExitCode  int           `json:"exit_code"`
	Duration  time.Duration `json:"duration"`
	WorkingDir string       `json:"working_dir"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: goteleport-agent.exe <config-file>")
	}

	agent, err := NewGoTeleportAgent(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	agent.Start()
}

func NewGoTeleportAgent(configFile string) (*GoTeleportAgent, error) {
	// Read config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Set defaults
	if config.Platform == "" {
		config.Platform = runtime.GOOS
	}
	if config.WorkingDir == "" {
		config.WorkingDir, _ = os.Getwd()
	}

	// Setup logger
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	logger := log.New(logFile, "", log.LstdFlags)

	agent := &GoTeleportAgent{
		config:    &config,
		logger:    logger,
		sessions:  make(map[string]*AgentSession),
		dbProxies: make(map[string]*DatabaseProxy),
	}

	// Initialize database proxies
	if err := agent.initDatabaseProxies(); err != nil {
		logger.Printf("Warning: Failed to initialize database proxies: %v", err)
	}

	return agent, nil
}

func (a *GoTeleportAgent) Start() {
	a.logEvent("AGENT_START", "GoTeleport Agent starting", a.config.AgentName)

	for {
		if err := a.connect(); err != nil {
			a.logEvent("ERROR", "Connection failed", err.Error())
			fmt.Printf("❌ Connection failed: %v\n", err)
			fmt.Println("🔄 Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}

		// Connection lost, retry
		fmt.Println("🔄 Connection lost, retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func (a *GoTeleportAgent) connect() error {
	fmt.Printf("🔗 Connecting to server: %s\n", a.config.ServerURL)

	// Connect to server
	conn, _, err := websocket.DefaultDialer.Dial(a.config.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	a.conn = conn

	// Register with server
	if err := a.register(); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	fmt.Printf("✅ Connected and registered as: %s\n", a.config.AgentName)
	a.logEvent("AGENT_CONNECT", "Connected to server", a.config.ServerURL)

	// Start heartbeat
	go a.heartbeat()

	// Handle messages
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			a.logEvent("AGENT_DISCONNECT", "Disconnected from server", err.Error())
			return fmt.Errorf("connection lost: %v", err)
		}

		a.handleMessage(&msg)
	}
}

func (a *GoTeleportAgent) register() error {
	regMsg := Message{
		Type:      "register",
		AgentID:   a.generateID(),
		Metadata: map[string]interface{}{
			"name":       a.config.AgentName,  // Changed from "agent_name" to "name"
			"agent_name": a.config.AgentName,  // Keep for backward compatibility
			"platform":   a.config.Platform,
			"auth_token": a.config.AuthToken,
			"metadata":   a.config.Metadata,
		},
		Timestamp: time.Now(),
	}

	a.agentID = regMsg.AgentID
	return a.conn.WriteJSON(regMsg)
}

func (a *GoTeleportAgent) generateID() string {
	return fmt.Sprintf("agent_%d", time.Now().UnixNano())
}

func (a *GoTeleportAgent) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if a.conn == nil {
			return
		}

		msg := Message{
			Type:      "heartbeat",
			AgentID:   a.agentID,
			Timestamp: time.Now(),
		}

		if err := a.conn.WriteJSON(msg); err != nil {
			a.logEvent("ERROR", "Failed to send heartbeat", err.Error())
			return
		}
	}
}

func (a *GoTeleportAgent) handleMessage(msg *Message) {
	switch msg.Type {
	case "command":
		a.executeCommand(msg)
	default:
		a.logEvent("MESSAGE", "Unknown message type", msg.Type)
	}
}

func (a *GoTeleportAgent) executeCommand(msg *Message) {
	sessionID := msg.SessionID
	command := msg.Command

	// Log command execution
	a.logEvent("CMD_EXEC", "Command execution", fmt.Sprintf("Session: %s, Command: %s", sessionID, command))

	// Execute command
	result := a.runCommand(command)

	// Send result back
	responseMsg := Message{
		Type:      "command_result",
		SessionID: sessionID,
		AgentID:   a.agentID,
		Data:      a.formatCommandResult(result),
		Metadata: map[string]interface{}{
			"command":    result.Command,
			"exit_code":  result.ExitCode,
			"duration":   result.Duration.Milliseconds(),
			"working_dir": result.WorkingDir,
		},
		Timestamp: time.Now(),
	}

	if err := a.conn.WriteJSON(responseMsg); err != nil {
		a.logEvent("ERROR", "Failed to send command result", err.Error())
	}
}

func (a *GoTeleportAgent) runCommand(command string) *CommandResult {
	start := time.Now()
	workingDir := a.config.WorkingDir
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Dir = workingDir
	output, err := cmd.CombinedOutput()

	result := &CommandResult{
		Command:    command,
		Output:     string(output),
		Duration:   time.Since(start),
		WorkingDir: workingDir,
	}

	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result
}

func (a *GoTeleportAgent) formatCommandResult(result *CommandResult) string {
	var output strings.Builder
	
	if result.Output != "" {
		output.WriteString(result.Output)
	}
	
	if result.Error != "" {
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		output.WriteString("Error: " + result.Error)
	}
	
	output.WriteString(fmt.Sprintf("\n[Exit: %d, Duration: %v, Dir: %s]", 
		result.ExitCode, result.Duration, result.WorkingDir))
	
	return output.String()
}

func (a *GoTeleportAgent) logEvent(eventType, description, details string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "system"
	}

	logEntry := fmt.Sprintf("[%s] [%s] User: %s | Agent: %s | Event: %s | Details: %s",
		timestamp, eventType, user, a.config.AgentName, description, details)

	if a.logger != nil {
		a.logger.Println(logEntry)
	}
}

// Database Proxy Functions
func (a *GoTeleportAgent) initDatabaseProxies() error {
	for _, proxyConfig := range a.config.DatabaseProxies {
		if !proxyConfig.Enabled {
			continue
		}

		proxy := &DatabaseProxy{
			Config: proxyConfig,
			Agent:  a,
			Logger: a.logger,
		}

		if err := proxy.Start(); err != nil {
			a.logger.Printf("Failed to start database proxy %s: %v", proxyConfig.Name, err)
			continue
		}

		a.mutex.Lock()
		a.dbProxies[proxyConfig.Name] = proxy
		a.mutex.Unlock()

		a.logger.Printf("Database proxy %s started on port %d -> %s:%d", 
			proxyConfig.Name, proxyConfig.LocalPort, proxyConfig.TargetHost, proxyConfig.TargetPort)
	}

	return nil
}

func (dp *DatabaseProxy) Start() error {
	addr := fmt.Sprintf(":%d", dp.Config.LocalPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}

	dp.Listener = listener
	dp.Active = true

	go dp.acceptConnections()
	return nil
}

func (dp *DatabaseProxy) acceptConnections() {
	for {
		conn, err := dp.Listener.Accept()
		if err != nil {
			if dp.Active {
				dp.Logger.Printf("Error accepting connection: %v", err)
			}
			return
		}

		go dp.handleConnection(conn)
	}
}

func (dp *DatabaseProxy) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Connect to target database
	targetAddr := fmt.Sprintf("%s:%d", dp.Config.TargetHost, dp.Config.TargetPort)
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		dp.Logger.Printf("Failed to connect to target %s: %v", targetAddr, err)
		return
	}
	defer targetConn.Close()

	sessionID := fmt.Sprintf("db_%s_%d", dp.Config.Name, time.Now().Unix())
	clientIP := clientConn.RemoteAddr().String()

	dp.Logger.Printf("Database connection established: %s -> %s (Session: %s)", 
		clientIP, targetAddr, sessionID)

	// Start packet inspection for command logging
	go dp.inspectAndForward(clientConn, targetConn, "client_to_server", sessionID, clientIP)
	go dp.inspectAndForward(targetConn, clientConn, "server_to_client", sessionID, clientIP)

	// Keep connection alive until one side closes
	done := make(chan bool, 2)
	go func() {
		io.Copy(clientConn, targetConn)
		done <- true
	}()
	go func() {
		io.Copy(targetConn, clientConn)
		done <- true
	}()

	<-done
	dp.Logger.Printf("Database connection closed: Session %s", sessionID)
}

func (dp *DatabaseProxy) inspectAndForward(src, dst net.Conn, direction, sessionID, clientIP string) {
	buffer := make([]byte, 4096)
	var dataBuffer bytes.Buffer

	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				dp.Logger.Printf("Error reading from %s: %v", direction, err)
			}
			return
		}

		data := buffer[:n]
		dataBuffer.Write(data)

		// Inspect data for SQL commands if it's from client to server
		if direction == "client_to_server" {
			if dp.Config.Protocol == "mysql" {
				dp.inspectMySQLCommands(dataBuffer.Bytes(), sessionID, clientIP)
			} else if dp.Config.Protocol == "postgres" {
				dp.inspectPostgreSQLCommands(dataBuffer.Bytes(), sessionID, clientIP)
			}
		}

		// Forward the data
		if _, err := dst.Write(data); err != nil {
			dp.Logger.Printf("Error writing to %s: %v", direction, err)
			return
		}

		// Clear buffer if it gets too large
		if dataBuffer.Len() > 10240 {
			dataBuffer.Reset()
		}
	}
}

func (dp *DatabaseProxy) inspectMySQLCommands(data []byte, sessionID, clientIP string) {
	// Enhanced MySQL command detection with protocol parsing
	if len(data) < 5 {
		return
	}

	// First try to parse as MySQL protocol packets
	offset := 0
	for offset < len(data) {
		if offset+4 >= len(data) {
			break
		}

		// MySQL packet format: [length(3)] [sequence(1)] [payload]
		packetLen := int(data[offset]) | int(data[offset+1])<<8 | int(data[offset+2])<<16
		if packetLen == 0 || offset+4+packetLen > len(data) {
			break
		}

		payload := data[offset+4 : offset+4+packetLen]
		if len(payload) > 0 {
			// Check if this is a command packet (COM_QUERY = 0x03)
			if payload[0] == 0x03 && len(payload) > 1 {
				sqlQuery := string(payload[1:])
				if dp.isSQLCommand(sqlQuery) {
					dbCmd := DatabaseCommand{
						SessionID: sessionID,
						Command:   strings.TrimSpace(sqlQuery),
						Protocol:  dp.Config.Protocol,
						ClientIP:  clientIP,
						Timestamp: time.Now(),
						ProxyName: dp.Config.Name,
					}
					dp.logDatabaseCommand(dbCmd)
					dp.sendCommandToServer(dbCmd)
				}
			}
		}
		offset += 4 + packetLen
	}

	// Fallback: try regex-based detection on raw data
	dataStr := string(data)
	sqlCommands := dp.extractSQLCommands(dataStr)
	for _, cmd := range sqlCommands {
		dbCmd := DatabaseCommand{
			SessionID: sessionID,
			Command:   cmd,
			Protocol:  dp.Config.Protocol,
			ClientIP:  clientIP,
			Timestamp: time.Now(),
			ProxyName: dp.Config.Name,
		}
		dp.logDatabaseCommand(dbCmd)
		dp.sendCommandToServer(dbCmd)
	}
}

func (dp *DatabaseProxy) isSQLCommand(query string) bool {
	query = strings.TrimSpace(strings.ToUpper(query))
	if len(query) == 0 {
		return false
	}

	sqlKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", 
		"ALTER", "SHOW", "DESCRIBE", "DESC", "USE", "EXPLAIN",
		"GRANT", "REVOKE", "SET", "CALL", "EXECUTE",
	}

	for _, keyword := range sqlKeywords {
		if strings.HasPrefix(query, keyword+" ") || query == keyword {
			return true
		}
	}
	return false
}

func (dp *DatabaseProxy) extractSQLCommands(data string) []string {
	var commands []string
	
	// Common SQL command patterns
	sqlPatterns := []string{
		`(?i)\b(SELECT\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(INSERT\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(UPDATE\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(DELETE\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(CREATE\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(DROP\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(ALTER\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(SHOW\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(DESCRIBE\s+.+?)(?:\s*;|\s*$)`,
		`(?i)\b(USE\s+.+?)(?:\s*;|\s*$)`,
	}

	for _, pattern := range sqlPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}

		matches := re.FindAllStringSubmatch(data, -1)
		for _, match := range matches {
			if len(match) > 1 {
				cmd := strings.TrimSpace(match[1])
				if len(cmd) > 0 {
					commands = append(commands, cmd)
				}
			}
		}
	}

	return commands
}

func (dp *DatabaseProxy) inspectPostgreSQLCommands(data []byte, sessionID, clientIP string) {
	// PostgreSQL command detection with protocol parsing
	if len(data) < 5 {
		return
	}

	// PostgreSQL protocol: Try to detect Query messages (type 'Q')
	offset := 0
	for offset < len(data) {
		if offset+5 >= len(data) {
			break
		}

		// Look for Query message type 'Q' (0x51)
		if data[offset] == 0x51 {
			// Next 4 bytes are message length (big-endian)
			msgLen := int(data[offset+1])<<24 | int(data[offset+2])<<16 | int(data[offset+3])<<8 | int(data[offset+4])
			
			if msgLen > 4 && offset+msgLen+1 <= len(data) {
				// Extract SQL query (null-terminated string after the length)
				queryStart := offset + 5
				queryEnd := queryStart
				for queryEnd < offset+msgLen+1 && queryEnd < len(data) && data[queryEnd] != 0 {
					queryEnd++
				}
				
				if queryEnd > queryStart {
					sqlQuery := string(data[queryStart:queryEnd])
					if dp.isSQLCommand(sqlQuery) {
						dbCmd := DatabaseCommand{
							SessionID: sessionID,
							Command:   strings.TrimSpace(sqlQuery),
							Protocol:  dp.Config.Protocol,
							ClientIP:  clientIP,
							Timestamp: time.Now(),
							ProxyName: dp.Config.Name,
						}
						dp.logDatabaseCommand(dbCmd)
						dp.sendCommandToServer(dbCmd)
					}
				}
			}
			offset += msgLen + 1
		} else {
			offset++
		}
	}

	// Fallback: try regex-based detection on raw data
	dataStr := string(data)
	sqlCommands := dp.extractSQLCommands(dataStr)
	for _, cmd := range sqlCommands {
		dbCmd := DatabaseCommand{
			SessionID: sessionID,
			Command:   cmd,
			Protocol:  dp.Config.Protocol,
			ClientIP:  clientIP,
			Timestamp: time.Now(),
			ProxyName: dp.Config.Name,
		}
		dp.logDatabaseCommand(dbCmd)
		dp.sendCommandToServer(dbCmd)
	}
}

func (dp *DatabaseProxy) logDatabaseCommand(cmd DatabaseCommand) {
	timestamp := cmd.Timestamp.Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] [DB_COMMAND] Agent: %s | Proxy: %s | Session: %s | Client: %s | Protocol: %s | Command: %s",
		timestamp, dp.Agent.config.AgentName, cmd.ProxyName, cmd.SessionID, cmd.ClientIP, cmd.Protocol, cmd.Command)

	if dp.Logger != nil {
		dp.Logger.Println(logEntry)
	}
}

func (dp *DatabaseProxy) sendCommandToServer(cmd DatabaseCommand) {
	if dp.Agent.conn == nil {
		return
	}

	msg := Message{
		Type:      "database_command",
		SessionID: cmd.SessionID,
		AgentID:   dp.Agent.agentID,
		Command:   cmd.Command,
		Metadata: map[string]interface{}{
			"proxy_name": cmd.ProxyName,
			"protocol":   cmd.Protocol,
			"client_ip":  cmd.ClientIP,
		},
		Timestamp: cmd.Timestamp,
	}

	if err := dp.Agent.conn.WriteJSON(msg); err != nil {
		dp.Logger.Printf("Failed to send database command to server: %v", err)
	}
}

func (dp *DatabaseProxy) Stop() error {
	dp.mutex.Lock()
	defer dp.mutex.Unlock()

	dp.Active = false
	if dp.Listener != nil {
		return dp.Listener.Close()
	}
	return nil
}
