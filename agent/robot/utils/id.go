package utils

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// NewID generates a new unique ID using nanoid
func NewID() string {
	id, err := gonanoid.New()
	if err != nil {
		// Fallback to nanoid with default alphabet if error occurs
		return gonanoid.Must()
	}
	return id
}

// NewIDWithPrefix generates a new ID with a prefix
func NewIDWithPrefix(prefix string) string {
	return prefix + NewID()
}
