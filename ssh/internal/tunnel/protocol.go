package tunnel

import (
	"encoding/json"
	"time"
)

// Message types for tunnel communication
const (
	// Agent messages
	MsgAgentRegister   = "agent_register"
	MsgAgentHeartbeat  = "agent_heartbeat"
	MsgAgentDisconnect = "agent_disconnect"
	
	// Client messages
	MsgClientConnect    = "client_connect"
	MsgClientDisconnect = "client_disconnect"
	MsgTunnelRequest    = "tunnel_request"
	MsgTunnelClose      = "tunnel_close"
	
	// Server responses
	MsgAgentRegistered  = "agent_registered"
	MsgTunnelCreated    = "tunnel_created"
	MsgTunnelSuccess    = "tunnel_success"
	MsgTunnelError      = "tunnel_error"
	MsgAgentList        = "agent_list"
	
	// Stream messages
	MsgStreamOpen  = "stream_open"
	MsgStreamClose = "stream_close"
	MsgStreamData  = "stream_data"
)

type Message struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	TunnelID  string                 `json:"tunnel_id,omitempty"`
	StreamID  string                 `json:"stream_id,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Error     string                 `json:"error,omitempty"`
}

type AgentInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Platform  string            `json:"platform"`
	Version   string            `json:"version"`
	Status    string            `json:"status"`
	Targets   []string          `json:"targets"`
	LastSeen  time.Time         `json:"last_seen"`
	Metadata  map[string]string `json:"metadata"`
	Endpoints []string          `json:"endpoints"`
}

type TunnelRequest struct {
	TunnelID     string `json:"tunnel_id"`
	AgentID      string `json:"agent_id"`
	ClientID     string `json:"client_id"`
	LocalHost    string `json:"local_host"`
	LocalPort    int    `json:"local_port"`
	RemoteHost   string `json:"remote_host"`
	RemotePort   int    `json:"remote_port"`
	Protocol     string `json:"protocol"` // tcp, udp
	Description  string `json:"description,omitempty"`
}

type TunnelInfo struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	ClientID    string    `json:"client_id"`
	LocalAddr   string    `json:"local_addr"`
	RemoteAddr  string    `json:"remote_addr"`
	Protocol    string    `json:"protocol"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	Connections int       `json:"connections"`
}

func NewMessage(msgType string) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: time.Now(),
	}
}

func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Message) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

func (m *Message) SetData(data interface{}) *Message {
	m.Data = data
	return m
}

func (m *Message) SetError(err string) *Message {
	m.Error = err
	return m
}

func (m *Message) SetAgentID(agentID string) *Message {
	m.AgentID = agentID
	return m
}

func (m *Message) SetClientID(clientID string) *Message {
	m.ClientID = clientID
	return m
}

func (m *Message) SetTunnelID(tunnelID string) *Message {
	m.TunnelID = tunnelID
	return m
}

func (m *Message) SetMetadata(key string, value interface{}) *Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[key] = value
	return m
}

// MapToStruct converts a map[string]interface{} to a struct using JSON marshaling
func MapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}
