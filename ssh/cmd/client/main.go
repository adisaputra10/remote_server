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
    "os/signal"
    "strings"
    "sync"
    "syscall"
    "time"

    "ssh-tunnel/internal/common"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
)

type Client struct {
    id         string
    name       string
    relayURL   string
    conn       *websocket.Conn
    sessions   map[string]net.Conn
    dbLoggers  map[string]*common.DatabaseQueryLogger
    targets    map[string]string  // sessionID -> target
    agentIDs   map[string]string  // sessionID -> agentID
    mutex      sync.RWMutex
    logger     *common.Logger
    running    bool
    heartbeat  *time.Ticker
}

type QueryLogData struct {
    SessionID   string `json:"session_id"`
    ClientID    string `json:"client_id"`
    AgentID     string `json:"agent_id"`
    Direction   string `json:"direction"`
    Protocol    string `json:"protocol"`
    Operation   string `json:"operation"`
    TableName   string `json:"table_name"`
    QueryText   string `json:"query_text"`
}

func NewClient(id, name, relayURL string) *Client {
    return &Client{
        id:        id,
        name:      name,
        relayURL:  relayURL,
        sessions:  make(map[string]net.Conn),
        dbLoggers: make(map[string]*common.DatabaseQueryLogger),
        targets:   make(map[string]string),
        agentIDs:  make(map[string]string),
        logger:    common.NewLogger(fmt.Sprintf("CLIENT-%s", id)),
    }
}

func (c *Client) connect() error {
    _, err := url.Parse(c.relayURL)
    if err != nil {
        return fmt.Errorf("invalid relay URL: %v", err)
    }

    c.logger.Info("Connecting to relay server: %s", c.relayURL)
    
    conn, _, err := websocket.DefaultDialer.Dial(c.relayURL, nil)
    if err != nil {
        return fmt.Errorf("failed to connect to relay: %v", err)
    }

    c.conn = conn
    c.running = true

    // Register with relay
    registerMsg := common.NewMessage(common.MsgTypeRegister)
    registerMsg.ClientID = c.id
    registerMsg.ClientName = c.name
    if err := c.sendMessage(registerMsg); err != nil {
        return fmt.Errorf("failed to register: %v", err)
    }

    c.logger.Info("Successfully connected and registered with relay")
    return nil
}

func (c *Client) start() error {
    if err := c.connect(); err != nil {
        return err
    }

    // Start heartbeat
    c.heartbeat = time.NewTicker(30 * time.Second)
    go c.heartbeatLoop()

    // Start message handler
    go c.messageLoop()

    c.logger.Info("Client started successfully")
    return nil
}

func (c *Client) stop() {
    c.running = false
    
    if c.heartbeat != nil {
        c.heartbeat.Stop()
    }

    if c.conn != nil {
        c.conn.Close()
    }

    // Close all sessions
    c.mutex.Lock()
    for sessionID, conn := range c.sessions {
        conn.Close()
        delete(c.sessions, sessionID)
    }
    c.mutex.Unlock()

    c.logger.Info("Client stopped")
}

func (c *Client) heartbeatLoop() {
    for c.running {
        select {
        case <-c.heartbeat.C:
            heartbeatMsg := common.NewMessage(common.MsgTypeHeartbeat)
            heartbeatMsg.ClientID = c.id
            if err := c.sendMessage(heartbeatMsg); err != nil {
                c.logger.Error("Failed to send heartbeat: %v", err)
            }
        }
    }
}

func (c *Client) messageLoop() {
    defer c.stop()

    for c.running {
        _, messageData, err := c.conn.ReadMessage()
        if err != nil {
            if c.running {
                c.logger.Error("Failed to read message: %v", err)
            }
            break
        }

        message, err := common.FromJSON(messageData)
        if err != nil {
            c.logger.Error("Failed to parse message: %v", err)
            continue
        }

        c.handleMessage(message)
    }
}

func (c *Client) handleMessage(msg *common.Message) {
    c.logger.Debug("Received message: %s", msg.String())

    switch msg.Type {
    case common.MsgTypeRegister:
        c.logger.Debug("Registration confirmation received")
    case common.MsgTypeData:
        c.handleData(msg)
    case common.MsgTypeClose:
        c.handleClose(msg)
    case common.MsgTypeError:
        c.handleError(msg)
    case common.MsgTypeHeartbeat:
        // Heartbeat response received
        c.logger.Debug("Heartbeat response received")
    default:
        c.logger.Error("Unknown message type: %s", msg.Type)
    }
}

