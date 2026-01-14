package utils

import (
	"fmt"
	"regexp"
)

var (
	// Email regex pattern
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	// Time pattern (HH:MM)
	timeRegex = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)
)

// IsEmpty checks if a string is empty or whitespace only
func IsEmpty(s string) bool {
	return len(s) == 0
}

// IsValidEmail validates email format
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// IsValidTime validates time format (HH:MM)
func IsValidTime(timeStr string) bool {
	return timeRegex.MatchString(timeStr)
}

// ValidateRequired checks if required fields are present
func ValidateRequired(fieldName string, value interface{}) error {
	if value == nil {
		return fmt.Errorf("%s is required", fieldName)
	}

	switch v := value.(type) {
	case string:
		if IsEmpty(v) {
			return fmt.Errorf("%s is required", fieldName)
		}
	case []string:
		if len(v) == 0 {
			return fmt.Errorf("%s is required", fieldName)
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return fmt.Errorf("%s is required", fieldName)
		}
	}

	return nil
}

// ValidateRange checks if a number is within range
func ValidateRange(fieldName string, value, min, max int) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d", fieldName, min, max)
	}
	return nil
}

// ValidateOneOf checks if value is one of allowed values
func ValidateOneOf(fieldName string, value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %v", fieldName, allowed)
}

// ValidateEmail validates email and returns error if invalid
func ValidateEmail(fieldName string, email string) error {
	if !IsValidEmail(email) {
		return fmt.Errorf("%s is not a valid email", fieldName)
	}
	return nil
}

// ValidateTimeFormat validates time format (HH:MM)
func ValidateTimeFormat(fieldName string, timeStr string) error {
	if !IsValidTime(timeStr) {
		return fmt.Errorf("%s must be in HH:MM format", fieldName)
	}
	return nil
}
