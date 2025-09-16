package main

import (
    "bufio"
    "bytes"
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
    clientID    string
    clientName  string
    relayURL    string
    apiURL      string
    conn        *websocket.Conn
    sessionID   string
    agentID     string
    remoteHost  string
    remoteUser  string
    logger      *common.Logger
    connected   bool
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
        clientID:   clientID,
        clientName: clientName,
        relayURL:   relayURL,
        apiURL:     apiURL,
        agentID:    agentID,
        remoteHost: remoteHost,
        remoteUser: remoteUser,
        logger:     common.NewLogger("SHELL"),
        connected:  false,
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
    
    fmt.Printf("\n🔗 Interactive Shell Connected\n")
    fmt.Printf("📡 Agent: %s\n", s.agentID)
    fmt.Printf("🖥️  Host: %s@%s\n", s.remoteUser, s.remoteHost)
    fmt.Printf("🆔 Session: %s\n", s.sessionID[:8])
    fmt.Printf("💡 Type 'exit' to quit, 'help' for commands\n\n")

    // Start message handler
    go s.handleMessages()

    // Interactive command loop
    scanner := bufio.NewScanner(os.Stdin)
    
    for {
        fmt.Printf("%s@%s:~$ ", s.remoteUser, s.remoteHost)
        
        if !scanner.Scan() {
            break
        }
        
        command := strings.TrimSpace(scanner.Text())
        
        if command == "" {
            continue
        }
        
        if command == "exit" || command == "quit" {
            fmt.Println("👋 Goodbye!")
            break
        }
        
        if command == "help" {
            s.showHelp()
            continue
        }
        
        if command == "status" {
            s.showStatus()
            continue
        }
        
        // Execute command on remote agent
        s.executeCommand(command)
    }
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
                // Display command output
                if msg.Data != "" {
                    fmt.Print(msg.Data)
                }
                
                // Log inbound response
                s.logCommand(msg.DBQuery, "inbound")
                s.connected = true
            }
            
        case "shell_error":
            if msg.SessionID == s.sessionID {
                fmt.Printf("❌ Error: %s\n", msg.Data)
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

func (s *InteractiveShell) showHelp() {
    fmt.Printf(`
📚 Interactive Shell Help:

🔧 Built-in Commands:
  help     - Show this help message
  status   - Show connection status
  exit     - Exit the shell
  quit     - Exit the shell

🐧 Linux Commands (examples):
  ls       - List files
  pwd      - Show current directory
  cd <dir> - Change directory
  cat <file> - Show file content
  ps aux   - List processes
  top      - System monitor
  df -h    - Disk usage
  free -m  - Memory usage
  uname -a - System information

💡 Tips:
  - All commands are executed on the remote agent
  - Command history is logged to the dashboard
  - Use Ctrl+C to interrupt long-running commands

`)
}

func (s *InteractiveShell) showStatus() {
    fmt.Printf(`
📊 Connection Status:

🆔 Client ID: %s
📡 Agent ID: %s
🖥️  Remote: %s@%s
🔗 Session: %s
📡 Relay: %s
✅ Status: Connected

`, s.clientID, s.agentID, s.remoteUser, s.remoteHost, s.sessionID[:8], s.relayURL)
}