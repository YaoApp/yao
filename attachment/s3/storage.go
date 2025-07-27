package s3

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// DefaultExpiration default expiration time for presigned URLs (5 minutes)
const DefaultExpiration = 5 * time.Minute

// MaxImageSize maximum image size (1920x1080)
const MaxImageSize = 1920

// Storage the S3 storage driver
type Storage struct {
	Endpoint    string        `json:"endpoint" yaml:"endpoint"`
	Region      string        `json:"region" yaml:"region"`
	Key         string        `json:"key" yaml:"key"`
	Secret      string        `json:"secret" yaml:"secret"`
	Bucket      string        `json:"bucket" yaml:"bucket"`
	Expiration  time.Duration `json:"expiration" yaml:"expiration"`
	CacheDir    string        `json:"cache_dir" yaml:"cache_dir"`
	client      *s3.Client
	prefix      string
	compression bool
}

// New create a new S3 storage
func New(options map[string]interface{}) (*Storage, error) {
	storage := &Storage{
		Region:      "auto",
		Expiration:  DefaultExpiration,
		compression: true,
	}

	if endpoint, ok := options["endpoint"].(string); ok {
		storage.Endpoint = endpoint
	}

	if region, ok := options["region"].(string); ok {
		storage.Region = region
	}

	if key, ok := options["key"].(string); ok {
		storage.Key = key
	}

	if secret, ok := options["secret"].(string); ok {
		storage.Secret = secret
	}

	if bucket, ok := options["bucket"].(string); ok {
		storage.Bucket = bucket
	}

	if prefix, ok := options["prefix"].(string); ok {
		storage.prefix = prefix
	}

	if cacheDir, ok := options["cache_dir"].(string); ok {
		storage.CacheDir = cacheDir
	} else {
		// Use system temp directory as default
		storage.CacheDir = os.TempDir()
	}

	if exp, ok := options["expiration"].(time.Duration); ok {
		storage.Expiration = exp
	}

	if compression, ok := options["compression"].(bool); ok {
		storage.compression = compression
	}

	// Validate required fields
	if storage.Key == "" || storage.Secret == "" {
		return nil, fmt.Errorf("key and secret are required")
	}

	if storage.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	// Create S3 client
	opts := s3.Options{
		Region:       storage.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(storage.Key, storage.Secret, ""),
		UsePathStyle: true,
	}

	if storage.Endpoint != "" {
		// Remove bucket name from endpoint if present
		endpoint := storage.Endpoint
		if strings.Contains(endpoint, "/"+storage.Bucket) {
			endpoint = strings.TrimSuffix(endpoint, "/"+storage.Bucket)
		}
		opts.BaseEndpoint = aws.String(endpoint)
	}

	storage.client = s3.New(opts)

	// Ensure cache directory exists
	if err := os.MkdirAll(storage.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", storage.CacheDir, err)
	}

	return storage, nil
}

// Upload upload file to S3
func (storage *Storage) Upload(ctx context.Context, path string, reader io.Reader, contentType string) (string, error) {
	if storage.client == nil {
		return "", fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, path)

	// Upload file
	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(storage.Bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file %s: %w", path, err)
	}

	return path, nil
}

// UploadChunk uploads a chunk of a file to S3
func (storage *Storage) UploadChunk(ctx context.Context, path string, chunkIndex int, reader io.Reader, contentType string) error {
	if storage.client == nil {
		return fmt.Errorf("s3 client not initialized")
	}

	// Store chunks with a special prefix
	chunkKey := filepath.Join(storage.prefix, ".chunks", path, fmt.Sprintf("chunk_%d", chunkIndex))

	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(storage.Bucket),
		Key:         aws.String(chunkKey),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload chunk %s %d: %w", path, chunkIndex, err)
	}

	return nil
}

