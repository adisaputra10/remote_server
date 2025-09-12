package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"remote-tunnel/internal/logger"
)

type Transport struct {
	conn     *websocket.Conn
	mux      *yamux.Session
	logger   *logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	closed   bool
	streams  map[string]net.Conn
}

func NewTransport(wsConn *websocket.Conn, isClient bool, log *logger.Logger) (*Transport, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	transport := &Transport{
		conn:    wsConn,
		logger:  log,
		ctx:     ctx,
		cancel:  cancel,
		streams: make(map[string]net.Conn),
	}

	// Create yamux session over WebSocket
	wsWrapper := &WebSocketWrapper{conn: wsConn, logger: log}
	
	var muxSession *yamux.Session
	var err error
	
	if isClient {
		muxSession, err = yamux.Client(wsWrapper, yamux.DefaultConfig())
		transport.logger.Info("Created yamux client session")
	} else {
		muxSession, err = yamux.Server(wsWrapper, yamux.DefaultConfig())
		transport.logger.Info("Created yamux server session")
	}
	
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create yamux session: %w", err)
	}
	
	transport.mux = muxSession
	
	// Start background goroutines
	go transport.handleIncomingStreams()
	go transport.keepAlive()
	
	return transport, nil
}

// WebSocketWrapper wraps WebSocket to implement net.Conn interface for yamux
type WebSocketWrapper struct {
	conn   *websocket.Conn
	reader io.Reader
	logger *logger.Logger
	mu     sync.Mutex
}

func (w *WebSocketWrapper) Read(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.reader == nil {
		msgType, data, err := w.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		
		if msgType != websocket.BinaryMessage {
			return 0, fmt.Errorf("expected binary message, got %d", msgType)
		}
		
		w.reader = &dataReader{data: data}
	}
	
	n, err := w.reader.Read(p)
	if err == io.EOF {
		w.reader = nil
		err = nil
	}
	
	return n, err
}

func (w *WebSocketWrapper) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	err := w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	
	return len(p), nil
}

func (w *WebSocketWrapper) Close() error {
	return w.conn.Close()
}

func (w *WebSocketWrapper) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *WebSocketWrapper) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *WebSocketWrapper) SetDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *WebSocketWrapper) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *WebSocketWrapper) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

type dataReader struct {
	data   []byte
	offset int
}

func (r *dataReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	
	n := copy(p, r.data[r.offset:])
	r.offset += n
	
	return n, nil
}

func (t *Transport) handleIncomingStreams() {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}
		
		stream, err := t.mux.Accept()
		if err != nil {
			if !t.isClosed() {
				t.logger.Error("Failed to accept stream: %v", err)
			}
			return
		}
		
		streamID := fmt.Sprintf("stream_%d", time.Now().UnixNano())
		t.mu.Lock()
		t.streams[streamID] = stream
		t.mu.Unlock()
		
		t.logger.Debug("Accepted new stream: %s", streamID)
		go t.handleStream(streamID, stream)
	}
}

func (t *Transport) handleStream(streamID string, stream net.Conn) {
	defer func() {
		stream.Close()
		t.mu.Lock()
		delete(t.streams, streamID)
		t.mu.Unlock()
		t.logger.Debug("Stream closed: %s", streamID)
	}()
	
	// Handle stream data
	buffer := make([]byte, 32*1024)
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}
		
		n, err := stream.Read(buffer)
		if err != nil {
			if err != io.EOF {
				t.logger.Error("Stream read error [%s]: %v", streamID, err)
			}
			return
		}
		
		t.logger.Debug("Stream data [%s]: %d bytes", streamID, n)
		// Process stream data here
	}
}

func (t *Transport) OpenStream() (net.Conn, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.closed {
		return nil, fmt.Errorf("transport is closed")
	}
	
	stream, err := t.mux.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	
	streamID := fmt.Sprintf("stream_%d", time.Now().UnixNano())
	t.streams[streamID] = stream
	
	t.logger.Debug("Opened new stream: %s", streamID)
	return stream, nil
}

