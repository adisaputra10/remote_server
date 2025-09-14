package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "ssh-tunnel/internal/common"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
)

type RelayServer struct {
    agents   map[string]*websocket.Conn
    clients  map[string]*websocket.Conn
    sessions map[string]*Session
    mutex    sync.RWMutex
    logger   *common.Logger
}

type Session struct {
    ID       string
    AgentID  string
    ClientID string
    Target   string
    Created  time.Time
}

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow connections from any origin
    },
}

func NewRelayServer() *RelayServer {
    return &RelayServer{
        agents:   make(map[string]*websocket.Conn),
        clients:  make(map[string]*websocket.Conn),
        sessions: make(map[string]*Session),
        logger:   common.NewLogger("RELAY"),
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
    default:
        rs.logger.Error("Unknown message type: %s", msg.Type)
    }
}

func (rs *RelayServer) handleRegister(conn *websocket.Conn, msg *common.Message) {
    rs.mutex.Lock()
    defer rs.mutex.Unlock()

    if msg.AgentID != "" {
        rs.agents[msg.AgentID] = conn
        rs.logger.Info("Agent registered: %s", msg.AgentID)
    } else if msg.ClientID != "" {
        rs.clients[msg.ClientID] = conn
        rs.logger.Info("Client registered: %s", msg.ClientID)
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
        rs.sendMessage(agentConn, connectMsg)
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
        targetConn = rs.agents[session.AgentID]
        rs.logger.Debug("Forwarding client data to agent %s", session.AgentID)
    } else if msg.AgentID != "" {
        // Data from agent to client
        targetConn = rs.clients[session.ClientID]
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
            targetConn = rs.agents[session.AgentID]
        } else if msg.AgentID != "" {
            targetConn = rs.clients[session.ClientID]
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
    for agentID, agentConn := range rs.agents {
        if agentConn == conn {
            delete(rs.agents, agentID)
            rs.logger.Info("Agent disconnected: %s", agentID)
            break
        }
    }

    // Remove from clients
    for clientID, clientConn := range rs.clients {
        if clientConn == conn {
            delete(rs.clients, clientID)
            rs.logger.Info("Client disconnected: %s", clientID)
            break
        }
    }

    // Clean up sessions
    for sessionID, session := range rs.sessions {
        // Check if this connection belongs to this session
        agentConn := rs.agents[session.AgentID]
        clientConn := rs.clients[session.ClientID]
        if agentConn == conn || clientConn == conn {
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
        rs.logger.Close()
        os.Exit(0)
    }()

    http.HandleFunc("/ws/agent", rs.handleWebSocket)
    http.HandleFunc("/ws/client", rs.handleWebSocket)
    
    // Health check endpoint
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        rs.mutex.RLock()
        stats := map[string]interface{}{
            "agents":   len(rs.agents),
            "clients":  len(rs.clients),
            "sessions": len(rs.sessions),
        }
        rs.mutex.RUnlock()
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(stats)
    })

    addr := fmt.Sprintf(":%d", port)
    rs.logger.Info("Starting relay server on %s", addr)
    rs.logger.Info("WebSocket endpoints:")
    rs.logger.Info("  - Agent: ws://localhost%s/ws/agent", addr)
    rs.logger.Info("  - Client: ws://localhost%s/ws/client", addr)
    rs.logger.Info("  - Health: http://localhost%s/health", addr)
    
    log.Fatal(http.ListenAndServe(addr, nil))
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