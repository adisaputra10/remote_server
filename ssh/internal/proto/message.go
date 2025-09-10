package proto

import (
	"encoding/json"
	"time"
)

// MessageType defines the type of message
type MessageType string

const (
	// Agent messages
	MessageTypeAgentRegister   MessageType = "agent_register"
	MessageTypeAgentHeartbeat  MessageType = "agent_heartbeat"
	MessageTypeAgentDisconnect MessageType = "agent_disconnect"

	// Client messages
	MessageTypeClientConnect MessageType = "client_connect"
	MessageTypeClientCommand MessageType = "client_command"
	MessageTypeClientTunnel  MessageType = "client_tunnel"

	// Tunnel messages
	MessageTypeTunnelStart MessageType = "tunnel_start"
	MessageTypeTunnelData  MessageType = "tunnel_data"
	MessageTypeTunnelClose MessageType = "tunnel_close"
	MessageTypeTunnelReady MessageType = "tunnel_ready"
	MessageTypeTunnelError MessageType = "tunnel_error"

	// Database messages
	MessageTypeDatabaseCommand MessageType = "database_command"
	MessageTypeDatabaseResult  MessageType = "database_result"

	// Response messages
	MessageTypeResponse MessageType = "response"
	MessageTypeError    MessageType = "error"
)

// Message represents a protocol message
type Message struct {
	Type      MessageType            `json:"type"`
	ID        string                 `json:"id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Error     string                 `json:"error,omitempty"`
}

// AgentInfo represents agent information
type AgentInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Platform     string            `json:"platform"`
	Version      string            `json:"version"`
	Status       string            `json:"status"`
	Capabilities []string          `json:"capabilities"`
	Hostname     string            `json:"hostname"`
	IPAddress    string            `json:"ip_address"`
	Metadata     map[string]string `json:"metadata"`
	LastSeen     time.Time         `json:"last_seen"`
}

// AgentRegisterData for agent registration
type AgentRegisterData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
	Token    string `json:"token"`
}

// ClientRegisterData for client registration
type ClientRegisterData struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// HeartbeatData for heartbeat messages
type HeartbeatData struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// TunnelRequestData for tunnel requests
type TunnelRequestData struct {
	TunnelID   string `json:"tunnel_id"`
	AgentID    string `json:"agent_id"`
	Type       string `json:"type"`
	LocalPort  int    `json:"local_port"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
}

// TunnelResponseData for tunnel responses
type TunnelResponseData struct {
	TunnelID string `json:"tunnel_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// TunnelDataMessage for tunnel data
type TunnelDataMessage struct {
	TunnelID string `json:"tunnel_id"`
	Data     []byte `json:"data"`
}

// TunnelCloseMessage for tunnel close
type TunnelCloseMessage struct {
	TunnelID string `json:"tunnel_id"`
	Reason   string `json:"reason"`
}

// AgentListData for agent list responses
type AgentListData struct {
	Agents []AgentInfo `json:"agents"`
}

// AgentInfoRequestData for agent info requests
type AgentInfoRequestData struct {
	AgentID string `json:"agent_id"`
}

// ClientInfo represents client information
type ClientInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Username string            `json:"username"`
	IP       string            `json:"ip"`
	Metadata map[string]string `json:"metadata"`
	LastSeen time.Time         `json:"last_seen"`
}

// TunnelRequest represents a tunnel request
type TunnelRequest struct {
	AgentID    string `json:"agent_id"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port"`
	LocalPort  int    `json:"local_port,omitempty"`
	Protocol   string `json:"protocol"` // tcp, udp, mysql, postgres
}

// TunnelInfo represents tunnel information
type TunnelInfo struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agent_id"`
	ClientID   string    `json:"client_id"`
	TargetHost string    `json:"target_host"`
	TargetPort int       `json:"target_port"`
	LocalPort  int       `json:"local_port"`
	Protocol   string    `json:"protocol"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// DatabaseCommand represents a database command execution
type DatabaseCommand struct {
	SessionID string                 `json:"session_id"`
	AgentID   string                 `json:"agent_id"`
	Command   string                 `json:"command"`
	Protocol  string                 `json:"protocol"`
	ClientIP  string                 `json:"client_ip"`
	ProxyName string                 `json:"proxy_name"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// ToJSON converts message to JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON creates message from JSON
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