func (t *Transport) SendMessage(msg *Message) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.closed {
		return fmt.Errorf("transport is closed")
	}
	
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	err = t.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	t.logger.Debug("Sent message: %s", msg.Type)
	return nil
}

func (t *Transport) ReceiveMessage() (*Message, error) {
	msgType, data, err := t.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}
	
	if msgType != websocket.TextMessage {
		return nil, fmt.Errorf("expected text message for control, got %d", msgType)
	}
	
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	t.logger.Debug("Received message: %s", msg.Type)
	return &msg, nil
}

func (t *Transport) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			if err := t.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				t.logger.Error("Ping failed: %v", err)
				return
			}
			t.logger.Debug("Sent ping")
		}
	}
}

func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return nil
	}
	
	t.closed = true
	t.cancel()
	
	// Close all streams
	for streamID, stream := range t.streams {
		stream.Close()
		t.logger.Debug("Closed stream: %s", streamID)
	}
	
	// Close mux session
	if t.mux != nil {
		t.mux.Close()
	}
	
	// Close WebSocket
	t.conn.Close()
	
	t.logger.Info("Transport closed")
	return nil
}

func (t *Transport) isClosed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.closed
}

// TunnelTransport handles a single tunnel connection for agent
type TunnelTransport struct {
	id         string
	remoteHost string
	remotePort int
	logger     *logger.Logger
	transport  *Transport
	active     bool
	mu         sync.RWMutex
}

func NewTunnelTransport(transport *Transport, tunnelID, remoteHost string, remotePort int, logger *logger.Logger) (*TunnelTransport, error) {
	return &TunnelTransport{
		id:         tunnelID,
		remoteHost: remoteHost,
		remotePort: remotePort,
		logger:     logger,
		transport:  transport,
		active:     true,
	}, nil
}

func (tt *TunnelTransport) Start() {
	tt.logger.Info("Starting tunnel transport: %s", tt.id)
	
	// This is a simplified implementation
	// In a real implementation, you would:
	// 1. Accept incoming streams from the transport
	// 2. For each stream, connect to the remote target
	// 3. Bridge the stream data with the target connection
}

func (tt *TunnelTransport) Close() {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	
	if !tt.active {
		return
	}
	
	tt.active = false
	tt.logger.Info("Tunnel transport closed: %s", tt.id)
}

// ClientTunnel handles a tunnel on the client side
type ClientTunnel struct {
	ID         string
	LocalAddr  string
	TargetAddr string
	listener   net.Listener
	transport  *Transport
	logger     *logger.Logger
	active     bool
	mu         sync.RWMutex
}

func NewClientTunnel(id, localAddr, targetAddr string, listener net.Listener, transport *Transport, logger *logger.Logger) *ClientTunnel {
	return &ClientTunnel{
		ID:         id,
		LocalAddr:  localAddr,
		TargetAddr: targetAddr,
		listener:   listener,
		transport:  transport,
		logger:     logger,
		active:     true,
	}
}

func (ct *ClientTunnel) Start() {
	ct.logger.Info("Starting client tunnel: %s -> %s", ct.LocalAddr, ct.TargetAddr)
	
	for ct.IsActive() {
		conn, err := ct.listener.Accept()
		if err != nil {
			if ct.IsActive() {
				ct.logger.Error("Failed to accept connection: %v", err)
			}
			break
		}
		
		go ct.handleConnection(conn)
	}
}

func (ct *ClientTunnel) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	ct.logger.Info("Handling new connection for tunnel: %s", ct.ID)
	
	// This is a simplified implementation
	// In a real implementation, you would:
	// 1. Open a new stream through the transport
	// 2. Bridge the local connection with the stream
}

func (ct *ClientTunnel) IsActive() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.active
}

func (ct *ClientTunnel) Close() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if !ct.active {
		return
	}
	
	ct.active = false
	
	if ct.listener != nil {
		ct.listener.Close()
	}
	
	ct.logger.Info("Client tunnel closed: %s", ct.ID)
}
