package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "strings"
    "time"

    "ssh-tunnel/internal/common"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
)

type SSHClient struct {
    clientID    string
    clientName  string
    relayURL    string
    conn        *websocket.Conn
    sessionID   string
    agentID     string
    sshHost     string
    sshPort     string
    sshUser     string
    sshPassword string
    localPort   string
    logger      *common.Logger
    dbLogger    *common.DatabaseQueryLogger
}

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

type Message struct {
    Type        string `json:"type"`
    ClientID    string `json:"client_id,omitempty"`
    ClientName  string `json:"client_name,omitempty"`
    AgentID     string `json:"agent_id,omitempty"`
    Target      string `json:"target,omitempty"`
    SessionID   string `json:"session_id,omitempty"`
    Data        []byte `json:"data,omitempty"`
    Protocol    string `json:"protocol,omitempty"`
}

func main() {
    var rootCmd = &cobra.Command{
        Use:   "ssh-tunnel-client",
        Short: "SSH Tunnel Client with command logging",
        Run:   runSSHClient,
    }

    rootCmd.Flags().StringP("client-id", "c", "ssh-client-1", "Client ID")
    rootCmd.Flags().StringP("client-name", "n", "SSH Client", "Client name")
    rootCmd.Flags().StringP("relay", "r", "ws://168.231.119.242:8080/ws/client", "Relay server WebSocket URL")
    rootCmd.Flags().StringP("agent", "a", "agent-linux", "Target agent ID")
    rootCmd.Flags().StringP("ssh-host", "H", "127.0.0.1", "SSH target host")
    rootCmd.Flags().StringP("ssh-port", "P", "22", "SSH target port")
    rootCmd.Flags().StringP("ssh-user", "u", "root", "SSH username")
    rootCmd.Flags().StringP("ssh-password", "w", "1qazxsw2", "SSH password")
    rootCmd.Flags().StringP("local-port", "p", "2222", "Local port to listen")

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func runSSHClient(cmd *cobra.Command, args []string) {
    clientID, _ := cmd.Flags().GetString("client-id")
    clientName, _ := cmd.Flags().GetString("client-name")
    relayURL, _ := cmd.Flags().GetString("relay")
    agentID, _ := cmd.Flags().GetString("agent")
    sshHost, _ := cmd.Flags().GetString("ssh-host")
    sshPort, _ := cmd.Flags().GetString("ssh-port")
    sshUser, _ := cmd.Flags().GetString("ssh-user")
    sshPassword, _ := cmd.Flags().GetString("ssh-password")
    localPort, _ := cmd.Flags().GetString("local-port")

    client := &SSHClient{
        clientID:    clientID,
        clientName:  clientName,
        relayURL:    relayURL,
        agentID:     agentID,
        sshHost:     sshHost,
        sshPort:     sshPort,
        sshUser:     sshUser,
        sshPassword: sshPassword,
        localPort:   localPort,
        logger:     common.NewLogger(fmt.Sprintf("SSH-CLIENT-%s", clientID)),
    }

    // Initialize database logger for SSH commands (simplified)
    client.dbLogger = common.NewDatabaseQueryLogger(client.logger, "ssh")

    if err := client.connect(); err != nil {
        client.logger.Error("Failed to connect: %v", err)
        os.Exit(1)
    }

    // Start local SSH server
    client.startLocalSSHServer()
}

func (c *SSHClient) logSSHActivity(operation, query string) {
    c.logger.Info("SSH Activity logged: %s - %s", operation, query)
}

func (c *SSHClient) connect() error {
    var err error
    c.conn, _, err = websocket.DefaultDialer.Dial(c.relayURL, nil)
    if err != nil {
        return fmt.Errorf("failed to connect to relay: %v", err)
    }

    // Register client
    registerMsg := Message{
        Type:       "register",
        ClientID:   c.clientID,
        ClientName: c.clientName,
    }

    if err := c.conn.WriteJSON(registerMsg); err != nil {
        return fmt.Errorf("failed to register: %v", err)
    }

    c.logger.Info("Connected to relay server as %s (%s)", c.clientID, c.clientName)
    return nil
}

func (c *SSHClient) startLocalSSHServer() {
    listener, err := net.Listen("tcp", ":"+c.localPort)
    if err != nil {
        c.logger.Error("Failed to listen on port %s: %v", c.localPort, err)
        os.Exit(1)
    }
    defer listener.Close()

    c.logger.Info("SSH tunnel listening on port %s", c.localPort)
    c.logger.Info("Connect using: ssh %s@localhost -p %s", c.sshUser, c.localPort)

    for {
        conn, err := listener.Accept()
        if err != nil {
            c.logger.Error("Failed to accept connection: %v", err)
            continue
        }

        go c.handleSSHConnection(conn)
    }
}

func (c *SSHClient) handleSSHConnection(conn net.Conn) {
    defer conn.Close()

    c.logger.Info("New SSH connection from %s", conn.RemoteAddr())

    // Create tunnel session
    c.sessionID = fmt.Sprintf("ssh_%d", time.Now().UnixNano())
    target := fmt.Sprintf("%s:%s", c.sshHost, c.sshPort)

    // Request tunnel through relay
    connectMsg := Message{
        Type:      "connect",
        ClientID:  c.clientID,
        AgentID:   c.agentID,
        Target:    target,
        SessionID: c.sessionID,
        Protocol:  "ssh",
    }

    if err := c.conn.WriteJSON(connectMsg); err != nil {
        c.logger.Error("Failed to request tunnel: %v", err)
        return
    }

    // Handle SSH protocol detection and logging
    go c.handleSSHLogging(conn)

    // Handle data forwarding
    c.forwardData(conn)
}

func (c *SSHClient) handleSSHLogging(conn net.Conn) {
    reader := bufio.NewReader(conn)
    
    for {
        // Read SSH data
        data := make([]byte, 4096)
        n, err := reader.Read(data)
        if err != nil {
            if err != io.EOF {
                c.logger.Error("Error reading SSH data: %v", err)
            }
            break
        }

        if n > 0 {
            // Analyze SSH data for commands
            c.analyzeAndLogSSHData(string(data[:n]), "outbound")
        }
    }
}

func (c *SSHClient) analyzeAndLogSSHData(data, direction string) {
    // Simple SSH command detection
    // This is a basic implementation - in practice you'd need more sophisticated SSH protocol parsing
    
    if strings.Contains(data, "SSH-") {
        // SSH version exchange
        c.logSSHCommand("SSH_VERSION", direction, fmt.Sprintf("SSH version: %s", strings.TrimSpace(data)))
    } else if isSSHCommand(data) {
        // Extract command from SSH data
        command := extractSSHCommand(data)
        if command != "" {
            c.logSSHCommand(command, direction, data)
        }
    }
}

func isSSHCommand(data string) bool {
    // Simple command detection patterns
    commands := []string{"ls", "cd", "pwd", "cat", "grep", "ps", "top", "ssh", "scp", "vim", "nano", "tail", "head"}
    
    for _, cmd := range commands {
        if strings.Contains(strings.ToLower(data), cmd) {
            return true
        }
    }
    return false
}

func extractSSHCommand(data string) string {
    // Basic command extraction - this would need to be more sophisticated in practice
    data = strings.TrimSpace(data)
    
    // Remove control characters and non-printable characters
    var result strings.Builder
    for _, r := range data {
        if r >= 32 && r <= 126 { // Printable ASCII
            result.WriteRune(r)
        }
    }
    
    cleaned := result.String()
    
    // If it looks like a command, return first word
    words := strings.Fields(cleaned)
    if len(words) > 0 && len(words[0]) > 0 {
        return words[0]
    }
    
    return ""
}

func (c *SSHClient) logSSHCommand(command, direction, data string) {
    // Log to local logger first
    c.logger.Info("SSH Command: %s [%s] -> %s", direction, command, c.sshHost)
    
    logReq := SSHLogRequest{
        SessionID: c.sessionID,
        ClientID:  c.clientID,
        AgentID:   c.agentID,
        Direction: direction,
        User:      c.sshUser,
        Host:      c.sshHost,
        Port:      c.sshPort,
        Command:   command,
        Data:      data,
    }

    // Send to relay server
    jsonData, err := json.Marshal(logReq)
    if err != nil {
        c.logger.Error("Failed to marshal SSH log: %v", err)
        return
    }

    // Post to relay API
    relayAPIURL := strings.Replace(c.relayURL, "ws://", "http://", 1)
    relayAPIURL = strings.Replace(relayAPIURL, "/ws/client", "/api/log-ssh", 1)

    resp, err := http.Post(relayAPIURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        c.logger.Error("Failed to send SSH log to relay: %v", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode == 200 {
        c.logger.Debug("SSH command logged: %s -> %s", direction, command)
    } else {
        c.logger.Error("Failed to log SSH command: HTTP %d", resp.StatusCode)
    }
}

func (c *SSHClient) forwardData(conn net.Conn) {
    // Handle bidirectional data forwarding through WebSocket using binary messages
    done := make(chan bool, 2)

    // Forward from local connection to relay
    go func() {
        defer func() { done <- true }()
        
        buffer := make([]byte, 4096)
        for {
            n, err := conn.Read(buffer)
            if err != nil {
                if err != io.EOF {
                    c.logger.Error("Error reading from local connection: %v", err)
                }
                return
            }

            c.logger.Debug("=== SENDING TO RELAY ===")
            c.logger.Debug("Read %d bytes from local connection", n)
            c.logger.Debug("SessionID: %s", c.sessionID)
            c.logger.Debug("ClientID: %s", c.clientID)

            // Create a protocol frame for binary data
            // Format: [TYPE:4][CLIENT_ID_LEN:4][CLIENT_ID][SESSION_ID_LEN:4][SESSION_ID][DATA]
            frame := &bytes.Buffer{}
            
            // Type (4 bytes)
            frame.WriteString("DATA")
            
            // ClientID length and data
            clientIDBytes := []byte(c.clientID)
            frame.Write([]byte{byte(len(clientIDBytes))})
            frame.Write(clientIDBytes)
            
            // SessionID length and data  
            sessionIDBytes := []byte(c.sessionID)
            frame.Write([]byte{byte(len(sessionIDBytes))})
            frame.Write(sessionIDBytes)
            
            // Actual SSH data
            frame.Write(buffer[:n])

            // Send as binary WebSocket message
            if err := c.conn.WriteMessage(websocket.BinaryMessage, frame.Bytes()); err != nil {
                c.logger.Error("Error sending binary data to relay: %v", err)
                return
            }
            
            c.logger.Debug("✅ Sent %d bytes to relay (binary)", n)
        }
    }()

    // Forward from relay to local connection
    go func() {
        defer func() { done <- true }()
        
        for {
            messageType, messageData, err := c.conn.ReadMessage()
            if err != nil {
                c.logger.Error("Error reading from relay: %v", err)
                return
            }

            c.logger.Debug("=== RECEIVED FROM RELAY ===")
            c.logger.Debug("Message type: %d", messageType)
            c.logger.Debug("Message length: %d", len(messageData))

            if messageType == websocket.BinaryMessage {
                // Parse binary frame for our session
                if len(messageData) < 5 {
                    continue
                }
                
                reader := bytes.NewReader(messageData)
                
                // Read type (4 bytes)
                typeBytes := make([]byte, 4)
                reader.Read(typeBytes)
                if string(typeBytes) != "DATA" {
                    continue
                }
                
                // Read client ID
                clientIDLen := make([]byte, 1)
                reader.Read(clientIDLen)
                clientIDBytes := make([]byte, clientIDLen[0])
                reader.Read(clientIDBytes)
                
                // Read session ID
                sessionIDLen := make([]byte, 1)
                reader.Read(sessionIDLen)
                sessionIDBytes := make([]byte, sessionIDLen[0])
                reader.Read(sessionIDBytes)
                
                // Check if this is for our session
                if string(sessionIDBytes) == c.sessionID {
                    // Read remaining data
                    sshData := make([]byte, reader.Len())
                    reader.Read(sshData)
                    
                    c.logger.Debug("✅ Writing %d bytes to local connection", len(sshData))
                    if _, err := conn.Write(sshData); err != nil {
                        c.logger.Error("Error writing to local connection: %v", err)
                        return
                    }
                    
                    // Log received data
                    if len(sshData) > 0 {
                        c.analyzeAndLogSSHData(string(sshData), "inbound")
                    }
                } else {
                    c.logger.Debug("⚠️ Ignoring binary message for session %s (expected %s)", string(sessionIDBytes), c.sessionID)
                }
            } else if messageType == websocket.TextMessage {
                // Handle JSON messages for control commands
                var msg Message
                if err := json.Unmarshal(messageData, &msg); err == nil {
                    c.logger.Debug("Received JSON message: Type=%s, SessionID=%s", msg.Type, msg.SessionID)
                }
            }
        }
    }()

    // Wait for either direction to complete
    <-done
    c.logger.Info("SSH session %s ended", c.sessionID)
}