package types

// Configuration Error definitions
var (
	ErrInvalidConfiguration     = &ErrorResponse{Code: "invalid_configuration", ErrorDescription: "Invalid OAuth service configuration"}
	ErrStoreMissing             = &ErrorResponse{Code: "store_missing", ErrorDescription: "Store is required for OAuth service"}
	ErrIssuerURLMissing         = &ErrorResponse{Code: "issuer_url_missing", ErrorDescription: "Issuer URL is required for OAuth service"}
	ErrCertificateMissing       = &ErrorResponse{Code: "certificate_missing", ErrorDescription: "JWT signing certificate and key paths must both be provided or both be empty"}
	ErrInvalidTokenLifetime     = &ErrorResponse{Code: "invalid_token_lifetime", ErrorDescription: "Token lifetime must be greater than 0"}
	ErrPKCEConfigurationInvalid = &ErrorResponse{Code: "pkce_configuration_invalid", ErrorDescription: "PKCE configuration is invalid"}
)

// Authentication & Authorization Error definitions
var (
	// Token related errors
	ErrUnauthorized        = &ErrorResponse{Code: "unauthorized", ErrorDescription: "Authentication is required to access this resource"}
	ErrInvalidToken        = &ErrorResponse{Code: "invalid_token", ErrorDescription: "The access token provided is invalid, expired or malformed"}
	ErrTokenExpired        = &ErrorResponse{Code: "token_expired", ErrorDescription: "The access token has expired"}
	ErrTokenMissing        = &ErrorResponse{Code: "token_missing", ErrorDescription: "No access token provided in the request"}
	ErrInvalidRefreshToken = &ErrorResponse{Code: "invalid_refresh_token", ErrorDescription: "The refresh token provided is invalid or expired"}
	ErrRefreshTokenMissing = &ErrorResponse{Code: "refresh_token_missing", ErrorDescription: "No refresh token provided in the request"}

	// Permission related errors
	ErrForbidden         = &ErrorResponse{Code: "forbidden", ErrorDescription: "You do not have permission to access this resource"}
	ErrInsufficientScope = &ErrorResponse{Code: "insufficient_scope", ErrorDescription: "The access token does not have the required scope"}
	ErrAccessDenied      = &ErrorResponse{Code: "access_denied", ErrorDescription: "Access to this resource has been denied"}

	// ACL related errors
	ErrACLCheckFailed   = &ErrorResponse{Code: "acl_check_failed", ErrorDescription: "ACL verification failed"}
	ErrACLInternalError = &ErrorResponse{Code: "acl_internal_error", ErrorDescription: "Internal error occurred during ACL verification"}

	// Rate limiting errors
	ErrRateLimitExceeded = &ErrorResponse{Code: "rate_limit_exceeded", ErrorDescription: "Too many requests. Please try again later"}
	ErrTooManyRequests   = &ErrorResponse{Code: "too_many_requests", ErrorDescription: "Request rate limit exceeded"}

	// Resource related errors
	ErrResourceNotFound = &ErrorResponse{Code: "resource_not_found", ErrorDescription: "The requested resource was not found"}
	ErrMethodNotAllowed = &ErrorResponse{Code: "method_not_allowed", ErrorDescription: "The HTTP method is not allowed for this resource"}

	// Server errors
	ErrInternalServerError = &ErrorResponse{Code: "internal_server_error", ErrorDescription: "An internal server error occurred"}
	ErrServiceUnavailable  = &ErrorResponse{Code: "service_unavailable", ErrorDescription: "The service is temporarily unavailable"}
)
