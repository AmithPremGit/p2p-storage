package node

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"p2p-storage/internal/crypto"
	"p2p-storage/internal/network"
	"p2p-storage/internal/protocol"
	"p2p-storage/internal/storage"

	"github.com/fsnotify/fsnotify"
)

// Node represents a P2P node
type PeerInfo struct {
	ID      string
	Address string
}

type Node struct {
	ID          string
	transport   *network.Transport
	store       *storage.Store
	localKey    crypto.Key
	networkKey  crypto.Key
	isFirstNode bool
	watchDir    string
	watcher     *fsnotify.Watcher
	peers       map[string]PeerInfo
	transfers   map[string]*transferState
	done        chan struct{}
	mu          sync.RWMutex
	keyReady    chan struct{} // Channel to signal network key is ready
}

type transferState struct {
	tempFile  *os.File
	chunks    map[int]bool
	received  int
	fromWatch bool
}

// NewNode creates a new P2P node
func NewNode(nodeID, address, storeDir, watchDir string) (*Node, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	store, err := storage.NewStore(storeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	node := &Node{
		ID:          nodeID,
		localKey:    key,
		networkKey:  key,
		isFirstNode: len(os.Args) <= 3,
		store:       store,
		watchDir:    watchDir,
		peers:       make(map[string]PeerInfo),
		transfers:   make(map[string]*transferState),
		done:        make(chan struct{}),
		keyReady:    make(chan struct{}),
	}

	// If this is the first node, mark key as ready immediately
	if node.isFirstNode {
		close(node.keyReady)
	}

	transport, err := network.NewTransport(nodeID, address, node)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}
	node.transport = transport

	return node, nil
}

