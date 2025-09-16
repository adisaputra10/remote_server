package main

import (
    "bufio"
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "ssh-tunnel/internal/common"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
)

type InteractiveShell struct {
    clientID      string
    clientName    string
    relayURL      string
    apiURL        string
    conn          *websocket.Conn
    sessionID     string
    agentID       string
    remoteHost    string
    remoteUser    string
    logger        *common.Logger
    connected     bool
    currentPrompt string
    currentDir    string
    waitingForResponse bool
    responseChan  chan string
}

type ShellMessage struct {
    Type        string `json:"type"`
    ClientID    string `json:"client_id,omitempty"`
    ClientName  string `json:"client_name,omitempty"`
    AgentID     string `json:"agent_id,omitempty"`
    SessionID   string `json:"session_id,omitempty"`
    Command     string `json:"command,omitempty"`
    Data        string `json:"data,omitempty"`
    Status      string `json:"status,omitempty"`
    Protocol    string `json:"protocol,omitempty"`
    Direction   string `json:"direction,omitempty"`
    DBQuery     string `json:"db_query,omitempty"`  // Use for shell commands
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

func main() {
    var rootCmd = &cobra.Command{
        Use:   "interactive-shell",
        Short: "Interactive shell client for remote agent",
        Run:   runInteractiveShell,
    }

    rootCmd.Flags().StringP("client-id", "c", "shell-client-1", "Client ID")
    rootCmd.Flags().StringP("client-name", "n", "Interactive Shell", "Client name")
    rootCmd.Flags().StringP("relay", "r", "ws://168.231.119.242:8080/ws/client", "Relay server WebSocket URL")
    rootCmd.Flags().StringP("agent", "a", "agent-linux", "Target agent ID")
    rootCmd.Flags().StringP("remote-host", "H", "127.0.0.1", "Remote host")
    rootCmd.Flags().StringP("remote-user", "u", "root", "Remote user")

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func runInteractiveShell(cmd *cobra.Command, args []string) {
    clientID, _ := cmd.Flags().GetString("client-id")
    clientName, _ := cmd.Flags().GetString("client-name")
    relayURL, _ := cmd.Flags().GetString("relay")
    agentID, _ := cmd.Flags().GetString("agent")
    remoteHost, _ := cmd.Flags().GetString("remote-host")
    remoteUser, _ := cmd.Flags().GetString("remote-user")

    // Convert WebSocket URL to HTTP API URL
    apiURL := strings.Replace(relayURL, "ws://", "http://", 1)
    apiURL = strings.Replace(apiURL, "/ws/client", "", 1)

    shell := &InteractiveShell{
        clientID:      clientID,
        clientName:    clientName,
        relayURL:      relayURL,
        apiURL:        apiURL,
        agentID:       agentID,
        remoteHost:    remoteHost,
        remoteUser:    remoteUser,
        logger:        common.NewLogger("SHELL"),
        connected:     false,
        currentPrompt: fmt.Sprintf("%s@%s:~$ ", remoteUser, remoteHost),
        currentDir:    "~",
        responseChan:  make(chan string, 1),
    }

    if err := shell.connect(); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer shell.conn.Close()

    shell.startInteractiveSession()
}

func (s *InteractiveShell) connect() error {
    var err error
    s.conn, _, err = websocket.DefaultDialer.Dial(s.relayURL, nil)
    if err != nil {
        return fmt.Errorf("failed to connect to relay: %v", err)
    }

    // Register client
    registerMsg := ShellMessage{
        Type:       "register",
        ClientID:   s.clientID,
        ClientName: s.clientName,
    }

    if err := s.conn.WriteJSON(registerMsg); err != nil {
        return fmt.Errorf("failed to register: %v", err)
    }

    s.logger.Info("Connected to relay server as %s (%s)", s.clientID, s.clientName)
    return nil
}

func (s *InteractiveShell) startInteractiveSession() {
    s.sessionID = fmt.Sprintf("shell_%d", time.Now().UnixNano())
    
    fmt.Printf("\n🔗 Remote Shell Connected\n")
    fmt.Printf("📡 Agent: %s\n", s.agentID)
    fmt.Printf("🖥️  Remote: %s@%s\n", s.remoteUser, s.remoteHost)
    fmt.Printf("🆔 Session: %s\n", s.sessionID[:8])
    fmt.Printf("💡 Full remote terminal emulation - Type 'exit' to quit\n\n")

    // Start message handler
    go s.handleMessages()

    // Initialize remote shell by getting current working directory and prompt
    s.initializeRemoteShell()

    // Interactive command loop with dynamic prompt
    scanner := bufio.NewScanner(os.Stdin)
    
    for {
        // Display current prompt (updated by remote server)
        fmt.Print(s.currentPrompt)
        
        if !scanner.Scan() {
            break
        }
        
        command := strings.TrimSpace(scanner.Text())
        
        if command == "" {
            continue
        }
        
        if command == "exit" || command == "quit" {
            fmt.Println("👋 Connection closed")
            break
        }
        
        // Execute command on remote server and wait for response
        s.executeCommandAndWait(command)
    }
}

func (s *InteractiveShell) initializeRemoteShell() {
    // Get current working directory
    s.executeCommandAndWait("pwd")
    
    // Get hostname for better prompt
    s.executeCommandAndWait("hostname")
    
    // Setup shell environment
    s.executeCommandAndWait("export PS1='\\u@\\h:\\w\\$ '")
}

func (s *InteractiveShell) executeCommandAndWait(command string) {
    s.waitingForResponse = true
    s.executeCommand(command)
    
    // Wait for response with timeout
    select {
    case response := <-s.responseChan:
        s.displayResponse(response, command)
    case <-time.After(10 * time.Second):
        fmt.Println("⚠️  Command timeout")
        s.waitingForResponse = false
    }
}

func (s *InteractiveShell) displayResponse(response, command string) {
    s.waitingForResponse = false
    
    if response != "" {
        // Clean and display response
        cleanResponse := strings.TrimSpace(response)
        if cleanResponse != "" {
            fmt.Print(cleanResponse)
            if !strings.HasSuffix(cleanResponse, "\n") {
                fmt.Println()
            }
        }
    }
    
    // Update prompt based on command
    s.updatePromptFromCommand(command, response)
}

func (s *InteractiveShell) updatePromptFromCommand(command, response string) {
    if strings.HasPrefix(command, "cd ") {
        // Update current directory after cd command
        s.executeCommand("pwd")
        // pwd response will update the prompt
    } else if command == "pwd" {
        // Update current directory from pwd response
        if response != "" {
            s.currentDir = strings.TrimSpace(response)
            s.updatePrompt()
        }
    } else if command == "hostname" {
        // Update hostname from response
        if response != "" {
            hostname := strings.TrimSpace(response)
            if hostname != "" {
                s.remoteHost = hostname
                s.updatePrompt()
            }
        }
    }
}

func (s *InteractiveShell) updatePrompt() {
    // Create dynamic prompt based on current state
    dir := s.currentDir
    if dir == "" {
        dir = "~"
    }
    
    // Shorten long paths
    if len(dir) > 30 {
        parts := strings.Split(dir, "/")
        if len(parts) > 2 {
            dir = fmt.Sprintf(".../%s", parts[len(parts)-1])
        }
    }
    
    s.currentPrompt = fmt.Sprintf("%s@%s:%s$ ", s.remoteUser, s.remoteHost, dir)
}

func (s *InteractiveShell) executeCommand(command string) {
    // Log outbound command
    s.logCommand(command, "outbound")
    
    // Send command to agent via relay using the correct format
    msg := ShellMessage{
        Type:      "shell_command",
        ClientID:  s.clientID,
        AgentID:   s.agentID,
        SessionID: s.sessionID,
        DBQuery:   command,  // Use DBQuery field for shell commands
        Protocol:  "shell",
        Direction: "outbound",
    }

    if err := s.conn.WriteJSON(msg); err != nil {
        fmt.Printf("❌ Error sending command: %v\n", err)
        return
    }
    
    s.logger.Info("Command sent: %s", command)
}

func (s *InteractiveShell) handleMessages() {
    for {
        var msg ShellMessage
        err := s.conn.ReadJSON(&msg)
        if err != nil {
            s.logger.Error("Error reading message: %v", err)
            return
        }

        switch msg.Type {
        case "shell_response":
            if msg.SessionID == s.sessionID {
                // Decode response
                var response string
                if msg.Data != "" {
                    // Try to decode base64 output
                    decodedOutput, err := base64.StdEncoding.DecodeString(msg.Data)
                    if err != nil {
                        // If decode fails, use raw data
                        response = msg.Data
                    } else {
                        response = string(decodedOutput)
                    }
                }
                
                // Log inbound response
                s.logCommand(msg.DBQuery, "inbound")
                s.connected = true
                
                // Send response to waiting command if any
                if s.waitingForResponse {
                    select {
                    case s.responseChan <- response:
                    default:
                        // Channel full, display directly
                        if response != "" {
                            fmt.Print(response)
                            if !strings.HasSuffix(response, "\n") {
                                fmt.Println()
                            }
                        }
                    }
                } else {
                    // Not waiting, display directly (for async commands)
                    if response != "" {
                        fmt.Print(response)
                        if !strings.HasSuffix(response, "\n") {
                            fmt.Println()
                        }
                    }
                }
            }
            
        case "shell_error":
            if msg.SessionID == s.sessionID {
                errorMsg := fmt.Sprintf("❌ Error: %s", msg.Data)
                
                if s.waitingForResponse {
                    select {
                    case s.responseChan <- errorMsg:
                    default:
                        fmt.Println(errorMsg)
                    }
                } else {
                    fmt.Println(errorMsg)
                }
                s.connected = false
            }
            
        case "ping":
            // Respond to ping
            pongMsg := ShellMessage{
                Type:     "pong",
                ClientID: s.clientID,
            }
            s.conn.WriteJSON(pongMsg)
            
        case "agent_disconnected":
            if msg.AgentID == s.agentID {
                fmt.Printf("\n❌ Agent %s disconnected!\n", s.agentID)
                fmt.Printf("Reconnecting...\n")
                s.connected = false
            }
            
        default:
            s.logger.Debug("Received unknown message type: %s", msg.Type)
        }
    }
}

func (s *InteractiveShell) logCommand(command, direction string) {
    logReq := SSHLogRequest{
        SessionID: s.sessionID,
        ClientID:  s.clientID,
        AgentID:   s.agentID,
        Direction: direction,
        User:      s.remoteUser,
        Host:      s.remoteHost,
        Port:      "22",
        Command:   command,
        Data:      command,
    }

    // Send to relay API
    go func() {
        jsonData, err := json.Marshal(logReq)
        if err != nil {
            s.logger.Error("Failed to marshal SSH log: %v", err)
            return
        }

        apiURL := s.apiURL + "/api/log-ssh"

        resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
        if err != nil {
            s.logger.Error("Failed to send command log: %v", err)
            return
        }
        defer resp.Body.Close()

        if resp.StatusCode == 200 {
            s.logger.Debug("Command logged: %s -> %s", direction, command)
        }
    }()
}