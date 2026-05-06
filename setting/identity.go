package setting

// Identity represents a scoped caller for setting resolution.
// GetMerged(userID, teamID, ns) uses these to cascade: system <- team <- user.
type Identity interface {
	GetUserID() string
	GetTeamID() string
}
