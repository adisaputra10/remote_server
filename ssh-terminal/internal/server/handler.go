package server

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"ssh-terminal/internal/proto"
)

// handleAgentMessages processes messages from agents
func (s *Server) handleAgentMessages(agent *AgentConnection) {
	defer func() {
		s.logger.Printf("Agent %s disconnected", agent.ID)
		s.removeAgent(agent.ID)
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Read message from agent - use websocket connection
		_, data, err := agent.Session.GetWebSocketConn().ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Printf("Agent %s websocket error: %v", agent.ID, err)
			}
			return
		}

		// Parse message
		var msg proto.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			s.logger.Printf("Failed to parse message from agent %s: %v", agent.ID, err)
			continue
		}

		// Update last seen
		agent.LastSeen = time.Now()

		// Handle message
		if err := s.handleAgentMessage(agent, &msg); err != nil {
			s.logger.Printf("Error handling message from agent %s: %v", agent.ID, err)
		}
	}
}

// handleAgentMessage handles a specific message from an agent
func (s *Server) handleAgentMessage(agent *AgentConnection, msg *proto.Message) error {
	switch msg.Type {
	case "heartbeat":
		return s.handleAgentHeartbeat(agent, msg)
	case "tunnel_response":
		return s.handleTunnelResponse(agent, msg)
	case "tunnel_data":
		return s.handleTunnelData(agent, msg)
	case "tunnel_close":
		return s.handleTunnelClose(agent, msg)
	case "agent_info":
		return s.handleAgentInfo(agent, msg)
	default:
		s.logger.Printf("Unknown message type from agent %s: %s", agent.ID, msg.Type)
		return nil
	}
}

// handleClientMessages processes messages from clients
func (s *Server) handleClientMessages(client *ClientConnection) {
	defer func() {
		s.logger.Printf("Client %s disconnected", client.ID)
		s.removeClient(client.ID)
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Read message from client
		_, data, err := client.Session.GetWebSocketConn().ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Printf("Client %s websocket error: %v", client.ID, err)
			}
			return
		}

		// Parse message
		var msg proto.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			s.logger.Printf("Failed to parse message from client %s: %v", client.ID, err)
			continue
		}

		// Handle message
		if err := s.handleClientMessage(client, &msg); err != nil {
			s.logger.Printf("Error handling message from client %s: %v", client.ID, err)
		}
	}
}

// handleClientMessage handles a specific message from a client
func (s *Server) handleClientMessage(client *ClientConnection, msg *proto.Message) error {
	switch msg.Type {
	case "tunnel_request":
		return s.handleTunnelRequest(client, msg)
	case "tunnel_data":
		return s.handleClientTunnelData(client, msg)
	case "tunnel_close":
		return s.handleClientTunnelClose(client, msg)
	case "list_agents":
		return s.handleListAgents(client, msg)
	case "agent_info_request":
		return s.handleAgentInfoRequest(client, msg)
	default:
		s.logger.Printf("Unknown message type from client %s: %s", client.ID, msg.Type)
		return nil
	}
}

// handleAgentHeartbeat handles heartbeat from agent
func (s *Server) handleAgentHeartbeat(agent *AgentConnection, msg *proto.Message) error {
	var heartbeat proto.HeartbeatData
	if data, ok := msg.Data.(json.RawMessage); ok {
		if err := json.Unmarshal(data, &heartbeat); err != nil {
			return fmt.Errorf("failed to unmarshal heartbeat: %w", err)
		}
	} else {
		return fmt.Errorf("invalid heartbeat data type")
	}

	s.logger.Printf("Heartbeat from agent %s: %s", agent.ID, heartbeat.Status)

	// Send heartbeat response
	response := proto.Message{
		Type: "heartbeat_response",
		Data: json.RawMessage(`{"status": "ok"}`),
	}

	return s.sendToAgent(agent.ID, &response)
}

