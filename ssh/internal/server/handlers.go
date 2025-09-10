package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"ssh-terminal/internal/proto"
)

// handleLegacyTunnel handles legacy tunnel connections for existing client compatibility
func (s *Server) handleLegacyTunnel(conn *websocket.Conn, remoteAddr string) {
	s.logger.Printf("🔗 Legacy tunnel connection from: %s", remoteAddr)
	defer conn.Close()

	// Set up message reading
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Read tunnel request
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		s.logger.Printf("❌ Failed to read legacy tunnel request: %v", err)
		return
	}

	msg, err := proto.FromJSON(msgBytes)
	if err != nil {
		s.logger.Printf("❌ Failed to parse legacy tunnel request: %v", err)
		return
	}

	s.logger.Printf("📋 Legacy tunnel request: %s", msg.Type)

	// Send acknowledgment for now
	ackMsg := &proto.Message{
		Type:      proto.MessageTypeTunnelReady,
		SessionID: msg.SessionID,
		Timestamp: time.Now().Unix(),
	}

	ackBytes, _ := ackMsg.ToJSON()
	if err := conn.WriteMessage(websocket.TextMessage, ackBytes); err != nil {
		s.logger.Printf("❌ Failed to send legacy tunnel ack: %v", err)
		return
	}

	// Keep connection alive for now
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			s.logger.Printf("📝 Legacy tunnel connection closed: %v", err)
			break
		}
	}
}

// handleTunnelRequest handles tunnel requests from clients
func (s *Server) handleTunnelRequest(client *ClientConnection, msg *proto.Message, stream net.Conn) {
	tunnelInfo, ok := msg.Data.(*proto.TunnelInfo)
	if !ok {
		s.logger.Printf("❌ Invalid tunnel request data")
		return
	}

	s.logger.Printf("🚇 Tunnel request from %s: %s:%d -> %s:%d",
		client.Info.Name, tunnelInfo.LocalHost, tunnelInfo.LocalPort,
		tunnelInfo.RemoteHost, tunnelInfo.RemotePort)

	// Find available agent for the tunnel
	s.mu.RLock()
	var targetAgent *AgentConnection
	for _, agent := range s.agents {
		// For now, use the first available agent
		// TODO: Implement agent selection logic
		targetAgent = agent
		break
	}
	s.mu.RUnlock()

	if targetAgent == nil {
		s.logger.Printf("❌ No agents available for tunnel")

		// Send error response
		errorMsg := &proto.Message{
			Type:      proto.MessageTypeTunnelError,
			SessionID: msg.SessionID,
			Error:     "No agents available",
			Timestamp: time.Now().Unix(),
		}
		errorBytes, _ := errorMsg.ToJSON()
		stream.Write(errorBytes)
		return
	}

	s.logger.Printf("🎯 Selected agent: %s for tunnel", targetAgent.Info.Name)

	// Create tunnel info
	tunnelID := fmt.Sprintf("tunnel_%d", time.Now().UnixNano())
	tunnel := &TunnelConnection{
		ID:       tunnelID,
		AgentID:  targetAgent.ID,
		ClientID: client.ID,
		Info:     tunnelInfo,
	}

	s.mu.Lock()
	s.tunnels[tunnelID] = tunnel
	s.mu.Unlock()

	// Send tunnel request to agent
	tunnelMsg := &proto.Message{
		Type:      proto.MessageTypeClientTunnel,
		SessionID: msg.SessionID,
		Data:      tunnelInfo,
		Timestamp: time.Now().Unix(),
	}

	agentStream, err := targetAgent.Session.OpenStream()
	if err != nil {
		s.logger.Printf("❌ Failed to open agent stream: %v", err)
		return
	}
	defer agentStream.Close()

	tunnelBytes, _ := tunnelMsg.ToJSON()
	if _, err := agentStream.Write(tunnelBytes); err != nil {
		s.logger.Printf("❌ Failed to send tunnel request to agent: %v", err)
		return
	}

	// Forward data between client and agent streams
	s.forwardTunnelData(stream, agentStream, tunnel)
}

// handleCommandRequest handles command requests from clients
func (s *Server) handleCommandRequest(client *ClientConnection, msg *proto.Message, stream net.Conn) {
	s.logger.Printf("📋 Command request from %s: %s", client.Info.Name, msg.Type)

	// Create response based on command type
	var response *proto.Message

	switch msg.Type {
	case proto.MessageTypeClientCommand:
		// List agents or other commands
		agentList := make([]*proto.AgentInfo, 0)
		s.mu.RLock()
		for _, agent := range s.agents {
			agentList = append(agentList, agent.Info)
		}
		s.mu.RUnlock()

		response = &proto.Message{
			Type:      proto.MessageTypeServerResponse,
			SessionID: msg.SessionID,
			Data:      agentList,
			Timestamp: time.Now().Unix(),
		}

	default:
		response = &proto.Message{
			Type:      proto.MessageTypeServerResponse,
			SessionID: msg.SessionID,
			Error:     "Unknown command",
			Timestamp: time.Now().Unix(),
		}
	}

	// Send response
	respBytes, _ := response.ToJSON()
	stream.Write(respBytes)
}

// forwardTunnelData forwards data between client and agent streams
func (s *Server) forwardTunnelData(clientStream, agentStream net.Conn, tunnel *TunnelConnection) {
	s.logger.Printf("🔀 Starting data forwarding for tunnel %s", tunnel.ID)

	// Forward client -> agent
	go func() {
		defer agentStream.Close()
		buffer := make([]byte, 32768)
		for {
			n, err := clientStream.Read(buffer)
			if err != nil {
				s.logger.Printf("📝 Client stream closed: %v", err)
				break
			}

			if _, err := agentStream.Write(buffer[:n]); err != nil {
				s.logger.Printf("❌ Failed to write to agent: %v", err)
				break
			}
		}
	}()

	// Forward agent -> client
	go func() {
		defer clientStream.Close()
		buffer := make([]byte, 32768)
		for {
			n, err := agentStream.Read(buffer)
			if err != nil {
				s.logger.Printf("📝 Agent stream closed: %v", err)
				break
			}

			if _, err := clientStream.Write(buffer[:n]); err != nil {
				s.logger.Printf("❌ Failed to write to client: %v", err)
				break
			}
		}
	}()
}

// API Handlers

// handleAgentsAPI handles agents API endpoint
func (s *Server) handleAgentsAPI(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	agentList := make([]*proto.AgentInfo, 0)
	for _, agent := range s.agents {
		agentList = append(agentList, agent.Info)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agentList,
		"count":  len(agentList),
	})
}

// handleClientsAPI handles clients API endpoint
func (s *Server) handleClientsAPI(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	clientList := make([]*proto.ClientInfo, 0)
	for _, client := range s.clients {
		clientList = append(clientList, client.Info)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clients": clientList,
		"count":   len(clientList),
	})
}

// handleTunnelsAPI handles tunnels API endpoint
func (s *Server) handleTunnelsAPI(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tunnelList := make([]*proto.TunnelInfo, 0)
	for _, tunnel := range s.tunnels {
		tunnelList = append(tunnelList, tunnel.Info)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tunnels": tunnelList,
		"count":   len(tunnelList),
	})
}
