package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"remote-tunnel/internal/logger"
	"remote-tunnel/internal/tunnel"
)

type TunnelAgent struct {
	id         string
	name       string
	relayURL   string
	endpoints  []string
	logger     *logger.Logger
	transport  *tunnel.Transport
	tunnels    map[string]*ActiveTunnel
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

type ActiveTunnel struct {
	ID         string
	LocalAddr  string
	RemoteAddr string
	Listener   net.Listener
	Info       *tunnel.TunnelInfo
	Connections map[string]net.Conn
	mu         sync.RWMutex
}

func main() {
	var (
		id       = flag.String("id", "", "Agent ID (auto-generated if empty)")
		name     = flag.String("name", "", "Agent name")
		relayURL = flag.String("relay-url", "ws://localhost:8443/ws/agent", "Relay server WebSocket URL")
		allow    = flag.String("allow", "127.0.0.1:22,127.0.0.1:3306,127.0.0.1:5432", "Allowed target addresses")
	)
	flag.Parse()

	log := logger.New("AGENT")
	
	if *id == "" {
		*id = fmt.Sprintf("agent_%d", time.Now().UnixNano())
	}
	
	if *name == "" {
		hostname, _ := os.Hostname()
		*name = fmt.Sprintf("%s_%s", hostname, runtime.GOOS)
	}

	log.Info("🚀 Starting tunnel agent")
	log.Info("📋 Agent ID: %s", *id)
	log.Info("📋 Agent Name: %s", *name)
	log.Info("📋 Relay URL: %s", *relayURL)
	log.Command("STARTUP", "allow", *allow)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agent := &TunnelAgent{
		id:        *id,
		name:      *name,
		relayURL:  *relayURL,
		endpoints: parseEndpoints(*allow),
		logger:    log,
		tunnels:   make(map[string]*ActiveTunnel),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Connect to relay
	if err := agent.connectToRelay(); err != nil {
		log.Error("Failed to connect to relay: %v", err)
		os.Exit(1)
	}

	// Start heartbeat
	go agent.heartbeat()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info("🛑 Shutting down agent...")
	agent.shutdown()
	log.Info("👋 Agent stopped")
}

func parseEndpoints(allow string) []string {
	// Simple parsing - in production you'd want more robust parsing
	endpoints := []string{}
	// Parse comma-separated endpoints
	return endpoints
}

func (a *TunnelAgent) connectToRelay() error {
	a.logger.Info("🔗 Connecting to relay: %s", a.relayURL)

	u, err := url.Parse(a.relayURL)
	if err != nil {
		return fmt.Errorf("invalid relay URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(a.relayURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	transport, err := tunnel.NewTransport(conn, true, a.logger)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create transport: %w", err)
	}

	a.transport = transport

	// Register with relay
	if err := a.register(); err != nil {
		transport.Close()
		return fmt.Errorf("registration failed: %w", err)
	}

	// Start message handler
	go a.handleMessages()

	a.logger.Info("✅ Connected to relay successfully")
	return nil
}

func (a *TunnelAgent) register() error {
	agentInfo := &tunnel.AgentInfo{
		ID:       a.id,
		Name:     a.name,
		Platform: runtime.GOOS,
		Version:  "1.0.0",
		Status:   "online",
		LastSeen: time.Now(),
		Metadata: map[string]string{
			"arch":      runtime.GOARCH,
			"go_version": runtime.Version(),
		},
		Endpoints: a.endpoints,
	}

	msg := tunnel.NewMessage(tunnel.MsgAgentRegister).
		SetAgentID(a.id).
		SetData(agentInfo)

	a.logger.Command("REGISTER", "sending agent info")
	return a.transport.SendMessage(msg)
}

func (a *TunnelAgent) handleMessages() {
	defer a.transport.Close()

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
		}

		msg, err := a.transport.ReceiveMessage()
		if err != nil {
			a.logger.Error("Message receive error: %v", err)
			return
		}

		a.logger.Command("RECEIVED", msg.Type, msg.TunnelID)

		switch msg.Type {
		case tunnel.MsgAgentRegistered:
			a.logger.Info("✅ Registration confirmed by relay")

		case tunnel.MsgTunnelRequest:
			a.handleTunnelRequest(msg)

		case tunnel.MsgTunnelClose:
			a.handleTunnelClose(msg)

		default:
			a.logger.Warn("Unknown message type: %s", msg.Type)
		}
	}
}

