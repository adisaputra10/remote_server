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
    "strings"
    "sync"
    "syscall"
    "time"

    "ssh-tunnel/internal/common"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
    _ "github.com/go-sql-driver/mysql"
)

type RelayServer struct {
    agents   map[string]*Agent
    clients  map[string]*Client
    sessions map[string]*Session
    mutex    sync.RWMutex
    logger   *common.Logger
    db       *sql.DB
    webSessions map[string]*WebSession // Enhanced session storage with user info
}

type WebSession struct {
    Username string
    Role     string
    LoginTime time.Time
}

type Agent struct {
    ID         string    `json:"id"`
    Connection *websocket.Conn `json:"-"`
    ConnectedAt time.Time `json:"connected_at"`
    LastPing   time.Time `json:"last_ping"`
    Status     string    `json:"status"`
}

type Client struct {
    ID         string    `json:"id"`
    Name       string    `json:"name"`
    Connection *websocket.Conn `json:"-"`
    ConnectedAt time.Time `json:"connected_at"`
    LastPing   time.Time `json:"last_ping"`
    AgentID    string    `json:"agent_id"`
    LocalPort  string    `json:"local_port"`
    TargetAddr string    `json:"target_addr"`
    Status     string    `json:"status"`
}

type Session struct {
    ID       string
    AgentID  string
    ClientID string
    Target   string
    Created  time.Time
}

type QueryLogRequest struct {
    SessionID   string `json:"session_id"`
    ClientID    string `json:"client_id"`
    AgentID     string `json:"agent_id"`
    Direction   string `json:"direction"`
    Protocol    string `json:"protocol"`
    Operation   string `json:"operation"`
    TableName   string `json:"table_name"`
    QueryText   string `json:"query_text"`
}

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow connections from any origin
    },
}

func NewRelayServer() *RelayServer {
    rs := &RelayServer{
        agents:   make(map[string]*Agent),
        clients:  make(map[string]*Client),
        sessions: make(map[string]*Session),
        logger:   common.NewLogger("RELAY"),
        webSessions: make(map[string]*WebSession),
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
        dbName = "logs"
    }
    
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
    }
    
    for _, query := range queries {
        _, err := rs.db.Exec(query)
        if err != nil {
            rs.logger.Error("Failed to create table: %v", err)
        }
    }
    
    // Add new columns to tunnel_logs if they don't exist
    alterQueries := []string{
        `ALTER TABLE tunnel_logs ADD COLUMN agent_id VARCHAR(100) AFTER session_id`,
        `ALTER TABLE tunnel_logs ADD COLUMN client_id VARCHAR(100) AFTER agent_id`,
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

func (rs *RelayServer) logTunnelQuery(sessionID, agentID, clientID, direction, protocol, operation, tableName, queryText string) {
    // Clean all string parameters before processing
    sessionID = rs.cleanString(sessionID)
    agentID = rs.cleanString(agentID)
    clientID = rs.cleanString(clientID)
    direction = rs.cleanString(direction)
    protocol = rs.cleanString(protocol)
    operation = rs.cleanOperation(operation) // Use special cleaning for operation
    tableName = rs.cleanString(tableName)
    queryText = rs.cleanString(queryText)
    
    // Check if this operation should be saved to database
    if !rs.isAllowedOperation(operation) {
        rs.logger.Debug("Skipping operation '%s' - not in allowed list", operation)
        return
    }
    
    _, err := rs.db.Exec(
        "INSERT INTO tunnel_logs (session_id, agent_id, client_id, direction, protocol, operation, table_name, query_text) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
        sessionID, agentID, clientID, direction, protocol, operation, tableName, queryText,
    )
    if err != nil {
        rs.logger.Error("Failed to log tunnel query: %v", err)
    } else {
        rs.logger.Debug("Logged operation '%s' to database", operation)
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
        _, messageData, err := conn.ReadMessage()
        if err != nil {
            rs.logger.Error("Failed to read message from %s: %v", r.RemoteAddr, err)
            break
        }

        rs.logger.Debug("Received raw message: %s", string(messageData))

        message, err := common.FromJSON(messageData)
        if err != nil {
            rs.logger.Error("Failed to parse message from %s: %v", r.RemoteAddr, err)
            continue
        }

        rs.handleMessage(conn, message)
    }

    // Clean up connection
    rs.logger.Info("Cleaning up connection from %s", r.RemoteAddr)
    rs.cleanupConnection(conn)
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
    default:
        rs.logger.Error("Unknown message type: %s", msg.Type)
    }
}

