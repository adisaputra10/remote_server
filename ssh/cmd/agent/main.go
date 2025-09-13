package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"remote-tunnel/internal/logger"
	"remote-tunnel/internal/tunnel"
)

type TunnelAgent struct {
	id        string
	name      string
	relayURL  string
	logger    *logger.Logger
	insecure  bool
	allows    []string
	transport *tunnel.Transport
	
	// Database query logging
	dbLogger    *logger.Logger
	queryLog    bool
	queryFilter []string
}

func main() {
	var (
		id          = flag.String("id", "", "Agent ID (auto-generated if empty)")
		name        = flag.String("name", "", "Agent name (defaults to hostname)")
		relayURL    = flag.String("relay-url", "wss://localhost:8443/ws/agent", "Relay server WebSocket URL")
		insecure    = flag.Bool("insecure", false, "Skip TLS certificate verification")
		allow       = flag.String("allow", "127.0.0.1:22,127.0.0.1:3306,127.0.0.1:5432", "Comma-separated list of allowed target addresses")
		logQueries  = flag.Bool("log-queries", true, "Enable database query logging")
		queryFilter = flag.String("query-filter", "SELECT,INSERT,UPDATE,DELETE", "Comma-separated list of query types to log")
	)
	flag.Parse()

	log := logger.New("AGENT")
	
	if *insecure {
		log.Warn("🔓 INSECURE mode enabled - TLS certificate verification disabled!")
	}

	// Generate ID if not provided
	agentID := *id
	if agentID == "" {
		agentID = fmt.Sprintf("agent_%d", time.Now().UnixNano())
	}

	// Get agent name
	agentName := *name
	if agentName == "" {
		hostname, _ := os.Hostname()
		agentName = hostname
		if agentName == "" {
			agentName = "unknown"
		}
	}

	// Parse allowed targets
	allows := strings.Split(*allow, ",")
	for i, addr := range allows {
		allows[i] = strings.TrimSpace(addr)
	}

	// Parse query filter
	var filters []string
	if *logQueries {
		filters = strings.Split(*queryFilter, ",")
		for i, filter := range filters {
			filters[i] = strings.TrimSpace(strings.ToUpper(filter))
		}
	}

	agent := &TunnelAgent{
		id:          agentID,
		name:        agentName,
		relayURL:    *relayURL,
		logger:      log,
		insecure:    *insecure,
		allows:      allows,
		dbLogger:    logger.New("DB-QUERY"),
		queryLog:    *logQueries,
		queryFilter: filters,
	}

	log.Info("🚀 Starting tunnel agent")
	log.Info("📋 Agent ID: %s", agentID)
	log.Info("📋 Agent Name: %s", agentName)
	log.Info("📋 Relay URL: %s", *relayURL)
	log.Info("📋 Allowed targets: %v", allows)
	
	if *logQueries {
		log.Info("📊 Database query logging: ENABLED")
		log.Info("📊 Query filters: %v", filters)
	} else {
		log.Info("📊 Database query logging: DISABLED")
	}

	// Connect to relay
	if err := agent.connect(); err != nil {
		log.Error("❌ Failed to connect to relay: %v", err)
		os.Exit(1)
	}

	// Handle interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info("🛑 Shutting down agent...")
	agent.disconnect()
	log.Info("👋 Agent stopped")
}

func (a *TunnelAgent) connect() error {
	a.logger.Info("🔗 Connecting to relay server...")

	// Setup WebSocket dialer
	dialer := websocket.DefaultDialer
	if a.insecure {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		a.logger.Warn("⚠️ TLS certificate verification disabled")
	}

	// Connect to WebSocket
	conn, _, err := dialer.Dial(a.relayURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket dial failed: %v", err)
	}

	a.logger.Info("✅ Connected to relay server")

	// Create transport
	transport, err := tunnel.NewTransport(conn, true, a.logger)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create transport: %v", err)
	}

	a.transport = transport

	// Set up database query logging if enabled
	if a.queryLog {
		transport.SetQueryLogger(func(data []byte, targetAddr string) {
			a.logDatabaseQuery(data, targetAddr)
		})
		a.logger.Info("📊 Query logger attached to transport")
	}

	// Register with relay
	if err := a.register(); err != nil {
		a.transport.Close()
		return fmt.Errorf("registration failed: %v", err)
	}

	// Start message handler
	go a.handleMessages()

	// Start heartbeat
	go a.heartbeat()

	return nil
}

