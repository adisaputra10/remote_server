package main

import (
    "fmt"
    "io"
    "log"
    "net"
    "os"
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
    agentID     string
    localPort   string
    logger      *common.Logger
    dbLogger    *common.DatabaseQueryLogger
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
        Short: "SSH Tunnel Client - Pure tunnel with logging at agent level",
        Run:   runSSHClient,
    }

    rootCmd.Flags().StringP("client-id", "c", "ssh-client-1", "Client ID")
    rootCmd.Flags().StringP("client-name", "n", "SSH Tunnel Client", "Client name")
    rootCmd.Flags().StringP("relay", "r", "ws://localhost:8080/ws/client", "Relay server WebSocket URL")
    rootCmd.Flags().StringP("agent", "a", "ssh-agent", "Target agent ID")
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
    localPort, _ := cmd.Flags().GetString("local-port")

    client := &SSHClient{
        clientID:   clientID,
        clientName: clientName,
        relayURL:   relayURL,
        agentID:    agentID,
        localPort:  localPort,
        logger:     common.NewLogger("SSH-CLIENT-" + clientID),
        dbLogger:   common.NewDatabaseQueryLogger(common.NewLogger("SSH-CLIENT-" + clientID), clientID),
    }

    if err := client.connect(); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }

    // Start SSH tunnel server
    client.startSSHTunnelServer()
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

func (c *SSHClient) startSSHTunnelServer() {
    listener, err := net.Listen("tcp", ":"+c.localPort)
    if err != nil {
        log.Fatalf("Failed to listen on port %s: %v", c.localPort, err)
    }
    defer listener.Close()

    c.logger.Info("SSH tunnel listening on port %s", c.localPort)
    c.logger.Info("Target agent: %s", c.agentID)
    c.logger.Info("SSH target: 127.0.0.1:22 (through agent)")
    c.logger.Info("Connect using: ssh username@localhost -p %s", c.localPort)

    for {
        conn, err := listener.Accept()
        if err != nil {
            c.logger.Error("Failed to accept connection: %v", err)
            continue
        }

        go c.handleSSHTunnelConnection(conn)
    }
}

func (c *SSHClient) handleSSHTunnelConnection(conn net.Conn) {
    defer conn.Close()

    c.logger.Info("New SSH tunnel connection from %s", conn.RemoteAddr())

    // Create unique session ID
    sessionID := fmt.Sprintf("ssh_%d", time.Now().UnixNano())
    
    // Target is SSH port on remote server through agent
    target := "127.0.0.1:22"

    // Request tunnel through relay
    connectMsg := Message{
        Type:      "connect",
        ClientID:  c.clientID,
        AgentID:   c.agentID,
        Target:    target,
        SessionID: sessionID,
        Protocol:  "ssh",
    }

    if err := c.conn.WriteJSON(connectMsg); err != nil {
        c.logger.Error("Failed to request tunnel: %v", err)
        return
    }

    c.logger.Info("Tunnel requested - Session: %s, Agent: %s, Target: %s", sessionID, c.agentID, target)

    // Log connection attempt
    c.dbLogger.LogData([]byte(fmt.Sprintf("SSH tunnel to %s via %s", target, c.agentID)), "CONNECT", sessionID)

    // Handle bidirectional data forwarding
    c.forwardSSHData(conn, sessionID)
}

func (c *SSHClient) forwardSSHData(conn net.Conn, sessionID string) {
    done := make(chan bool, 2)

    // Forward from local SSH client to relay (client -> agent -> target)
    go func() {
        defer func() { done <- true }()
        
        buffer := make([]byte, 4096)
        totalBytes := 0
        
        for {
            n, err := conn.Read(buffer)
            if err != nil {
                if err != io.EOF {
                    c.logger.Debug("Error reading from local SSH client: %v", err)
                }
                return
            }

            totalBytes += n

            // Send data through WebSocket to relay
            dataMsg := Message{
                Type:      "data",
                SessionID: sessionID,
                Data:      buffer[:n],
            }

            if err := c.conn.WriteJSON(dataMsg); err != nil {
                c.logger.Error("Error sending data to relay: %v", err)
                return
            }

            c.logger.Debug("Forwarded %d bytes to agent (total: %d)", n, totalBytes)
        }
    }()

    // Forward from relay to local SSH client (target -> agent -> client)
    go func() {
        defer func() { done <- true }()
        
        totalBytes := 0
        
        for {
            var msg Message
            if err := c.conn.ReadJSON(&msg); err != nil {
                c.logger.Debug("Error reading from relay: %v", err)
                return
            }

            if msg.Type == "data" && msg.SessionID == sessionID {
                if len(msg.Data) > 0 {
                    totalBytes += len(msg.Data)
                    
                    if _, err := conn.Write(msg.Data); err != nil {
                        c.logger.Error("Error writing to local SSH client: %v", err)
                        return
                    }

                    c.logger.Debug("Forwarded %d bytes from agent (total: %d)", len(msg.Data), totalBytes)
                }
            } else if msg.Type == "error" {
                c.logger.Error("Tunnel error: %s", string(msg.Data))
                return
            } else if msg.Type == "close" {
                c.logger.Info("Tunnel closed by agent")
                return
            }
        }
    }()

    // Wait for either direction to complete
    <-done
    
    // Log session end
    c.dbLogger.LogData([]byte(fmt.Sprintf("SSH session %s ended", sessionID)), "DISCONNECT", sessionID)
    c.logger.Info("SSH tunnel session %s ended", sessionID)
}