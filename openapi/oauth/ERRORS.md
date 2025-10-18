# OAuth Error Handling Documentation

This document describes the error handling specifications and ACL error definitions for OAuth services.

## Overview

OAuth services use a standardized error response format. All errors follow the `ErrorResponse` structure:

```go
type ErrorResponse struct {
    Code             string `json:"error"`
    ErrorDescription string `json:"error_description,omitempty"`
    ErrorURI         string `json:"error_uri,omitempty"`
    State            string `json:"state,omitempty"`
}
```

## Configuration Errors

These errors are used during OAuth service initialization and configuration:

| Error Code                   | Variable Name                 | Description                                                          |
| ---------------------------- | ----------------------------- | -------------------------------------------------------------------- |
| `invalid_configuration`      | `ErrInvalidConfiguration`     | Invalid OAuth service configuration                                  |
| `store_missing`              | `ErrStoreMissing`             | Store is required for OAuth service                                  |
| `issuer_url_missing`         | `ErrIssuerURLMissing`         | Issuer URL is missing                                                |
| `certificate_missing`        | `ErrCertificateMissing`       | JWT signing certificate and key paths must both be provided or empty |
| `invalid_token_lifetime`     | `ErrInvalidTokenLifetime`     | Token lifetime must be greater than 0                                |
| `pkce_configuration_invalid` | `ErrPKCEConfigurationInvalid` | PKCE configuration is invalid                                        |

## Authentication & Authorization Errors

### Token Related Errors

| Error Code              | Variable Name            | HTTP Status | Description                                        |
| ----------------------- | ------------------------ | ----------- | -------------------------------------------------- |
| `token_missing`         | `ErrTokenMissing`        | 401         | No access token provided in the request            |
| `invalid_token`         | `ErrInvalidToken`        | 401         | The access token is invalid, expired or malformed  |
| `token_expired`         | `ErrTokenExpired`        | 401         | The access token has expired                       |
| `unauthorized`          | `ErrUnauthorized`        | 401         | Authentication is required to access this resource |
| `refresh_token_missing` | `ErrRefreshTokenMissing` | 401         | No refresh token provided in the request           |
| `invalid_refresh_token` | `ErrInvalidRefreshToken` | 401         | The refresh token is invalid or expired            |

### Permission Related Errors

| Error Code           | Variable Name          | HTTP Status | Description                                        |
| -------------------- | ---------------------- | ----------- | -------------------------------------------------- |
| `forbidden`          | `ErrForbidden`         | 403         | You do not have permission to access this resource |
| `access_denied`      | `ErrAccessDenied`      | 403         | Access to this resource has been denied            |
| `insufficient_scope` | `ErrInsufficientScope` | 403         | The access token does not have the required scope  |

### ACL Errors

| Error Code           | Variable Name         | HTTP Status | Description                              |
| -------------------- | --------------------- | ----------- | ---------------------------------------- |
| `acl_check_failed`   | `ErrACLCheckFailed`   | 500         | ACL verification failed                  |
| `acl_internal_error` | `ErrACLInternalError` | 500         | Internal error occurred during ACL check |

### Rate Limiting Errors

| Error Code            | Variable Name          | HTTP Status | Description                               |
| --------------------- | ---------------------- | ----------- | ----------------------------------------- |
| `rate_limit_exceeded` | `ErrRateLimitExceeded` | 429         | Too many requests. Please try again later |
| `too_many_requests`   | `ErrTooManyRequests`   | 429         | Request rate limit exceeded               |

### Resource Errors

| Error Code           | Variable Name         | HTTP Status | Description                                      |
| -------------------- | --------------------- | ----------- | ------------------------------------------------ |
| `resource_not_found` | `ErrResourceNotFound` | 404         | The requested resource was not found             |
| `method_not_allowed` | `ErrMethodNotAllowed` | 405         | The HTTP method is not allowed for this resource |

### Server Errors

| Error Code              | Variable Name            | HTTP Status | Description                            |
| ----------------------- | ------------------------ | ----------- | -------------------------------------- |
| `internal_server_error` | `ErrInternalServerError` | 500         | An internal server error occurred      |
| `service_unavailable`   | `ErrServiceUnavailable`  | 503         | The service is temporarily unavailable |

## ACL Error Types

ACL implementations can return the following error types, which the Guard middleware automatically converts to appropriate HTTP responses:

### acl.Error Structure

```go
type Error struct {
    Type        ErrorType                  // Error type
    Message     string                     // Error message
    Details     map[string]interface{}     // Additional error details
    RetryAfter  int                        // Retry wait time (seconds)
}
```

### ACL Error Type Definitions

