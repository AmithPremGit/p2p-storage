package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Store manages the content-addressable storage
type Store struct {
	baseDir string
	tempDir string
	mu      sync.RWMutex
}

// NewStore creates a new storage instance
func NewStore(baseDir string) (*Store, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	// Create temp directory for in-progress transfers
	tempDir := filepath.Join(baseDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	return &Store{
		baseDir: baseDir,
		tempDir: tempDir,
	}, nil
}

// Store stores a file in the content-addressable storage
func (s *Store) Store(contentHash string, r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create temporary file
	tempFile, err := os.CreateTemp(s.tempDir, "store-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up temp file on error

	// Copy content to temporary file
	if _, err := io.Copy(tempFile, r); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write content: %w", err)
	}
	tempFile.Close()

	// Create hash directory structure
	hashPath := s.hashToPath(contentHash)
	if err := os.MkdirAll(filepath.Dir(hashPath), 0755); err != nil {
		return fmt.Errorf("failed to create hash directory: %w", err)
	}

	// Move temporary file to final location
	if err := os.Rename(tempPath, hashPath); err != nil {
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	return nil
}

// Load retrieves a file from storage by its content hash
func (s *Store) Load(contentHash string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hashPath := s.hashToPath(contentHash)
	file, err := os.Open(hashPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Exists checks if a file exists in storage
func (s *Store) Exists(contentHash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.hashToPath(contentHash))
	return err == nil
}

// Delete removes a file from storage
func (s *Store) Delete(contentHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hashPath := s.hashToPath(contentHash)
	if err := os.Remove(hashPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Try to remove empty parent directories
	dir := filepath.Dir(hashPath)
	for dir != s.baseDir {
		if err := os.Remove(dir); err != nil {
			break // Directory not empty or other error
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// hashToPath converts a content hash to a file path
func (s *Store) hashToPath(contentHash string) string {
	// Use first 4 characters as directory names for better distribution
	// Example: abc123... -> base/ab/c1/23...
	return filepath.Join(
		s.baseDir,
		contentHash[0:2],
		contentHash[2:4],
		contentHash[4:],
	)
}

// CreateTemp creates a temporary file for in-progress operations
func (s *Store) CreateTemp() (*os.File, error) {
	return os.CreateTemp(s.tempDir, "transfer-*")
}

// CleanTemp removes all temporary files
func (s *Store) CleanTemp() error {
	dir, err := os.Open(s.tempDir)
	if err != nil {
		return err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err := os.Remove(filepath.Join(s.tempDir, name))
		if err != nil {
			fmt.Printf("Failed to remove temp file %s: %v\n", name, err)
		}
	}

	return nil
}

// List returns a list of all content hashes in storage
func (s *Store) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var hashes []string
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Dir(path) != s.tempDir {
			relPath, err := filepath.Rel(s.baseDir, path)
			if err != nil {
				return err
			}
			// Reconstruct hash from path
			hash := filepath.Join(filepath.Dir(relPath), info.Name())
			hash = filepath.ToSlash(hash) // Normalize separator
			hash = filepath.Clean(hash)   // Clean the path
			hashes = append(hashes, hash)
		}
		return nil
	})

	return hashes, err
}
