package proto

import (
	"encoding/json"
	"testing"
)

func TestControlMarshaling(t *testing.T) {
	tests := []struct {
		name string
		ctrl Control
	}{
		{
			name: "Register message",
			ctrl: Control{
				Type:    MsgRegister,
				AgentID: "test-agent",
				Token:   "test-token",
			},
		},
		{
			name: "Dial message",
			ctrl: Control{
				Type:       MsgDial,
				AgentID:    "test-agent",
				StreamID:   "stream-123",
				TargetAddr: "127.0.0.1:22",
			},
		},
		{
			name: "Accept message",
			ctrl: Control{
				Type:     MsgAccept,
				StreamID: "stream-123",
			},
		},
		{
			name: "Error message",
			ctrl: Control{
				Type:  MsgError,
				Error: "Connection failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.ctrl)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal back
			var result Control
			err = json.Unmarshal(data, &result)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Compare
			if result.Type != tt.ctrl.Type {
				t.Errorf("Type mismatch: got %s, want %s", result.Type, tt.ctrl.Type)
			}
			if result.AgentID != tt.ctrl.AgentID {
				t.Errorf("AgentID mismatch: got %s, want %s", result.AgentID, tt.ctrl.AgentID)
			}
			if result.StreamID != tt.ctrl.StreamID {
				t.Errorf("StreamID mismatch: got %s, want %s", result.StreamID, tt.ctrl.StreamID)
			}
			if result.TargetAddr != tt.ctrl.TargetAddr {
				t.Errorf("TargetAddr mismatch: got %s, want %s", result.TargetAddr, tt.ctrl.TargetAddr)
			}
		})
	}
}
