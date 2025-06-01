package attachment

import (
	"strconv"
	"strings"
)

// UID is the uid of the file, it is the unique identifier of the file
func (fileheader *FileHeader) UID() string {
	return fileheader.Header.Get("Content-Uid")
}

// Fingerprint is the fingerprint of the file, it is the fingerprint of the file
func (fileheader *FileHeader) Fingerprint() string {
	return fileheader.Header.Get("Content-Fingerprint")
}

// Range is the range of the file, it is the start and end of the file
func (fileheader *FileHeader) Range() string {
	return fileheader.Header.Get("Content-Range")
}

// Sync is the sync of the file, it is the sync of the file
func (fileheader *FileHeader) Sync() bool {
	return fileheader.Header.Get("Content-Sync") == "true"
}

// IsChunk is the chunk of the file, it is the chunk of the file
func (fileheader *FileHeader) IsChunk() bool {
	return fileheader.Range() != ""
}

// Complete checks if the chunk upload is completed
// For non-chunk files, it returns true
// For chunk files, it parses the Content-Range header to determine if this is the last chunk
func (fileheader *FileHeader) Complete() bool {
	if !fileheader.IsChunk() {
		return true
	}

	// Parse Content-Range header: "bytes start-end/total"
	rangeHeader := fileheader.Range()
	if rangeHeader == "" {
		return false
	}

	// Remove "bytes " prefix
	rangeStr := strings.TrimPrefix(rangeHeader, "bytes ")

	// Split by "/"
	parts := strings.Split(rangeStr, "/")
	if len(parts) != 2 {
		return false
	}

	// Parse total size
	total, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}

	// Parse range "start-end"
	rangeParts := strings.Split(parts[0], "-")
	if len(rangeParts) != 2 {
		return false
	}

	end, err := strconv.ParseInt(rangeParts[1], 10, 64)
	if err != nil {
		return false
	}

	// Check if this is the last chunk: end + 1 == total
	return end+1 == total
}

// GetChunkInfo returns the chunk information parsed from Content-Range header
// Returns start, end, total, and error
func (fileheader *FileHeader) GetChunkInfo() (start, end, total int64, err error) {
	if !fileheader.IsChunk() {
		return 0, 0, 0, nil
	}

	rangeHeader := fileheader.Range()
	if rangeHeader == "" {
		return 0, 0, 0, nil
	}

	// Remove "bytes " prefix
	rangeStr := strings.TrimPrefix(rangeHeader, "bytes ")

	// Split by "/"
	parts := strings.Split(rangeStr, "/")
	if len(parts) != 2 {
		return 0, 0, 0, nil
	}

	// Parse total size
	total, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	// Parse range "start-end"
	rangeParts := strings.Split(parts[0], "-")
	if len(rangeParts) != 2 {
		return 0, 0, 0, nil
	}

	start, err = strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	end, err = strconv.ParseInt(rangeParts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	return start, end, total, nil
}

// GetTotalSize returns the total file size from Content-Range header
func (fileheader *FileHeader) GetTotalSize() int64 {
	_, _, total, err := fileheader.GetChunkInfo()
	if err != nil {
		return 0
	}
	return total
}

// GetChunkSize returns the current chunk size
func (fileheader *FileHeader) GetChunkSize() int64 {
	start, end, _, err := fileheader.GetChunkInfo()
	if err != nil {
		return 0
	}
	return end - start + 1
}
