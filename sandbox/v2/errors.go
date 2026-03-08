package sandbox

import "errors"

var (
	ErrNotAvailable = errors.New("sandbox: not available (no pools configured)")
	ErrNotFound     = errors.New("sandbox: not found")
	ErrPoolNotFound = errors.New("sandbox: pool not found")
	ErrPoolMissing  = errors.New("sandbox: pool name is required")
)
