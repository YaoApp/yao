package types

import (
	"time"
)

// Map converts the OIDCUserInfo to a map[string]interface{}, excluding empty values
func (user OIDCUserInfo) Map() map[string]interface{} {
	result := make(map[string]interface{})

	// Only add non-empty string fields
	if user.Sub != "" {
		result["sub"] = user.Sub
	}
	if user.Name != "" {
		result["name"] = user.Name
	}
	if user.GivenName != "" {
		result["given_name"] = user.GivenName
	}
	if user.FamilyName != "" {
		result["family_name"] = user.FamilyName
	}
	if user.MiddleName != "" {
		result["middle_name"] = user.MiddleName
	}
	if user.Nickname != "" {
		result["nickname"] = user.Nickname
	}
	if user.PreferredUsername != "" {
		result["preferred_username"] = user.PreferredUsername
	}
	if user.Profile != "" {
		result["profile"] = user.Profile
	}
	if user.Picture != "" {
		result["picture"] = user.Picture
	}
	if user.Website != "" {
		result["website"] = user.Website
	}
	if user.Email != "" {
		result["email"] = user.Email
	}
	if user.Gender != "" {
		result["gender"] = user.Gender
	}
	if user.Birthdate != "" {
		result["birthdate"] = user.Birthdate
	}
	if user.Zoneinfo != "" {
		result["zoneinfo"] = user.Zoneinfo
	}
	if user.Locale != "" {
		result["locale"] = user.Locale
	}
	if user.PhoneNumber != "" {
		result["phone_number"] = user.PhoneNumber
	}

	// Only add non-nil boolean pointer fields
	if user.EmailVerified != nil {
		result["email_verified"] = user.EmailVerified
	}
	if user.PhoneNumberVerified != nil {
		result["phone_number_verified"] = user.PhoneNumberVerified
	}

	// Convert and add UpdatedAt if not nil
	if converted := unixToMySQL(user.UpdatedAt); converted != nil {
		result["updated_at"] = converted
	}

	// Add address if present and has content
	if user.Address != nil {
		addressMap := make(map[string]interface{})
		if user.Address.Formatted != "" {
			addressMap["formatted"] = user.Address.Formatted
		}
		if user.Address.StreetAddress != "" {
			addressMap["street_address"] = user.Address.StreetAddress
		}
		if user.Address.Locality != "" {
			addressMap["locality"] = user.Address.Locality
		}
		if user.Address.Region != "" {
			addressMap["region"] = user.Address.Region
		}
		if user.Address.PostalCode != "" {
			addressMap["postal_code"] = user.Address.PostalCode
		}
		if user.Address.Country != "" {
			addressMap["country"] = user.Address.Country
		}
		if len(addressMap) > 0 {
			result["address"] = addressMap
		}
	}

	// Add Yao custom fields with namespace
	if user.YaoUserID != "" {
		result["yao:user_id"] = user.YaoUserID
	}
	if user.YaoTenantID != "" {
		result["yao:tenant_id"] = user.YaoTenantID
	}
	if user.YaoTeamID != "" {
		result["yao:team_id"] = user.YaoTeamID
	}
	if user.YaoIsOwner != nil {
		result["yao:is_owner"] = user.YaoIsOwner
	}
	if user.YaoTypeID != "" {
		result["yao:type_id"] = user.YaoTypeID
	}

	// Add Yao team info if present and has content
	if user.YaoTeam != nil {
		teamMap := make(map[string]interface{})
		if user.YaoTeam.TeamID != "" {
			teamMap["team_id"] = user.YaoTeam.TeamID
		}
		if user.YaoTeam.Logo != "" {
			teamMap["logo"] = user.YaoTeam.Logo
		}
		if user.YaoTeam.Name != "" {
			teamMap["name"] = user.YaoTeam.Name
		}
		if user.YaoTeam.OwnerID != "" {
			teamMap["owner_id"] = user.YaoTeam.OwnerID
		}
		if user.YaoTeam.Description != "" {
			teamMap["description"] = user.YaoTeam.Description
		}
		if converted := unixToMySQL(user.YaoTeam.UpdatedAt); converted != nil {
			teamMap["updated_at"] = converted
		}
		if len(teamMap) > 0 {
			result["yao:team"] = teamMap
		}
	}

	// Add Yao type info if present and has content
	if user.YaoType != nil {
		typeMap := make(map[string]interface{})
		if user.YaoType.TypeID != "" {
			typeMap["type_id"] = user.YaoType.TypeID
		}
		if user.YaoType.Name != "" {
			typeMap["name"] = user.YaoType.Name
		}
		if user.YaoType.Locale != "" {
			typeMap["locale"] = user.YaoType.Locale
		}
		if len(typeMap) > 0 {
			result["yao:type"] = typeMap
		}
	}

	// Add Yao member info if present and has content (for team context)
	if user.YaoMember != nil {
		memberMap := make(map[string]interface{})
		if user.YaoMember.MemberID != "" {
			memberMap["member_id"] = user.YaoMember.MemberID
		}
		if user.YaoMember.DisplayName != "" {
			memberMap["display_name"] = user.YaoMember.DisplayName
		}
		if user.YaoMember.Bio != "" {
			memberMap["bio"] = user.YaoMember.Bio
		}
		if user.YaoMember.Avatar != "" {
			memberMap["avatar"] = user.YaoMember.Avatar
		}
		if user.YaoMember.Email != "" {
			memberMap["email"] = user.YaoMember.Email
		}
		if len(memberMap) > 0 {
			result["yao:member"] = memberMap
		}
	}

	// Include raw data if available
	// if user.Raw != nil {
	// 	// Merge raw data, but let structured fields take precedence
	// 	for k, v := range user.Raw {
	// 		if _, exists := result[k]; !exists && v != nil && v != "" {
	// 			result[k] = v
	// 		}
	// 	}
	// }

	return result
}