func (c *Client) handleData(msg *common.Message) {
    c.mutex.RLock()
    conn, exists := c.sessions[msg.SessionID]
    dbLogger, dbExists := c.dbLoggers[msg.SessionID]
    c.mutex.RUnlock()

    if !exists {
        c.logger.Error("Session not found: %s - message from agent may be late", msg.SessionID)
        c.logger.Debug("Available sessions: %d", len(c.sessions))
        return
    }

    c.logger.Debug("Received data for session %s: %d bytes", msg.SessionID, len(msg.Data))

    // Log database response if logger exists
    if dbExists {
        dbLogger.LogData(msg.Data, "AGENT->CLIENT", msg.SessionID)
        
        // Database response logging is now handled by the DatabaseQueryLogger callback
        // No need for manual parsing here
    }

    // Forward data to local connection
    if _, err := conn.Write(msg.Data); err != nil {
        c.logger.Error("Failed to write to local connection for session %s: %v", msg.SessionID, err)
        c.closeSession(msg.SessionID)
    } else {
        c.logger.Debug("Successfully forwarded %d bytes to local connection", len(msg.Data))
    }
}

func (c *Client) handleClose(msg *common.Message) {
    c.logger.Info("Remote closed session: %s", msg.SessionID)
    c.closeSession(msg.SessionID)
}

func (c *Client) handleError(msg *common.Message) {
    c.logger.Error("Received error: %s", msg.Error)
    if msg.SessionID != "" {
        c.closeSession(msg.SessionID)
    }
}

func (c *Client) startLocalListener(localAddr, agentID, target string) error {
    listener, err := net.Listen("tcp", localAddr)
    if err != nil {
        return fmt.Errorf("failed to listen on %s: %v", localAddr, err)
    }
    defer listener.Close()

    c.logger.Info("Local tunnel listening on %s -> Agent: %s, Target: %s", localAddr, agentID, target)

    for {
        conn, err := listener.Accept()
        if err != nil {
            c.logger.Error("Failed to accept connection: %v", err)
            continue
        }

        go c.handleLocalConnection(conn, agentID, target)
    }
}

func (c *Client) handleLocalConnection(conn net.Conn, agentID, target string) {
    sessionID := common.GenerateID()
    
    c.mutex.Lock()
    c.sessions[sessionID] = conn
    // Create database query logger for this session
    dbLogger := common.NewDatabaseQueryLogger(c.logger, target)
    // Set callback to send clean query data to relay
    dbLogger.SetCallback(func(sessionID, operation, tableName, query, protocol, direction string) {
        c.mutex.RLock()
        agentID := c.agentIDs[sessionID]
        c.mutex.RUnlock()
        
        if agentID != "" {
            logData := QueryLogData{
                SessionID: sessionID,
                ClientID:  c.id,
                AgentID:   agentID,
                Direction: direction,
                Protocol:  protocol,
                Operation: operation,
                TableName: tableName,
                QueryText: query,
            }
            c.sendQueryLogToAPI(logData)
        }
    })
    c.dbLoggers[sessionID] = dbLogger
    // Store target and agent for this session
    c.targets[sessionID] = target
    c.agentIDs[sessionID] = agentID
    c.mutex.Unlock()

    c.logger.Info("New local connection accepted, session: %s, target: %s", sessionID, target)
    c.logger.Debug("Local connection from: %s", conn.RemoteAddr().String())

    // Send connect request to relay
    connectMsg := common.NewMessage(common.MsgTypeConnect)
    connectMsg.SessionID = sessionID
    connectMsg.ClientID = c.id
    connectMsg.AgentID = agentID
    connectMsg.Target = target

    c.logger.Info("Sending connect request to relay - Agent: %s, Target: %s", agentID, target)

    if err := c.sendMessage(connectMsg); err != nil {
        c.logger.Error("Failed to send connect message: %v", err)
        conn.Close()
        c.closeSession(sessionID)
        return
    }

    // Start forwarding data from local connection to relay
    c.forwardFromLocal(sessionID, conn)
}

