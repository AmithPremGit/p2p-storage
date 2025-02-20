package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"p2p-storage/internal/crypto"
	"p2p-storage/internal/node"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: demo <node-id> <port> [peer-address]")
		os.Exit(1)
	}

	nodeID := os.Args[1]
	port := os.Args[2]
	baseDir := filepath.Join("data", nodeID)
	storeDir := filepath.Join(baseDir, "store")
	watchDir := filepath.Join(baseDir, "watch")

	// Create directories
	os.MkdirAll(storeDir, 0755)
	os.MkdirAll(watchDir, 0755)

	// Create node
	n, err := node.NewNode(
		nodeID,
		fmt.Sprintf(":%s", port),
		storeDir,
		watchDir,
	)
	if err != nil {
		fmt.Printf("Failed to create node: %v\n", err)
		os.Exit(1)
	}

	// Start node
	if err := n.Start(); err != nil {
		fmt.Printf("Failed to start node: %v\n", err)
		os.Exit(1)
	}
	defer n.Stop()

	// Connect to peer if provided
	if len(os.Args) > 3 {
		peerAddr := os.Args[3]
		fmt.Printf("Connecting to peer at %s...\n", peerAddr)
		if err := n.Connect(peerAddr); err != nil {
			fmt.Printf("Failed to connect to peer: %v\n", err)
		}
	}

	fmt.Printf("Node %s started. Watch directory: %s\n", nodeID, watchDir)
	fmt.Println("Available commands:")
	fmt.Println("  store <file>  - Store a file")
	fmt.Println("  get <hash>    - Get a file by hash")
	fmt.Println("  list          - List stored files")
	fmt.Println("  connect <addr> - Connect to a peer")
	fmt.Println("  quit          - Exit the program")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		cmd := scanner.Text()
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "store":
			if len(parts) < 2 {
				fmt.Println("Usage: store <file>")
				continue
			}
			filePath := parts[1]
			hash, err := n.StoreFile(filePath)
			if err != nil {
				fmt.Printf("Failed to store file: %v\n", err)
			} else {
				fmt.Printf("File stored with hash: %s\n", hash)
			}

		case "get":
			if len(parts) < 2 {
				fmt.Println("Usage: get <hash>")
				continue
			}
			hash := parts[1]
			reader, key, err := n.GetFile(hash)
			if err != nil {
				fmt.Printf("Failed to get file: %v\n", err)
				continue
			}
			defer reader.Close()

			// Create downloads directory
			os.MkdirAll("downloads", 0755)
			outPath := filepath.Join("downloads", hash)

			// Create temporary file for decrypted content
			tempFile, err := os.CreateTemp("downloads", "decrypted-*")
			if err != nil {
				fmt.Printf("Failed to create temporary file: %v\n", err)
				continue
			}
			tempPath := tempFile.Name()
			defer tempFile.Close()

			// Decrypt using the appropriate key
			if err := crypto.DecryptStream(key, reader, tempFile); err != nil {
				fmt.Printf("Failed to decrypt file: %v\n", err)
				os.Remove(tempPath)
				continue
			}

			// Close temp file before renaming
			tempFile.Close()

			// Move the decrypted file to final location
			if err := os.Rename(tempPath, outPath); err != nil {
				fmt.Printf("Failed to move decrypted file: %v\n", err)
				os.Remove(tempPath)
				continue
			}

			fmt.Printf("File decrypted and saved to: %s\n", outPath)

		case "list":
			files, err := n.List()
			if err != nil {
				fmt.Printf("Failed to list files: %v\n", err)
				continue
			}
			if len(files) == 0 {
				fmt.Println("No files stored")
				continue
			}
			fmt.Println("Stored files:")
			for _, hash := range files {
				fmt.Printf("  %s\n", hash)
			}

		case "connect":
			if len(parts) < 2 {
				fmt.Println("Usage: connect <address>")
				continue
			}
			addr := parts[1]
			if err := n.Connect(addr); err != nil {
				fmt.Printf("Failed to connect: %v\n", err)
			} else {
				fmt.Printf("Connected to %s\n", addr)
			}

		case "quit":
			return

		default:
			fmt.Println("Unknown command")
		}
	}
}
