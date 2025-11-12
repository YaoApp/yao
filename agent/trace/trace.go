package trace

import (
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// GenTraceID generate a new trace ID using NanoID algorithm
// safe: optional parameter, reserved for future safe mode implementation (collision detection)
func GenTraceID(safe ...bool) string {
	// TODO: Implement safe mode with collision detection when needed
	// For now, NanoID provides sufficient uniqueness without collision checking

	// URL-safe alphabet (no ambiguous characters like 0/O, 1/l/I)
	const alphabet = "1234567890"
	const length = 8 // 8 characters provides good balance of uniqueness and readability

	id, err := gonanoid.Generate(alphabet, length)
	if err != nil {
		// Fallback to timestamp-based ID if NanoID generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return id
}