func (c *Client) forwardFromLocal(sessionID string, conn net.Conn) {
    defer c.closeSession(sessionID)

    c.logger.Debug("Starting data forwarding from local connection for session %s", sessionID)
    buffer := make([]byte, 32*1024) // 32KB buffer

    for {
        n, err := conn.Read(buffer)
        if err != nil {
            if err != io.EOF {
                c.logger.Error("Failed to read from local connection for session %s: %v", sessionID, err)
            } else {
                c.logger.Debug("Local connection closed for session %s", sessionID)
            }
            break
        }

        if n > 0 {
            c.logger.Debug("Read %d bytes from local connection for session %s", n, sessionID)
            
            // Log database query if logger exists
            c.mutex.RLock()
            dbLogger, dbExists := c.dbLoggers[sessionID]
            c.mutex.RUnlock()
            
            if dbExists {
                dbLogger.LogData(buffer[:n], "CLIENT->AGENT", sessionID)
                
                // Database query logging is now handled by the DatabaseQueryLogger callback
                // No need for manual parsing here
            }
            
            dataMsg := common.NewMessage(common.MsgTypeData)
            dataMsg.SessionID = sessionID
            dataMsg.ClientID = c.id
            dataMsg.Data = make([]byte, n)
            copy(dataMsg.Data, buffer[:n])

            if err := c.sendMessage(dataMsg); err != nil {
                c.logger.Error("Failed to forward data to relay for session %s: %v", sessionID, err)
                break
            }
            c.logger.Debug("Successfully forwarded %d bytes to relay", n)
        }
    }

    // Connection closed, notify relay
    c.logger.Info("Local connection closed for session %s, notifying relay", sessionID)
    closeMsg := common.NewMessage(common.MsgTypeClose)
    closeMsg.SessionID = sessionID
    closeMsg.ClientID = c.id
    c.sendMessage(closeMsg)
}

func (c *Client) closeSession(sessionID string) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if conn, exists := c.sessions[sessionID]; exists {
        conn.Close()
        delete(c.sessions, sessionID)
        c.logger.Info("Session closed: %s", sessionID)
    }
    
    // Clean up database logger
    if dbLogger, exists := c.dbLoggers[sessionID]; exists {
        _ = dbLogger // Use the variable to avoid unused warning
        delete(c.dbLoggers, sessionID)
        c.logger.Debug("Database logger cleaned up for session: %s", sessionID)
    }
    
    // Clean up target mapping
    if _, exists := c.targets[sessionID]; exists {
        delete(c.targets, sessionID)
        c.logger.Debug("Target mapping cleaned up for session: %s", sessionID)
    }
    
    // Clean up agent ID mapping
    if _, exists := c.agentIDs[sessionID]; exists {
        delete(c.agentIDs, sessionID)
        c.logger.Debug("Agent ID mapping cleaned up for session: %s", sessionID)
    }
}

func (c *Client) sendMessage(msg *common.Message) error {
    data, err := msg.ToJSON()
    if err != nil {
        return fmt.Errorf("failed to serialize message: %v", err)
    }

    return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) sendQueryLogToAPI(logData QueryLogData) {
    // Parse relay URL to get the base URL
    u, err := url.Parse(c.relayURL)
    if err != nil {
        c.logger.Error("Failed to parse relay URL: %v", err)
        return
    }
    
    // Convert ws:// to http://
    apiURL := strings.Replace(u.String(), "ws://", "http://", 1)
    apiURL = strings.Replace(apiURL, "/ws/client", "/api/log-query", 1)
    
    jsonData, err := json.Marshal(logData)
    if err != nil {
        c.logger.Error("Failed to marshal query log data: %v", err)
        return
    }
    
    resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        c.logger.Error("Failed to send query log to API: %v", err)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        c.logger.Error("API returned error status: %d", resp.StatusCode)
        return
    }
    
    c.logger.Debug("Successfully sent query log via API: %s %s", logData.Operation, logData.Protocol)
}

func (c *Client) extractSQLInfo(queryText string) (operation, tableName string) {
    // Simple SQL parser - extract operation and table name
    query := strings.TrimSpace(strings.ToUpper(queryText))
    
    // Skip MySQL protocol headers and binary data
    if len(query) < 10 || query[0] < 32 {
        return "", ""
    }
    
    words := strings.Fields(query)
    if len(words) < 2 {
        return "", ""
    }
    
    operation = words[0]
    
    switch operation {
    case "SELECT":
        // Find FROM keyword
        for i, word := range words {
            if word == "FROM" && i+1 < len(words) {
                tableName = strings.Trim(words[i+1], ",();")
                break
            }
        }
    case "INSERT":
        // Find INTO keyword
        for i, word := range words {
            if word == "INTO" && i+1 < len(words) {
                tableName = strings.Trim(words[i+1], ",();")
                break
            }
        }
    case "UPDATE", "DELETE":
        // Table name usually follows UPDATE or DELETE FROM
        if len(words) > 1 {
            if operation == "DELETE" && words[1] == "FROM" && len(words) > 2 {
                tableName = strings.Trim(words[2], ",();")
            } else if operation == "UPDATE" {
                tableName = strings.Trim(words[1], ",();")
            }
        }
    case "CREATE", "DROP", "ALTER":
        // Find table name after TABLE keyword
        for i, word := range words {
            if word == "TABLE" && i+1 < len(words) {
                tableName = strings.Trim(words[i+1], ",();")
                break
            }
        }
    }
    
    return operation, tableName
}

