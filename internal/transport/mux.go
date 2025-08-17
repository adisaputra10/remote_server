package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"remote-tunnel/internal/proto"
)

const (
	ControlStreamID = 0
	DialTimeout     = 30 * time.Second
)

// MuxSession wraps yamux session with control channel
type MuxSession struct {
	session     *yamux.Session
	controlConn net.Conn
	mu          sync.RWMutex
	closed      bool
}

func NewMuxServer(conn net.Conn) (*MuxSession, error) {
	return NewMuxServerWithCompression(conn, false)
}

func NewMuxServerWithCompression(conn net.Conn, enableCompression bool) (*MuxSession, error) {
	// Apply compression if enabled
	if enableCompression {
		conn = EnableCompression(conn)
	}
	
	config := yamux.DefaultConfig()
	config.KeepAliveInterval = 30 * time.Second
	config.ConnectionWriteTimeout = 10 * time.Second
	config.EnableKeepAlive = true
	config.MaxStreamWindowSize = 256 * 1024
	config.StreamOpenTimeout = 10 * time.Second
	config.StreamCloseTimeout = 5 * time.Second
	
	session, err := yamux.Server(conn, config)
	if err != nil {
		return nil, fmt.Errorf("yamux server: %w", err)
	}
	
	// Accept control stream with timeout
	acceptTimeout := time.After(10 * time.Second)
	acceptChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)
	
	go func() {
		conn, err := session.Accept()
		if err != nil {
			errChan <- err
		} else {
			acceptChan <- conn
		}
	}()
	
	var controlConn net.Conn
	select {
	case controlConn = <-acceptChan:
		// Success
	case err := <-errChan:
		session.Close()
		return nil, fmt.Errorf("accept control stream: %w", err)
	case <-acceptTimeout:
		session.Close()
		return nil, fmt.Errorf("timeout accepting control stream")
	}
	
	return &MuxSession{
		session:     session,
		controlConn: controlConn,
	}, nil
}

func NewMuxClient(conn net.Conn) (*MuxSession, error) {
	return NewMuxClientWithCompression(conn, false)
}

func NewMuxClientWithCompression(conn net.Conn, enableCompression bool) (*MuxSession, error) {
	// Apply compression if enabled
	if enableCompression {
		conn = EnableCompression(conn)
	}
	
	config := yamux.DefaultConfig()
	config.KeepAliveInterval = 30 * time.Second
	config.ConnectionWriteTimeout = 10 * time.Second
	config.EnableKeepAlive = true
	config.MaxStreamWindowSize = 256 * 1024
	config.StreamOpenTimeout = 10 * time.Second
	config.StreamCloseTimeout = 5 * time.Second
	
	session, err := yamux.Client(conn, config)
	if err != nil {
		return nil, fmt.Errorf("yamux client: %w", err)
	}
	
	// Add a small delay to avoid race conditions
	time.Sleep(100 * time.Millisecond)
	
	// Open control stream with retry
	var controlConn net.Conn
	for i := 0; i < 3; i++ {
		controlConn, err = session.Open()
		if err == nil {
			break
		}
		log.Printf("Failed to open control stream (attempt %d/3): %v", i+1, err)
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}
	
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("open control stream after retries: %w", err)
	}
	
	return &MuxSession{
		session:     session,
		controlConn: controlConn,
	}, nil
}

func (m *MuxSession) OpenStream() (net.Conn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("session closed")
	}
	
	return m.session.Open()
}

func (m *MuxSession) AcceptStream() (net.Conn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("session closed")
	}
	
	return m.session.Accept()
}

func (m *MuxSession) SendControl(msg *proto.Control) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return fmt.Errorf("session closed")
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal control: %w", err)
	}
	
	// Add newline delimiter for easier parsing
	data = append(data, '\n')
	
	_, err = m.controlConn.Write(data)
	if err != nil {
		return fmt.Errorf("write control: %w", err)
	}
	
	return nil
}

func (m *MuxSession) ReceiveControl() (*proto.Control, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("session closed")
	}
	
	// Read until newline delimiter
	buffer := make([]byte, 1)
	var msgBytes []byte
	
	for {
		n, err := m.controlConn.Read(buffer)
		if err != nil {
			return nil, fmt.Errorf("read control: %w", err)
		}
		
		if n > 0 {
			if buffer[0] == '\n' {
				break
			}
			msgBytes = append(msgBytes, buffer[0])
		}
		
		// Safety check to prevent infinite loop
		if len(msgBytes) > 8192 {
			return nil, fmt.Errorf("control message too large")
		}
	}
	
	var msg proto.Control
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal control: %w", err)
	}
	
	return &msg, nil
}

func (m *MuxSession) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return nil
	}
	
	m.closed = true
	
	if m.controlConn != nil {
		m.controlConn.Close()
	}
	
	return m.session.Close()
}

func (m *MuxSession) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// Bridge copies data between two connections bidirectionally
func Bridge(ctx context.Context, conn1, conn2 net.Conn) error {
	errCh := make(chan error, 2)
	
	go func() {
		_, err := io.Copy(conn1, conn2)
		errCh <- err
	}()
	
	go func() {
		_, err := io.Copy(conn2, conn1)
		errCh <- err
	}()
	
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
