package network

import (
	"fmt"
	"net"
	"sync"

	"p2p-storage/internal/protocol"
)

// Transport handles the network communication
type Transport struct {
	listener net.Listener
	nodeID   string
	address  string
	peers    map[string]*Peer
	handler  MessageHandler
	mu       sync.RWMutex
	done     chan struct{}
}

// MessageHandler handles incoming messages
type MessageHandler interface {
	HandleMessage(peer *Peer, msg *protocol.Message) error
}

// NewTransport creates a new transport
func NewTransport(nodeID, address string, handler MessageHandler) (*Transport, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	return &Transport{
		listener: listener,
		nodeID:   nodeID,
		address:  address,
		peers:    make(map[string]*Peer),
		handler:  handler,
		done:     make(chan struct{}),
	}, nil
}

// Start starts the transport
func (t *Transport) Start() {
	go t.acceptLoop()
}

// Stop stops the transport
func (t *Transport) Stop() {
	close(t.done)
	t.listener.Close()

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, peer := range t.peers {
		peer.Close()
	}
}

// In transport.go, modify Connect:
func (t *Transport) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Connection error: %v\n", err)
		return err
	}

	peer := NewPeer(conn, t.handler)

	t.mu.Lock()
	t.peers[peer.ID()] = peer
	t.mu.Unlock()

	// Start peer handling
	peer.Start()

	// Create and send handshake immediately
	handshaker := protocol.NewHandshaker(t.nodeID, t.address, []string{})
	msg, err := handshaker.CreateHandshake()
	if err != nil {
		fmt.Printf("Handshake creation error: %v\n", err)
		return err
	}

	if err := peer.Send(msg); err != nil {
		fmt.Printf("Handshake send error: %v\n", err)
		return err
	}

	return nil
}

// Broadcast sends a message to all connected peers
func (t *Transport) Broadcast(msg *protocol.Message) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, peer := range t.peers {
		if err := peer.Send(msg); err != nil {
			fmt.Printf("Failed to send message to peer %s: %v\n", peer.ID(), err)
		}
	}
	return nil
}

func (t *Transport) acceptLoop() {
	for {
		select {
		case <-t.done:
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				continue
			}

			peer := NewPeer(conn, t.handler)

			t.mu.Lock()
			t.peers[peer.ID()] = peer
			t.mu.Unlock()

			go peer.Start()
		}
	}
}

// RemovePeer removes a peer from the transport
func (t *Transport) RemovePeer(peerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if peer, exists := t.peers[peerID]; exists {
		peer.Close()
		delete(t.peers, peerID)
	}
}

// Send sends a message to a specific peer
func (t *Transport) Send(peerID string, msg *protocol.Message) error {
	t.mu.RLock()
	peer, exists := t.peers[peerID]
	t.mu.RUnlock()

	if !exists {
		return fmt.Errorf("peer %s not found", peerID)
	}

	return peer.Send(msg)
}

// Address returns the transport's address
func (t *Transport) Address() string {
	return t.address
}
