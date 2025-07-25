package attachment

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// GzipCompressor supports chunked compression for Gzip
type GzipCompressor struct {
	writer *gzip.Writer
	buffer *bytes.Buffer
	file   *os.File // optional file handle
}

// NewGzipCompressor creates a new Gzip compressor
func NewGzipCompressor() *GzipCompressor {
	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	return &GzipCompressor{
		writer: gz,
		buffer: buf,
		file:   nil,
	}
}

// NewGzipCompressorFromFile creates a Gzip compressor from file, supports streaming read
func NewGzipCompressorFromFile(filePath string) (*GzipCompressor, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	return &GzipCompressor{
		writer: gz,
		buffer: buf,
		file:   file,
	}, nil
}

// ReadChunk reads a chunk of specified size from file and compresses it
func (gc *GzipCompressor) ReadChunk(chunkSize int) (bool, error) {
	if gc.file == nil {
		return false, fmt.Errorf("no file associated with this compressor")
	}

	chunk := make([]byte, chunkSize)
	n, err := gc.file.Read(chunk)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read from file: %w", err)
	}

	if n > 0 {
		if err := gc.Write(chunk[:n]); err != nil {
			return false, err
		}
	}

	// return whether there is more data
	return err != io.EOF, nil
}

// CompressFileInChunks compresses the entire file in chunks
func (gc *GzipCompressor) CompressFileInChunks(chunkSize int) error {
	if gc.file == nil {
		return fmt.Errorf("no file associated with this compressor")
	}

	for {
		hasMore, err := gc.ReadChunk(chunkSize)
		if err != nil {
			return err
		}
		if !hasMore {
			break
		}
	}
	return nil
}

// Write writes data for compression (supports chunked writing)
func (gc *GzipCompressor) Write(data []byte) error {
	_, err := gc.writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data to gzip: %w", err)
	}
	return nil
}

// Flush flushes the buffer but does not close the compressor
func (gc *GzipCompressor) Flush() error {
	return gc.writer.Flush()
}

// Close closes the compressor and returns the final compressed data
func (gc *GzipCompressor) Close() ([]byte, error) {
	err := gc.writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// if there is an associated file, close it too
	if gc.file != nil {
		gc.file.Close()
		gc.file = nil
	}

	return gc.buffer.Bytes(), nil
}

// GetCompressedData gets the current compressed data (without closing the compressor)
func (gc *GzipCompressor) GetCompressedData() []byte {
	// flush the buffer first
	gc.writer.Flush()
	return gc.buffer.Bytes()
}

// Reset resets the compressor for reuse
func (gc *GzipCompressor) Reset() {
	gc.buffer.Reset()
	gc.writer.Reset(gc.buffer)
	if gc.file != nil {
		gc.file.Close()
		gc.file = nil
	}
}

// Gzip compresses data in one go
func Gzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to gzip data: %w", err)
	}
	err = gz.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return buf.Bytes(), nil
}

// GzipChunks compresses multiple data chunks
func GzipChunks(chunks [][]byte) ([]byte, error) {
	compressor := NewGzipCompressor()

	for _, chunk := range chunks {
		if err := compressor.Write(chunk); err != nil {
			return nil, err
		}
	}

	return compressor.Close()
}

// GzipFromReader compresses data from Reader stream
func GzipFromReader(reader io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := io.Copy(gz, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to copy data to gzip: %w", err)
	}

	err = gz.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Gunzip decompresses gzip data
func Gunzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return buf.Bytes(), nil
}

// GzipToWriter writes compressed data to Writer
func GzipToWriter(data []byte, writer io.Writer) error {
	gz := gzip.NewWriter(writer)
	defer gz.Close()

	_, err := gz.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write gzip data: %w", err)
	}

	return nil
}

// GzipFromReaderToWriter reads data from Reader and writes compressed data to Writer
func GzipFromReaderToWriter(reader io.Reader, writer io.Writer) error {
	gz := gzip.NewWriter(writer)
	defer gz.Close()

	_, err := io.Copy(gz, reader)
	if err != nil {
		return fmt.Errorf("failed to copy and compress data: %w", err)
	}

	return nil
}

// GzipFile compresses entire file (loads into memory at once)
func GzipFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return Gzip(data)
}

// GzipFileInChunks compresses file in chunks (memory friendly)
func GzipFileInChunks(filePath string, chunkSize int) ([]byte, error) {
	compressor, err := NewGzipCompressorFromFile(filePath)
	if err != nil {
		return nil, err
	}
	defer compressor.Close()

	err = compressor.CompressFileInChunks(chunkSize)
	if err != nil {
		return nil, err
	}

	return compressor.Close()
}

// GzipFileToFile compresses file and saves to another file
func GzipFileToFile(srcPath, dstPath string, chunkSize int) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	return GzipFromReaderToWriter(srcFile, dstFile)
}

// GzipFileStream compresses file in streaming mode, returns Reader interface
func GzipFileStream(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		defer file.Close()

		gz := gzip.NewWriter(pw)
		defer gz.Close()

		_, err := io.Copy(gz, file)
		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, nil
}