| Error Type             | HTTP Status | Description                                       | Retryable |
| ---------------------- | ----------- | ------------------------------------------------- | --------- |
| `permission_denied`    | 403         | User does not have required permissions           | No        |
| `rate_limit_exceeded`  | 429         | Request rate limit exceeded                       | Yes       |
| `insufficient_scope`   | 403         | Token scope is insufficient                       | No        |
| `resource_not_allowed` | 403         | Access to the resource is not allowed             | No        |
| `method_not_allowed`   | 405         | The HTTP method is not allowed                    | No        |
| `ip_blocked`           | 403         | IP address is blocked                             | No        |
| `geo_restricted`       | 403         | Access is restricted based on geographic location | No        |
| `time_restricted`      | 403         | Access is restricted based on time                | Yes       |
| `quota_exceeded`       | 429         | Usage quota has been exceeded                     | Yes       |
| `invalid_request`      | 400         | Request is invalid                                | No        |
| `internal_error`       | 500         | Internal error occurred during ACL check          | Yes       |

### ACL Error Creation Functions

```go
// Basic error creation
acl.NewError(errorType, message)

// Permission denied
acl.NewPermissionDeniedError("User does not have admin role")

// Rate limit error (with retry time)
acl.NewRateLimitError("Too many requests", 60) // Retry after 60 seconds

// Insufficient scope
acl.NewInsufficientScopeError("Missing required scope", []string{"read", "write"})

// Resource not allowed
acl.NewResourceNotAllowedError("/admin/users")

// Method not allowed
acl.NewMethodNotAllowedError("DELETE", []string{"GET", "POST"})

// IP blocked
acl.NewIPBlockedError("192.168.1.1")

// Quota exceeded
acl.NewQuotaExceededError("API quota exceeded", "api_calls", 1000, 1050)

// Internal error
acl.NewInternalError("Failed to load ACL rules")
```

## Usage Examples

### Using in Guard Middleware

The Guard middleware automatically handles all error types:

```go
func (s *Service) Guard(c *gin.Context) {
    // Token validation
    token := s.getAccessToken(c)
    if token == "" {
        c.JSON(http.StatusUnauthorized, types.ErrTokenMissing)
        c.Abort()
        return
    }

    // ACL check
    ok, err := acl.Global.Enforce(c)
    if err != nil {
        s.handleACLError(c, err) // Automatically handles different types of ACL errors
        return
    }
}
```

### Implementing ACL Enforce Method

```go
func (a *MyACL) Enforce(c *gin.Context) (bool, error) {
    // Check rate limit
    if rateLimitExceeded {
        return false, acl.NewRateLimitError("Too many requests", 60)
    }

    // Check permissions
    if !hasPermission {
        return false, acl.NewPermissionDeniedError("User does not have required permission")
    }

    // Check IP
    if ipBlocked {
        return false, acl.NewIPBlockedError(clientIP)
    }

    // Check quota
    if quotaExceeded {
        return false, acl.NewQuotaExceededError("API quota exceeded", "api_calls", limit, current)
    }

    return true, nil
}
```

### Error Response Examples

#### Standard Error Response

```json
{
  "error": "token_missing",
  "error_description": "No access token provided in the request"
}
```

#### Rate Limit Error Response (with Retry-After header)

```
HTTP/1.1 429 Too Many Requests
Retry-After: 60

{
  "error": "rate_limit_exceeded",
  "error_description": "Too many requests. Please try again later"
}
```

#### Forbidden Response

```json
{
  "error": "forbidden",
  "error_description": "You do not have permission to access this resource"
}
```

## Error Handling Best Practices

1. **Use Predefined Error Constants**: Always use `types.Err*` constants instead of manually creating error responses
2. **Provide Detailed Error Information**: For ACL errors, use the Details field to provide additional context
3. **Set Appropriate HTTP Status Codes**: The Guard middleware handles this automatically, but ensure correct status codes elsewhere
4. **Set Retry-After for Retryable Errors**: For rate limiting and quota errors, provide retry time
5. **Log Error Details**: Log detailed error information for debugging before returning errors
6. **Avoid Leaking Sensitive Information**: Error messages should be user-friendly but not expose internal system details

## Standard HTTP Status Code Mapping

- **200 OK**: Request successful
- **400 Bad Request**: Invalid request parameters
- **401 Unauthorized**: Not authenticated or authentication failed
- **403 Forbidden**: Authenticated but not authorized to access
- **404 Not Found**: Resource does not exist
- **405 Method Not Allowed**: HTTP method not allowed
- **429 Too Many Requests**: Rate limit or quota exceeded
- **500 Internal Server Error**: Internal server error
- **503 Service Unavailable**: Service temporarily unavailable

## Testing Error Handling

It is recommended to test the following scenarios:

1. Missing Token
2. Invalid Token
3. Expired Token
4. Insufficient Permissions
5. Rate Limit Exceeded
6. Quota Exceeded
7. IP Blocked
8. Resource Not Found
9. Method Not Allowed
10. ACL Internal Error

Each scenario should return the appropriate HTTP status code and error response.
