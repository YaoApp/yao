package types

// Error definitions
var (
	ErrInvalidConfiguration     = &ErrorResponse{Code: "invalid_configuration", ErrorDescription: "Invalid OAuth service configuration"}
	ErrStoreMissing             = &ErrorResponse{Code: "store_missing", ErrorDescription: "Store is required for OAuth service"}
	ErrIssuerURLMissing         = &ErrorResponse{Code: "issuer_url_missing", ErrorDescription: "Issuer URL is required for OAuth service"}
	ErrCertificateMissing       = &ErrorResponse{Code: "certificate_missing", ErrorDescription: "JWT signing certificate and key paths must both be provided or both be empty"}
	ErrInvalidTokenLifetime     = &ErrorResponse{Code: "invalid_token_lifetime", ErrorDescription: "Token lifetime must be greater than 0"}
	ErrPKCEConfigurationInvalid = &ErrorResponse{Code: "pkce_configuration_invalid", ErrorDescription: "PKCE configuration is invalid"}
)
