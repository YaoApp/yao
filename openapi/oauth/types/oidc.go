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
