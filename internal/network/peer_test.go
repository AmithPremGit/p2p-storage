package network

import (
	"net"
	"sync"
	"testing"
	"time"

	"p2p-storage/internal/protocol"
)

type mockAddr struct{}

func (a *mockAddr) Network() string { return "mock" }
func (a *mockAddr) String() string  { return "mock:1234" }

type mockConn struct {
	mu        sync.Mutex
	closed    bool
	readData  []byte
	writeData []byte
	addr      net.Addr
}

func newMockConn() *mockConn {
	return &mockConn{
		addr: &mockAddr{},
	}
}

func (c *mockConn) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, net.ErrClosed
	}
	if len(c.readData) == 0 {
		return 0, nil
	}
	n = copy(b, c.readData)
	c.readData = c.readData[n:]
	return n, nil
}

func (c *mockConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, net.ErrClosed
	}
	c.writeData = append(c.writeData, b...)
	return len(b), nil
}

func (c *mockConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *mockConn) LocalAddr() net.Addr                { return c.addr }
func (c *mockConn) RemoteAddr() net.Addr               { return c.addr }
func (c *mockConn) SetDeadline(t time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type mockHandler struct{}

func (h *mockHandler) HandleMessage(peer *Peer, msg *protocol.Message) error {
	return nil
}

func TestNewPeer(t *testing.T) {
	conn := newMockConn()
	handler := &mockHandler{}
	peer := NewPeer(conn, handler)

	if peer == nil {
		t.Fatal("NewPeer returned nil")
	}

	if peer.ID() != conn.RemoteAddr().String() {
		t.Errorf("Peer ID = %v, want %v", peer.ID(), conn.RemoteAddr().String())
	}
}

func TestPeer_Send(t *testing.T) {
	conn := newMockConn()
	handler := &mockHandler{}
	peer := NewPeer(conn, handler)

	msg, err := protocol.NewMessage(protocol.MessageTypeData, "test", nil)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	if err := peer.Send(msg); err != nil {
		t.Errorf("Failed to send message: %v", err)
	}

	// Give some time for the message to be processed
	time.Sleep(100 * time.Millisecond)

	conn.mu.Lock()
	if len(conn.writeData) == 0 {
		t.Error("No data was written to connection")
	}
	conn.mu.Unlock()
}

func TestPeer_Close(t *testing.T) {
	conn := newMockConn()
	handler := &mockHandler{}
	peer := NewPeer(conn, handler)

	if err := peer.Close(); err != nil {
		t.Errorf("Failed to close peer: %v", err)
	}

	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if !closed {
		t.Error("Connection not marked as closed")
	}
}
