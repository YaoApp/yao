package types

// Map converts the OIDCUserInfo to a map[string]interface{}
func (user OIDCUserInfo) Map() map[string]interface{} {
	return map[string]interface{}{
		"sub":                user.Sub,
		"name":               user.Name,
		"given_name":         user.GivenName,
		"family_name":        user.FamilyName,
		"middle_name":        user.MiddleName,
		"nickname":           user.Nickname,
		"preferred_username": user.PreferredUsername,
		"profile":            user.Profile,
		"picture":            user.Picture,
		"website":            user.Website,
		"email":              user.Email,
		"email_verified":     user.EmailVerified,
		"gender":             user.Gender,
	}
}