// MergeChunks merges all chunks into the final file in S3
func (storage *Storage) MergeChunks(ctx context.Context, path string, totalChunks int) error {
	if storage.client == nil {
		return fmt.Errorf("s3 client not initialized")
	}

	finalKey := filepath.Join(storage.prefix, path)

	// Create a buffer to hold the merged content
	var mergedContent bytes.Buffer
	var contentType string

	// Download and merge chunks in order
	for i := 0; i < totalChunks; i++ {
		chunkKey := filepath.Join(storage.prefix, ".chunks", path, fmt.Sprintf("chunk_%d", i))

		result, err := storage.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(storage.Bucket),
			Key:    aws.String(chunkKey),
		})
		if err != nil {
			return fmt.Errorf("failed to get chunk %d: %w", i, err)
		}

		// Get content type from the first chunk
		if i == 0 && result.ContentType != nil {
			contentType = *result.ContentType
		}

		_, err = io.Copy(&mergedContent, result.Body)
		result.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to copy chunk %s %d: %w", path, i, err)
		}
	}

	// Default content type if not found
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload the merged content as the final file with proper content type
	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(storage.Bucket),
		Key:         aws.String(finalKey),
		Body:        bytes.NewReader(mergedContent.Bytes()),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload merged file %s: %w", path, err)
	}

	// Clean up chunks
	for i := 0; i < totalChunks; i++ {
		chunkKey := filepath.Join(storage.prefix, ".chunks", path, fmt.Sprintf("chunk_%d", i))
		storage.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(storage.Bucket),
			Key:    aws.String(chunkKey),
		})
	}

	return nil
}

// Reader read file from S3
func (storage *Storage) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	if storage.client == nil {
		return nil, fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, path)

	result, err := storage.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s: %w", path, err)
	}

	// If the file is a gzip file, decompress it
	if strings.HasSuffix(path, ".gz") {
		reader, err := gzip.NewReader(result.Body)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	return result.Body, nil
}

// Download download file from S3
func (storage *Storage) Download(ctx context.Context, path string) (io.ReadCloser, string, error) {
	if storage.client == nil {
		return nil, "", fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, path)

	// Get object
	result, err := storage.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file %s: %w", path, err)
	}

	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	// Try to detect content type from file extension
	ext := filepath.Ext(strings.TrimSuffix(path, ".gz"))
	switch strings.ToLower(ext) {
	case ".txt":
		contentType = "text/plain"
	case ".html":
		contentType = "text/html"
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".json":
		contentType = "application/json"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".pdf":
		contentType = "application/pdf"
	case ".mp4":
		contentType = "video/mp4"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".wav":
		contentType = "audio/wav"
	case ".ogg":
		contentType = "audio/ogg"
	case ".webm":
		contentType = "video/webm"
	case ".webp":
		contentType = "image/webp"
	case ".zip":
	}

	// If the file is a gzip file, decompress it
	if strings.HasSuffix(path, ".gz") {
		reader, err := gzip.NewReader(result.Body)
		if err != nil {
			return nil, "", err
		}
		return reader, contentType, nil
	}

	return result.Body, contentType, nil
}

