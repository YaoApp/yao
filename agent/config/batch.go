package config

// ResolveBatch resolves config for multiple assistants in one pass.
// ChatID is empty so only L1 (DSL) + L2 (user preferences) are applied.
func ResolveBatch(assistantIDs []string, userID, teamID string) map[string]*Resolved {
	result := make(map[string]*Resolved, len(assistantIDs))
	for _, id := range assistantIDs {
		resolved, err := Resolve(ResolveOptions{
			AssistantID: id,
			UserID:      userID,
			TeamID:      teamID,
		})
		if err != nil {
			result[id] = &Resolved{}
			continue
		}
		result[id] = resolved
	}
	return result
}
