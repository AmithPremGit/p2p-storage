package crypto

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if len(key1) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(key1))
	}

	// Generate another key to ensure they're different
	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate second key: %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("Generated keys are identical")
	}
}

func TestGenerateIV(t *testing.T) {
	iv1, err := GenerateIV()
	if err != nil {
		t.Fatalf("Failed to generate IV: %v", err)
	}

	if len(iv1) != IVSize {
		t.Errorf("Expected IV size %d, got %d", IVSize, len(iv1))
	}

	// Generate another IV to ensure they're different
	iv2, err := GenerateIV()
	if err != nil {
		t.Fatalf("Failed to generate second IV: %v", err)
	}

	if bytes.Equal(iv1, iv2) {
		t.Error("Generated IVs are identical")
	}
}

func TestContentHash(t *testing.T) {
	// Test with known content
	content := "test content"
	reader := strings.NewReader(content)

	hash1, err := ContentHash(reader)
	if err != nil {
		t.Fatalf("Failed to generate hash: %v", err)
	}

	// Test with same content again
	reader = strings.NewReader(content)
	hash2, err := ContentHash(reader)
	if err != nil {
		t.Fatalf("Failed to generate second hash: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Hashes of identical content are different")
	}

	// Test with different content
	reader = strings.NewReader("different content")
	hash3, err := ContentHash(reader)
	if err != nil {
		t.Fatalf("Failed to generate third hash: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Hashes of different content are identical")
	}
}

func TestEncryptDecryptStream(t *testing.T) {
	// Generate key
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Test data
	plaintext := "This is a test message for encryption and decryption"
	plaintextReader := strings.NewReader(plaintext)

	// Encrypt
	var encryptedBuf bytes.Buffer
	err = EncryptStream(key, plaintextReader, &encryptedBuf)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Decrypt
	var decryptedBuf bytes.Buffer
	err = DecryptStream(key, &encryptedBuf, &decryptedBuf)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	// Compare results
	decrypted := decryptedBuf.String()
	if decrypted != plaintext {
		t.Errorf("Decrypted text doesn't match original.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}

func TestEncryptDecryptStreamLargeData(t *testing.T) {
	// Generate key
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Generate large test data (larger than ChunkSize)
	dataSize := ChunkSize * 3
	largeData := make([]byte, dataSize)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Encrypt
	plaintextReader := bytes.NewReader(largeData)
	var encryptedBuf bytes.Buffer
	err = EncryptStream(key, plaintextReader, &encryptedBuf)
	if err != nil {
		t.Fatalf("Failed to encrypt large data: %v", err)
	}

	// Decrypt
	var decryptedBuf bytes.Buffer
	err = DecryptStream(key, &encryptedBuf, &decryptedBuf)
	if err != nil {
		t.Fatalf("Failed to decrypt large data: %v", err)
	}

	// Compare results
	decrypted := decryptedBuf.Bytes()
	if !bytes.Equal(decrypted, largeData) {
		t.Error("Decrypted data doesn't match original for large data")
	}
}

func TestEncryptStreamInvalidKey(t *testing.T) {
	invalidKey := make([]byte, KeySize-1) // Invalid key size
	reader := strings.NewReader("test")
	var writer bytes.Buffer

	err := EncryptStream(invalidKey, reader, &writer)
	if err == nil {
		t.Error("Expected error for invalid key size, got nil")
	}
}

func TestDecryptStreamInvalidKey(t *testing.T) {
	invalidKey := make([]byte, KeySize-1) // Invalid key size
	reader := strings.NewReader("test")
	var writer bytes.Buffer

	err := DecryptStream(invalidKey, reader, &writer)
	if err == nil {
		t.Error("Expected error for invalid key size, got nil")
	}
}

type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestContentHashErrorHandling(t *testing.T) {
	reader := errorReader{}
	_, err := ContentHash(reader)
	if err == nil {
		t.Error("Expected error for failed read, got nil")
	}
}
