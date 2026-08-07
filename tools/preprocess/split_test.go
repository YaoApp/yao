package preprocess

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func makeWAV(durationSec float64, sampleRate, channels, bitsPerSample int) []byte {
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	numSamples := int(float64(sampleRate) * durationSec)
	dataSize := numSamples * blockAlign

	buf := make([]byte, 44+dataSize)

	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(buf[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(buf[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(buf[34:36], uint16(bitsPerSample))
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))

	return buf
}

func TestSplitAudio_NonWAV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(path, []byte("fake mp3"), 0644); err != nil {
		t.Fatal(err)
	}

	paths, cleanup, err := SplitAudio(path, 30)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if len(paths) != 1 || paths[0] != path {
		t.Errorf("non-WAV should return original path, got %v", paths)
	}
}

func TestSplitAudio_WAV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audio.wav")
	wavData := makeWAV(5.0, 44100, 1, 16)
	if err := os.WriteFile(path, wavData, 0644); err != nil {
		t.Fatal(err)
	}

	paths, cleanup, err := SplitAudio(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if len(paths) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(paths))
	}

	for i, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("chunk %d not found: %v", i, err)
		}
		if info.Size() < 44 {
			t.Errorf("chunk %d too small: %d bytes", i, info.Size())
		}
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
			t.Errorf("chunk %d is not a valid WAV file", i)
		}
	}
}

func TestSplitAudio_ZeroChunkSec(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audio.wav")
	if err := os.WriteFile(path, makeWAV(1.0, 8000, 1, 16), 0644); err != nil {
		t.Fatal(err)
	}

	paths, cleanup, err := SplitAudio(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if len(paths) != 1 || paths[0] != path {
		t.Errorf("chunkSec=0 should return original path, got %v", paths)
	}
}

func TestParseWAV(t *testing.T) {
	data := makeWAV(1.0, 44100, 2, 16)
	headerSize, byteRate, audioData, err := parseWAV(data)
	if err != nil {
		t.Fatal(err)
	}
	if headerSize != 44 {
		t.Errorf("headerSize = %d, want 44", headerSize)
	}
	if byteRate != 44100*2*16/8 {
		t.Errorf("byteRate = %d, want %d", byteRate, 44100*2*16/8)
	}
	expectedDataSize := 44100 * 2 * 16 / 8
	if len(audioData) != expectedDataSize {
		t.Errorf("audioData len = %d, want %d", len(audioData), expectedDataSize)
	}
}
