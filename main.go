package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const uploadDir = "./tmp"

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH, HEAD")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Upload-Length, Upload-Offset, Upload-Metadata, Upload-Name")
	w.Header().Set("Access-Control-Expose-Headers", "Upload-Offset, Location")
}

func upload(w http.ResponseWriter, req *http.Request) {
	setCORSHeaders(w)

	// Handle preflight OPTIONS request
	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	patchID := req.URL.Query().Get("patch")

	switch req.Method {
	case "POST":
		handlePost(w, req)
	case "PATCH":
		if patchID == "" {
			http.Error(w, "Missing patch ID", http.StatusBadRequest)
			return
		}
		handlePatch(w, req, patchID)
	case "HEAD":
		if patchID == "" {
			http.Error(w, "Missing patch ID", http.StatusBadRequest)
			return
		}
		handleHead(w, req, patchID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST: Create unique transfer location
func handlePost(w http.ResponseWriter, req *http.Request) {
	uploadLength := req.Header.Get("Upload-Length")
	if uploadLength == "" {
		http.Error(w, "Upload-Length header required", http.StatusBadRequest)
		return
	}

	// Generate unique ID
	transferID := generateID()
	transferDir := filepath.Join(uploadDir, transferID)

	// Create temporary directory for this transfer
	if err := os.MkdirAll(transferDir, 0755); err != nil {
		log.Printf("Error creating transfer directory: %v", err)
		http.Error(w, "Failed to create transfer location", http.StatusInternalServerError)
		return
	}

	log.Printf("Created transfer %s with length %s bytes", transferID, uploadLength)

	// Store upload metadata
	metadataFile := filepath.Join(transferDir, "metadata.txt")
	metadata := fmt.Sprintf("Upload-Length: %s\nUpload-Metadata: %s\n",
		uploadLength, req.Header.Get("Upload-Metadata"))
	os.WriteFile(metadataFile, []byte(metadata), 0644)

	// Return the unique ID as text/plain
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, transferID)
}

// PATCH: Receive file chunk
func handlePatch(w http.ResponseWriter, req *http.Request, transferID string) {
	transferDir := filepath.Join(uploadDir, transferID)

	// Check if transfer exists
	if _, err := os.Stat(transferDir); os.IsNotExist(err) {
		http.Error(w, "Transfer not found", http.StatusNotFound)
		return
	}

	offsetStr := req.Header.Get("Upload-Offset")
	lengthStr := req.Header.Get("Upload-Length")
	fileName := req.Header.Get("Upload-Name")

	if offsetStr == "" || lengthStr == "" {
		http.Error(w, "Upload-Offset and Upload-Length headers required", http.StatusBadRequest)
		return
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Upload-Offset", http.StatusBadRequest)
		return
	}

	totalLength, err := strconv.ParseInt(lengthStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Upload-Length", http.StatusBadRequest)
		return
	}

	// Save chunk
	chunkFile := filepath.Join(transferDir, "data")
	file, err := os.OpenFile(chunkFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening chunk file: %v", err)
		http.Error(w, "Failed to save chunk", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Seek to offset
	if _, err := file.Seek(offset, 0); err != nil {
		log.Printf("Error seeking to offset: %v", err)
		http.Error(w, "Failed to write chunk", http.StatusInternalServerError)
		return
	}

	// Write chunk data
	written, err := io.Copy(file, req.Body)
	if err != nil {
		log.Printf("Error writing chunk: %v", err)
		http.Error(w, "Failed to write chunk", http.StatusInternalServerError)
		return
	}

	newOffset := offset + written
	log.Printf("Transfer %s: wrote %d bytes at offset %d (new offset: %d/%d)",
		transferID, written, offset, newOffset, totalLength)

	// Check if upload is complete
	if newOffset >= totalLength {
		// Upload complete - move file to final location
		finalName := fileName
		if finalName == "" {
			finalName = "uploaded_file"
		}

		finalPath := filepath.Join(uploadDir, finalName)
		if err := os.Rename(chunkFile, finalPath); err != nil {
			log.Printf("Error moving file to final location: %v", err)
		} else {
			log.Printf("Transfer %s complete: saved as %s", transferID, finalName)
		}
	}

	// Return current offset
	w.Header().Set("Upload-Offset", strconv.FormatInt(newOffset, 10))
	w.WriteHeader(http.StatusNoContent)
}

// HEAD: Get current upload offset (for resume)
func handleHead(w http.ResponseWriter, req *http.Request, transferID string) {
	transferDir := filepath.Join(uploadDir, transferID)
	chunkFile := filepath.Join(transferDir, "data")

	// Check if transfer exists
	if _, err := os.Stat(transferDir); os.IsNotExist(err) {
		http.Error(w, "Transfer not found", http.StatusNotFound)
		return
	}

	// Get current file size (= next expected offset)
	var currentOffset int64 = 0
	if info, err := os.Stat(chunkFile); err == nil {
		currentOffset = info.Size()
	}

	log.Printf("Transfer %s: current offset is %d", transferID, currentOffset)

	w.Header().Set("Upload-Offset", strconv.FormatInt(currentOffset, 10))
	w.WriteHeader(http.StatusOK)
}

func health(w http.ResponseWriter, req *http.Request) {
	setCORSHeaders(w)
	
	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

func main() {
	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	http.HandleFunc("/upload", upload)
	http.HandleFunc("/health", health)

	log.Println("Server starting on :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
