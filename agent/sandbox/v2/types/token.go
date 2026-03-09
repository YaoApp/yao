package types

import "time"

// SandboxToken is a short-lived JWT issued for a sandbox computer.
type SandboxToken struct {
	Token     string
	ExpiresAt time.Time
}
