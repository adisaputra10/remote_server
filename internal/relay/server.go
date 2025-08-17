package relay

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"remote-tunnel/internal/proto"
	"remote-tunnel/internal/transport"
)

type Server struct {
	token   string
	agents  map[string]*AgentSession
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

type AgentSession struct {
	ID      string
	Session *transport.MuxSession
	ctx     context.Context
	cancel  context.CancelFunc
}

type ClientSession struct {
	Session *transport.MuxSession
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewServer(token string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		token:  token,
		agents: make(map[string]*AgentSession),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Server) HandleAgent(w http.ResponseWriter, r *http.Request) {
	wsConn, err := transport.AcceptWS(w, r, s.token)
	if err != nil {
		log.Printf("Agent websocket accept failed: %v", err)
		return
	}
	defer wsConn.Close()

	wsConn.StartPingPong()

	session, err := transport.NewMuxServer(wsConn)
	if err != nil {
		log.Printf("Agent mux server failed: %v", err)
		return
	}
	defer session.Close()

	// Wait for REGISTER message
	msg, err := session.ReceiveControl()
	if err != nil {
		log.Printf("Agent receive register failed: %v", err)
		return
	}

	if msg.Type != proto.MsgRegister {
		log.Printf("Expected REGISTER, got %s", msg.Type)
		return
	}

	if msg.Token != s.token {
		log.Printf("Invalid agent token")
		session.SendControl(&proto.Control{
			Type:  proto.MsgError,
			Error: "Invalid token",
		})
		return
	}

	agentID := msg.AgentID
	if agentID == "" {
		log.Printf("Empty agent ID")
		session.SendControl(&proto.Control{
			Type:  proto.MsgError,
			Error: "Empty agent ID",
		})
		return
	}

	log.Printf("Agent %s registered", agentID)

	ctx, cancel := context.WithCancel(s.ctx)
	agentSession := &AgentSession{
		ID:      agentID,
		Session: session,
		ctx:     ctx,
		cancel:  cancel,
	}

	s.mu.Lock()
	s.agents[agentID] = agentSession
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.agents, agentID)
		s.mu.Unlock()
		cancel()
		log.Printf("Agent %s disconnected", agentID)
	}()

	// Handle agent requests
	s.handleAgentRequests(agentSession)
}

func (s *Server) HandleClient(w http.ResponseWriter, r *http.Request) {
	wsConn, err := transport.AcceptWS(w, r, s.token)
	if err != nil {
		log.Printf("Client websocket accept failed: %v", err)
		return
	}
	defer wsConn.Close()

	wsConn.StartPingPong()

	session, err := transport.NewMuxClient(wsConn)
	if err != nil {
		log.Printf("Client mux client failed: %v", err)
		return
	}
	defer session.Close()

	ctx, cancel := context.WithCancel(s.ctx)
	clientSession := &ClientSession{
		Session: session,
		ctx:     ctx,
		cancel:  cancel,
	}
	defer cancel()

	log.Printf("Client connected")

	// Handle client requests
	s.handleClientRequests(clientSession)
}

func (s *Server) handleAgentRequests(agent *AgentSession) {
	for {
		select {
		case <-agent.ctx.Done():
			return
		default:
		}

		msg, err := agent.Session.ReceiveControl()
		if err != nil {
			log.Printf("Agent %s control receive error: %v", agent.ID, err)
			return
		}

		switch msg.Type {
		case proto.MsgPing:
			agent.Session.SendControl(&proto.Control{Type: proto.MsgPong})
		case proto.MsgPong:
			// Keep alive received
		default:
			log.Printf("Agent %s unexpected message: %s", agent.ID, msg.Type)
		}
	}
}

func (s *Server) handleClientRequests(client *ClientSession) {
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
		}

		msg, err := client.Session.ReceiveControl()
		if err != nil {
			log.Printf("Client control receive error: %v", err)
			return
		}

		switch msg.Type {
		case proto.MsgDial:
			go s.handleDial(client, msg)
		case proto.MsgPing:
			client.Session.SendControl(&proto.Control{Type: proto.MsgPong})
		case proto.MsgPong:
			// Keep alive received
		default:
			log.Printf("Client unexpected message: %s", msg.Type)
		}
	}
}

func (s *Server) handleDial(client *ClientSession, dialMsg *proto.Control) {
	agentID := dialMsg.AgentID
	streamID := dialMsg.StreamID
	targetAddr := dialMsg.TargetAddr

	if streamID == "" {
		streamID = uuid.New().String()
	}

	log.Printf("Dial request: agent=%s, target=%s, stream=%s", agentID, targetAddr, streamID)

	s.mu.RLock()
	agent, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		client.Session.SendControl(&proto.Control{
			Type:     proto.MsgRefuse,
			StreamID: streamID,
			Error:    "Agent not found",
		})
		return
	}

	// Send DIAL to agent
	err := agent.Session.SendControl(&proto.Control{
		Type:       proto.MsgDial,
		StreamID:   streamID,
		TargetAddr: targetAddr,
	})
	if err != nil {
		log.Printf("Failed to send dial to agent: %v", err)
		client.Session.SendControl(&proto.Control{
			Type:     proto.MsgRefuse,
			StreamID: streamID,
			Error:    "Failed to contact agent",
		})
		return
	}

	// Open stream to client
	clientStream, err := client.Session.OpenStream()
	if err != nil {
		log.Printf("Failed to open client stream: %v", err)
		return
	}
	defer clientStream.Close()

	// Accept stream from agent
	agentStream, err := agent.Session.AcceptStream()
	if err != nil {
		log.Printf("Failed to accept agent stream: %v", err)
		client.Session.SendControl(&proto.Control{
			Type:     proto.MsgRefuse,
			StreamID: streamID,
			Error:    "Failed to establish agent stream",
		})
		return
	}
	defer agentStream.Close()

	// Send ACCEPT to client
	client.Session.SendControl(&proto.Control{
		Type:     proto.MsgAccept,
		StreamID: streamID,
	})

	log.Printf("Bridging streams for %s", streamID)

	// Bridge the streams
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transport.Bridge(ctx, clientStream, agentStream)
}

func (s *Server) Close() error {
	s.cancel()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, agent := range s.agents {
		agent.cancel()
		agent.Session.Close()
	}
	
	return nil
}