func (a *TunnelAgent) handleTunnelRequest(msg *tunnel.Message) {
	var req tunnel.TunnelRequest
	
	if data, ok := msg.Data.(map[string]interface{}); ok {
		reqBytes, _ := json.Marshal(data)
		json.Unmarshal(reqBytes, &req)
	}

	a.logger.Tunnel("REQUEST", req.TunnelID, fmt.Sprintf("%s:%d -> %s:%d", 
		req.LocalHost, req.LocalPort, req.RemoteHost, req.RemotePort))

	// Validate target endpoint
	targetAddr := fmt.Sprintf("%s:%d", req.RemoteHost, req.RemotePort)
	if !a.isAllowedTarget(targetAddr) {
		a.logger.Error("Target not allowed: %s", targetAddr)
		a.sendTunnelError(req.TunnelID, msg.ClientID, "Target address not allowed")
		return
	}

	// Create tunnel
	if err := a.createTunnel(&req, msg.ClientID); err != nil {
		a.logger.Error("Failed to create tunnel: %v", err)
		a.sendTunnelError(req.TunnelID, msg.ClientID, err.Error())
		return
	}

	a.logger.Tunnel("CREATED", req.TunnelID, fmt.Sprintf("listening on local port"))
}

func (a *TunnelAgent) createTunnel(req *tunnel.TunnelRequest, clientID string) error {
	// Listen on local port for relay connections
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", req.LocalHost, req.LocalPort))
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	tunnelInfo := &tunnel.TunnelInfo{
		ID:          req.TunnelID,
		AgentID:     a.id,
		ClientID:    clientID,
		LocalAddr:   listener.Addr().String(),
		RemoteAddr:  fmt.Sprintf("%s:%d", req.RemoteHost, req.RemotePort),
		Protocol:    req.Protocol,
		Status:      "active",
		CreatedAt:   time.Now(),
		Connections: 0,
	}

	activeTunnel := &ActiveTunnel{
		ID:          req.TunnelID,
		LocalAddr:   listener.Addr().String(),
		RemoteAddr:  fmt.Sprintf("%s:%d", req.RemoteHost, req.RemotePort),
		Listener:    listener,
		Info:        tunnelInfo,
		Connections: make(map[string]net.Conn),
	}

	a.mu.Lock()
	a.tunnels[req.TunnelID] = activeTunnel
	a.mu.Unlock()

	// Send success response
	response := tunnel.NewMessage(tunnel.MsgTunnelCreated).
		SetTunnelID(req.TunnelID).
		SetClientID(clientID).
		SetAgentID(a.id).
		SetData(tunnelInfo)

	if err := a.transport.SendMessage(response); err != nil {
		listener.Close()
		a.mu.Lock()
		delete(a.tunnels, req.TunnelID)
		a.mu.Unlock()
		return fmt.Errorf("failed to send tunnel created response: %w", err)
	}

	// Start accepting connections
	go a.handleTunnelConnections(activeTunnel)

	return nil
}

func (a *TunnelAgent) handleTunnelConnections(tunnel *ActiveTunnel) {
	defer func() {
		tunnel.Listener.Close()
		a.mu.Lock()
		delete(a.tunnels, tunnel.ID)
		a.mu.Unlock()
		a.logger.Tunnel("CLOSED", tunnel.ID, "connection handler stopped")
	}()

	for {
		conn, err := tunnel.Listener.Accept()
		if err != nil {
			a.logger.Error("Accept error for tunnel %s: %v", tunnel.ID, err)
			return
		}

		connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
		tunnel.mu.Lock()
		tunnel.Connections[connID] = conn
		tunnel.Info.Connections++
		tunnel.mu.Unlock()

		a.logger.Debug("New connection [%s] for tunnel %s", connID, tunnel.ID)

		go a.handleTunnelConnection(tunnel, connID, conn)
	}
}

