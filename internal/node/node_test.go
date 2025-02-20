package node

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "node-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewNode(t *testing.T) {
	baseDir, cleanup := setupTestDir(t)
	defer cleanup()

	storeDir := filepath.Join(baseDir, "store")
	watchDir := filepath.Join(baseDir, "watch")

	// Create watch directory first
	if err := os.MkdirAll(watchDir, 0755); err != nil {
		t.Fatalf("Failed to create watch directory: %v", err)
	}

	node, err := NewNode("test-node", ":0", storeDir, watchDir)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	if node.ID != "test-node" {
		t.Errorf("Node ID = %v, want %v", node.ID, "test-node")
	}

	// Verify directories were created
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		t.Error("Store directory was not created")
	}
	if _, err := os.Stat(watchDir); os.IsNotExist(err) {
		t.Error("Watch directory was not created")
	}

	// Verify node components were initialized
	if node.store == nil {
		t.Error("Store was not initialized")
	}
	if node.transport == nil {
		t.Error("Transport was not initialized")
	}
}

func TestNode_List(t *testing.T) {
	baseDir, cleanup := setupTestDir(t)
	defer cleanup()

	storeDir := filepath.Join(baseDir, "store")
	watchDir := filepath.Join(baseDir, "watch")

	if err := os.MkdirAll(watchDir, 0755); err != nil {
		t.Fatalf("Failed to create watch directory: %v", err)
	}

	node, err := NewNode("test-node", ":0", storeDir, watchDir)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	// List files should return empty list initially
	files, err := node.List()
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected empty list, got %d files", len(files))
	}
}