func (rs *RelayServer) handleRegister(conn *websocket.Conn, msg *common.Message) {
    rs.mutex.Lock()
    defer rs.mutex.Unlock()

    if msg.AgentID != "" {
        agent := &Agent{
            ID:         msg.AgentID,
            Connection: conn,
            ConnectedAt: time.Now(),
            LastPing:   time.Now(),
            Status:     "connected",
        }
        rs.agents[msg.AgentID] = agent
        rs.logger.Info("Agent registered: %s", msg.AgentID)
        
        // Log to database
        rs.logConnection("agent", msg.AgentID, "", "connected", "")
    } else if msg.ClientID != "" {
        client := &Client{
            ID:         msg.ClientID,
            Name:       msg.ClientName,
            Connection: conn,
            ConnectedAt: time.Now(),
            LastPing:   time.Now(),
            AgentID:    msg.AgentID,
            Status:     "connected",
        }
        rs.clients[msg.ClientID] = client
        rs.logger.Info("Client registered: %s (name: %s)", msg.ClientID, msg.ClientName)
        
        // Log to database
        rs.logConnection("client", "", msg.ClientID, "connected", fmt.Sprintf("target_agent: %s", msg.AgentID))
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

    // Forward connect message to agent
    if agentConn, exists := rs.agents[msg.AgentID]; exists {
        connectMsg := common.NewMessage(common.MsgTypeConnect)
        connectMsg.SessionID = session.ID
        connectMsg.ClientID = msg.ClientID
        connectMsg.Target = msg.Target
        rs.logger.Debug("Forwarding connect message to agent %s", msg.AgentID)
        rs.sendMessage(agentConn.Connection, connectMsg)
    } else {
        rs.logger.Error("Agent not found: %s", msg.AgentID)
        errorMsg := common.NewMessage(common.MsgTypeError)
        errorMsg.SessionID = session.ID
        errorMsg.Error = "Agent not available"
        rs.sendMessage(conn, errorMsg)
    }
}

func (rs *RelayServer) handleData(conn *websocket.Conn, msg *common.Message) {
    rs.mutex.RLock()
    session, exists := rs.sessions[msg.SessionID]
    rs.mutex.RUnlock()

    if !exists {
        rs.logger.Error("Session not found: %s", msg.SessionID)
        return
    }

    rs.logger.Debug("Forwarding data for session %s: %d bytes", msg.SessionID, len(msg.Data))

    var targetConn *websocket.Conn
    rs.mutex.RLock()
    if msg.ClientID != "" {
        // Data from client to agent
        if agent, exists := rs.agents[session.AgentID]; exists {
            targetConn = agent.Connection
        }
        rs.logger.Debug("Forwarding client data to agent %s", session.AgentID)
    } else if msg.AgentID != "" {
        // Data from agent to client
        if client, exists := rs.clients[session.ClientID]; exists {
            targetConn = client.Connection
        }
        rs.logger.Debug("Forwarding agent data to client %s", session.ClientID)
    }
    rs.mutex.RUnlock()

    if targetConn != nil {
        rs.sendMessage(targetConn, msg)
    } else {
        rs.logger.Error("Target connection not found for session: %s", msg.SessionID)
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
    // Log database query to tunnel_logs table
    rs.logTunnelQuery(msg.SessionID, msg.AgentID, msg.ClientID, "inbound", msg.DBProtocol, msg.DBOperation, msg.DBTable, msg.DBQuery)
    
    // Also log to connection_logs for visibility
    clientID := msg.ClientID
    if clientID == "" {
        clientID = "unknown"
    }
    agentID := msg.AgentID
    if agentID == "" {
        agentID = "unknown"
    }
    
    details := fmt.Sprintf("Query: %s, Table: %s, Protocol: %s", 
        msg.DBQuery, msg.DBTable, msg.DBProtocol)
    rs.logConnection("database", agentID, clientID, "database_query", details)
    
    rs.logger.Info("Database query logged from client %s: %s %s", 
        clientID, msg.DBOperation, msg.DBTable)
}

func (rs *RelayServer) sendMessage(conn *websocket.Conn, msg *common.Message) {
    data, err := msg.ToJSON()
    if err != nil {
        rs.logger.Error("Failed to serialize message: %v", err)
        return
    }

    if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
        rs.logger.Error("Failed to send message: %v", err)
    }
}

func (rs *RelayServer) cleanupConnection(conn *websocket.Conn) {
    rs.mutex.Lock()
    defer rs.mutex.Unlock()

    // Remove from agents
    for agentID, agent := range rs.agents {
        if agent.Connection == conn {
            delete(rs.agents, agentID)
            rs.logger.Info("Agent disconnected: %s", agentID)
            rs.logConnection("agent", agentID, "", "disconnected", "")
            break
        }
    }

    // Remove from clients
    for clientID, client := range rs.clients {
        if client.Connection == conn {
            delete(rs.clients, clientID)
            rs.logger.Info("Client disconnected: %s", clientID)
            rs.logConnection("client", "", clientID, "disconnected", "")
            break
        }
    }

    // Clean up sessions
    for sessionID, session := range rs.sessions {
        // Check if this connection belongs to this session
        var sessionBelongsToConn bool
        if agent, exists := rs.agents[session.AgentID]; exists && agent.Connection == conn {
            sessionBelongsToConn = true
        }
        if client, exists := rs.clients[session.ClientID]; exists && client.Connection == conn {
            sessionBelongsToConn = true
        }
        if sessionBelongsToConn {
            delete(rs.sessions, sessionID)
            rs.logger.Info("Session cleaned up: %s", sessionID)
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
    http.HandleFunc("/api/clients", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPIClients)))
    http.HandleFunc("/api/logs", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPILogs)))
    http.HandleFunc("/api/tunnel-logs", rs.corsMiddleware(rs.requireAPIAuth(rs.handleAPITunnelLogs)))
    http.HandleFunc("/api/log-query", rs.corsMiddleware(rs.handleAPILogQuery))
    
    // Health endpoint
    http.HandleFunc("/health", rs.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
        response := map[string]interface{}{
            "status": "healthy",
            "timestamp": time.Now(),
            "agents": len(rs.agents),
            "clients": len(rs.clients),
            "sessions": len(rs.sessions),
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }))
}

