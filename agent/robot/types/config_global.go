package types

import "sync"

// Global configuration for robot agent
// These values can be set during agent initialization

var (
	// defaultEmailChannel - default messenger channel name for sending emails
	// Can be configured via SetDefaultEmailChannel()
	// Default: "email" (maps to messengers/channels.yao configuration)
	defaultEmailChannel = "email"

	// configMu protects global configuration
	configMu sync.RWMutex
)

// DefaultEmailChannel returns the default email channel name
func DefaultEmailChannel() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return defaultEmailChannel
}

// SetDefaultEmailChannel sets the default messenger channel for email delivery
// This should be called during agent initialization
// The channel name must match a channel defined in messengers/channels.yao
func SetDefaultEmailChannel(channel string) {
	if channel == "" {
		return
	}
	configMu.Lock()
	defer configMu.Unlock()
	defaultEmailChannel = channel
}
