package types

// SandboxToken holds credentials for a sandbox execution session.
// Expiry is managed by the LRU store TTL, not stored here.
type SandboxToken struct {
	Token        string // access token → YAO_TOKEN
	RefreshToken string // refresh token → YAO_REFRESH_TOKEN
}
