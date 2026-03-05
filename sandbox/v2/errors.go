package sandbox

import "errors"

var (
	ErrNotAvailable  = errors.New("sandbox: not available (no pools configured)")
	ErrNotFound      = errors.New("sandbox: not found")
	ErrLimitExceeded = errors.New("sandbox: limit exceeded")
	ErrPoolNotFound  = errors.New("sandbox: pool not found")
	ErrPoolInUse     = errors.New("sandbox: pool has running boxes")
)
