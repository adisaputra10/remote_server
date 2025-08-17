package proto

type MsgType string

const (
	MsgRegister MsgType = "REGISTER"
	MsgDial     MsgType = "DIAL"
	MsgAccept   MsgType = "ACCEPT"
	MsgRefuse   MsgType = "REFUSE"
	MsgPing     MsgType = "PING"
	MsgPong     MsgType = "PONG"
	MsgError    MsgType = "ERROR"
)

type Control struct {
	Type       MsgType `json:"type"`
	AgentID    string  `json:"agent_id,omitempty"`
	StreamID   string  `json:"stream_id,omitempty"`
	TargetAddr string  `json:"target_addr,omitempty"`
	Token      string  `json:"token,omitempty"`
	Error      string  `json:"error,omitempty"`
}