func (c *Client) isValidSQL(queryText string) bool {
    // Check if string contains mostly printable characters and SQL keywords
    if len(queryText) < 3 {
        return false
    }
    
    // Check for common SQL keywords
    upperQuery := strings.ToUpper(strings.TrimSpace(queryText))
    sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "SHOW", "DESCRIBE", "EXPLAIN", "COMMIT", "ROLLBACK", "START", "BEGIN", "PREPARE", "EXECUTE"}
    
    for _, keyword := range sqlKeywords {
        if strings.HasPrefix(upperQuery, keyword) {
            return true
        }
    }
    
    return false
}

func (c *Client) detectProtocol(target string) string {
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

func main() {
    var (
        clientID   string
        clientName string
        relayURL   string
        localAddr  string
        agentID    string
        target     string
        interactive bool
    )

    var rootCmd = &cobra.Command{
        Use:   "tunnel-client",
        Short: "SSH Tunnel Client",
        Long:  "A client that creates local tunnels and forwards connections through a relay to remote agents",
        Run: func(cmd *cobra.Command, args []string) {
            if clientID == "" {
                clientID = common.GenerateID()
            }
            
            if clientName == "" {
                clientName = fmt.Sprintf("client-%s", clientID[:8])
            }

            client := NewClient(clientID, clientName, relayURL)
            
            // Setup signal handling for graceful shutdown
            sigChan := make(chan os.Signal, 1)
            signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
            
            go func() {
                <-sigChan
                client.logger.Info("Received shutdown signal, stopping client...")
                client.stop()
                client.logger.Close()
                os.Exit(0)
            }()
            
            if err := client.start(); err != nil {
                log.Fatalf("Failed to start client: %v", err)
            }

            if interactive {
                fmt.Println("Interactive mode - Available commands:")
                fmt.Println("  help    - Show this help")
                fmt.Println("  tunnel  - Create a new tunnel")
                fmt.Println("  list    - List active sessions")
                fmt.Println("  quit    - Exit the client")
                
                // Simple interactive mode implementation
                for {
                    fmt.Print("> ")
                    var command string
                    fmt.Scanln(&command)
                    
                    switch command {
                    case "help":
                        fmt.Println("Available commands: help, tunnel, list, quit")
                    case "tunnel":
                        fmt.Print("Local address (e.g., :2222): ")
                        fmt.Scanln(&localAddr)
                        fmt.Print("Agent ID: ")
                        fmt.Scanln(&agentID)
                        fmt.Print("Target (e.g., localhost:22): ")
                        fmt.Scanln(&target)
                        
                        go func() {
                            if err := client.startLocalListener(localAddr, agentID, target); err != nil {
                                fmt.Printf("Failed to start tunnel: %v\n", err)
                            }
                        }()
                        fmt.Printf("Tunnel created: %s -> %s:%s\n", localAddr, agentID, target)
                    case "list":
                        client.mutex.RLock()
                        fmt.Printf("Active sessions: %d\n", len(client.sessions))
                        for sessionID := range client.sessions {
                            fmt.Printf("  - %s\n", sessionID)
                        }
                        client.mutex.RUnlock()
                    case "quit":
                        client.stop()
                        return
                    default:
                        fmt.Println("Unknown command. Type 'help' for available commands.")
                    }
                }
            } else {
                // Single tunnel mode
                if localAddr == "" || agentID == "" || target == "" {
                    log.Fatal("Local address (-L), agent ID (-agent), and target (-target) are required for single tunnel mode")
                }
                
                if err := client.startLocalListener(localAddr, agentID, target); err != nil {
                    log.Fatalf("Failed to start local listener: %v", err)
                }
            }
        },
    }

    rootCmd.Flags().StringVarP(&clientID, "client-id", "c", "", "Client ID (auto-generated if not provided)")
    rootCmd.Flags().StringVarP(&clientName, "name", "n", "", "Client name (auto-generated if not provided)")
    rootCmd.Flags().StringVarP(&relayURL, "relay-url", "r", "ws://localhost:8080/ws/client", "Relay server WebSocket URL")
    rootCmd.Flags().StringVarP(&localAddr, "local", "L", "", "Local address to listen on (e.g., :2222)")
    rootCmd.Flags().StringVarP(&agentID, "agent", "a", "", "Target agent ID")
    rootCmd.Flags().StringVarP(&target, "target", "t", "", "Target address (e.g., localhost:22)")
    rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run in interactive mode")

    if err := rootCmd.Execute(); err != nil {
        log.Fatal(err)
    }
}