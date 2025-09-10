package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type Agent struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Platform string                 `json:"platform"`
	Status   string                 `json:"status"`
	LastSeen time.Time              `json:"last_seen"`
	Conn     *websocket.Conn        `json:"-"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Client struct {
	ID   string
	Conn *websocket.Conn
	Name string
}

type Message struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	Command   string                 `json:"command,omitempty"`
	Data      string                 `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type Server struct {
	agents  map[string]*Agent
	clients map[string]*Client
	mutex   sync.RWMutex
	logger  *log.Logger
}

func NewServer(logger *log.Logger) *Server {
	return &Server{
		agents:  make(map[string]*Agent),
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// Simple server for now - using same structure as old server but modular
func main() {
	var (
		port = flag.Int("port", 8080, "Server port")
		host = flag.String("host", "0.0.0.0", "Server host")
	)
	flag.Parse()

	logger := log.New(os.Stdout, "[SERVER] ", log.LstdFlags|log.Lshortfile)

	server := NewServer(logger)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	mux := http.NewServeMux()

	// WebSocket handlers
	mux.HandleFunc("/agent", func(w http.ResponseWriter, r *http.Request) {
		server.handleAgentConnection(w, r, &upgrader)
	})

	mux.HandleFunc("/client", func(w http.ResponseWriter, r *http.Request) {
		server.handleClientConnection(w, r, &upgrader)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	addr := fmt.Sprintf("%s:%d", *host, *port)
	logger.Printf("🚀 Server starting on %s", addr)
	logger.Printf("📋 Server configuration:")
	logger.Printf("   - Host: %s", *host)
	logger.Printf("   - Port: %d", *port)
	logger.Printf("   - Agent endpoint: /agent")
	logger.Printf("   - Client endpoint: /client")
	logger.Printf("   - Health endpoint: /health")

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("🛑 Shutting down server...")
		httpServer.Close()
	}()

	logger.Printf("✅ Server ready! Waiting for connections...")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("❌ Server failed: %v", err)
	}

	logger.Println("👋 Server stopped")
}

func (s *Server) handleAgentConnection(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader) {
	s.logger.Printf("🔗 New agent connection attempt from %s", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("❌ Agent upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	s.logger.Printf("✅ Agent connected successfully from %s", r.RemoteAddr)

	var agent *Agent

	// Handle messages
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			s.logger.Printf("❌ Agent read error from %s: %v", r.RemoteAddr, err)
			break
		}

		s.logger.Printf("📨 Agent message from %s: %v", r.RemoteAddr, msg.Type)

		switch msg.Type {
		case "register":
			// Register new agent
			agentID := msg.AgentID
			if agentID == "" {
				agentID = fmt.Sprintf("agent_%d", time.Now().Unix())
			}

			agent = &Agent{
				ID:       agentID,
				Name:     fmt.Sprintf("agent-%s", agentID[:8]),
				Platform: "windows",
				Status:   "online",
				LastSeen: time.Now(),
				Conn:     conn,
				Metadata: msg.Metadata,
			}

			if metadata := msg.Metadata; metadata != nil {
				if name, ok := metadata["name"].(string); ok {
					agent.Name = name
				}
				if platform, ok := metadata["platform"].(string); ok {
					agent.Platform = platform
				}
			}

			s.mutex.Lock()
			s.agents[agentID] = agent
			s.mutex.Unlock()

			s.logger.Printf("✅ Agent registered: %s (%s)", agent.Name, agentID)

			// Send registration confirmation
			response := Message{
				Type:      "registered",
				AgentID:   agentID,
				Data:      "Agent registered successfully",
				Timestamp: time.Now(),
			}
			conn.WriteJSON(response)

		case "heartbeat":
			if agent != nil {
				agent.LastSeen = time.Now()
				agent.Status = "online"

				// Send heartbeat response
				response := Message{
					Type:      "heartbeat",
					AgentID:   agent.ID,
					Timestamp: time.Now(),
				}
				conn.WriteJSON(response)
			}
		case "tunnel_response":
			// Forward tunnel response from agent to client
			s.logger.Printf("📤 Forwarding tunnel response from agent %s", msg.AgentID)
			
			// Find the client that requested this tunnel (for now, forward to all clients)
			s.mutex.RLock()
			for _, client := range s.clients {
				if err := client.Conn.WriteJSON(msg); err != nil {
					s.logger.Printf("❌ Failed to forward tunnel response to client: %v", err)
				} else {
					s.logger.Printf("✅ Tunnel response forwarded to client")
				}
			}
			s.mutex.RUnlock()
		}
	}

	// Remove agent on disconnect
	if agent != nil {
		s.mutex.Lock()
		delete(s.agents, agent.ID)
		s.mutex.Unlock()
		s.logger.Printf("🔌 Agent disconnected: %s (%s)", agent.Name, agent.ID)
	}
}

func (s *Server) handleClientConnection(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader) {
	s.logger.Printf("🔗 New client connection attempt from %s", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("❌ Client upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	s.logger.Printf("✅ Client connected successfully from %s", r.RemoteAddr)

	var client *Client

	// Handle messages
	for {
		// Read as raw JSON first to preserve all fields
		var rawMsg map[string]interface{}
		if err := conn.ReadJSON(&rawMsg); err != nil {
			s.logger.Printf("❌ Client read error from %s: %v", r.RemoteAddr, err)
			break
		}

		msgType, ok := rawMsg["type"].(string)
		if !ok {
			s.logger.Printf("❌ Invalid message type from client")
			continue
		}

		s.logger.Printf("📨 Client message from %s: %v", r.RemoteAddr, msgType)
		s.logger.Printf("🔍 Full client message: %+v", rawMsg)

		// For tunnel_request, preserve all fields
		if msgType == "tunnel_request" {
			s.logger.Printf("🚇 TUNNEL REQUEST from client - preserving all fields")
			
			agentID, _ := rawMsg["agent_id"].(string)
			s.mutex.RLock()
			agent, exists := s.agents[agentID]
			s.mutex.RUnlock()

			if !exists {
				response := map[string]interface{}{
					"type":      "tunnel_error",
					"client_id": rawMsg["client_id"],
					"data":      "Agent not found",
					"timestamp": time.Now().Format(time.RFC3339),
				}
				conn.WriteJSON(response)
				s.logger.Printf("❌ Tunnel request failed: agent %s not found", agentID)
			} else {
				// Forward the raw message to preserve all tunnel parameters
				s.logger.Printf("📤 Forwarding tunnel request to agent %s with all parameters", agentID)
				s.logger.Printf("🔍 Forwarding message: %+v", rawMsg)
				if err := agent.Conn.WriteJSON(rawMsg); err != nil {
					s.logger.Printf("❌ Failed to forward tunnel request to agent %s: %v", agentID, err)
					response := map[string]interface{}{
						"type":      "tunnel_error",
						"client_id": rawMsg["client_id"],
						"data":      "Failed to forward request to agent",
						"timestamp": time.Now().Format(time.RFC3339),
					}
					conn.WriteJSON(response)
				} else {
					s.logger.Printf("✅ Tunnel request forwarded to agent %s", agentID)
				}
			}
			continue
		}

		// For other message types, convert back to Message struct
		var msg Message
		if jsonData, err := json.Marshal(rawMsg); err == nil {
			json.Unmarshal(jsonData, &msg)
		} else {
			s.logger.Printf("❌ Failed to convert message: %v", err)
			continue
		}

		s.logger.Printf("🔍 Message details: AgentID=%s, ClientID=%s, Data=%s", msg.AgentID, msg.ClientID, msg.Data)

		switch msg.Type {
		case "register":
			// Register new client
			clientID := msg.ClientID
			if clientID == "" {
				clientID = fmt.Sprintf("client_%d", time.Now().Unix())
			}

			clientName := "unknown"
			if metadata := msg.Metadata; metadata != nil {
				if name, ok := metadata["name"].(string); ok {
					clientName = name
				}
			}

			client = &Client{
				ID:   clientID,
				Conn: conn,
				Name: clientName,
			}

			s.mutex.Lock()
			s.clients[clientID] = client
			s.mutex.Unlock()

			s.logger.Printf("✅ Client registered: %s (%s)", client.Name, clientID)

			// Send registration confirmation
			response := Message{
				Type:      "registered",
				ClientID:  clientID,
				Data:      "Client registered successfully",
				Timestamp: time.Now(),
			}
			conn.WriteJSON(response)

		case "get_agents":
			// Send agent list to client
			s.mutex.RLock()
			agentList := make([]Agent, 0, len(s.agents))
			for _, agent := range s.agents {
				agentList = append(agentList, *agent)
			}
			s.mutex.RUnlock()

			response := Message{
				Type:      "agent_list",
				ClientID:  msg.ClientID,
				Metadata:  map[string]interface{}{"agents": agentList},
				Timestamp: time.Now(),
			}
			conn.WriteJSON(response)

		case "connect_agent":
			// Handle agent connection request
			agentID := msg.AgentID
			s.mutex.RLock()
			_, exists := s.agents[agentID]
			s.mutex.RUnlock()

			if !exists {
				response := Message{
					Type:      "access_denied",
					ClientID:  msg.ClientID,
					Data:      "Agent not found",
					Timestamp: time.Now(),
				}
				conn.WriteJSON(response)
			} else {
				sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
				response := Message{
					Type:      "session_created",
					SessionID: sessionID,
					AgentID:   agentID,
					ClientID:  msg.ClientID,
					Data:      "Session created successfully",
					Timestamp: time.Now(),
				}
				conn.WriteJSON(response)
				s.logger.Printf("✅ Session created: %s for client %s -> agent %s", sessionID, msg.ClientID, agentID)
			}
		case "tunnel_data":
			// Forward tunnel data to agent
			s.logger.Printf("📡 Received tunnel data from client")
			
			agentID, _ := rawMsg["agent_id"].(string)
			tunnelID, _ := rawMsg["tunnel_id"].(string)
			
			s.mutex.RLock()
			agent, exists := s.agents[agentID]
			s.mutex.RUnlock()

			if !exists {
				s.logger.Printf("❌ Tunnel data failed: agent %s not found", agentID)
			} else {
				// Forward the tunnel data to the agent
				s.logger.Printf("📤 Forwarding tunnel data to agent %s for tunnel %s", agentID, tunnelID)
				if err := agent.Conn.WriteJSON(rawMsg); err != nil {
					s.logger.Printf("❌ Failed to forward tunnel data to agent %s: %v", agentID, err)
				} else {
					s.logger.Printf("✅ Tunnel data forwarded to agent %s", agentID)
				}
			}
		}
	}

	// Remove client on disconnect
	if client != nil {
		s.mutex.Lock()
		delete(s.clients, client.ID)
		s.mutex.Unlock()
		s.logger.Printf("🔌 Client disconnected: %s (%s)", client.Name, client.ID)
	}
}