// Start starts the node
func (n *Node) Start() error {
	n.transport.Start()
	if err := n.startWatcher(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	return nil
}

// Stop stops the node
func (n *Node) Stop() {
	close(n.done)
	n.transport.Stop()
	if n.watcher != nil {
		n.watcher.Close()
	}
}

// HandleMessage implements the MessageHandler interface
func (n *Node) HandleMessage(peer *network.Peer, msg *protocol.Message) error {
	switch msg.Type {
	case protocol.MessageTypeHandshake:
		return n.handleHandshake(peer, msg)
	case protocol.MessageTypeData:
		return n.handleData(peer, msg)
	case protocol.MessageTypeDiscovery:
		return n.handleDiscovery(peer, msg)
	case protocol.MessageTypeDataRequest:
		return n.handleDataRequest(peer, msg)
	case protocol.MessageTypeDataTransfer:
		return n.handleDataTransfer(peer, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (n *Node) handleHandshake(peer *network.Peer, msg *protocol.Message) error {
	var payload protocol.HandshakePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return fmt.Errorf("failed to parse handshake: %w", err)
	}

	n.mu.Lock()
	// Store peer information
	n.peers[payload.NodeID] = PeerInfo{
		ID:      payload.NodeID,
		Address: payload.Address,
	}

	// Key exchange logic
	if n.isFirstNode {
		// fmt.Printf("DEBUG: First node handling handshake from %s\n", payload.NodeID)
		// fmt.Printf("DEBUG: Sending network key: %v\n", n.networkKey != nil)
	} else {
		if payload.Key != nil {
			n.networkKey = payload.Key
			// fmt.Printf("Adopted network key from peer %s\n", payload.NodeID)
			// Signal that key is ready
			select {
			case <-n.keyReady: // Channel already closed
			default:
				close(n.keyReady)
			}
		}
	}
	n.mu.Unlock()

	// Prepare response
	response := protocol.HandshakePayload{
		NodeID:     n.ID,
		Address:    n.transport.Address(),
		KnownPeers: n.getKnownPeers(),
	}

	// Only the first node sends its key
	if n.isFirstNode {
		n.mu.RLock()
		response.Key = n.networkKey
		n.mu.RUnlock()
	}

	responseMsg, err := protocol.NewMessage(protocol.MessageTypeHandshake, n.ID, response)
	if err != nil {
		return err
	}

	return peer.Send(responseMsg)
}

func (n *Node) handleNewFile(path string) {
	fmt.Printf("\nDEBUG: Starting to handle new file: %s\n", path)

	// Wait for key to be ready before processing
	if err := n.waitForKey(10 * time.Second); err != nil {
		fmt.Printf("DEBUG: Failed waiting for network key: %v\n", err)
		return
	}

	// Add a small delay to ensure file is completely written
	time.Sleep(100 * time.Millisecond)

	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("DEBUG: Failed to open file: %v\n", err)
		return
	}
	defer file.Close()

	tempFile, err := n.store.CreateTemp()
	if err != nil {
		fmt.Printf("DEBUG: Failed to create temp file: %v\n", err)
		return
	}
	defer tempFile.Close()

	n.mu.RLock()
	key := n.networkKey
	fmt.Printf("DEBUG: Network key present: %v\n", key != nil)
	n.mu.RUnlock()

	fmt.Printf("DEBUG: Attempting to encrypt file...\n")
	if err := crypto.EncryptStream(key, file, tempFile); err != nil {
		fmt.Printf("DEBUG: Failed to encrypt file: %v\n", err)
		return
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		fmt.Printf("DEBUG: Failed to reset file pointer for hashing: %v\n", err)
		return
	}

	fmt.Printf("DEBUG: Calculating hash...\n")
	hash, err := crypto.ContentHash(tempFile)
	if err != nil {
		fmt.Printf("DEBUG: Failed to calculate hash: %v\n", err)
		return
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		fmt.Printf("DEBUG: Failed to reset file pointer for storage: %v\n", err)
		return
	}

	fmt.Printf("DEBUG: Storing file with hash: %s\n", hash)
	if err := n.store.Store(hash, tempFile); err != nil {
		fmt.Printf("DEBUG: Failed to store file: %v\n", err)
		return
	}

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("DEBUG: Failed to get file info: %v\n", err)
		return
	}

	payload := protocol.DataPayload{
		ContentHash: hash,
		FileName:    filepath.Base(path),
		Size:        fileInfo.Size(),
		Encrypted:   true,
		FromWatch:   true,
	}

	msg, err := protocol.NewMessage(protocol.MessageTypeData, n.ID, payload)
	if err != nil {
		// fmt.Printf("DEBUG: Failed to create message: %v\n", err)
		return
	}

	fmt.Printf("DEBUG: Broadcasting file %s with hash %s\n", filepath.Base(path), hash)
	n.mu.RLock()
	peerCount := len(n.peers)
	n.mu.RUnlock()
	fmt.Printf("DEBUG: Number of connected peers: %d\n", peerCount)

	if err := n.transport.Broadcast(msg); err != nil {
		fmt.Printf("DEBUG: Failed to broadcast message: %v\n", err)
		return
	}
	// fmt.Printf("DEBUG: File processing complete\n")
}

func (n *Node) handleData(peer *network.Peer, msg *protocol.Message) error {
	var payload protocol.DataPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	if n.store.Exists(payload.ContentHash) {
		return nil
	}

	request := protocol.DataRequest{
		ContentHash: payload.ContentHash,
		FromWatch:   payload.FromWatch,
	}
	requestMsg, err := protocol.NewMessage(protocol.MessageTypeDataRequest, n.ID, request)
	if err != nil {
		return fmt.Errorf("failed to create data request: %w", err)
	}

	return peer.Send(requestMsg)
}

func (n *Node) handleDataRequest(peer *network.Peer, msg *protocol.Message) error {
	var request protocol.DataRequest
	if err := msg.ParsePayload(&request); err != nil {
		return fmt.Errorf("failed to parse data request: %w", err)
	}

	file, err := n.store.Load(request.ContentHash)
	if err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 1024*1024) // 1MB chunks
	chunkIndex := 0
	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		transfer := protocol.DataTransfer{
			ContentHash: request.ContentHash,
			Data:        buffer[:bytesRead],
			ChunkIndex:  chunkIndex,
			FinalChunk:  bytesRead < len(buffer),
			FromWatch:   request.FromWatch,
		}

		transferMsg, err := protocol.NewMessage(protocol.MessageTypeDataTransfer, n.ID, transfer)
		if err != nil {
			return fmt.Errorf("failed to create transfer message: %w", err)
		}

		if err := peer.Send(transferMsg); err != nil {
			return fmt.Errorf("failed to send chunk: %w", err)
		}

		chunkIndex++
	}

	return nil
}

