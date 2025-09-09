package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GoTeleportAgent struct {
	config              *AgentConfig
	agentID             string
	conn                *websocket.Conn
	sessions            map[string]*AgentSession
	tunnelConnections   map[string]net.Conn
	mutex               sync.RWMutex
	logger              *log.Logger
	logFile             *os.File
	proxies             []*DatabaseProxy
}

type AgentConfig struct {
	AgentName     string           `json:"agent_name"`
	ServerURL     string           `json:"server_url"`
	AuthToken     string           `json:"auth_token"`
	Platform      string           `json:"platform"`
	WorkingDir    string           `json:"working_dir"`
	LogFile       string           `json:"log_file"`
	DatabaseProxy []*DatabaseProxy `json:"database_proxy"`
}

type AgentSession struct {
	ID          string            `json:"id"`
	ClientID    string            `json:"client_id"`
	Environment map[string]string `json:"environment"`
	CreatedAt   time.Time         `json:"created_at"`
	LastUsed    time.Time         `json:"last_used"`
}

type CommandResult struct {
	Command    string        `json:"command"`
	Output     string        `json:"output"`
	Error      string        `json:"error"`
	ExitCode   int           `json:"exit_code"`
	Duration   time.Duration `json:"duration"`
	WorkingDir string        `json:"working_dir"`
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

type DatabaseProxy struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	LocalPort  int    `json:"local_port"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port"`
	agentRef   *GoTeleportAgent
	listener   net.Listener
}

func generateAgentID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func NewGoTeleportAgent(configFile string) (*GoTeleportAgent, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config AgentConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Setup logging
	var logFile *os.File
	var logger *log.Logger

	if config.LogFile != "" {
		logFile, err = os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	agent := &GoTeleportAgent{
		config:            &config,
		agentID:           generateAgentID(),
		sessions:          make(map[string]*AgentSession),
		tunnelConnections: make(map[string]net.Conn),
		logger:            logger,
		logFile:           logFile,
	}

	// Setup database proxies
	for _, proxyConfig := range config.DatabaseProxy {
		proxy := &DatabaseProxy{
			Name:       proxyConfig.Name,
			Protocol:   proxyConfig.Protocol,
			LocalPort:  proxyConfig.LocalPort,
			TargetHost: proxyConfig.TargetHost,
			TargetPort: proxyConfig.TargetPort,
			agentRef:   agent,
		}
		agent.proxies = append(agent.proxies, proxy)
	}

	return agent, nil
}

func (a *GoTeleportAgent) Start() error {
	a.logger.Printf("🚀 Starting GoTeleport Agent: %s (ID: %s)", a.config.AgentName, a.agentID)

	// Start database proxies
	for _, proxy := range a.proxies {
		if err := proxy.Start(); err != nil {
			a.logger.Printf("❌ Failed to start proxy %s: %v", proxy.Name, err)
			return err
		}
		a.logger.Printf("✅ Database proxy started: %s on port %d", proxy.Name, proxy.LocalPort)
	}

	// Connect to server
	if err := a.connectToServer(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	a.logger.Printf("✅ Agent started successfully")
	return nil
}

func (a *GoTeleportAgent) connectToServer() error {
	serverURL := strings.Replace(a.config.ServerURL, "http://", "ws://", 1)
	serverURL = strings.Replace(serverURL, "https://", "wss://", 1)
	if !strings.Contains(serverURL, "/ws/agent") {
		serverURL += "/ws/agent"
	}

	a.logger.Printf("🔌 Connecting to server: %s", serverURL)

	// Configure dialer with timeout
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second
	
	conn, _, err := dialer.Dial(serverURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	// Configure WebSocket for keep-alive
	conn.SetReadLimit(512 * 1024) // 512KB max message size
	
	// Setup ping/pong handlers
	conn.SetPongHandler(func(string) error {
		a.logger.Printf("🏓 AGENT: Received pong from server")
		return nil
	})
	
	conn.SetPingHandler(func(message string) error {
		a.logger.Printf("🏓 AGENT: Received ping from server, sending pong")
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := conn.WriteMessage(websocket.PongMessage, []byte(message))
		conn.SetWriteDeadline(time.Time{})
		return err
	})

	a.conn = conn
	a.logger.Printf("✅ Connected to server successfully")

	// Send registration message
	regMsg := Message{
		Type:    "register",
		AgentID: a.agentID,
		Metadata: map[string]interface{}{
			"name":     a.config.AgentName,
			"platform": a.config.Platform,
			"token":    a.config.AuthToken,
		},
		Timestamp: time.Now(),
	}

	if err := a.conn.WriteJSON(regMsg); err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}

	a.logger.Printf("📤 Registration sent to server")

	// Start ping routine for keep-alive
	go a.startPingRoutine()

	// Start message handler
	go a.handleMessages()

	return nil
}

// startPingRoutine sends periodic ping messages to keep the connection alive
func (a *GoTeleportAgent) startPingRoutine() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	
	a.logger.Printf("🏓 AGENT: Starting ping routine")
	
	for range ticker.C {
		if a.conn == nil {
			a.logger.Printf("🏓 AGENT: Connection is nil, stopping ping routine")
			return
		}
		
		a.logger.Printf("🏓 AGENT: Sending ping to server")
		a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := a.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			a.logger.Printf("❌ AGENT: Ping failed: %v", err)
			return
		}
		a.conn.SetWriteDeadline(time.Time{})
	}
}

