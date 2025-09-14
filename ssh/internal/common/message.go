package common

import (
    "encoding/json"
    "fmt"
)

// MessageType defines the type of message being sent
type MessageType string

const (
    // Message types
    MsgTypeRegister    MessageType = "register"
    MsgTypeConnect     MessageType = "connect"
    MsgTypeData        MessageType = "data"
    MsgTypeClose       MessageType = "close"
    MsgTypeHeartbeat   MessageType = "heartbeat"
    MsgTypeError       MessageType = "error"
)

// Message represents a message exchanged between relay, agent, and client
type Message struct {
    Type      MessageType `json:"type"`
    AgentID   string      `json:"agent_id,omitempty"`
    ClientID  string      `json:"client_id,omitempty"`
    SessionID string      `json:"session_id,omitempty"`
    Target    string      `json:"target,omitempty"`
    Data      []byte      `json:"data,omitempty"`
    Error     string      `json:"error,omitempty"`
    Timestamp int64       `json:"timestamp"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType) *Message {
    return &Message{
        Type:      msgType,
        Timestamp: GetCurrentTimestamp(),
    }
}

// ToJSON converts message to JSON bytes
func (m *Message) ToJSON() ([]byte, error) {
    return json.Marshal(m)
}

// FromJSON parses JSON bytes to message
func FromJSON(data []byte) (*Message, error) {
    var msg Message
    err := json.Unmarshal(data, &msg)
    return &msg, err
}

// String returns string representation of message
func (m *Message) String() string {
    sessionInfo := m.SessionID
    if sessionInfo == "" {
        sessionInfo = "<empty>"
    }
    return fmt.Sprintf("Message{Type: %s, AgentID: %s, ClientID: %s, SessionID: %s, Target: %s, DataLen: %d}",
        m.Type, m.AgentID, m.ClientID, sessionInfo, m.Target, len(m.Data))
}