func (a *TunnelAgent) register() error {
	a.logger.Command("SEND", "REGISTER", a.id)

	info := &tunnel.AgentInfo{
		ID:      a.id,
		Name:    a.name,
		Status:  "online",
		Targets: a.allows,
		LastSeen: time.Now(),
	}

	msg := tunnel.NewMessage(tunnel.MsgAgentRegister).
		SetAgentID(a.id).
		SetData(info)

	if err := a.transport.SendMessage(msg); err != nil {
		return fmt.Errorf("send register message: %v", err)
	}

	// Wait for registration response
	response, err := a.transport.ReceiveMessage()
	if err != nil {
		return fmt.Errorf("receive register response: %v", err)
	}

	a.logger.Command("RECV", response.Type, response.AgentID)

	if response.Type != tunnel.MsgAgentRegistered {
		return fmt.Errorf("unexpected response: %s", response.Type)
	}

	a.logger.Info("✅ Agent registered successfully")
	return nil
}

func (a *TunnelAgent) handleMessages() {
	defer a.transport.Close()

	for {
		msg, err := a.transport.ReceiveMessage()
		if err != nil {
			a.logger.Error("❌ Message receive error: %v", err)
			break
		}

		a.logger.Command("RECV", msg.Type, msg.TunnelID)

		switch msg.Type {
		case tunnel.MsgTunnelRequest:
			go a.handleTunnelRequest(msg)

		case tunnel.MsgTunnelClose:
			a.handleTunnelClose(msg)

		case tunnel.MsgAgentDisconnect:
			a.logger.Info("🔌 Disconnect request from relay")
			return

		default:
			a.logger.Warn("⚠️ Unknown message type: %s", msg.Type)
		}
	}
}

func (a *TunnelAgent) handleTunnelRequest(msg *tunnel.Message) {
	var req tunnel.TunnelRequest
	
	if data, ok := msg.Data.(map[string]interface{}); ok {
		tunnel.MapToStruct(data, &req)
	}

	a.logger.Tunnel("REQUEST", req.TunnelID, fmt.Sprintf("%s:%d", req.RemoteHost, req.RemotePort))

	// Check if target is allowed
	target := fmt.Sprintf("%s:%d", req.RemoteHost, req.RemotePort)
	if !a.isTargetAllowed(target) {
		a.logger.Error("❌ Target not allowed: %s", target)
		
		response := tunnel.NewMessage(tunnel.MsgTunnelError).
			SetTunnelID(req.TunnelID).
			SetClientID(req.ClientID).
			SetError(fmt.Sprintf("Target %s not allowed", target))
		
		a.transport.SendMessage(response)
		return
	}

	// Create tunnel
	tunnelTransport, err := tunnel.NewTunnelTransport(a.transport, req.TunnelID, req.RemoteHost, req.RemotePort, a.logger)
	if err != nil {
		a.logger.Error("❌ Failed to create tunnel: %v", err)
		
		response := tunnel.NewMessage(tunnel.MsgTunnelError).
			SetTunnelID(req.TunnelID).
			SetClientID(req.ClientID).
			SetError(fmt.Sprintf("Failed to create tunnel: %v", err))
		
		a.transport.SendMessage(response)
		return
	}

	// Send success response
	response := tunnel.NewMessage(tunnel.MsgTunnelSuccess).
		SetTunnelID(req.TunnelID).
		SetClientID(req.ClientID).
		SetData("Tunnel created successfully")

	if err := a.transport.SendMessage(response); err != nil {
		a.logger.Error("❌ Failed to send tunnel response: %v", err)
		tunnelTransport.Close()
		return
	}

	a.logger.Tunnel("CREATED", req.TunnelID, fmt.Sprintf("to %s", target))

	// Start tunnel transport
	go tunnelTransport.Start()
}

func (a *TunnelAgent) handleTunnelClose(msg *tunnel.Message) {
	tunnelID := msg.TunnelID
	a.logger.Tunnel("CLOSE", tunnelID, "from relay")
	
	// Tunnel cleanup is handled by transport layer
	// Just log the event
}

func (a *TunnelAgent) isTargetAllowed(target string) bool {
	for _, allowed := range a.allows {
		if allowed == target {
			return true
		}
	}
	return false
}

func (a *TunnelAgent) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		msg := tunnel.NewMessage(tunnel.MsgAgentHeartbeat).SetAgentID(a.id)
		
		if err := a.transport.SendMessage(msg); err != nil {
			a.logger.Error("❌ Heartbeat failed: %v", err)
			break
		}
		
		a.logger.Debug("💓 Heartbeat sent")
	}
}