// handleTunnelRequest handles tunnel request from client
func (s *Server) handleTunnelRequest(client *ClientConnection, msg *proto.Message) error {
	var req proto.TunnelRequestData
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel request: %w", err)
	}

	s.logger.Printf("Tunnel request from client %s: %+v", client.ID, req)

	// Find target agent
	s.mu.RLock()
	agent, exists := s.agents[req.AgentID]
	s.mu.RUnlock()

	if !exists {
		// Send error response to client
		response := proto.Message{
			Type: "tunnel_response",
			Data: json.RawMessage(fmt.Sprintf(`{"tunnel_id": "%s", "status": "error", "message": "Agent not found"}`, req.TunnelID)),
		}
		return s.sendToClient(client.ID, &response)
	}

	// Create tunnel record
	tunnel := &TunnelConnection{
		ID:         req.TunnelID,
		AgentID:    req.AgentID,
		ClientID:   client.ID,
		Type:       req.Type,
		LocalPort:  req.LocalPort,
		RemoteHost: req.RemoteHost,
		RemotePort: req.RemotePort,
		Active:     false,
	}

	s.mu.Lock()
	s.tunnels[tunnel.ID] = tunnel
	s.mu.Unlock()

	// Forward request to agent
	return s.sendToAgent(agent.ID, msg)
}

// handleTunnelResponse handles tunnel response from agent
func (s *Server) handleTunnelResponse(agent *AgentConnection, msg *proto.Message) error {
	var resp proto.TunnelResponseData
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel response: %w", err)
	}

	s.logger.Printf("Tunnel response from agent %s: %+v", agent.ID, resp)

	// Update tunnel status
	s.mu.Lock()
	if tunnel, exists := s.tunnels[resp.TunnelID]; exists {
		tunnel.Active = (resp.Status == "success")
	}
	s.mu.Unlock()

	// Find target client and forward response
	s.mu.RLock()
	var targetClientID string
	if tunnel, exists := s.tunnels[resp.TunnelID]; exists {
		targetClientID = tunnel.ClientID
	}
	s.mu.RUnlock()

	if targetClientID != "" {
		return s.sendToClient(targetClientID, msg)
	}

	return nil
}

// handleTunnelData handles tunnel data from agent
func (s *Server) handleTunnelData(agent *AgentConnection, msg *proto.Message) error {
	var data proto.TunnelDataMessage
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel data: %w", err)
	}

	// Forward to client
	s.mu.RLock()
	var targetClientID string
	if tunnel, exists := s.tunnels[data.TunnelID]; exists {
		targetClientID = tunnel.ClientID
	}
	s.mu.RUnlock()

	if targetClientID != "" {
		return s.sendToClient(targetClientID, msg)
	}

	return nil
}

// handleClientTunnelData handles tunnel data from client
func (s *Server) handleClientTunnelData(client *ClientConnection, msg *proto.Message) error {
	var data proto.TunnelDataMessage
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel data: %w", err)
	}

	// Forward to agent
	s.mu.RLock()
	var targetAgentID string
	if tunnel, exists := s.tunnels[data.TunnelID]; exists {
		targetAgentID = tunnel.AgentID
	}
	s.mu.RUnlock()

	if targetAgentID != "" {
		return s.sendToAgent(targetAgentID, msg)
	}

	return nil
}

// handleTunnelClose handles tunnel close from agent
func (s *Server) handleTunnelClose(agent *AgentConnection, msg *proto.Message) error {
	var closeMsg proto.TunnelCloseMessage
	if err := json.Unmarshal(msg.Data, &closeMsg); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel close: %w", err)
	}

	s.logger.Printf("Tunnel close from agent %s: %s", agent.ID, closeMsg.TunnelID)

	// Remove tunnel
	s.mu.Lock()
	var targetClientID string
	if tunnel, exists := s.tunnels[closeMsg.TunnelID]; exists {
		targetClientID = tunnel.ClientID
		delete(s.tunnels, closeMsg.TunnelID)
	}
	s.mu.Unlock()

	// Forward to client
	if targetClientID != "" {
		return s.sendToClient(targetClientID, msg)
	}

	return nil
}

// handleClientTunnelClose handles tunnel close from client
func (s *Server) handleClientTunnelClose(client *ClientConnection, msg *proto.Message) error {
	var closeMsg proto.TunnelCloseMessage
	if err := json.Unmarshal(msg.Data, &closeMsg); err != nil {
		return fmt.Errorf("failed to unmarshal tunnel close: %w", err)
	}

	s.logger.Printf("Tunnel close from client %s: %s", client.ID, closeMsg.TunnelID)

	// Remove tunnel
	s.mu.Lock()
	var targetAgentID string
	if tunnel, exists := s.tunnels[closeMsg.TunnelID]; exists {
		targetAgentID = tunnel.AgentID
		delete(s.tunnels, closeMsg.TunnelID)
	}
	s.mu.Unlock()

	// Forward to agent
	if targetAgentID != "" {
		return s.sendToAgent(targetAgentID, msg)
	}

	return nil
}

