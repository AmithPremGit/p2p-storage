# P2P Storage

A secure peer-to-peer file sharing system implemented in Go. This project enables encrypted file sharing across a network of peers with automatic discovery and synchronization capabilities.

## Features

- **Secure File Sharing**
  - AES-256 encryption for all file transfers
  - Stream-based encryption handling for large files
  - Secure key distribution across the network

- **Intelligent Storage**
  - Content-addressable storage system
  - Automatic deduplication
  - Efficient directory sharding for large collections

- **Network Features**
  - Automatic peer discovery
  - Watch directory for automatic file sharing
  - TCP-based transport with reliable message delivery
  - Chunked file transfers for better performance

## Installation

```bash
# Clone the repository
git clone https://github.com/AmithPremGit/p2p-storage
cd p2p-storage

# Install dependencies
go mod tidy

# Build the project
go build
```

## Usage

### Starting the First Node

```bash
# Start the first node (node ID and port)
go run cmd/main.go node1 3000
```

### Joining the Network

```bash
# Start additional nodes (specify node ID and port)
go run cmd/main.go node2 3001

# You can start multiple nodes on different ports
go run cmd/main.go node3 3002
```

### File Sharing

The system automatically creates and manages several directories:

- `watch/` - Place files here to automatically share them
- `store/` - Where encrypted files are stored
- `downloads/` - Where received files are decrypted and saved

To share files:
1. Place any file in the `watch/` directory
2. The file will be automatically encrypted and shared with other nodes
3. Other nodes will receive and store the encrypted file
4. Files can be retrieved and will be decrypted to the `downloads/` directory

## Architecture

The system consists of several key components:

- **Node**: Main coordinator managing peers and file transfers
- **Transport**: Handles network communication and peer connections
- **Store**: Manages the content-addressable storage system
- **Crypto**: Handles encryption/decryption and key management

## Development

### Testing

The project includes comprehensive test coverage across all components:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Tests are provided for:
- Store operations and file handling
- Network transport and peer communication
- Protocol message handling
- Cryptographic operations
- Error cases and edge conditions

Each package contains its own test suite:
- `crypto_test.go`: Tests for encryption/decryption and hashing
- `store_test.go`: Tests for file storage operations
- `transport_test.go`: Tests for network communication
- `peer_test.go`: Tests for peer management
- `message_test.go`: Tests for protocol messages
- `handshake_test.go`: Tests for peer handshake process

### Project Structure

```
.
├── internal/
│   ├── crypto/     # Encryption and hashing
│   ├── network/    # Network transport and peers 
│   ├── protocol/   # Message protocols
│   └── storage/    # File storage system
└── cmd/
    └── main        # Main application
```

## Security Considerations

- All file transfers are encrypted using AES-256
- Content integrity is verified using SHA-1 hashing
- Network keys are distributed securely
- Files are stored in encrypted format
