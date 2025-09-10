package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"ssh-terminal/internal/proto"
	"ssh-terminal/internal/transport"
)

// Config holds client configuration
type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	ClientID  string `json:"client_id"`
	LogFile   string `json:"log_file"`
	Timeout   int    `json:"timeout"`
}

// Client represents the tunnel client
type Client struct {
	config  *Config
	logger  *log.Logger
	conn    *websocket.Conn
	session *yamux.Session
	tunnels map[string]*Tunnel
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	ui      *UI
}

// Tunnel represents an active tunnel
type Tunnel struct {
	ID         string
	AgentID    string
	Type       string
	LocalPort  int
	RemoteHost string
	RemotePort int
	Active     bool
	mu         sync.RWMutex
}

// LoadConfig loads client configuration from file
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &Config{}
	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Set defaults
	if config.ClientID == "" {
		config.ClientID = generateClientID()
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}

	return config, nil
}

// New creates a new client instance
func New(config *Config) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	logger := log.New(os.Stdout, "[CLIENT] ", log.LstdFlags|log.Lshortfile)
	
	client := &Client{
		config:  config,
		logger:  logger,
		tunnels: make(map[string]*Tunnel),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Create UI
	ui, err := NewUI(client)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create UI: %w", err)
	}
	client.ui = ui

	return client, nil
}

// Start starts the client
func (c *Client) Start(ctx context.Context) error {
	c.logger.Printf("Starting client, connecting to %s", c.config.ServerURL)

	// Connect to server
	if err := c.connect(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Start UI
	go c.ui.Run()

	// Start message handler
	go c.handleMessages()

	// Keep connection alive
	c.keepAlive()

	return nil
}

// Stop stops the client
func (c *Client) Stop() error {
	c.logger.Printf("Stopping client...")
	
	c.cancel()
	
	// Close UI
	if c.ui != nil {
		c.ui.Stop()
	}
	
	// Close all tunnels
	c.mu.Lock()
	for _, tunnel := range c.tunnels {
		c.closeTunnel(tunnel.ID)
	}
	c.mu.Unlock()
	
	// Close connection
	if c.session != nil {
		c.session.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	
	return nil
}

// connect establishes connection to server
func (c *Client) connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Duration(c.config.Timeout) * time.Second

	conn, _, err := dialer.Dial(c.config.ServerURL+"/client", nil)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	c.conn = conn

	// Create yamux session
	transport := transport.NewWebSocketTransport(conn)
	session, err := yamux.Client(transport, nil)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create yamux session: %w", err)
	}

	c.session = session

	// Register with server
	if err := c.register(); err != nil {
		session.Close()
		conn.Close()
		return fmt.Errorf("failed to register with server: %w", err)
	}

	c.logger.Printf("Connected to server successfully")
	return nil
}

// register registers the client with the server
func (c *Client) register() error {
	regData := proto.ClientRegisterData{
		ID:    c.config.ClientID,
		Token: c.config.Token,
	}

	msg := proto.Message{
		Type: "client_register",
		Data: mustMarshal(regData),
	}

	data, _ := json.Marshal(msg)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}

	// Wait for response
	_, respData, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read registration response: %w", err)
	}

	var response proto.Message
	if err := json.Unmarshal(respData, &response); err != nil {
		return fmt.Errorf("failed to unmarshal registration response: %w", err)
	}

	if response.Type != "client_register_response" {
		return fmt.Errorf("unexpected response type: %s", response.Type)
	}

	var respData2 map[string]interface{}
	if err := json.Unmarshal(response.Data, &respData2); err != nil {
		return fmt.Errorf("failed to unmarshal response data: %w", err)
	}

	if status, ok := respData2["status"].(string); !ok || status != "success" {
		return fmt.Errorf("registration failed: %v", respData2)
	}

	return nil
}

// handleMessages handles incoming messages from server
func (c *Client) handleMessages() {
	defer c.logger.Printf("Message handler stopped")

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Printf("WebSocket error: %v", err)
			}
			return
		}

		var msg proto.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		if err := c.handleMessage(&msg); err != nil {
			c.logger.Printf("Error handling message: %v", err)
		}
	}
}

// handleMessage handles a specific message
func (c *Client) handleMessage(msg *proto.Message) error {
	switch msg.Type {
	case "tunnel_response":
		return c.handleTunnelResponse(msg)
	case "tunnel_data":
		return c.handleTunnelData(msg)
	case "tunnel_close":
		return c.handleTunnelClose(msg)
	case "agent_list":
		return c.handleAgentList(msg)
	case "agent_info_response":
		return c.handleAgentInfoResponse(msg)
	default:
		c.logger.Printf("Unknown message type: %s", msg.Type)
		return nil
	}
}

// CreateTunnel creates a new tunnel
func (c *Client) CreateTunnel(agentID, tunnelType string, localPort int, remoteHost string, remotePort int) error {
	tunnelID := generateTunnelID()

	tunnel := &Tunnel{
		ID:         tunnelID,
		AgentID:    agentID,
		Type:       tunnelType,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		Active:     false,
	}

	c.mu.Lock()
	c.tunnels[tunnelID] = tunnel
	c.mu.Unlock()

	// Send tunnel request
	reqData := proto.TunnelRequestData{
		TunnelID:   tunnelID,
		AgentID:    agentID,
		Type:       tunnelType,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
	}

	msg := proto.Message{
		Type: "tunnel_request",
		Data: mustMarshal(reqData),
	}

	data, _ := json.Marshal(msg)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.mu.Lock()
		delete(c.tunnels, tunnelID)
		c.mu.Unlock()
		return fmt.Errorf("failed to send tunnel request: %w", err)
	}

	c.logger.Printf("Tunnel request sent: %s", tunnelID)
	return nil
}

