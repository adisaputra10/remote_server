package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"remote-tunnel/internal/proto"
	streamutil "remote-tunnel/internal/stream"
	"remote-tunnel/internal/transport"
)

type Agent struct {
	id          string
	relayURL    string
	token       string
	allowedHosts []string
	session     *transport.MuxSession
	ctx         context.Context
	cancel      context.CancelFunc
	insecure    bool
	compress    bool
}

func NewAgent(id, relayURL, token string, allowedHosts []string) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		id:           id,
		relayURL:     relayURL,
		token:        token,
		allowedHosts: allowedHosts,
		ctx:          ctx,
		cancel:       cancel,
		insecure:     false,
		compress:     false,
	}
}

func (a *Agent) SetInsecure(insecure bool) {
	a.insecure = insecure
}

func (a *Agent) SetCompression(compress bool) {
	a.compress = compress
}

func (a *Agent) Run() error {
	for {
		select {
		case <-a.ctx.Done():
			return a.ctx.Err()
		default:
		}

		err := a.connect()
		if err != nil {
			log.Printf("Connection failed: %v", err)
			log.Printf("Reconnecting in 5 seconds...")
			
			select {
			case <-time.After(5 * time.Second):
				continue
			case <-a.ctx.Done():
				return a.ctx.Err()
			}
		}
	}
}

func (a *Agent) connect() error {
	log.Printf("Connecting to relay: %s", a.relayURL)

	wsConn, err := transport.DialWSInsecureWithCompression(a.ctx, a.relayURL, a.token, a.insecure, a.compress)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer wsConn.Close()

	wsConn.StartPingPong()

	session, err := transport.NewMuxClientWithCompression(wsConn, a.compress)
	if err != nil {
		return fmt.Errorf("mux client: %w", err)
	}
	defer session.Close()

	a.session = session

	// Send REGISTER
	err = session.SendControl(&proto.Control{
		Type:    proto.MsgRegister,
		AgentID: a.id,
		Token:   a.token,
	})
	if err != nil {
		return fmt.Errorf("send register: %w", err)
	}

	log.Printf("Agent %s registered", a.id)

	// Handle control messages
	return a.handleControl()
}

func (a *Agent) handleControl() error {
	for {
		select {
		case <-a.ctx.Done():
			return a.ctx.Err()
		default:
		}

		msg, err := a.session.ReceiveControl()
		if err != nil {
			return fmt.Errorf("receive control: %w", err)
		}

		switch msg.Type {
		case proto.MsgDial:
			go a.handleDial(msg)
		case proto.MsgPing:
			a.session.SendControl(&proto.Control{Type: proto.MsgPong})
		case proto.MsgPong:
			// Keep alive received
		case proto.MsgError:
			log.Printf("Received error: %s", msg.Error)
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}

func (a *Agent) handleDial(msg *proto.Control) {
	streamID := msg.StreamID
	targetAddr := msg.TargetAddr

	log.Printf("Dial request: target=%s, stream=%s", targetAddr, streamID)

	// Check if target is allowed
	if !a.isAllowed(targetAddr) {
		log.Printf("Target %s not allowed", targetAddr)
		return
	}

	// Open stream to relay
	stream, err := a.session.OpenStream()
	if err != nil {
		log.Printf("Failed to open stream: %v", err)
		return
	}
	defer stream.Close()

	// Dial to target
	conn, err := net.DialTimeout("tcp", targetAddr, 30*time.Second)
	if err != nil {
		log.Printf("Failed to dial target %s: %v", targetAddr, err)
		return
	}
	defer conn.Close()

	log.Printf("Connected to target %s, bridging with stream %s", targetAddr, streamID)

	// Create stream processor with compression if enabled  
	streamOpts := streamutil.DefaultStreamOptions()
	streamOpts.EnableCompression = a.compress
	processor := streamutil.NewStreamProcessor(streamOpts)

	// Bridge connections with optional compression
	errCh := make(chan error, 2)

	// Copy stream -> conn
	go func() {
		_, err := processor.CopyWithCompression(conn, stream)
		errCh <- err
	}()

	// Copy conn -> stream
	go func() {
		_, err := processor.CopyWithCompression(stream, conn)
		errCh <- err
	}()

	// Wait for first error or completion
	err = <-errCh
	if err != nil {
		log.Printf("Bridge error: %v", err)
	}

	log.Printf("Stream %s closed", streamID)
}

func (a *Agent) isAllowed(targetAddr string) bool {
	for _, allowed := range a.allowedHosts {
		if strings.HasPrefix(targetAddr, allowed) {
			return true
		}
	}
	return false
}

func (a *Agent) Close() error {
	a.cancel()
	if a.session != nil {
		return a.session.Close()
	}
	return nil
}