// API Authentication Middleware (supports Basic Auth)
func (rs *RelayServer) requireAPIAuth(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
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
                "token": token,
                "user": map[string]string{
                    "username": username,
                    "role": dbRole,
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
    rs.mutex.RLock()
    agents := make([]*Agent, 0, len(rs.agents))
    for _, agent := range rs.agents {
        agents = append(agents, agent)
    }
    rs.mutex.RUnlock()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(agents)
}

func (rs *RelayServer) handleAPIClients(w http.ResponseWriter, r *http.Request) {
    rs.mutex.RLock()
    clients := make([]*Client, 0, len(rs.clients))
    for _, client := range rs.clients {
        clients = append(clients, client)
    }
    rs.mutex.RUnlock()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(clients)
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
    rows, err := rs.db.Query("SELECT session_id, agent_id, client_id, direction, protocol, operation, table_name, query_text, timestamp FROM tunnel_logs ORDER BY timestamp DESC LIMIT 100")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var logs []map[string]interface{}
    for rows.Next() {
        var sessionID, agentID, clientID, direction, protocol, operation, tableName, queryText sql.NullString
        var timestamp time.Time
        
        err := rows.Scan(&sessionID, &agentID, &clientID, &direction, &protocol, &operation, &tableName, &queryText, &timestamp)
        if err != nil {
            continue
        }
        
        // Clean HTML entities and trim whitespace from query text
        cleanedQueryText := rs.cleanString(cleanHTMLEntities(queryText.String))
        
        log := map[string]interface{}{
            "session_id":  rs.cleanString(sessionID.String),
            "agent_id":    rs.cleanString(agentID.String),
            "client_id":   rs.cleanString(clientID.String),
            "direction":   rs.cleanString(direction.String),
            "protocol":    rs.cleanString(protocol.String),
            "operation":   rs.cleanOperation(operation.String),
            "table_name":  rs.cleanString(tableName.String),
            "query_text":  cleanedQueryText,
            "timestamp":   timestamp,
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
    rs.logTunnelQuery(req.SessionID, req.AgentID, req.ClientID, req.Direction, req.Protocol, req.Operation, req.TableName, req.QueryText)
    
    // Return success response
    response := map[string]interface{}{
        "status":  "success",
        "message": "Query logged successfully",
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