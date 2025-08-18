package relay

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"remote-tunnel/internal/proto"
	"remote-tunnel/internal/transport"
)

type Server struct {
	token    string
	agents   map[string]*AgentSession
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	compress bool
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
		token:    token,
		agents:   make(map[string]*AgentSession),
		ctx:      ctx,
		cancel:   cancel,
		compress: false,
	}
}

func NewServerWithCompression(token string, compress bool) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		token:    token,
		agents:   make(map[string]*AgentSession),
		ctx:      ctx,
		cancel:   cancel,
		compress: compress,
	}
}

func (s *Server) HandleAgent(w http.ResponseWriter, r *http.Request) {
	wsConn, err := transport.AcceptWSWithCompression(w, r, s.token, s.compress)
	if err != nil {
		log.Printf("Agent websocket accept failed: %v", err)
		return
	}
	defer wsConn.Close()

	wsConn.StartPingPong()

	session, err := transport.NewMuxServerWithCompression(wsConn, s.compress)
	if err != nil {
		log.Printf("Agent mux server failed: %v", err)
		return
	}
	defer session.Close()

	// Wait for REGISTER message from agent
	log.Printf("Waiting for REGISTER message from agent...")
	msg, err := session.ReceiveControl()
	if err != nil {
		log.Printf("Agent receive register failed: %v", err)
		return
	}

	log.Printf("Received message type: %s", msg.Type)
	if msg.Type != proto.MsgRegister {
		log.Printf("Expected REGISTER, got %s", msg.Type)
		// Send error and close connection
		session.SendControl(&proto.Control{
			Type:  proto.MsgError,
			Error: fmt.Sprintf("Expected REGISTER, got %s", msg.Type),
		})
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
	wsConn, err := transport.AcceptWSWithCompression(w, r, s.token, s.compress)
	if err != nil {
		log.Printf("Client websocket accept failed: %v", err)
		return
	}
	defer wsConn.Close()

	wsConn.StartPingPong()

	session, err := transport.NewMuxServerWithCompression(wsConn, s.compress)
	if err != nil {
		log.Printf("Client mux server failed: %v", err)
		return
	}
	defer session.Close()

	log.Printf("Client connected")

	ctx, cancel := context.WithCancel(s.ctx)
	clientSession := &ClientSession{
		Session: session,
		ctx:     ctx,
		cancel:  cancel,
	}

	defer func() {
		cancel()
		log.Printf("Client disconnected")
	}()

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
	defer log.Printf("Client request handler stopped")
	
	for {
		select {
		case <-client.ctx.Done():
			log.Printf("Client request handler exiting due to context cancellation")
			return
		default:
		}

		msg, err := client.Session.ReceiveControl()
		if err != nil {
			log.Printf("Client control receive error: %v", err)
			// Check if it's a clean shutdown or connection issue
			select {
			case <-client.ctx.Done():
				log.Printf("Client context cancelled during receive")
			default:
				log.Printf("Client connection issue detected")
			}
			return
		}

		switch msg.Type {
		case proto.MsgDial:
			go s.handleDial(client, msg)
		case proto.MsgPing:
			err := client.Session.SendControl(&proto.Control{Type: proto.MsgPong})
			if err != nil {
				log.Printf("Failed to send pong to client: %v", err)
				return
			}
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