// MakeOIDCUserInfo creates a new OIDCUserInfo from a map[string]interface{}
func MakeOIDCUserInfo(user map[string]interface{}) *OIDCUserInfo {
	userInfo := &OIDCUserInfo{
		Raw: user, // Store original response
	}

	// String fields with safe type assertion
	if sub, ok := user["sub"].(string); ok {
		userInfo.Sub = sub
	}
	if name, ok := user["name"].(string); ok {
		userInfo.Name = name
	}
	if givenName, ok := user["given_name"].(string); ok {
		userInfo.GivenName = givenName
	}
	if familyName, ok := user["family_name"].(string); ok {
		userInfo.FamilyName = familyName
	}
	if middleName, ok := user["middle_name"].(string); ok {
		userInfo.MiddleName = middleName
	}
	if nickname, ok := user["nickname"].(string); ok {
		userInfo.Nickname = nickname
	}
	if preferredUsername, ok := user["preferred_username"].(string); ok {
		userInfo.PreferredUsername = preferredUsername
	}
	if profile, ok := user["profile"].(string); ok {
		userInfo.Profile = profile
	}
	if picture, ok := user["picture"].(string); ok {
		userInfo.Picture = picture
	}
	if website, ok := user["website"].(string); ok {
		userInfo.Website = website
	}
	if email, ok := user["email"].(string); ok {
		userInfo.Email = email
	}
	if gender, ok := user["gender"].(string); ok {
		userInfo.Gender = gender
	}
	if birthdate, ok := user["birthdate"].(string); ok {
		userInfo.Birthdate = birthdate
	}
	if zoneinfo, ok := user["zoneinfo"].(string); ok {
		userInfo.Zoneinfo = zoneinfo
	}
	if locale, ok := user["locale"].(string); ok {
		userInfo.Locale = locale
	}
	if phoneNumber, ok := user["phone_number"].(string); ok {
		userInfo.PhoneNumber = phoneNumber
	}

	// Boolean pointer fields
	if emailVerified, ok := user["email_verified"].(bool); ok {
		userInfo.EmailVerified = &emailVerified
	}
	if phoneVerified, ok := user["phone_number_verified"].(bool); ok {
		userInfo.PhoneNumberVerified = &phoneVerified
	}

	// Updated_at field
	if updatedAt, ok := user["updated_at"]; ok {
		if converted := toUnixTimestamp(updatedAt); converted != nil {
			if unixTime, ok := converted.(int64); ok {
				userInfo.UpdatedAt = &unixTime
			}
		}
	}

	// Address field (nested object)
	if addressData, ok := user["address"].(map[string]interface{}); ok {
		address := &OIDCAddress{}
		if formatted, ok := addressData["formatted"].(string); ok {
			address.Formatted = formatted
		}
		if streetAddress, ok := addressData["street_address"].(string); ok {
			address.StreetAddress = streetAddress
		}
		if locality, ok := addressData["locality"].(string); ok {
			address.Locality = locality
		}
		if region, ok := addressData["region"].(string); ok {
			address.Region = region
		}
		if postalCode, ok := addressData["postal_code"].(string); ok {
			address.PostalCode = postalCode
		}
		if country, ok := addressData["country"].(string); ok {
			address.Country = country
		}
		userInfo.Address = address
	}

	// Yao custom fields with namespace
	if userID, ok := user["yao:user_id"].(string); ok {
		userInfo.YaoUserID = userID
	}
	if tenantID, ok := user["yao:tenant_id"].(string); ok {
		userInfo.YaoTenantID = tenantID
	}
	if teamID, ok := user["yao:team_id"].(string); ok {
		userInfo.YaoTeamID = teamID
	}
	if isOwner, ok := user["yao:is_owner"].(bool); ok {
		userInfo.YaoIsOwner = &isOwner
	}
	if typeID, ok := user["yao:type_id"].(string); ok {
		userInfo.YaoTypeID = typeID
	}

	// Yao team info (nested object)
	if teamData, ok := user["yao:team"].(map[string]interface{}); ok {
		team := &OIDCTeamInfo{}
		if teamID, ok := teamData["team_id"].(string); ok {
			team.TeamID = teamID
		}
		if logo, ok := teamData["logo"].(string); ok {
			team.Logo = logo
		}
		if name, ok := teamData["name"].(string); ok {
			team.Name = name
		}
		if ownerID, ok := teamData["owner_id"].(string); ok {
			team.OwnerID = ownerID
		}
		if description, ok := teamData["description"].(string); ok {
			team.Description = description
		}
		if updatedAt, ok := teamData["updated_at"]; ok {
			if converted := toUnixTimestamp(updatedAt); converted != nil {
				if unixTime, ok := converted.(int64); ok {
					team.UpdatedAt = &unixTime
				}
			}
		}
		userInfo.YaoTeam = team
	}

	// Yao type info (nested object)
	if typeData, ok := user["yao:type"].(map[string]interface{}); ok {
		typeInfo := &OIDCTypeInfo{}
		if typeID, ok := typeData["type_id"].(string); ok {
			typeInfo.TypeID = typeID
		}
		if name, ok := typeData["name"].(string); ok {
			typeInfo.Name = name
		}
		if locale, ok := typeData["locale"].(string); ok {
			typeInfo.Locale = locale
		}
		userInfo.YaoType = typeInfo
	}

	// Yao member info (nested object, for team context)
	if memberData, ok := user["yao:member"].(map[string]interface{}); ok {
		member := &OIDCMemberInfo{}
		if memberID, ok := memberData["member_id"].(string); ok {
			member.MemberID = memberID
		}
		if displayName, ok := memberData["display_name"].(string); ok {
			member.DisplayName = displayName
		}
		if bio, ok := memberData["bio"].(string); ok {
			member.Bio = bio
		}
		if avatar, ok := memberData["avatar"].(string); ok {
			member.Avatar = avatar
		}
		if email, ok := memberData["email"].(string); ok {
			member.Email = email
		}
		userInfo.YaoMember = member
	}

	return userInfo
}