func (a *GoTeleportAgent) handleMessages() {
	defer a.conn.Close()

	for {
		// Set read deadline for message reading
		a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		
		var msg Message
		if err := a.conn.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				a.logger.Printf("📥 AGENT: WebSocket closed normally")
			} else {
				a.logger.Printf("❌ AGENT: WebSocket read error: %v", err)
			}
			break
		}
		
		a.conn.SetReadDeadline(time.Time{}) // Remove deadline after successful read

		a.logger.Printf("📨 AGENT: Received message: Type=%s, SessionID=%s", msg.Type, msg.SessionID)
		a.handleMessage(&msg)
	}
}

func (a *GoTeleportAgent) handleMessage(msg *Message) {
	switch msg.Type {
	case "command":
		a.handleCommand(msg)
	case "tunnel_start":
		a.handleTunnelStart(msg)
	case "tunnel_data":
		a.handleTunnelData(msg)
	case "tunnel_close":
		a.handleTunnelClose(msg)
	default:
		a.logger.Printf("⚠️ AGENT: Unknown message type: %s", msg.Type)
	}
}

func (a *GoTeleportAgent) handleCommand(msg *Message) {
	sessionID := msg.SessionID
	command := msg.Command

	a.logger.Printf("🔧 AGENT: Executing command: %s (Session: %s)", command, sessionID)

	result := a.runCommand(command)
	
	// Log command execution
	a.logEvent("COMMAND_EXEC", command, fmt.Sprintf("Agent: %s | Client: %s | ExitCode: %d", 
		a.config.AgentName, msg.ClientID, result.ExitCode))

	response := Message{
		Type:      "command_result",
		SessionID: sessionID,
		AgentID:   a.agentID,
		Data:      result.Output,
		Metadata: map[string]interface{}{
			"exit_code": result.ExitCode,
			"error":     result.Error,
			"duration":  result.Duration.Milliseconds(),
		},
		Timestamp: time.Now(),
	}

	if err := a.conn.WriteJSON(response); err != nil {
		a.logger.Printf("❌ AGENT: Failed to send command result: %v", err)
	}
}