func (n *Node) handleDataTransfer(peer *network.Peer, msg *protocol.Message) error {
	var transfer protocol.DataTransfer
	if err := msg.ParsePayload(&transfer); err != nil {
		return fmt.Errorf("failed to parse data transfer: %w", err)
	}

	transferKey := fmt.Sprintf("%s-%s", peer.ID(), transfer.ContentHash)

	n.mu.Lock()
	state, exists := n.transfers[transferKey]
	if !exists {
		tempFile, err := n.store.CreateTemp()
		if err != nil {
			n.mu.Unlock()
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		state = &transferState{
			tempFile:  tempFile,
			chunks:    make(map[int]bool),
			fromWatch: transfer.FromWatch,
		}
		n.transfers[transferKey] = state
	}
	n.mu.Unlock()

	offset := int64(transfer.ChunkIndex * 1024 * 1024)
	if _, err := state.tempFile.WriteAt(transfer.Data, offset); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}

	n.mu.Lock()
	state.chunks[transfer.ChunkIndex] = true
	state.received++
	n.mu.Unlock()

	if transfer.FinalChunk {
		if state.fromWatch {
			// For watch transfers, just store in store directory
			if err := n.finalizeWatchTransfer(transferKey, transfer.ContentHash); err != nil {
				return fmt.Errorf("failed to finalize watch transfer: %w", err)
			}
		} else {
			// For manual get requests, decrypt to downloads directory
			if err := n.finalizeDownload(transferKey, transfer.ContentHash); err != nil {
				return fmt.Errorf("failed to finalize download: %w", err)
			}
		}
	}

	return nil
}

func (n *Node) finalizeWatchTransfer(transferKey, expectedHash string) error {
	n.mu.Lock()
	state, exists := n.transfers[transferKey]
	if !exists {
		n.mu.Unlock()
		return fmt.Errorf("transfer state not found")
	}
	delete(n.transfers, transferKey)
	n.mu.Unlock()

	// cleanup temporary files
	defer func() {
		state.tempFile.Close()
		os.Remove(state.tempFile.Name())
	}()

	defer state.tempFile.Close()

	// Verify hash
	if _, err := state.tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	hash, err := crypto.ContentHash(state.tempFile)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	if hash != expectedHash {
		return fmt.Errorf("content hash mismatch")
	}

	// Store in store directory without decrypting
	if _, err := state.tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	if err := n.store.Store(expectedHash, state.tempFile); err != nil {
		return fmt.Errorf("failed to store file: %w", err)
	}

	fmt.Printf("File stored in store directory with hash: %s\n", expectedHash)
	return nil
}

func (n *Node) finalizeDownload(transferKey, expectedHash string) error {
	n.mu.Lock()
	state, exists := n.transfers[transferKey]
	if !exists {
		n.mu.Unlock()
		return fmt.Errorf("transfer state not found")
	}
	delete(n.transfers, transferKey)
	n.mu.Unlock()

	// cleanup temporary files
	defer func() {
		state.tempFile.Close()
		os.Remove(state.tempFile.Name())
	}()

	defer state.tempFile.Close()

	if _, err := state.tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	hash, err := crypto.ContentHash(state.tempFile)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	if hash != expectedHash {
		return fmt.Errorf("content hash mismatch")
	}

	finalPath := filepath.Join("downloads", expectedHash)
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	if _, err := state.tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	n.mu.RLock()
	key := n.networkKey
	n.mu.RUnlock()

	if err := crypto.DecryptStream(key, state.tempFile, finalFile); err != nil {
		os.Remove(finalPath)
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	os.Remove(state.tempFile.Name())
	fmt.Printf("File downloaded and decrypted to: %s\n", finalPath)
	return nil
}

func (n *Node) handleDiscovery(peer *network.Peer, msg *protocol.Message) error {
	var payload protocol.DiscoveryPayload
	if err := msg.ParsePayload(&payload); err != nil {
		fmt.Printf("Received discovery from peer %s: failed to parse payload: %v\n", peer.ID(), err)
		return fmt.Errorf("failed to parse discovery payload from peer %s: %w", peer.ID(), err)
	}

	// Skip if it's our own node
	if payload.NodeID == n.ID {
		fmt.Printf("Received discovery from peer %s: skipping own node ID\n", peer.ID())
		return nil
	}

	n.mu.RLock()
	_, alreadyConnected := n.peers[payload.NodeID]
	n.mu.RUnlock()

	if !alreadyConnected {
		fmt.Printf("Discovered new peer %s through peer %s\n", payload.NodeID, peer.ID())
		go func() {
			if err := n.Connect(payload.Address); err != nil {
				fmt.Printf("Failed to connect to discovered peer %s (through %s): %v\n",
					payload.NodeID, peer.ID(), err)
			} else {
				fmt.Printf("Successfully connected to discovered peer %s (through %s)\n",
					payload.NodeID, peer.ID())
			}
		}()
	} else {
		fmt.Printf("Received discovery from peer %s: already connected to %s\n",
			peer.ID(), payload.NodeID)
	}

	return nil
}

// waitForKey waits for network key to be ready
func (n *Node) waitForKey(timeout time.Duration) error {
	if n.isFirstNode {
		return nil // First node already has the key
	}

	select {
	case <-n.keyReady:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for network key after %v", timeout)
	}
}

func (n *Node) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	n.watcher = watcher

	if err := os.MkdirAll(n.watchDir, 0755); err != nil {
		return err
	}

	if err := watcher.Add(n.watchDir); err != nil {
		return err
	}

	fmt.Printf("Started watching directory: %s\n", n.watchDir)
	go n.watchLoop()
	return nil
}