// CloseTunnel closes a tunnel
func (c *Client) CloseTunnel(tunnelID string) error {
	return c.closeTunnel(tunnelID)
}

// closeTunnel closes a tunnel (internal)
func (c *Client) closeTunnel(tunnelID string) error {
	c.mu.Lock()
	tunnel, exists := c.tunnels[tunnelID]
	if exists {
		delete(c.tunnels, tunnelID)
	}
	c.mu.Unlock()

	if !exists {
		return fmt.Errorf("tunnel not found: %s", tunnelID)
	}

	// Send close message
	closeMsg := proto.TunnelCloseMessage{
		TunnelID: tunnelID,
		Reason:   "client_requested",
	}

	msg := proto.Message{
		Type: "tunnel_close",
		Data: mustMarshal(closeMsg),
	}

	data, _ := json.Marshal(msg)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.logger.Printf("Failed to send tunnel close: %v", err)
	}

	c.logger.Printf("Tunnel closed: %s", tunnelID)
	return nil
}

// ListAgents requests list of agents from server
func (c *Client) ListAgents() error {
	msg := proto.Message{
		Type: "list_agents",
		Data: json.RawMessage("{}"),
	}

	data, _ := json.Marshal(msg)
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// GetAgentInfo requests agent info from server
func (c *Client) GetAgentInfo(agentID string) error {
	reqData := proto.AgentInfoRequestData{
		AgentID: agentID,
	}

	msg := proto.Message{
		Type: "agent_info_request",
		Data: mustMarshal(reqData),
	}

	data, _ := json.Marshal(msg)
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// GetTunnels returns current tunnels
func (c *Client) GetTunnels() map[string]*Tunnel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*Tunnel)
	for k, v := range c.tunnels {
		result[k] = v
	}
	return result
}

// handleTunnelResponse handles tunnel response from server
func (c *Client) handleTunnelResponse(msg *proto.Message) error {
	var resp proto.TunnelResponseData
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel response: %w", err)
	}

	c.mu.Lock()
	if tunnel, exists := c.tunnels[resp.TunnelID]; exists {
		tunnel.Active = (resp.Status == "success")
	}
	c.mu.Unlock()

	if resp.Status == "success" {
		c.logger.Printf("Tunnel established: %s", resp.TunnelID)
	} else {
		c.logger.Printf("Tunnel failed: %s - %s", resp.TunnelID, resp.Message)
	}

	// Notify UI
	if c.ui != nil {
		c.ui.NotifyTunnelResponse(&resp)
	}

	return nil
}

// handleTunnelData handles tunnel data from server
func (c *Client) handleTunnelData(msg *proto.Message) error {
	var data proto.TunnelDataMessage
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel data: %w", err)
	}

	// Forward data to local connection
	// This would be implemented with actual local port forwarding
	c.logger.Printf("Received tunnel data: %s (%d bytes)", data.TunnelID, len(data.Data))

	return nil
}

// handleTunnelClose handles tunnel close from server
func (c *Client) handleTunnelClose(msg *proto.Message) error {
	var closeMsg proto.TunnelCloseMessage
	if err := json.Unmarshal(msg.Data, &closeMsg); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel close: %w", err)
	}

	c.mu.Lock()
	delete(c.tunnels, closeMsg.TunnelID)
	c.mu.Unlock()

	c.logger.Printf("Tunnel closed by server: %s - %s", closeMsg.TunnelID, closeMsg.Reason)

	// Notify UI
	if c.ui != nil {
		c.ui.NotifyTunnelClose(&closeMsg)
	}

	return nil
}

// handleAgentList handles agent list from server
func (c *Client) handleAgentList(msg *proto.Message) error {
	var agentList proto.AgentListData
	if err := json.Unmarshal(msg.Data, &agentList); err != nil {
		return fmt.Errorf("failed to unmarshal agent list: %w", err)
	}

	c.logger.Printf("Received agent list: %d agents", len(agentList.Agents))

	// Notify UI
	if c.ui != nil {
		c.ui.NotifyAgentList(&agentList)
	}

	return nil
}

// handleAgentInfoResponse handles agent info response from server
func (c *Client) handleAgentInfoResponse(msg *proto.Message) error {
	var info proto.AgentInfo
	if err := json.Unmarshal(msg.Data, &info); err != nil {
		return fmt.Errorf("failed to unmarshal agent info: %w", err)
	}

	c.logger.Printf("Received agent info: %s", info.ID)

	// Notify UI
	if c.ui != nil {
		c.ui.NotifyAgentInfo(&info)
	}

	return nil
}

// keepAlive maintains connection with server
func (c *Client) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Send ping
			if c.conn != nil {
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.logger.Printf("Failed to send ping: %v", err)
					return
				}
			}
		}
	}
}

// Helper functions
func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().Unix())
}

func generateTunnelID() string {
	return fmt.Sprintf("tunnel-%d", time.Now().UnixNano())
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return json.RawMessage("{}")
	}
	return json.RawMessage(data)
}
