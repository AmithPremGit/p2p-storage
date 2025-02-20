package protocol

import (
	"encoding/json"
)

// MessageType represents the type of message being sent
type MessageType string

const (
	MessageTypeHandshake    MessageType = "handshake"
	MessageTypeData         MessageType = "data"
	MessageTypeDiscovery    MessageType = "discovery"
	MessageTypeDataRequest  MessageType = "data_request"
	MessageTypeDataTransfer MessageType = "data_transfer"
)

// Message represents a protocol message
type Message struct {
	Type     MessageType     `json:"type"`
	SenderID string          `json:"sender_id"`
	Payload  json.RawMessage `json:"payload"`
}

// HandshakePayload represents the handshake message payload
type HandshakePayload struct {
	NodeID     string   `json:"node_id"`
	Address    string   `json:"address"`
	KnownPeers []string `json:"known_peers"`
	Key        []byte   `json:"key"`
}

// DataPayload represents a file transfer message
type DataPayload struct {
	ContentHash string `json:"content_hash"`
	FileName    string `json:"file_name"`
	Size        int64  `json:"size"`
	Encrypted   bool   `json:"encrypted"`
	IV          []byte `json:"iv"`
	FromWatch   bool   `json:"from_watch"`
}

// DataRequest represents a request for file data
type DataRequest struct {
	ContentHash string `json:"content_hash"`
	FromWatch   bool   `json:"from_watch"`
}

// DataTransfer represents a file data transfer
type DataTransfer struct {
	ContentHash string `json:"content_hash"`
	Data        []byte `json:"data"`
	ChunkIndex  int    `json:"chunk_index"`
	FinalChunk  bool   `json:"final_chunk"`
	IV          []byte `json:"iv,omitempty"` // IV included in first chunk
	FromWatch   bool   `json:"from_watch"`
}

// DiscoveryPayload represents a peer discovery message
type DiscoveryPayload struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

// NewMessage creates a new message with the given type and payload
func NewMessage(msgType MessageType, senderID string, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:     msgType,
		SenderID: senderID,
		Payload:  payloadBytes,
	}, nil
}

// ParsePayload parses the message payload into the given interface
func (m *Message) ParsePayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}