// GetContent gets file content as bytes
func (storage *Storage) GetContent(ctx context.Context, path string) ([]byte, error) {
	reader, err := storage.Reader(ctx, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// URL get file url with expiration
func (storage *Storage) URL(ctx context.Context, path string) string {
	if storage.client == nil {
		return ""
	}

	key := filepath.Join(storage.prefix, path)
	presignClient := s3.NewPresignClient(storage.client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(storage.Expiration))

	if err != nil {
		return ""
	}

	return request.URL
}

// Exists checks if a file exists in S3
func (storage *Storage) Exists(ctx context.Context, path string) bool {
	if storage.client == nil {
		return false
	}

	key := filepath.Join(storage.prefix, path)
	_, err := storage.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

// Delete deletes a file from S3
func (storage *Storage) Delete(ctx context.Context, path string) error {
	if storage.client == nil {
		return fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, path)
	_, err := storage.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (storage *Storage) makeID(filename string, ext string) string {
	date := time.Now().Format("20060102")
	name := strings.TrimSuffix(filepath.Base(filename), ext)
	return fmt.Sprintf("%s/%s-%d%s", date, name, time.Now().UnixNano(), ext)
}

// isImage checks if the content type is an image
func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// compressImage compresses the image while maintaining aspect ratio
func compressImage(data []byte, contentType string) ([]byte, error) {
	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate new dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	var newWidth, newHeight int

	if width > height {
		if width > MaxImageSize {
			newWidth = MaxImageSize
			newHeight = int(float64(height) * (float64(MaxImageSize) / float64(width)))
		} else {
			return data, nil // No need to resize
		}
	} else {
		if height > MaxImageSize {
			newHeight = MaxImageSize
			newWidth = int(float64(width) * (float64(MaxImageSize) / float64(height)))
		} else {
			return data, nil // No need to resize
		}
	}

	// Create new image with new dimensions
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Scale the image using bilinear interpolation
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := float64(x) * float64(width) / float64(newWidth)
			srcY := float64(y) * float64(height) / float64(newHeight)
			newImg.Set(x, y, img.At(int(srcX), int(srcY)))
		}
	}

	// Encode image
	var buf bytes.Buffer
	switch contentType {
	case "image/jpeg":
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 85})
	case "image/png":
		err = png.Encode(&buf, newImg)
	default:
		return data, nil // Unsupported format, return original
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// LocalPath downloads the file to cache directory and returns absolute path with content type
func (storage *Storage) LocalPath(ctx context.Context, path string) (string, string, error) {
	if storage.client == nil {
		return "", "", fmt.Errorf("s3 client not initialized")
	}

	// Create cache file path using the same structure as storage path
	cacheFilePath := filepath.Join(storage.CacheDir, "s3_cache", path)

	// Create directory for cache file
	dir := filepath.Dir(cacheFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Check if file already exists in cache and is not outdated
	if _, err := os.Stat(cacheFilePath); err == nil {
		// File exists in cache, detect content type and return
		contentType, err := detectContentType(cacheFilePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to detect content type: %w", err)
		}
		return cacheFilePath, contentType, nil
	}

	// Download file from S3 to cache
	key := filepath.Join(storage.prefix, path)
	result, err := storage.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to download file %s: %w", path, err)
	}
	defer result.Body.Close()

	// Create cache file
	cacheFile, err := os.Create(cacheFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cache file: %w", err)
	}
	defer cacheFile.Close()

	// Handle gzipped files - decompress during download
	var reader io.Reader = result.Body
	if strings.HasSuffix(path, ".gz") {
		gzipReader, err := gzip.NewReader(result.Body)
		if err != nil {
			return "", "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader

		// Remove .gz extension from cache file path since we're decompressing
		newCacheFilePath := strings.TrimSuffix(cacheFilePath, ".gz")
		cacheFile.Close()
		os.Remove(cacheFilePath)

		cacheFile, err = os.Create(newCacheFilePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to create decompressed cache file: %w", err)
		}
		defer cacheFile.Close()
		cacheFilePath = newCacheFilePath
	}

	// Copy file content to cache
	_, err = io.Copy(cacheFile, reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to copy file to cache: %w", err)
	}

	// For files that were decompressed from .gz, we need to detect the original content type
	var contentType string
	if strings.HasSuffix(path, ".gz") {
		// Original path was gzipped, detect content type of decompressed content
		originalPath := strings.TrimSuffix(path, ".gz")
		ext := filepath.Ext(originalPath)

		// First try to detect by original file extension
		contentType, err = detectContentTypeFromExtension(ext)
		if err != nil || contentType == "application/octet-stream" {
			// Fallback: detect from decompressed content
			contentType, err = detectContentType(cacheFilePath)
			if err != nil {
				return "", "", fmt.Errorf("failed to detect content type: %w", err)
			}
		}
	} else {
		// Regular file content type detection
		contentType, err = detectContentType(cacheFilePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to detect content type: %w", err)
		}
	}

	return cacheFilePath, contentType, nil
}

// detectContentType detects content type based on file extension and content
func detectContentType(filePath string) (string, error) {
	// First try to detect by file extension
	ext := strings.ToLower(filepath.Ext(filePath))

	// Common file extensions mapping
	switch ext {
	case ".txt":
		return "text/plain", nil
	case ".html", ".htm":
		return "text/html", nil
	case ".css":
		return "text/css", nil
	case ".js":
		return "application/javascript", nil
	case ".json":
		return "application/json", nil
	case ".xml":
		return "application/xml", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	case ".svg":
		return "image/svg+xml", nil
	case ".pdf":
		return "application/pdf", nil
	case ".doc":
		return "application/msword", nil
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
	case ".xls":
		return "application/vnd.ms-excel", nil
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	case ".ppt":
		return "application/vnd.ms-powerpoint", nil
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation", nil
	case ".zip":
		return "application/zip", nil
	case ".tar":
		return "application/x-tar", nil
	case ".gz":
		return "application/gzip", nil
	case ".mp3":
		return "audio/mpeg", nil
	case ".wav":
		return "audio/wav", nil
	case ".m4a":
		return "audio/mp4", nil
	case ".ogg":
		return "audio/ogg", nil
	case ".mp4":
		return "video/mp4", nil
	case ".avi":
		return "video/x-msvideo", nil
	case ".mov":
		return "video/quicktime", nil
	case ".webm":
		return "video/webm", nil
	case ".md", ".mdx":
		return "text/markdown", nil
	case ".yao":
		return "application/yao", nil
	case ".csv":
		return "text/csv", nil
	}

	// Try to detect by MIME package
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType, nil
	}

	// Fallback: detect by reading file content
	file, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream", nil // Default fallback
	}
	defer file.Close()

	// Read first 512 bytes for content detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "application/octet-stream", nil
	}

	// Use http.DetectContentType to detect based on content
	contentType := http.DetectContentType(buffer[:n])
	return contentType, nil
}

