package context

// getValidatedValue gets value from query, header, or default, and validates it
func getValidatedValue(queryValue, headerValue, defaultValue string, validator func(string) string) string {
	if queryValue != "" {
		return validator(queryValue)
	}
	if headerValue != "" {
		return validator(headerValue)
	}
	return defaultValue
}

// getValidatedAccept gets Accept from query, header, or parse from client type
func getValidatedAccept(queryValue, headerValue, clientType string) Accept {
	if queryValue != "" {
		return validateAccept(queryValue)
	}
	if headerValue != "" {
		return validateAccept(headerValue)
	}
	return parseAccept(clientType)
}

// validateReferer validates and returns a valid Referer, returns RefererAPI if invalid
func validateReferer(referer string) string {
	if ValidReferers[referer] {
		return referer
	}
	return RefererAPI
}

// validateAccept validates and returns a valid Accept type, returns AcceptStandard if invalid
func validateAccept(accept string) Accept {
	if ValidAccepts[accept] {
		return Accept(accept)
	}
	return AcceptStandard
}

// parseAccept determines the accept type based on client type
func parseAccept(clientType string) Accept {
	switch clientType {
	case "web":
		return AcceptWebCUI
	case "android", "ios":
		return AccepNativeCUI
	case "windows", "macos", "linux":
		return AcceptDesktopCUI
	default:
		return AcceptStandard
	}
}
