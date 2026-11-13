package computer

import (
	"fmt"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	computerIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	nodeIDAlphabet     = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// NewComputerID returns a 32-character computer ID.
func NewComputerID() string {
	value, err := gonanoid.Generate(computerIDAlphabet, 28)
	if err != nil {
		return fmt.Sprintf("cmp-%08x", fallbackSeed())
	}
	return "cmp-" + value
}

// NewNodeID returns node-<8 lowercase alphanumeric>.
func NewNodeID() string {
	value, err := gonanoid.Generate(nodeIDAlphabet, 8)
	if err != nil {
		return fmt.Sprintf("node-%08x", fallbackSeed())
	}
	return "node-" + value
}

func fallbackSeed() uint32 {
	// Only used if the CSPRNG-backed generator itself fails.
	return uint32(nowUnixNano())
}