// detectContentTypeFromExtension detects content type based only on file extension
func detectContentTypeFromExtension(ext string) (string, error) {
	ext = strings.ToLower(ext)

	// Common file extensions mapping
	switch ext {
	case ".txt":
		return "text/plain", nil
	case ".html", ".htm":
		return "text/html", nil
	case ".css":
		return "text/css", nil
	case ".js":
		return "application/javascript", nil
	case ".json":
		return "application/json", nil
	case ".xml":
		return "application/xml", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	case ".svg":
		return "image/svg+xml", nil
	case ".pdf":
		return "application/pdf", nil
	case ".doc":
		return "application/msword", nil
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
	case ".xls":
		return "application/vnd.ms-excel", nil
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	case ".ppt":
		return "application/vnd.ms-powerpoint", nil
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation", nil
	case ".zip":
		return "application/zip", nil
	case ".tar":
		return "application/x-tar", nil
	case ".mp3":
		return "audio/mpeg", nil
	case ".wav":
		return "audio/wav", nil
	case ".m4a":
		return "audio/mp4", nil
	case ".ogg":
		return "audio/ogg", nil
	case ".mp4":
		return "video/mp4", nil
	case ".avi":
		return "video/x-msvideo", nil
	case ".mov":
		return "video/quicktime", nil
	case ".webm":
		return "video/webm", nil
	case ".md", ".mdx":
		return "text/markdown", nil
	case ".yao":
		return "application/yao", nil
	case ".csv":
		return "text/csv", nil
	}

	// Try to detect by MIME package
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType, nil
	}

	// Return default if not found
	return "application/octet-stream", nil
}