func (a *GoTeleportAgent) handleTunnelStart(msg *Message) {
	sessionID := msg.SessionID
	targetPort := 3307 // default
	dbType := "mysql"  // default

	if msg.Metadata != nil {
		if port, ok := msg.Metadata["target_port"].(float64); ok {
			targetPort = int(port)
		}
		if dbTypeVal, ok := msg.Metadata["db_type"].(string); ok {
			dbType = dbTypeVal
		}
	}

	a.logger.Printf("🚇 AGENT: TUNNEL_START - SessionID=%s, TargetPort=%d, DBType=%s", sessionID, targetPort, dbType)

	// Create session
	session := &AgentSession{
		ID:          sessionID,
		ClientID:    msg.ClientID,
		Environment: map[string]string{
			"target_port": fmt.Sprintf("%d", targetPort),
			"db_type":    dbType,
		},
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	a.mutex.Lock()
	a.sessions[sessionID] = session
	a.mutex.Unlock()

	a.logEvent("TUNNEL_START", "Tunnel session created", 
		fmt.Sprintf("Agent: %s | SessionID: %s | TargetPort: %d | DBType: %s", 
			a.config.AgentName, sessionID, targetPort, dbType))

	// Send ready response
	response := Message{
		Type:      "tunnel_ready",
		SessionID: sessionID,
		AgentID:   a.agentID,
		Timestamp: time.Now(),
	}

	if err := a.conn.WriteJSON(response); err != nil {
		a.logger.Printf("❌ AGENT: Failed to send tunnel_ready: %v", err)
	} else {
		a.logger.Printf("✅ AGENT: TUNNEL_READY sent for session: %s", sessionID)
	}
}

func (a *GoTeleportAgent) handleTunnelData(msg *Message) {
	sessionID := msg.SessionID
	
	a.logger.Printf("📦 AGENT: TUNNEL_DATA - SessionID=%s, Base64Len=%d", sessionID, len(msg.Data))
	
	// Get tunnel session
	a.mutex.RLock()
	session, exists := a.sessions[sessionID]
	a.mutex.RUnlock()
	
	if !exists {
		a.logger.Printf("❌ AGENT: Session not found: %s", sessionID)
		a.logEvent("TUNNEL_ERROR", "Tunnel session not found", sessionID)
		return
	}
	
	// Get target port from session metadata
	targetPortStr, _ := session.Environment["target_port"]
	targetPort := 3307 // default MySQL port
	if targetPortStr != "" {
		if _, err := fmt.Sscanf(targetPortStr, "%d", &targetPort); err != nil {
			targetPort = 3307
		}
	}
	
	a.logger.Printf("🔌 AGENT: Connecting to proxy at 127.0.0.1:%d", targetPort)
	
	// Connect to local database proxy
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", targetPort)
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		a.logger.Printf("❌ AGENT: Failed to connect to proxy %s: %v", proxyAddr, err)
		a.logEvent("TUNNEL_ERROR", "Failed to connect to database proxy", 
			fmt.Sprintf("Address: %s, Error: %v", proxyAddr, err))
		return
	}
	defer conn.Close()
	
	a.logger.Printf("✅ AGENT: Connected to proxy")
	
	// Decode base64 data from server
	data, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		a.logger.Printf("❌ AGENT: Failed to decode base64: %v", err)
		a.logEvent("TUNNEL_ERROR", "Failed to decode tunnel data", err.Error())
		return
	}
	
	a.logger.Printf("🔄 AGENT: Decoded %d bytes, forwarding to proxy", len(data))
	
	// Forward data to database proxy
	if _, err := conn.Write(data); err != nil {
		a.logger.Printf("❌ AGENT: Failed to write to proxy: %v", err)
		a.logEvent("TUNNEL_ERROR", "Failed to write to database proxy", err.Error())
		return
	}
	
	// Read response from database proxy
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil && err != io.EOF {
		a.logger.Printf("❌ AGENT: Failed to read from proxy: %v", err)
		a.logEvent("TUNNEL_ERROR", "Failed to read from database proxy", err.Error())
		return
	}
	
	a.logger.Printf("🔄 AGENT: Received %d bytes from proxy", n)
	
	// Encode response as base64 before sending to server
	encodedResponse := base64.StdEncoding.EncodeToString(buffer[:n])
	a.logger.Printf("🔧 AGENT: Encoded %d bytes to base64, sending to server", n)
	
	// Send response back to server - encode with base64 for binary safety
	responseMsg := Message{
		Type:      "tunnel_data",
		SessionID: sessionID,
		AgentID:   a.agentID,
		Data:      encodedResponse,
		Timestamp: time.Now(),
	}
	
	if err := a.conn.WriteJSON(responseMsg); err != nil {
		a.logger.Printf("❌ AGENT: Failed to send response to server: %v", err)
		a.logEvent("TUNNEL_ERROR", "Failed to send tunnel response", err.Error())
	} else {
		a.logger.Printf("✅ AGENT: Response sent successfully to server")
	}
}

func (a *GoTeleportAgent) handleTunnelClose(msg *Message) {
	sessionID := msg.SessionID
	
	a.logger.Printf("🔒 AGENT: TUNNEL_CLOSE - SessionID=%s", sessionID)
	
	a.mutex.Lock()
	delete(a.sessions, sessionID)
	a.mutex.Unlock()
	
	a.logEvent("TUNNEL_CLOSE", "Tunnel session closed", sessionID)
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
	duration := time.Since(start)

	result := &CommandResult{
		Command:    command,
		Output:     string(output),
		Duration:   duration,
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

func (a *GoTeleportAgent) logEvent(eventType, action, details string) {
	logEntry := map[string]interface{}{
		"timestamp":  time.Now().Format("2006-01-02 15:04:05"),
		"event_type": eventType,
		"action":     action,
		"details":    details,
		"agent_id":   a.agentID,
		"agent_name": a.config.AgentName,
	}

	logLine, _ := json.Marshal(logEntry)
	a.logger.Printf("📝 AGENT: EVENT: %s", string(logLine))
}

// Database Proxy Implementation
func (dp *DatabaseProxy) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", dp.LocalPort))
	if err != nil {
		return err
	}

	dp.listener = listener
	go dp.acceptConnections()
	return nil
}

func (dp *DatabaseProxy) acceptConnections() {
	for {
		clientConn, err := dp.listener.Accept()
		if err != nil {
			dp.agentRef.logger.Printf("❌ AGENT: Proxy %s accept error: %v", dp.Name, err)
			continue
		}

		dp.agentRef.logger.Printf("🔗 AGENT: New connection to proxy %s from %s", dp.Name, clientConn.RemoteAddr())
		go dp.handleConnection(clientConn)
	}
}

func (dp *DatabaseProxy) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	sessionID := generateAgentID()
	clientIP := clientConn.RemoteAddr().String()

	// Connect to target database
	targetAddr := fmt.Sprintf("%s:%d", dp.TargetHost, dp.TargetPort)
	serverConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		dp.agentRef.logger.Printf("❌ AGENT: Failed to connect to target %s: %v", targetAddr, err)
		return
	}
	defer serverConn.Close()

	dp.agentRef.logger.Printf("✅ AGENT: Connected to target database %s", targetAddr)

	// Start bidirectional forwarding with inspection
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		dp.inspectAndForward(clientConn, serverConn, "client->server", sessionID, clientIP)
	}()

	go func() {
		defer wg.Done()
		dp.inspectAndForward(serverConn, clientConn, "server->client", sessionID, clientIP)
	}()

	wg.Wait()
	dp.agentRef.logger.Printf("🔒 AGENT: Proxy session %s closed", sessionID)
}

