package protocol

import (
	"encoding/json"
	"fmt"
	"io"
)

// Handshaker handles the handshake process
type Handshaker struct {
	NodeID     string
	Address    string
	KnownPeers []string
}

// NewHandshaker creates a new handshake handler
func NewHandshaker(nodeID, address string, knownPeers []string) *Handshaker {
	return &Handshaker{
		NodeID:     nodeID,
		Address:    address,
		KnownPeers: knownPeers,
	}
}

// CreateHandshake creates a handshake message
func (h *Handshaker) CreateHandshake() (*Message, error) {
	payload := HandshakePayload{
		NodeID:     h.NodeID,
		Address:    h.Address,
		KnownPeers: h.KnownPeers,
	}

	return NewMessage(MessageTypeHandshake, h.NodeID, payload)
}

// HandleHandshake processes a received handshake message
func (h *Handshaker) HandleHandshake(msg *Message) (*HandshakePayload, error) {
	if msg.Type != MessageTypeHandshake {
		return nil, fmt.Errorf("invalid message type: expected handshake, got %s", msg.Type)
	}

	var payload HandshakePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return nil, fmt.Errorf("failed to parse handshake payload: %w", err)
	}

	return &payload, nil
}

// WriteHandshake writes a handshake message to a writer
func (h *Handshaker) WriteHandshake(w io.Writer) error {
	msg, err := h.CreateHandshake()
	if err != nil {
		return err
	}

	return json.NewEncoder(w).Encode(msg)
}

// ReadHandshake reads and processes a handshake message from a reader
func (h *Handshaker) ReadHandshake(r io.Reader) (*HandshakePayload, error) {
	var msg Message
	if err := json.NewDecoder(r).Decode(&msg); err != nil {
		return nil, err
	}

	return h.HandleHandshake(&msg)
}
