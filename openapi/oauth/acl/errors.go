package acl

import "fmt"

// ErrorType represents different types of ACL errors
type ErrorType string

const (
	// ErrorTypePermissionDenied indicates the user does not have required permissions
	ErrorTypePermissionDenied ErrorType = "permission_denied"

	// ErrorTypeRateLimitExceeded indicates the request rate limit has been exceeded
	ErrorTypeRateLimitExceeded ErrorType = "rate_limit_exceeded"

	// ErrorTypeInsufficientScope indicates the token scope is insufficient
	ErrorTypeInsufficientScope ErrorType = "insufficient_scope"

	// ErrorTypeResourceNotAllowed indicates access to the resource is not allowed
	ErrorTypeResourceNotAllowed ErrorType = "resource_not_allowed"

	// ErrorTypeMethodNotAllowed indicates the HTTP method is not allowed
	ErrorTypeMethodNotAllowed ErrorType = "method_not_allowed"

	// ErrorTypeIPBlocked indicates the request IP is blocked
	ErrorTypeIPBlocked ErrorType = "ip_blocked"

	// ErrorTypeGeoRestricted indicates access is restricted based on geographic location
	ErrorTypeGeoRestricted ErrorType = "geo_restricted"

	// ErrorTypeTimeRestricted indicates access is restricted based on time
	ErrorTypeTimeRestricted ErrorType = "time_restricted"

	// ErrorTypeQuotaExceeded indicates the usage quota has been exceeded
	ErrorTypeQuotaExceeded ErrorType = "quota_exceeded"

	// ErrorTypeInvalidRequest indicates the request is invalid
	ErrorTypeInvalidRequest ErrorType = "invalid_request"

	// ErrorTypeInternal indicates an internal error occurred during ACL check
	ErrorTypeInternal ErrorType = "internal_error"
)

// Error represents an ACL-related error with additional context
type Error struct {
	Type       ErrorType
	Message    string
	Details    map[string]interface{}
	RetryAfter int              // seconds to wait before retrying (for rate limit errors)
	Stage      EnforcementStage // stage where the permission check failed
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("ACL error [%s]: %s", e.Type, e.Message)
	}
	return fmt.Sprintf("ACL error: %s", e.Type)
}

// IsRetryable returns true if the error is retryable (e.g., rate limit)
func (e *Error) IsRetryable() bool {
	return e.Type == ErrorTypeRateLimitExceeded || e.Type == ErrorTypeQuotaExceeded
}

// NewError creates a new ACL error
func NewError(errorType ErrorType, message string) *Error {
	return &Error{
		Type:    errorType,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// NewPermissionDeniedError creates a permission denied error
func NewPermissionDeniedError(message string) *Error {
	return NewError(ErrorTypePermissionDenied, message)
}

// NewRateLimitError creates a rate limit error with retry-after information
func NewRateLimitError(message string, retryAfter int) *Error {
	err := NewError(ErrorTypeRateLimitExceeded, message)
	err.RetryAfter = retryAfter
	return err
}

// NewInsufficientScopeError creates an insufficient scope error
func NewInsufficientScopeError(message string, requiredScopes []string) *Error {
	err := NewError(ErrorTypeInsufficientScope, message)
	err.Details["required_scopes"] = requiredScopes
	return err
}

// NewResourceNotAllowedError creates a resource not allowed error
func NewResourceNotAllowedError(resource string) *Error {
	err := NewError(ErrorTypeResourceNotAllowed, "Access to this resource is not allowed")
	err.Details["resource"] = resource
	return err
}

// NewMethodNotAllowedError creates a method not allowed error
func NewMethodNotAllowedError(method string, allowedMethods []string) *Error {
	err := NewError(ErrorTypeMethodNotAllowed, fmt.Sprintf("HTTP method '%s' is not allowed", method))
	err.Details["method"] = method
	err.Details["allowed_methods"] = allowedMethods
	return err
}

// NewIPBlockedError creates an IP blocked error
func NewIPBlockedError(ip string) *Error {
	err := NewError(ErrorTypeIPBlocked, "Access from this IP address is blocked")
	err.Details["ip"] = ip
	return err
}

// NewQuotaExceededError creates a quota exceeded error
func NewQuotaExceededError(message string, quotaType string, limit int64, current int64) *Error {
	err := NewError(ErrorTypeQuotaExceeded, message)
	err.Details["quota_type"] = quotaType
	err.Details["limit"] = limit
	err.Details["current"] = current
	return err
}

// NewInternalError creates an internal error
func NewInternalError(message string) *Error {
	return NewError(ErrorTypeInternal, message)
}