func (dp *DatabaseProxy) inspectAndForward(src, dst net.Conn, direction, sessionID, clientIP string) {
	buffer := make([]byte, 4096)
	
	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				dp.agentRef.logger.Printf("❌ AGENT: Read error in %s: %v", direction, err)
			}
			break
		}

		data := buffer[:n]

		// Inspect SQL commands (only from client to server)
		if direction == "client->server" {
			if dp.Protocol == "mysql" {
				dp.inspectMySQLCommands(data, sessionID, clientIP)
			} else if dp.Protocol == "postgresql" {
				dp.inspectPostgreSQLCommands(data, sessionID, clientIP)
			}
		}

		// Forward data
		if _, err := dst.Write(data); err != nil {
			dp.agentRef.logger.Printf("❌ AGENT: Write error in %s: %v", direction, err)
			break
		}
	}
}

func (dp *DatabaseProxy) inspectMySQLCommands(data []byte, sessionID, clientIP string) {
	if len(data) < 5 {
		return
	}

	// MySQL command packet structure: [length:3][seq:1][command:1][payload:n]
	if data[4] == 0x03 { // COM_QUERY
		sqlQuery := string(data[5:])
		sqlQuery = strings.TrimSpace(sqlQuery)
		
		if sqlQuery != "" {
			dp.agentRef.logDatabaseCommand(sessionID, sqlQuery, "mysql", clientIP, dp.Name)
		}
	}
}

func (dp *DatabaseProxy) inspectPostgreSQLCommands(data []byte, sessionID, clientIP string) {
	if len(data) < 5 {
		return
	}

	// PostgreSQL message format: [type:1][length:4][payload:n]
	messageType := data[0]
	
	switch messageType {
	case 'Q': // Simple query
		if len(data) > 5 {
			sqlQuery := string(data[5:])
			// Remove null terminator
			if idx := strings.Index(sqlQuery, "\x00"); idx != -1 {
				sqlQuery = sqlQuery[:idx]
			}
			sqlQuery = strings.TrimSpace(sqlQuery)
			
			if sqlQuery != "" {
				dp.agentRef.logDatabaseCommand(sessionID, sqlQuery, "postgresql", clientIP, dp.Name)
			}
		}
	case 'P': // Parse (prepared statement)
		if len(data) > 9 {
			// Extract statement name and query
			payload := data[5:]
			parts := strings.Split(string(payload), "\x00")
			if len(parts) > 1 {
				sqlQuery := strings.TrimSpace(parts[1])
				if sqlQuery != "" {
					dp.agentRef.logDatabaseCommand(sessionID, sqlQuery, "postgresql", clientIP, dp.Name)
				}
			}
		}
	}
}

func (a *GoTeleportAgent) logDatabaseCommand(sessionID, command, protocol, clientIP, proxyName string) {
	// Log to agent log with detailed format
	a.logEvent("DB_COMMAND", command, 
		fmt.Sprintf("Agent: %s | Protocol: %s | Proxy: %s | Client: %s | SessionID: %s", 
			a.config.AgentName, protocol, proxyName, clientIP, sessionID))

	// Send to server
	msg := Message{
		Type:    "db_command",
		AgentID: a.agentID,
		Data:    command,
		Metadata: map[string]interface{}{
			"session_id": sessionID,
			"protocol":   protocol,
			"client_ip":  clientIP,
			"proxy_name": proxyName,
			"agent_name": a.config.AgentName,
		},
		Timestamp: time.Now(),
	}

	if err := a.conn.WriteJSON(msg); err != nil {
		a.logger.Printf("❌ AGENT: Failed to send DB command to server: %v", err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: goteleport-agent-db <config_file>")
		os.Exit(1)
	}

	configFile := os.Args[1]
	agent, err := NewGoTeleportAgent(configFile)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	// Keep running
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "quit" || input == "exit" {
			break
		}
	}

	agent.logger.Printf("🛑 Agent stopping...")
}
