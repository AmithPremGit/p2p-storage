package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	KeySize   = 32 // AES-256
	IVSize    = aes.BlockSize
	ChunkSize = 1024 * 64 // 64KB chunks for streaming
)

// Key represents an encryption key
type Key []byte

// GenerateKey generates a new random AES-256 key
func GenerateKey() (Key, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateIV generates a new random initialization vector
func GenerateIV() ([]byte, error) {
	iv := make([]byte, IVSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}
	return iv, nil
}

// ContentHash generates a SHA-1 hash of the content
func ContentHash(r io.Reader) (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to hash content: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// EncryptStream encrypts data from reader and writes to writer using AES-CTR
func EncryptStream(key Key, r io.Reader, w io.Writer) error {
	if len(key) != KeySize {
		return fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate and write IV
	iv, err := GenerateIV()
	if err != nil {
		return err
	}
	if _, err := w.Write(iv); err != nil {
		return fmt.Errorf("failed to write IV: %w", err)
	}

	// Create stream cipher
	stream := cipher.NewCTR(block, iv)

	// Create buffer for encryption
	buf := make([]byte, ChunkSize)
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read plaintext: %w", err)
		}

		// Encrypt chunk
		encrypted := make([]byte, n)
		stream.XORKeyStream(encrypted, buf[:n])

		// Write encrypted chunk
		if _, err := w.Write(encrypted); err != nil {
			return fmt.Errorf("failed to write ciphertext: %w", err)
		}
	}

	return nil
}

// DecryptStream decrypts data from reader and writes to writer using AES-CTR
func DecryptStream(key Key, r io.Reader, w io.Writer) error {
	if len(key) != KeySize {
		return fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Read IV
	iv := make([]byte, IVSize)
	if _, err := io.ReadFull(r, iv); err != nil {
		return fmt.Errorf("failed to read IV: %w", err)
	}

	// Create stream cipher
	stream := cipher.NewCTR(block, iv)

	// Create buffer for decryption
	buf := make([]byte, ChunkSize)
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read ciphertext: %w", err)
		}

		// Decrypt chunk
		decrypted := make([]byte, n)
		stream.XORKeyStream(decrypted, buf[:n])

		// Write decrypted chunk
		if _, err := w.Write(decrypted); err != nil {
			return fmt.Errorf("failed to write plaintext: %w", err)
		}
	}

	return nil
}
