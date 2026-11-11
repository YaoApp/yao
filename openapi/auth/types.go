package auth

import "time"

// RefreshNodeTokenResponse is returned by the node-token refresh endpoint.
type RefreshNodeTokenResponse struct {
	NodeToken string    `json:"node_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RotateNodeTokenRequest identifies the computer whose token is rotated.
type RotateNodeTokenRequest struct {
	ComputerID string `json:"computer_id"`
}
