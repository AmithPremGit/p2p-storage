package storage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestStore(t *testing.T) (*Store, string, func()) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create store
	store, err := NewStore(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return store, tmpDir, cleanup
}

func TestStore_StoreAndLoad(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Test data
	content := "test content"
	contentHash := "testhash123"

	// Store the content
	err := store.Store(contentHash, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to store content: %v", err)
	}

	// Check if file exists
	if !store.Exists(contentHash) {
		t.Error("Stored file does not exist")
	}

	// Load and verify content
	reader, err := store.Load(contentHash)
	if err != nil {
		t.Fatalf("Failed to load content: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if string(data) != content {
		t.Errorf("Loaded content does not match stored content. Got %s, want %s", string(data), content)
	}
}

func TestStore_Delete(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Store test content
	contentHash := "deletehash123"
	err := store.Store(contentHash, strings.NewReader("delete test"))
	if err != nil {
		t.Fatalf("Failed to store content: %v", err)
	}

	// Delete the content
	err = store.Delete(contentHash)
	if err != nil {
		t.Fatalf("Failed to delete content: %v", err)
	}

	// Verify content no longer exists
	if store.Exists(contentHash) {
		t.Error("Content still exists after deletion")
	}
}

func TestStore_List(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Store multiple files
	files := map[string]string{
		"abc123456789": "content1",
		"def123456789": "content2",
		"ghi123456789": "content3",
	}

	for hash, content := range files {
		err := store.Store(hash, strings.NewReader(content))
		if err != nil {
			t.Fatalf("Failed to store content for hash %s: %v", hash, err)
		}
	}

	// List files
	list, err := store.List()
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	// Check if all files are listed
	if len(list) != len(files) {
		t.Errorf("Listed files count mismatch. Got %d, want %d", len(list), len(files))
	}

	// Verify each file exists in the list
	for hash := range files {
		found := false
		expectedPath := filepath.Join(hash[0:2], hash[2:4], hash[4:])
		expectedPath = filepath.ToSlash(expectedPath) // Normalize path separators
		for _, listedPath := range list {
			if listedPath == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Hash %s (path: %s) not found in listed files: %v", hash, expectedPath, list)
		}
	}
}
