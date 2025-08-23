package attachment

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"strings"

	"github.com/yaoapp/yao/attachment/s3"
)

// ExampleUsage demonstrates how to use the attachment package
func ExampleUsage() {
	// 1. Create a local storage manager
	localManager, err := New(ManagerOption{
		Driver:       "local",
		MaxSize:      "20M",
		ChunkSize:    "2M",
		AllowedTypes: []string{"text/*", "image/*", "application/pdf", ".txt", ".jpg", ".png", ".pdf"},
		Options: map[string]interface{}{
			"path":     "/var/uploads",
			"base_url": "https://example.com/files",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create local manager: %v", err)
	}

	// 2. Create an S3 storage manager
	s3Manager, err := New(ManagerOption{
		Driver:       "s3",
		MaxSize:      "100M",
		ChunkSize:    "5M",
		AllowedTypes: []string{"*"}, // Allow all types
		Options: map[string]interface{}{
			"endpoint": "https://s3.amazonaws.com",
			"region":   "us-east-1",
			"key":      "your-access-key",
			"secret":   "your-secret-key",
			"bucket":   "your-bucket-name",
			"prefix":   "attachments/",
		},
	})
	if err != nil {
		log.Printf("Failed to create S3 manager (this is expected without credentials): %v", err)
	} else {
		fmt.Printf("Created S3 manager successfully\n")
		// Demonstrate S3 manager usage if credentials are available
		if s3Manager != nil {
			fmt.Printf("S3 manager is ready for use with bucket: %s\n",
				s3Manager.storage.(*s3.Storage).Bucket)
		}
	}

	// 3. Register managers globally
	_, err = Register("local", "local", ManagerOption{
		Driver:       "local",
		MaxSize:      "20M",
		AllowedTypes: []string{"text/*", "image/*"},
		Options: map[string]interface{}{
			"path": "/var/uploads",
		},
	})
	if err != nil {
		log.Printf("Failed to register local manager: %v", err)
	}

	// Try to register S3 manager (will fail without credentials)
	_, err = Register("s3", "s3", ManagerOption{
		Driver:  "s3",
		MaxSize: "100M",
		Options: map[string]interface{}{
			"bucket": "your-bucket",
			"key":    "your-key",
			"secret": "your-secret",
		},
	})
	if err != nil {
		log.Printf("Failed to register S3 manager (expected without credentials): %v", err)
	}

	ctx := context.Background()

	// 4. Example: Simple file upload
	content := "Hello, World! This is a test file with some content to demonstrate the attachment package."
	fileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "hello.txt",
			Size:     int64(len(content)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	uploadOption := UploadOption{
		Groups: []string{"user123"},
		Gzip:   false, // No compression for small text files
	}

	file, err := localManager.Upload(ctx, fileHeader, strings.NewReader(content), uploadOption)
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}

	fmt.Printf("Uploaded file: %s (ID: %s, Size: %d bytes)\n", file.Filename, file.ID, file.Bytes)

	// 5. Example: File upload with gzip compression
	largeContent := strings.Repeat("This is a large text file that benefits from compression. ", 100)
	gzipFileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "large_text.txt",
			Size:     int64(len(largeContent)),
			Header:   make(map[string][]string),
		},
	}
	gzipFileHeader.Header.Set("Content-Type", "text/plain")

	gzipOption := UploadOption{
		Groups: []string{"user123"},
		Gzip:   true, // Enable compression
	}

	gzipFile, err := localManager.Upload(ctx, gzipFileHeader, strings.NewReader(largeContent), gzipOption)
	if err != nil {
		log.Fatalf("Failed to upload gzipped file: %v", err)
	}

	fmt.Printf("Uploaded compressed file: %s (ID: %s)\n", gzipFile.Filename, gzipFile.ID)

	// 6. Example: Image upload with compression
	imageUploadOption := UploadOption{
		Groups:        []string{"user123"},
		CompressImage: true,
		CompressSize:  1920, // Resize to max 1920px
		Gzip:          false,
	}

	// Simulate image upload (you would get this from multipart form)
	imageHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "photo.jpg",
			Size:     1024000, // 1MB
			Header:   make(map[string][]string),
		},
	}
	imageHeader.Header.Set("Content-Type", "image/jpeg")

	fmt.Printf("Image upload option configured: compress=%v, size=%d\n",
		imageUploadOption.CompressImage, imageUploadOption.CompressSize)

	// 6.5. Example: Multi-level groups
	fmt.Println("\n--- Multi-level Groups Examples ---")

	// Single level grouping
	singleGroupOption := UploadOption{
		Groups: []string{"knowledge"},
	}

	singleGroupHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "knowledge_doc.txt",
			Size:     int64(len("Knowledge base document")),
			Header:   make(map[string][]string),
		},
	}
	singleGroupHeader.Header.Set("Content-Type", "text/plain")

	singleFile, err := localManager.Upload(ctx, singleGroupHeader, strings.NewReader("Knowledge base document"), singleGroupOption)
	if err != nil {
		log.Printf("Failed to upload single group file: %v", err)
	} else {
		fmt.Printf("Single group file uploaded: %s (ID: %s)\n", singleFile.Filename, singleFile.ID)
	}

	// Multi-level grouping
	multiGroupOption := UploadOption{
		Groups: []string{"users", "user123", "chats", "chat456", "documents"},
	}

	multiGroupHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "chat_document.txt",
			Size:     int64(len("Document in user chat")),
			Header:   make(map[string][]string),
		},
	}
	multiGroupHeader.Header.Set("Content-Type", "text/plain")

	multiFile, err := localManager.Upload(ctx, multiGroupHeader, strings.NewReader("Document in user chat"), multiGroupOption)
	if err != nil {
		log.Printf("Failed to upload multi-group file: %v", err)
	} else {
		fmt.Printf("Multi-level group file uploaded: %s (ID: %s)\n", multiFile.Filename, multiFile.ID)
		fmt.Printf("File path includes hierarchy: users/user123/chats/chat456/documents\n")
	}

	// Knowledge base organization
	knowledgeOption := UploadOption{
		Groups: []string{"knowledge", "technical", "api", "documentation"},
	}

	knowledgeHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "api_guide.md",
			Size:     int64(len("# API Documentation\n\nThis is technical documentation.")),
			Header:   make(map[string][]string),
		},
	}
	knowledgeHeader.Header.Set("Content-Type", "text/markdown")

	knowledgeFile, err := localManager.Upload(ctx, knowledgeHeader,
		strings.NewReader("# API Documentation\n\nThis is technical documentation."), knowledgeOption)
	if err != nil {
		log.Printf("Failed to upload knowledge file: %v", err)
	} else {
		fmt.Printf("Knowledge base file uploaded: %s (ID: %s)\n", knowledgeFile.Filename, knowledgeFile.ID)
		fmt.Printf("Organized in: knowledge/technical/api/documentation\n")
	}

	// 7. Example: Chunked upload
	largeContent = strings.Repeat("This is a large file content that will be uploaded in chunks. ", 1000)
	chunkSize := 1024
	totalSize := len(largeContent)
	uid := "unique-large-file-123"

	fmt.Printf("Starting chunked upload: total size=%d, chunk size=%d\n", totalSize, chunkSize)

	var lastFile *File
	chunkCount := 0

	// Split into chunks and upload
	for i := 0; i < totalSize; i += chunkSize {
		end := i + chunkSize
		if end > totalSize {
			end = totalSize
		}
		chunk := largeContent[i:end]

		chunkHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "large_file.txt",
				Size:     int64(len(chunk)),
				Header:   make(map[string][]string),
			},
		}
		chunkHeader.Header.Set("Content-Type", "text/plain")
		chunkHeader.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", i, end-1, totalSize))
		chunkHeader.Header.Set("Content-Uid", uid)

		chunkOption := UploadOption{
			Groups: []string{"user123"},
			Gzip:   true, // Compress chunks
		}

		chunkFile, err := localManager.Upload(ctx, chunkHeader, strings.NewReader(chunk), chunkOption)
		if err != nil {
			log.Fatalf("Failed to upload chunk %d: %v", chunkCount, err)
		}

		chunkCount++
		lastFile = chunkFile

		// Check if this is the last chunk
		if chunkHeader.Complete() {
			fmt.Printf("Uploaded large file in %d chunks: %s (ID: %s)\n", chunkCount, chunkFile.Filename, chunkFile.ID)
			break
		}
	}

	// 8. Example: Download and read files
	if file != nil {
		// Download as stream
		response, err := localManager.Download(ctx, file.ID)
		if err != nil {
			log.Fatalf("Failed to download file: %v", err)
		}
		defer response.Reader.Close()

		fmt.Printf("Downloaded file content type: %s, extension: %s\n", response.ContentType, response.Extension)

		// Read as bytes
		data, err := localManager.Read(ctx, file.ID)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}

		fmt.Printf("File content length: %d bytes\n", len(data))
		if len(data) < 100 {
			fmt.Printf("File content: %s\n", string(data))
		} else {
			fmt.Printf("File content preview: %s...\n", string(data[:100]))
		}

		// Read as base64
		base64Data, err := localManager.ReadBase64(ctx, file.ID)
		if err != nil {
			log.Fatalf("Failed to read file as base64: %v", err)
		}

		fmt.Printf("File as base64 (first 50 chars): %s...\n", base64Data[:min(50, len(base64Data))])
	}

	// 9. Example: Read chunked file
	if lastFile != nil {
		chunkData, err := localManager.Read(ctx, lastFile.ID)
		if err != nil {
			log.Fatalf("Failed to read chunked file: %v", err)
		}

		// Since the chunks were compressed, we need to decompress
		decompressed, err := Gunzip(chunkData)
		if err != nil {
			log.Fatalf("Failed to decompress chunked file: %v", err)
		}

		fmt.Printf("Chunked file content length: %d bytes (decompressed)\n", len(decompressed))
		if len(decompressed) < 200 {
			fmt.Printf("Chunked file content: %s\n", string(decompressed))
		} else {
			fmt.Printf("Chunked file content preview: %s...\n", string(decompressed[:200]))
		}
	}

	// 10. Example: Using global managers
	globalManager := Managers["local"]
	if globalManager != nil {
		fmt.Println("Using global manager for local storage")

		// Test a simple upload with global manager
		testContent := "Test content using global manager"
		testHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "global_test.txt",
				Size:     int64(len(testContent)),
				Header:   make(map[string][]string),
			},
		}
		testHeader.Header.Set("Content-Type", "text/plain")

		testFile, err := globalManager.Upload(ctx, testHeader, strings.NewReader(testContent), UploadOption{
			Groups: []string{"global_user"},
		})
		if err != nil {
			log.Printf("Failed to upload with global manager: %v", err)
		} else {
			fmt.Printf("Global manager upload successful: %s\n", testFile.ID)
		}
	}

	// 11. Example: File validation
	fmt.Println("\n--- File Validation Examples ---")

	// Test file size validation
	tooLargeContent := strings.Repeat("x", 25*1024*1024) // 25MB, exceeds 20MB limit
	largeFileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "too_large.txt",
			Size:     int64(len(tooLargeContent)),
			Header:   make(map[string][]string),
		},
	}
	largeFileHeader.Header.Set("Content-Type", "text/plain")

	_, err = localManager.Upload(ctx, largeFileHeader, strings.NewReader(tooLargeContent), UploadOption{})
	if err != nil {
		fmt.Printf("Expected error for large file: %v\n", err)
	}

	// Test file type validation
	invalidFileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "script.exe",
			Size:     1024,
			Header:   make(map[string][]string),
		},
	}
	invalidFileHeader.Header.Set("Content-Type", "application/x-executable")

	_, err = localManager.Upload(ctx, invalidFileHeader, strings.NewReader("fake exe content"), UploadOption{})
	if err != nil {
		fmt.Printf("Expected error for invalid file type: %v\n", err)
	}

	fmt.Println("\n--- Example Usage Complete ---")
}

