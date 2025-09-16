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
    id         string
    relayURL   string
    conn       *websocket.Conn
    sessions   map[string]net.Conn
    dbLoggers  map[string]*common.DatabaseQueryLogger
    targets    map[string]string  // sessionID -> target
    mutex      sync.RWMutex
    logger     *common.Logger
    running    bool
    heartbeat  *time.Ticker
}

func NewAgent(id, relayURL string) *Agent {
    return &Agent{
        id:        id,
        relayURL:  relayURL,
        sessions:  make(map[string]net.Conn),
        dbLoggers: make(map[string]*common.DatabaseQueryLogger),
        targets:   make(map[string]string),
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
        _, messageData, err := a.conn.ReadMessage()
        if err != nil {
            if a.running {
                a.logger.Error("Failed to read message: %v", err)
            }
            break
        }

        message, err := common.FromJSON(messageData)
        if err != nil {
            a.logger.Error("Failed to parse message: %v", err)
            continue
        }

        a.handleMessage(message)
    }
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
        
        // Also extract and send query to relay if it looks like SQL
        queryText := string(msg.Data)
        if operation, tableName := a.extractSQLInfo(queryText); operation != "" {
            a.mutex.RLock()
            target := a.targets[msg.SessionID]
            a.mutex.RUnlock()
            protocol := a.detectProtocol(target)
            a.sendDatabaseQuery(msg.SessionID, queryText, operation, tableName, protocol)
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
            }
            
            dataMsg := common.NewMessage(common.MsgTypeData)
            dataMsg.SessionID = sessionID
            dataMsg.AgentID = a.id
            dataMsg.Data = make([]byte, n)
            copy(dataMsg.Data, buffer[:n])

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
}

func (a *Agent) sendMessage(msg *common.Message) error {
    data, err := msg.ToJSON()
    if err != nil {
        return fmt.Errorf("failed to serialize message: %v", err)
    }

    return a.conn.WriteMessage(websocket.TextMessage, data)
}

func (a *Agent) sendDatabaseQuery(sessionID, query, operation, tableName, protocol string) {
    msg := common.NewMessage(common.MsgTypeDBQuery)
    msg.AgentID = a.id
    msg.SessionID = sessionID
    msg.DBQuery = query
    msg.DBOperation = operation
    msg.DBTable = tableName
    msg.DBProtocol = protocol
    
    if err := a.sendMessage(msg); err != nil {
        a.logger.Error("Failed to send database query log: %v", err)
    } else {
        a.logger.Debug("Sent database query log: %s %s %s", operation, tableName, protocol)
    }
}

func (a *Agent) extractSQLInfo(queryText string) (operation, tableName string) {
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

// handleShellCommand executes shell commands on the agent machine
func (a *Agent) handleShellCommand(msg *common.Message) {
    command := msg.DBQuery // Use DBQuery field for command
    a.logger.Info("Executing shell command: %s", command)
    
    if command == "" {
        a.sendShellError(msg.SessionID, msg.ClientID, "Empty command")
        return
    }
    
    // Log SSH command to database
    a.logSSHCommand(msg.SessionID, msg.ClientID, command, "input")
    
    // Execute command using the system shell
    output, err := a.executeSystemCommand(command)
    
    // Log command output
    if err == nil {
        a.logSSHCommand(msg.SessionID, msg.ClientID, output, "output")
    }
    
    // Send response back
    responseMsg := common.NewMessage("shell_response")
    responseMsg.SessionID = msg.SessionID
    responseMsg.ClientID = msg.ClientID
    responseMsg.AgentID = a.id
    responseMsg.DBQuery = command // Store command in DBQuery field
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
    )

    var rootCmd = &cobra.Command{
        Use:   "tunnel-agent",
        Short: "SSH Tunnel Agent",
        Long:  "An agent that runs on the target server and forwards SSH connections through a relay",
        Run: func(cmd *cobra.Command, args []string) {
            if agentID == "" {
                agentID = common.GenerateID()
            }

            agent := NewAgent(agentID, relayURL)
            
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

    if err := rootCmd.Execute(); err != nil {
        log.Fatal(err)
    }
}