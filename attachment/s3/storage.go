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
	return storage, nil
}

// Upload upload file to S3
func (storage *Storage) Upload(ctx context.Context, fileID string, reader io.Reader, contentType string) (string, error) {
	if storage.client == nil {
		return "", fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, fileID)

	// Upload file
	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(storage.Bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return fileID, nil
}

// UploadChunk uploads a chunk of a file to S3
func (storage *Storage) UploadChunk(ctx context.Context, fileID string, chunkIndex int, reader io.Reader, contentType string) error {
	if storage.client == nil {
		return fmt.Errorf("s3 client not initialized")
	}

	// Store chunks with a special prefix
	chunkKey := filepath.Join(storage.prefix, ".chunks", fileID, fmt.Sprintf("chunk_%d", chunkIndex))

	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(storage.Bucket),
		Key:         aws.String(chunkKey),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload chunk %d: %w", chunkIndex, err)
	}

	return nil
}

// MergeChunks merges all chunks into the final file in S3
func (storage *Storage) MergeChunks(ctx context.Context, fileID string, totalChunks int) error {
	if storage.client == nil {
		return fmt.Errorf("s3 client not initialized")
	}

	finalKey := filepath.Join(storage.prefix, fileID)

	// Create a buffer to hold the merged content
	var mergedContent bytes.Buffer
	var contentType string

	// Download and merge chunks in order
	for i := 0; i < totalChunks; i++ {
		chunkKey := filepath.Join(storage.prefix, ".chunks", fileID, fmt.Sprintf("chunk_%d", i))

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
			return fmt.Errorf("failed to copy chunk %d: %w", i, err)
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
		return fmt.Errorf("failed to upload merged file: %w", err)
	}

	// Clean up chunks
	for i := 0; i < totalChunks; i++ {
		chunkKey := filepath.Join(storage.prefix, ".chunks", fileID, fmt.Sprintf("chunk_%d", i))
		storage.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(storage.Bucket),
			Key:    aws.String(chunkKey),
		})
	}

	return nil
}

// Reader read file from S3
func (storage *Storage) Reader(ctx context.Context, fileID string) (io.ReadCloser, error) {
	if storage.client == nil {
		return nil, fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, fileID)

	result, err := storage.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	// If the file is a gzip file, decompress it
	if strings.HasSuffix(fileID, ".gz") {
		reader, err := gzip.NewReader(result.Body)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	return result.Body, nil
}

// Download download file from S3
func (storage *Storage) Download(ctx context.Context, fileID string) (io.ReadCloser, string, error) {
	if storage.client == nil {
		return nil, "", fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, fileID)

	// Get object
	result, err := storage.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %w", err)
	}

	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	// Try to detect content type from file extension
	ext := filepath.Ext(strings.TrimSuffix(fileID, ".gz"))
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
	if strings.HasSuffix(fileID, ".gz") {
		reader, err := gzip.NewReader(result.Body)
		if err != nil {
			return nil, "", err
		}
		return reader, contentType, nil
	}

	return result.Body, contentType, nil
}

// URL get file url with expiration
func (storage *Storage) URL(ctx context.Context, fileID string) string {
	if storage.client == nil {
		return ""
	}

	key := filepath.Join(storage.prefix, fileID)
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
func (storage *Storage) Exists(ctx context.Context, fileID string) bool {
	if storage.client == nil {
		return false
	}

	key := filepath.Join(storage.prefix, fileID)
	_, err := storage.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

// Delete deletes a file from S3
func (storage *Storage) Delete(ctx context.Context, fileID string) error {
	if storage.client == nil {
		return fmt.Errorf("s3 client not initialized")
	}

	key := filepath.Join(storage.prefix, fileID)
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