// ExampleChunkedUpload demonstrates how to handle chunked uploads properly
func ExampleChunkedUpload(manager *Manager, filename string, totalSize int64, contentType string) error {
	ctx := context.Background()
	chunkSize := int64(1024 * 1024) // 1MB chunks
	uid := "unique-file-" + filename

	fmt.Printf("Starting chunked upload: file=%s, total=%d bytes, chunks=%d\n",
		filename, totalSize, (totalSize+chunkSize-1)/chunkSize)

	for offset := int64(0); offset < totalSize; offset += chunkSize {
		end := offset + chunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}

		chunkSize := end - offset + 1

		// Create chunk header
		chunkHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: filename,
				Size:     chunkSize,
				Header:   make(map[string][]string),
			},
		}
		chunkHeader.Header.Set("Content-Type", contentType)
		chunkHeader.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, end, totalSize))
		chunkHeader.Header.Set("Content-Uid", uid)

		// In real usage, you would read the actual chunk data from the source
		chunkData := make([]byte, chunkSize)
		// Fill with sample data for demonstration
		for i := range chunkData {
			chunkData[i] = byte('A' + (i % 26))
		}

		option := UploadOption{
			Groups: []string{"user123"},
			Gzip:   false, // Disable compression for this example
		}

		file, err := manager.Upload(ctx, chunkHeader, bytes.NewReader(chunkData), option)
		if err != nil {
			return fmt.Errorf("failed to upload chunk at offset %d: %w", offset, err)
		}

		fmt.Printf("Uploaded chunk %d-%d/%d\n", offset, end, totalSize)

		// Check if this was the last chunk
		if chunkHeader.Complete() {
			fmt.Printf("File upload completed: %s (ID: %s)\n", file.Filename, file.ID)

			// Verify the uploaded file
			data, err := manager.Read(ctx, file.ID)
			if err != nil {
				return fmt.Errorf("failed to read uploaded file: %w", err)
			}

			if int64(len(data)) != totalSize {
				return fmt.Errorf("uploaded file size mismatch: expected %d, got %d", totalSize, len(data))
			}

			fmt.Printf("File verification successful: %d bytes\n", len(data))
			break
		}
	}

	return nil
}

// ExampleS3Upload demonstrates S3-specific features
func ExampleS3Upload() {
	// This example requires actual S3 credentials
	s3Manager, err := New(ManagerOption{
		Driver:  "s3",
		MaxSize: "50M",
		Options: map[string]interface{}{
			"endpoint": "https://s3.amazonaws.com",
			"region":   "us-east-1",
			"key":      "your-access-key",
			"secret":   "your-secret-key",
			"bucket":   "your-bucket",
			"prefix":   "test-uploads/",
		},
	})
	if err != nil {
		log.Printf("S3 manager creation failed (expected without credentials): %v", err)
		return
	}

	ctx := context.Background()
	content := "Test content for S3 upload"

	fileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "s3_test.txt",
			Size:     int64(len(content)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	file, err := s3Manager.Upload(ctx, fileHeader, strings.NewReader(content), UploadOption{
		Groups: []string{"s3_user"},
	})
	if err != nil {
		log.Printf("S3 upload failed: %v", err)
		return
	}

	fmt.Printf("S3 upload successful: %s\n", file.ID)

	// Get presigned URL
	url := s3Manager.storage.URL(ctx, file.ID)
	fmt.Printf("Presigned URL: %s\n", url)
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
