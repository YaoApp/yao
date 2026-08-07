package preprocess

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SplitAudio splits a WAV file into chunks of chunkSec seconds.
// For non-WAV formats, returns the original path as a single-element slice.
// Returns (chunkPaths, cleanupFunc, error) where cleanupFunc removes the temp directory.
func SplitAudio(localPath string, chunkSec int) ([]string, func(), error) {
	noop := func() {}

	if chunkSec <= 0 {
		return []string{localPath}, noop, nil
	}

	ext := strings.ToLower(filepath.Ext(localPath))
	if ext != ".wav" {
		return []string{localPath}, noop, nil
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, noop, err
	}

	headerSize, byteRate, audioData, err := parseWAV(data)
	if err != nil {
		return nil, noop, fmt.Errorf("parse WAV: %w", err)
	}

	if len(audioData) == 0 {
		return []string{localPath}, noop, nil
	}

	bytesPerChunk := byteRate * chunkSec
	if bytesPerChunk <= 0 {
		return []string{localPath}, noop, nil
	}

	tmpDir, err := os.MkdirTemp("", "audio-split-*")
	if err != nil {
		return nil, noop, err
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	header := data[:headerSize]
	var paths []string
	chunkIdx := 0

	for offset := 0; offset < len(audioData); offset += bytesPerChunk {
		end := offset + bytesPerChunk
		if end > len(audioData) {
			end = len(audioData)
		}
		chunk := audioData[offset:end]
		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%03d.wav", chunkIdx))
		if err := writeWAVChunk(chunkPath, header, chunk); err != nil {
			cleanup()
			return nil, noop, err
		}
		paths = append(paths, chunkPath)
		chunkIdx++
	}

	if len(paths) == 0 {
		cleanup()
		return []string{localPath}, noop, nil
	}
	return paths, cleanup, nil
}

// parseWAV extracts the header size, byte rate, and raw audio data from a WAV file.
func parseWAV(data []byte) (headerSize int, byteRate int, audioData []byte, err error) {
	if len(data) < 44 {
		return 0, 0, nil, fmt.Errorf("file too short")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return 0, 0, nil, fmt.Errorf("not a RIFF/WAVE file")
	}

	offset := 12
	var fmtData []byte

	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkStart := offset + 8
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(data) {
			chunkEnd = len(data)
		}

		switch chunkID {
		case "fmt ":
			fmtData = data[chunkStart:chunkEnd]
		case "data":
			if len(fmtData) >= 12 {
				byteRate = int(binary.LittleEndian.Uint32(fmtData[8:12]))
			}
			return chunkStart, byteRate, data[chunkStart:chunkEnd], nil
		}

		offset = chunkStart + chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}

	return 0, 0, nil, fmt.Errorf("data chunk not found")
}

// writeWAVChunk writes a valid WAV file with the given fmt header and audio data.
func writeWAVChunk(path string, originalHeader []byte, audioData []byte) error {
	// Rebuild header up to (but not including) the original data chunk.
	headerSize, _, _, err := parseWAV(originalHeader)
	if err != nil {
		return err
	}

	// Copy everything before the data chunk payload.
	prefix := make([]byte, headerSize)
	copy(prefix, originalHeader[:headerSize])

	// Patch RIFF chunk size: total file size - 8.
	binary.LittleEndian.PutUint32(prefix[4:8], uint32(len(prefix)+len(audioData)-8))

	// Find and patch data chunk size within prefix.
	offset := 12
	for offset+8 <= len(prefix) {
		chunkID := string(prefix[offset : offset+4])
		if chunkID == "data" {
			binary.LittleEndian.PutUint32(prefix[offset+4:offset+8], uint32(len(audioData)))
			break
		}
		chunkSize := int(binary.LittleEndian.Uint32(prefix[offset+4 : offset+8]))
		offset += 8 + chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(prefix); err != nil {
		return err
	}
	_, err = f.Write(audioData)
	return err
}