func (a *TunnelAgent) disconnect() {
	if a.transport != nil {
		msg := tunnel.NewMessage(tunnel.MsgAgentDisconnect).SetAgentID(a.id)
		a.transport.SendMessage(msg)
		a.transport.Close()
	}
}

// Database Query Logging Functions

func (a *TunnelAgent) logDatabaseQuery(data []byte, targetAddr string) {
	if !a.queryLog {
		return
	}

	query := string(data)
	
	// Detect database type by port
	dbType := a.detectDatabaseType(targetAddr)
	
	// Extract and parse SQL queries
	if dbType == "mysql" || dbType == "postgresql" {
		a.parseSQLQuery(query, dbType, targetAddr)
	} else if dbType == "mongodb" {
		a.parseMongoQuery(query, targetAddr)
	}
}

func (a *TunnelAgent) detectDatabaseType(targetAddr string) string {
	if strings.Contains(targetAddr, ":3306") {
		return "mysql"
	} else if strings.Contains(targetAddr, ":5432") {
		return "postgresql"
	} else if strings.Contains(targetAddr, ":27017") {
		return "mongodb"
	} else if strings.Contains(targetAddr, ":1433") {
		return "sqlserver"
	} else if strings.Contains(targetAddr, ":1521") {
		return "oracle"
	}
	return "unknown"
}

func (a *TunnelAgent) parseSQLQuery(query, dbType, targetAddr string) {
	// Remove extra whitespace and newlines
	cleanQuery := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(query), " ")
	
	if len(cleanQuery) < 3 {
		return
	}

	// Extract query type (first word)
	queryType := strings.ToUpper(strings.Split(cleanQuery, " ")[0])
	
	// Check if query type should be logged
	if !a.shouldLogQueryType(queryType) {
		return
	}

	// Extract table name for common queries
	tableName := a.extractTableName(cleanQuery, queryType)
	
	// Log the query with metadata
	a.dbLogger.Info("🗄️ [%s] %s Query: %s", dbType, queryType, a.sanitizeQuery(cleanQuery))
	a.dbLogger.Info("📊 Target: %s | Table: %s | Size: %d bytes", targetAddr, tableName, len(query))
	
	// Log detailed query for debugging (truncated if too long)
	if len(cleanQuery) > 500 {
		a.dbLogger.Debug("📝 Full Query: %s...", cleanQuery[:500])
	} else {
		a.dbLogger.Debug("📝 Full Query: %s", cleanQuery)
	}
}

func (a *TunnelAgent) parseMongoQuery(query, targetAddr string) {
	a.dbLogger.Info("🍃 [MongoDB] Query to %s: %d bytes", targetAddr, len(query))
	
	// Try to extract MongoDB operations
	if strings.Contains(query, "find") {
		a.dbLogger.Info("📊 Operation: FIND")
	} else if strings.Contains(query, "insert") {
		a.dbLogger.Info("📊 Operation: INSERT")
	} else if strings.Contains(query, "update") {
		a.dbLogger.Info("📊 Operation: UPDATE")
	} else if strings.Contains(query, "delete") {
		a.dbLogger.Info("📊 Operation: DELETE")
	}
}

func (a *TunnelAgent) shouldLogQueryType(queryType string) bool {
	if len(a.queryFilter) == 0 {
		return true // Log all if no filter
	}
	
	for _, filter := range a.queryFilter {
		if filter == queryType {
			return true
		}
	}
	return false
}

func (a *TunnelAgent) extractTableName(query, queryType string) string {
	query = strings.ToUpper(query)
	
	switch queryType {
	case "SELECT":
		re := regexp.MustCompile(`FROM\s+([^\s]+)`)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 1 {
			return matches[1]
		}
	case "INSERT":
		re := regexp.MustCompile(`INSERT\s+INTO\s+([^\s]+)`)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 1 {
			return matches[1]
		}
	case "UPDATE":
		re := regexp.MustCompile(`UPDATE\s+([^\s]+)`)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 1 {
			return matches[1]
		}
	case "DELETE":
		re := regexp.MustCompile(`DELETE\s+FROM\s+([^\s]+)`)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return "unknown"
}

func (a *TunnelAgent) sanitizeQuery(query string) string {
	// Remove potential sensitive data (basic sanitization)
	sanitized := query
	
	// Replace common password patterns
	patterns := []string{
		`PASSWORD\s*=\s*'[^']+'`,
		`PASSWORD\s*=\s*"[^"]+"`,
		`PWD\s*=\s*'[^']+'`,
		`PWD\s*=\s*"[^"]+"`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		sanitized = re.ReplaceAllString(sanitized, "PASSWORD='***'")
	}
	
	return sanitized
}
