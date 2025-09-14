package main

import (
    "fmt"
    "io"
    "log"
    "net"
    "net/url"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "ssh-tunnel/internal/common"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
)

type Client struct {
    id         string
    relayURL   string
    conn       *websocket.Conn
    sessions   map[string]net.Conn
    dbLoggers  map[string]*common.DatabaseQueryLogger
    mutex      sync.RWMutex
    logger     *common.Logger
    running    bool
    heartbeat  *time.Ticker
}

func NewClient(id, relayURL string) *Client {
    return &Client{
        id:        id,
        relayURL:  relayURL,
        sessions:  make(map[string]net.Conn),
        dbLoggers: make(map[string]*common.DatabaseQueryLogger),
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
    c.dbLoggers[sessionID] = common.NewDatabaseQueryLogger(c.logger, target)
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
}

func (c *Client) sendMessage(msg *common.Message) error {
    data, err := msg.ToJSON()
    if err != nil {
        return fmt.Errorf("failed to serialize message: %v", err)
    }

    return c.conn.WriteMessage(websocket.TextMessage, data)
}

func main() {
    var (
        clientID  string
        relayURL  string
        localAddr string
        agentID   string
        target    string
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

            client := NewClient(clientID, relayURL)
            
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
    rootCmd.Flags().StringVarP(&relayURL, "relay-url", "r", "ws://localhost:8080/ws/client", "Relay server WebSocket URL")
    rootCmd.Flags().StringVarP(&localAddr, "local", "L", "", "Local address to listen on (e.g., :2222)")
    rootCmd.Flags().StringVarP(&agentID, "agent", "a", "", "Target agent ID")
    rootCmd.Flags().StringVarP(&target, "target", "t", "", "Target address (e.g., localhost:22)")
    rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run in interactive mode")

    if err := rootCmd.Execute(); err != nil {
        log.Fatal(err)
    }
}