package shared

import "bufio"

// MaxLineSize is the safety threshold for a single JSONL line.
// Lines exceeding this limit are drained and discarded to prevent
// unbounded memory growth (e.g. Claude CLI "result" events containing
// full conversation history).
const MaxLineSize = 50 * 1024 * 1024 // 50MB

// ReadJSONLine reads one complete line from a bufio.Reader.
//
// Returns:
//   - line: complete line bytes (without \n); nil when skipped
//   - skipped: true if line exceeded MaxLineSize and was drained
//   - err: io.EOF at end of stream, or other IO error
func ReadJSONLine(r *bufio.Reader) ([]byte, bool, error) {
	var buf []byte
	for {
		chunk, isPrefix, err := r.ReadLine()
		if err != nil {
			return nil, false, err
		}
		buf = append(buf, chunk...)
		if len(buf) > MaxLineSize {
			if !isPrefix {
				buf = nil
				return nil, true, nil
			}
			for isPrefix {
				_, isPrefix, err = r.ReadLine()
				if err != nil {
					return nil, true, err
				}
			}
			buf = nil
			return nil, true, nil
		}
		if !isPrefix {
			return buf, false, nil
		}
	}
}
