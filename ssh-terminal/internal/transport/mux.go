package transport

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

// MuxSession wraps yamux session over WebSocket
type MuxSession struct {
	conn    *websocket.Conn
	session *yamux.Session
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
}

// WSConn wraps WebSocket connection to implement net.Conn interface
type WSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (ws *WSConn) Read(b []byte) (int, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	_, data, err := ws.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	
	n := copy(b, data)
	return n, nil
}

func (ws *WSConn) Write(b []byte) (int, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	err := ws.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	
	return len(b), nil
}

func (ws *WSConn) Close() error {
	return ws.conn.Close()
}

func (ws *WSConn) LocalAddr() net.Addr {
	return ws.conn.LocalAddr()
}

func (ws *WSConn) RemoteAddr() net.Addr {
	return ws.conn.RemoteAddr()
}

func (ws *WSConn) SetDeadline(t time.Time) error {
	ws.conn.SetReadDeadline(t)
	ws.conn.SetWriteDeadline(t)
	return nil
}

func (ws *WSConn) SetReadDeadline(t time.Time) error {
	return ws.conn.SetReadDeadline(t)
}

func (ws *WSConn) SetWriteDeadline(t time.Time) error {
	return ws.conn.SetWriteDeadline(t)
}

// NewMuxSession creates a new multiplexed session over WebSocket
func NewMuxSession(conn *websocket.Conn, isClient bool) (*MuxSession, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Wrap WebSocket as net.Conn
	wsConn := &WSConn{conn: conn}
	
	// Create yamux session
	var session *yamux.Session
	var err error
	
	if isClient {
		session, err = yamux.Client(wsConn, nil)
	} else {
		session, err = yamux.Server(wsConn, nil)
	}
	
	if err != nil {
		cancel()
		return nil, err
	}
	
	return &MuxSession{
		conn:    conn,
		session: session,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// OpenStream opens a new stream
func (m *MuxSession) OpenStream() (net.Conn, error) {
	return m.session.Open()
}

// AcceptStream accepts a new stream
func (m *MuxSession) AcceptStream() (net.Conn, error) {
	return m.session.Accept()
}

// Close closes the session
func (m *MuxSession) Close() error {
	m.cancel()
	if m.session != nil {
		m.session.Close()
	}
	return m.conn.Close()
}

// Context returns the session context
func (m *MuxSession) Context() context.Context {
	return m.ctx
}

// NumStreams returns the number of active streams
func (m *MuxSession) NumStreams() int {
	return m.session.NumStreams()
}

// GetWebSocketConn returns the underlying WebSocket connection
func (m *MuxSession) GetWebSocketConn() *websocket.Conn {
	return m.conn
}
