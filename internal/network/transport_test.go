package network

import (
	"testing"

	"p2p-storage/internal/protocol"
)

func TestNewTransport(t *testing.T) {
	handler := &mockHandler{}
	transport, err := NewTransport("test-node", ":0", handler)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	defer transport.Stop()

	if transport.nodeID != "test-node" {
		t.Errorf("Transport nodeID = %v, want %v", transport.nodeID, "test-node")
	}
}

func TestTransport_Broadcast(t *testing.T) {
	handler := &mockHandler{}
	transport, err := NewTransport("test-node", ":0", handler)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	// Start the transport
	transport.Start()
	defer transport.Stop()

	// Create a single mock peer
	conn := newMockConn()
	peer := NewPeer(conn, handler)

	// Add peer to transport
	transport.mu.Lock()
	transport.peers[peer.ID()] = peer
	transport.mu.Unlock()

	// Create test message
	msg, err := protocol.NewMessage(protocol.MessageTypeData, "test-node", protocol.DataPayload{
		ContentHash: "test123",
		FileName:    "test.txt",
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Broadcast message
	if err := transport.Broadcast(msg); err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// Check if message was written
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if len(conn.writeData) == 0 {
		t.Error("Peer did not receive the message")
	}
}
