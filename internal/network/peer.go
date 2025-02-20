package network

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"p2p-storage/internal/protocol"
)

// Peer represents a connected peer
type Peer struct {
	conn    net.Conn
	handler MessageHandler
	done    chan struct{}
	mu      sync.Mutex
}

// NewPeer creates a new peer
func NewPeer(conn net.Conn, handler MessageHandler) *Peer {
	return &Peer{
		conn:    conn,
		handler: handler,
		done:    make(chan struct{}),
	}
}

// ID returns the peer's ID (using remote address for now)
func (p *Peer) ID() string {
	return p.conn.RemoteAddr().String()
}

// Start starts handling peer communication
func (p *Peer) Start() {
	go p.readLoop()
}

// Close closes the peer connection
func (p *Peer) Close() error {
	close(p.done)
	return p.conn.Close()
}

// Send sends a message to the peer
func (p *Peer) Send(msg *protocol.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return json.NewEncoder(p.conn).Encode(msg)
}

func (p *Peer) readLoop() {
	decoder := json.NewDecoder(p.conn)

	for {
		select {
		case <-p.done:
			return
		default:
			var msg protocol.Message
			if err := decoder.Decode(&msg); err != nil {
				fmt.Printf("Error reading message from peer %s: %v\n", p.ID(), err)
				p.Close()
				return
			}

			if err := p.handler.HandleMessage(p, &msg); err != nil {
				fmt.Printf("Error handling message from peer %s: %v\n", p.ID(), err)
			}
		}
	}
}

// Address returns the peer's address
func (p *Peer) Address() string {
	return p.conn.RemoteAddr().String()
}