// handleListAgents handles agent list request from client
func (s *Server) handleListAgents(client *ClientConnection, msg *proto.Message) error {
	s.mu.RLock()
	agentList := make([]proto.AgentInfo, 0, len(s.agents))
	for _, agent := range s.agents {
		agentList = append(agentList, *agent.Info)
	}
	s.mu.RUnlock()

	response := proto.Message{
		Type: "agent_list",
		Data: must(json.Marshal(proto.AgentListData{Agents: agentList})),
	}

	return s.sendToClient(client.ID, &response)
}

// handleAgentInfoRequest handles agent info request from client
func (s *Server) handleAgentInfoRequest(client *ClientConnection, msg *proto.Message) error {
	var req proto.AgentInfoRequestData
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return fmt.Errorf("failed to unmarshal agent info request: %w", err)
	}

	s.mu.RLock()
	agent, exists := s.agents[req.AgentID]
	s.mu.RUnlock()

	if !exists {
		response := proto.Message{
			Type: "agent_info_response",
			Data: json.RawMessage(`{"error": "Agent not found"}`),
		}
		return s.sendToClient(client.ID, &response)
	}

	response := proto.Message{
		Type: "agent_info_response",
		Data: must(json.Marshal(agent.Info)),
	}

	return s.sendToClient(client.ID, &response)
}

// handleAgentInfo handles agent info update
func (s *Server) handleAgentInfo(agent *AgentConnection, msg *proto.Message) error {
	var info proto.AgentInfo
	if err := json.Unmarshal(msg.Data, &info); err != nil {
		return fmt.Errorf("failed to unmarshal agent info: %w", err)
	}

	s.mu.Lock()
	agent.Info = &info
	s.mu.Unlock()

	s.logger.Printf("Updated info for agent %s", agent.ID)
	return nil
}

// sendToAgent sends a message to a specific agent
func (s *Server) sendToAgent(agentID string, msg *proto.Message) error {
	s.mu.RLock()
	agent, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	agent.mu.Lock()
	defer agent.mu.Unlock()

	if err := agent.Session.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message to agent: %w", err)
	}

	return nil
}

// sendToClient sends a message to a specific client
func (s *Server) sendToClient(clientID string, msg *proto.Message) error {
	s.mu.RLock()
	client, exists := s.clients[clientID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if err := client.Session.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message to client: %w", err)
	}

	return nil
}

// removeAgent removes an agent from the server
func (s *Server) removeAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if agent, exists := s.agents[agentID]; exists {
		// Close all tunnels for this agent
		for tunnelID, tunnel := range s.tunnels {
			if tunnel.AgentID == agentID {
				delete(s.tunnels, tunnelID)
			}
		}

		// Close connections
		if agent.Session != nil && agent.Session.Conn != nil {
			agent.Session.Conn.Close()
		}
		if agent.Session != nil && agent.Session.Session != nil {
			agent.Session.Session.Close()
		}

		delete(s.agents, agentID)
		s.logger.Printf("Removed agent %s", agentID)
	}
}

// removeClient removes a client from the server
func (s *Server) removeClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[clientID]; exists {
		// Close all tunnels for this client
		for tunnelID, tunnel := range s.tunnels {
			if tunnel.ClientID == clientID {
				delete(s.tunnels, tunnelID)
			}
		}

		// Close connections
		if client.Session != nil && client.Session.Conn != nil {
			client.Session.Conn.Close()
		}
		if client.Session != nil && client.Session.Session != nil {
			client.Session.Session.Close()
		}

		delete(s.clients, clientID)
		s.logger.Printf("Removed client %s", clientID)
	}
}

// must is a helper function for marshaling JSON
func must(data []byte, err error) json.RawMessage {
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return json.RawMessage("{}")
	}
	return json.RawMessage(data)
}