func (a *TunnelAgent) handleTunnelConnection(tunnel *ActiveTunnel, connID string, relayConn net.Conn) {
	defer func() {
		relayConn.Close()
		tunnel.mu.Lock()
		delete(tunnel.Connections, connID)
		tunnel.Info.Connections--
		tunnel.mu.Unlock()
		a.logger.Debug("Connection [%s] closed for tunnel %s", connID, tunnel.ID)
	}()

	// Connect to actual target
	targetConn, err := net.Dial("tcp", tunnel.RemoteAddr)
	if err != nil {
		a.logger.Error("Failed to connect to target %s: %v", tunnel.RemoteAddr, err)
		return
	}
	defer targetConn.Close()

	a.logger.Debug("Connected to target %s for tunnel %s", tunnel.RemoteAddr, tunnel.ID)

	// Bridge connections
	done := make(chan bool, 2)

	// Relay -> Target
	go func() {
		defer func() { done <- true }()
		written, err := io.Copy(targetConn, relayConn)
		if err != nil {
			a.logger.Debug("Relay->Target copy error [%s]: %v", connID, err)
		}
		tunnel.Info.BytesOut += written
		a.logger.Debug("Relay->Target [%s]: %d bytes", connID, written)
	}()

	// Target -> Relay
	go func() {
		defer func() { done <- true }()
		written, err := io.Copy(relayConn, targetConn)
		if err != nil {
			a.logger.Debug("Target->Relay copy error [%s]: %v", connID, err)
		}
		tunnel.Info.BytesIn += written
		a.logger.Debug("Target->Relay [%s]: %d bytes", connID, written)
	}()

	// Wait for one direction to close
	<-done
}

func (a *TunnelAgent) handleTunnelClose(msg *tunnel.Message) {
	tunnelID := msg.TunnelID
	a.logger.Tunnel("CLOSE", tunnelID, "requested by relay")

	a.mu.Lock()
	tunnel, exists := a.tunnels[tunnelID]
	if exists {
		delete(a.tunnels, tunnelID)
	}
	a.mu.Unlock()

	if exists {
		tunnel.Listener.Close()
		a.logger.Tunnel("CLOSED", tunnelID, "listener stopped")
	}
}

func (a *TunnelAgent) isAllowedTarget(targetAddr string) bool {
	// Simple validation - in production you'd want more sophisticated checking
	allowedTargets := []string{
		"127.0.0.1:22",   // SSH
		"127.0.0.1:3306", // MySQL
		"127.0.0.1:5432", // PostgreSQL
		"localhost:22",
		"localhost:3306",
		"localhost:5432",
	}

	for _, allowed := range allowedTargets {
		if targetAddr == allowed {
			return true
		}
	}

	a.logger.Warn("Target address not in allowed list: %s", targetAddr)
	return false
}

func (a *TunnelAgent) sendTunnelError(tunnelID, clientID, errorMsg string) {
	response := tunnel.NewMessage(tunnel.MsgTunnelError).
		SetTunnelID(tunnelID).
		SetClientID(clientID).
		SetAgentID(a.id).
		SetError(errorMsg)

	if err := a.transport.SendMessage(response); err != nil {
		a.logger.Error("Failed to send tunnel error: %v", err)
	}
}

func (a *TunnelAgent) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			msg := tunnel.NewMessage(tunnel.MsgAgentHeartbeat).SetAgentID(a.id)
			if err := a.transport.SendMessage(msg); err != nil {
				a.logger.Error("Heartbeat failed: %v", err)
				return
			}
			a.logger.Debug("💓 Heartbeat sent")
		}
	}
}

func (a *TunnelAgent) shutdown() {
	a.cancel()

	// Close all tunnels
	a.mu.Lock()
	for _, tunnel := range a.tunnels {
		tunnel.Listener.Close()
	}
	a.mu.Unlock()

	// Send disconnect message
	if a.transport != nil {
		msg := tunnel.NewMessage(tunnel.MsgAgentDisconnect).SetAgentID(a.id)
		a.transport.SendMessage(msg)
		time.Sleep(100 * time.Millisecond) // Give time for message to send
		a.transport.Close()
	}

	a.logger.Info("All tunnels closed")
}
