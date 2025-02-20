package protocol

import (
	"encoding/json"
	"testing"
)

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		senderID string
		payload  interface{}
		wantErr  bool
	}{
		{
			name:     "handshake message",
			msgType:  MessageTypeHandshake,
			senderID: "node1",
			payload: HandshakePayload{
				NodeID:     "node1",
				Address:    "localhost:8080",
				KnownPeers: []string{"peer1", "peer2"},
			},
			wantErr: false,
		},
		{
			name:     "data message",
			msgType:  MessageTypeData,
			senderID: "node1",
			payload: DataPayload{
				ContentHash: "abc123",
				FileName:    "test.txt",
				Size:        1024,
				Encrypted:   true,
			},
			wantErr: false,
		},
		{
			name:     "invalid payload",
			msgType:  MessageTypeData,
			senderID: "node1",
			payload:  make(chan int), // Channels cannot be marshaled to JSON
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewMessage(tt.msgType, tt.senderID, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && msg != nil {
				if msg.Type != tt.msgType {
					t.Errorf("NewMessage() type = %v, want %v", msg.Type, tt.msgType)
				}
				if msg.SenderID != tt.senderID {
					t.Errorf("NewMessage() senderID = %v, want %v", msg.SenderID, tt.senderID)
				}
			}
		})
	}
}

func TestMessage_ParsePayload(t *testing.T) {
	// Test HandshakePayload
	t.Run("handshake payload", func(t *testing.T) {
		originalPayload := HandshakePayload{
			NodeID:     "node1",
			Address:    "localhost:8080",
			KnownPeers: []string{"peer1", "peer2"},
			Key:        []byte("testkey"),
		}

		msg, err := NewMessage(MessageTypeHandshake, "node1", originalPayload)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}

		var parsedPayload HandshakePayload
		if err := msg.ParsePayload(&parsedPayload); err != nil {
			t.Fatalf("Failed to parse payload: %v", err)
		}

		if parsedPayload.NodeID != originalPayload.NodeID {
			t.Errorf("NodeID = %v, want %v", parsedPayload.NodeID, originalPayload.NodeID)
		}
		if parsedPayload.Address != originalPayload.Address {
			t.Errorf("Address = %v, want %v", parsedPayload.Address, originalPayload.Address)
		}
		if len(parsedPayload.KnownPeers) != len(originalPayload.KnownPeers) {
			t.Errorf("KnownPeers length = %v, want %v", len(parsedPayload.KnownPeers), len(originalPayload.KnownPeers))
		}
		if string(parsedPayload.Key) != string(originalPayload.Key) {
			t.Errorf("Key = %v, want %v", string(parsedPayload.Key), string(originalPayload.Key))
		}
	})

	// Test DataPayload
	t.Run("data payload", func(t *testing.T) {
		originalPayload := DataPayload{
			ContentHash: "abc123",
			FileName:    "test.txt",
			Size:        1024,
			Encrypted:   true,
			IV:          []byte("testiv"),
		}

		msg, err := NewMessage(MessageTypeData, "node1", originalPayload)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}

		var parsedPayload DataPayload
		if err := msg.ParsePayload(&parsedPayload); err != nil {
			t.Fatalf("Failed to parse payload: %v", err)
		}

		if parsedPayload.ContentHash != originalPayload.ContentHash {
			t.Errorf("ContentHash = %v, want %v", parsedPayload.ContentHash, originalPayload.ContentHash)
		}
		if parsedPayload.FileName != originalPayload.FileName {
			t.Errorf("FileName = %v, want %v", parsedPayload.FileName, originalPayload.FileName)
		}
		if parsedPayload.Size != originalPayload.Size {
			t.Errorf("Size = %v, want %v", parsedPayload.Size, originalPayload.Size)
		}
		if parsedPayload.Encrypted != originalPayload.Encrypted {
			t.Errorf("Encrypted = %v, want %v", parsedPayload.Encrypted, originalPayload.Encrypted)
		}
		if string(parsedPayload.IV) != string(originalPayload.IV) {
			t.Errorf("IV = %v, want %v", string(parsedPayload.IV), string(originalPayload.IV))
		}
	})

	// Test invalid payload parsing
	t.Run("invalid payload", func(t *testing.T) {
		msg := &Message{
			Type:     MessageTypeData,
			SenderID: "node1",
			Payload:  json.RawMessage(`invalid json`),
		}

		var payload DataPayload
		if err := msg.ParsePayload(&payload); err == nil {
			t.Error("Expected error for invalid payload, got nil")
		}
	})
}