func (n *Node) watchLoop() {
	fmt.Printf("Watch loop started for directory: %s\n", n.watchDir)
	for {
		select {
		case <-n.done:
			fmt.Printf("Watch loop terminating\n")
			return
		case event, ok := <-n.watcher.Events:
			if !ok {
				fmt.Printf("Watch event channel closed\n")
				return
			}
			fmt.Printf("Watch event received: %s %s\n", event.Op, event.Name)
			if event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Printf("Create event detected, calling handleNewFile for: %s\n", event.Name)
				go n.handleNewFile(event.Name)
			}
		case err, ok := <-n.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

// Connect connects to a peer
func (n *Node) Connect(address string) error {
	// When a non-first node connects, it should prepare to receive the network key
	if !n.isFirstNode {
		fmt.Printf("Connecting to established node to receive network key...\n")
	}
	return n.transport.Connect(address)
}

// List returns a list of stored files
func (n *Node) List() ([]string, error) {
	return n.store.List()
}

// StoreFile stores a file
func (n *Node) StoreFile(path string) (string, error) {
	// Wait for key to be ready before storing
	if err := n.waitForKey(10 * time.Second); err != nil {
		return "", fmt.Errorf("failed waiting for network key: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	tempFile, err := n.store.CreateTemp()
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	n.mu.RLock()
	key := n.networkKey
	n.mu.RUnlock()

	if err := crypto.EncryptStream(key, file, tempFile); err != nil {
		return "", fmt.Errorf("failed to encrypt file: %w", err)
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		return "", err
	}

	hash, err := crypto.ContentHash(tempFile)
	if err != nil {
		return "", err
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		return "", err
	}

	if err := n.store.Store(hash, tempFile); err != nil {
		return "", err
	}

	return hash, nil
}

// GetFile retrieves a file and its decryption key
func (n *Node) GetFile(contentHash string) (io.ReadCloser, crypto.Key, error) {
	// Create downloads directory if it doesn't exist
	if err := os.MkdirAll("downloads", 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// Wait for key to be ready before getting file
	if err := n.waitForKey(10 * time.Second); err != nil {
		return nil, nil, fmt.Errorf("failed waiting for network key: %w", err)
	}

	// First try local storage
	reader, err := n.store.Load(contentHash)
	if err == nil {
		n.mu.RLock()
		key := n.networkKey
		n.mu.RUnlock()
		return reader, key, nil
	}

	// If not found locally, request from peers
	request := protocol.DataRequest{
		ContentHash: contentHash,
	}

	requestMsg, err := protocol.NewMessage(protocol.MessageTypeDataRequest, n.ID, request)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request message: %w", err)
	}

	if err := n.transport.Broadcast(requestMsg); err != nil {
		return nil, nil, fmt.Errorf("failed to broadcast request: %w", err)
	}

	n.mu.RLock()
	key := n.networkKey
	n.mu.RUnlock()

	return nil, key, fmt.Errorf("file not found locally, request sent to peers")
}

func (n *Node) getKnownPeers() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]string, 0, len(n.peers))
	for _, p := range n.peers {
		peers = append(peers, p.Address)
	}
	return peers
}
