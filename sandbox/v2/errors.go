package sandbox

import "errors"

var (
	ErrNotAvailable = errors.New("sandbox: not available (no nodes registered)")
	ErrNotFound     = errors.New("sandbox: not found")
	ErrNodeNotFound = errors.New("sandbox: node not found")
	ErrNodeMissing  = errors.New("sandbox: node ID is required")
)
