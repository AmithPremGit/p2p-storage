package protocol

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewHandshaker(t *testing.T) {
	nodeID := "testNode"
	address := "localhost:8080"
	knownPeers := []string{"peer1", "peer2"}

	handshaker := NewHandshaker(nodeID, address, knownPeers)

	if handshaker.NodeID != nodeID {
		t.Errorf("NodeID = %v, want %v", handshaker.NodeID, nodeID)
	}
	if handshaker.Address != address {
		t.Errorf("Address = %v, want %v", handshaker.Address, address)
	}
	if len(handshaker.KnownPeers) != len(knownPeers) {
		t.Errorf("KnownPeers length = %v, want %v", len(handshaker.KnownPeers), len(knownPeers))
	}
}

func TestHandshaker_CreateHandshake(t *testing.T) {
	nodeID := "testNode"
	address := "localhost:8080"
	knownPeers := []string{"peer1", "peer2"}

	handshaker := NewHandshaker(nodeID, address, knownPeers)

	msg, err := handshaker.CreateHandshake()
	if err != nil {
		t.Fatalf("Failed to create handshake: %v", err)
	}

	if msg.Type != MessageTypeHandshake {
		t.Errorf("Message type = %v, want %v", msg.Type, MessageTypeHandshake)
	}
	if msg.SenderID != nodeID {
		t.Errorf("SenderID = %v, want %v", msg.SenderID, nodeID)
	}

	var payload HandshakePayload
	if err := msg.ParsePayload(&payload); err != nil {
		t.Fatalf("Failed to parse payload: %v", err)
	}

	if payload.NodeID != nodeID {
		t.Errorf("Payload NodeID = %v, want %v", payload.NodeID, nodeID)
	}
	if payload.Address != address {
		t.Errorf("Payload Address = %v, want %v", payload.Address, address)
	}
	if len(payload.KnownPeers) != len(knownPeers) {
		t.Errorf("Payload KnownPeers length = %v, want %v", len(payload.KnownPeers), len(knownPeers))
	}
}

func TestHandshaker_HandleHandshake(t *testing.T) {
	handshaker := NewHandshaker("testNode", "localhost:8080", []string{"peer1"})

	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
	}{
		{
			name: "valid handshake",
			msg: func() *Message {
				msg, _ := NewMessage(MessageTypeHandshake, "node1", HandshakePayload{
					NodeID:     "node1",
					Address:    "localhost:8081",
					KnownPeers: []string{"peer2"},
				})
				return msg
			}(),
			wantErr: false,
		},
		{
			name: "wrong message type",
			msg: func() *Message {
				msg, _ := NewMessage(MessageTypeData, "node1", DataPayload{})
				return msg
			}(),
			wantErr: true,
		},
		{
			name: "invalid payload",
			msg: &Message{
				Type:     MessageTypeHandshake,
				SenderID: "node1",
				Payload:  json.RawMessage(`invalid json`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := handshaker.HandleHandshake(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleHandshake() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && payload == nil {
				t.Error("HandleHandshake() returned nil payload for valid message")
			}
		})
	}
}

func TestHandshaker_WriteAndReadHandshake(t *testing.T) {
	nodeID := "testNode"
	address := "localhost:8080"
	knownPeers := []string{"peer1", "peer2"}

	handshaker := NewHandshaker(nodeID, address, knownPeers)

	// Test writing and reading handshake
	var buf bytes.Buffer

	// Write handshake
	if err := handshaker.WriteHandshake(&buf); err != nil {
		t.Fatalf("Failed to write handshake: %v", err)
	}

	// Read handshake
	payload, err := handshaker.ReadHandshake(&buf)
	if err != nil {
		t.Fatalf("Failed to read handshake: %v", err)
	}

	// Verify payload
	if payload.NodeID != nodeID {
		t.Errorf("Read NodeID = %v, want %v", payload.NodeID, nodeID)
	}
	if payload.Address != address {
		t.Errorf("Read Address = %v, want %v", payload.Address, address)
	}
	if len(payload.KnownPeers) != len(knownPeers) {
		t.Errorf("Read KnownPeers length = %v, want %v", len(payload.KnownPeers), len(knownPeers))
	}
}