// unixToMySQL converts interface{} to MySQL DATETIME string
func unixToMySQL(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	var unixTime int64
	switch v := val.(type) {
	case int64:
		unixTime = v
	case *int64:
		if v == nil {
			return nil
		}
		unixTime = *v
	case int:
		unixTime = int64(v)
	case float64:
		unixTime = int64(v)
	default:
		return nil
	}

	return time.Unix(unixTime, 0).UTC().Format("2006-01-02 15:04:05")
}

// mysqlToUnix converts interface{} to Unix timestamp
func mysqlToUnix(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	var dateTime string
	switch v := val.(type) {
	case string:
		dateTime = v
	case *string:
		if v == nil {
			return nil
		}
		dateTime = *v
	default:
		return nil
	}

	if dateTime == "" {
		return nil
	}

	// Try MySQL DATETIME format
	if t, err := time.Parse("2006-01-02 15:04:05", dateTime); err == nil {
		return t.Unix()
	}

	// Try ISO format as fallback
	if t, err := time.Parse("2006-01-02T15:04:05Z", dateTime); err == nil {
		return t.Unix()
	}

	return nil
}

// toUnixTimestamp converts any interface{} to Unix timestamp
func toUnixTimestamp(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case int64:
		return v
	case *int64:
		if v == nil {
			return nil
		}
		return *v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		// Handle MySQL DATETIME or ISO format
		return mysqlToUnix(v)
	case *string:
		if v == nil {
			return nil
		}
		return mysqlToUnix(*v)
	default:
		return nil
	}
}
