package main

import (
    "encoding/hex"
    "fmt"
    "io"
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
        Use:   "ssh-tunnel-client-debug",
        Short: "SSH Tunnel Client with detailed debugging",
        Run:   runSSHClient,
    }

    rootCmd.Flags().StringP("client-id", "c", "ssh-client-1", "Client ID")
    rootCmd.Flags().StringP("client-name", "n", "SSH Client", "Client name")
    rootCmd.Flags().StringP("relay", "r", "ws://168.231.119.242:8080/ws/client", "Relay server WebSocket URL")
    rootCmd.Flags().StringP("agent", "a", "agent-linux", "Target agent ID")
    rootCmd.Flags().StringP("ssh-host", "H", "168.231.119.242", "SSH target host")
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
        logger:     common.NewLogger(fmt.Sprintf("SSH-CLIENT-DEBUG-%s", clientID)),
    }

    if err := client.connect(); err != nil {
        client.logger.Error("Failed to connect: %v", err)
        os.Exit(1)
    }

    client.startLocalSSHServer()
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

    // Request tunnel through relay using JSON first
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

    c.logger.Info("✅ Requested tunnel for session %s to target %s", c.sessionID, target)

    // Handle data forwarding with detailed debugging
    c.forwardDataWithDebug(conn)
}

func (c *SSHClient) forwardDataWithDebug(conn net.Conn) {
    done := make(chan bool, 2)

    // Forward from local connection to relay
    go func() {
        defer func() { done <- true }()
        
        buffer := make([]byte, 1024) // Smaller buffer for debugging
        for {
            n, err := conn.Read(buffer)
            if err != nil {
                if err != io.EOF {
                    c.logger.Error("Error reading from local connection: %v", err)
                }
                return
            }

            c.logger.Info("📤 READ %d bytes from local SSH client", n)
            c.logger.Info("📤 HEX: %s", hex.EncodeToString(buffer[:n]))
            if n < 100 { // Only log small packets as text
                c.logger.Info("📤 TXT: %q", string(buffer[:n]))
            }

            // Send using simplified JSON format for debugging
            dataMsg := Message{
                Type:      "data",
                ClientID:  c.clientID,
                SessionID: c.sessionID,
                Data:      make([]byte, n),
            }
            copy(dataMsg.Data, buffer[:n])

            if err := c.conn.WriteJSON(dataMsg); err != nil {
                c.logger.Error("Error sending data to relay: %v", err)
                return
            }
            
            c.logger.Info("✅ Sent %d bytes to relay via JSON", n)
        }
    }()

    // Forward from relay to local connection
    go func() {
        defer func() { done <- true }()
        
        for {
            var msg Message
            if err := c.conn.ReadJSON(&msg); err != nil {
                c.logger.Error("Error reading from relay: %v", err)
                return
            }

            if msg.Type == "data" && msg.SessionID == c.sessionID {
                c.logger.Info("📥 RECEIVED %d bytes from relay", len(msg.Data))
                c.logger.Info("📥 HEX: %s", hex.EncodeToString(msg.Data))
                if len(msg.Data) < 100 { // Only log small packets as text
                    c.logger.Info("📥 TXT: %q", string(msg.Data))
                }

                if _, err := conn.Write(msg.Data); err != nil {
                    c.logger.Error("Error writing to local connection: %v", err)
                    return
                }
                
                c.logger.Info("✅ Wrote %d bytes to local SSH client", len(msg.Data))
            } else {
                c.logger.Debug("Ignoring message - Type: %s, SessionID match: %t", msg.Type, msg.SessionID == c.sessionID)
            }
        }
    }()

    // Wait for either direction to complete
    <-done
    c.logger.Info("SSH session %s ended", c.sessionID)
}