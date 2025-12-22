# OAuth 2.1 Testing Guide

## Overview

This guide provides comprehensive testing infrastructure for OAuth 2.1 authorization server implementation. The test environment includes standardized test data sets, environment setup functions, and complete test coverage for all OAuth functionality.

## Test Environment Architecture

### Core Components

- **OAuth Service Configuration**: Complete OAuth 2.1 configuration with all features enabled
- **Store Management**: Support for MongoDB and Xun (database-backed) stores with automatic fallback
- **Test Data Sets**: Pre-configured clients and users for comprehensive testing
- **Environment Setup**: Standardized initialization and cleanup procedures

### Test Data Sets

#### Standard Test Clients (3 clients)

1. **Confidential Client** (`test-confidential-client`)

   - **Purpose**: Authorization code flow testing
   - **Grant Types**: authorization_code, refresh_token
   - **Use Case**: Web applications with server-side authentication

2. **Public Client** (`test-public-client`)

   - **Purpose**: Mobile/SPA application testing
   - **Grant Types**: authorization_code (with PKCE)
   - **Use Case**: Single-page applications and mobile apps

3. **Client Credentials Client** (`test-credentials-client`)
   - **Purpose**: Server-to-server authentication
   - **Grant Types**: client_credentials
   - **Use Case**: API access and service authentication

#### Standard Test Users (10 users)

1. **Admin User** (`admin`)

   - **Privileges**: Full access with admin scope
   - **Features**: 2FA enabled, all verifications complete
   - **Use Case**: Administrative functionality testing

2. **Regular User** (`john.doe`)

   - **Privileges**: Basic user access
   - **Features**: Standard verification
   - **Use Case**: Standard user flow testing

3. **Enhanced User** (`jane.smith`)

   - **Privileges**: Basic user access with mobile verification
   - **Features**: Email and mobile verified
   - **Use Case**: Multi-factor authentication testing

4. **Pending User** (`pending.user`)

   - **Privileges**: Limited access
   - **Features**: Pending verification status
   - **Use Case**: User onboarding flow testing

5. **Inactive User** (`inactive.user`)

   - **Privileges**: Disabled account
   - **Features**: Inactive status
   - **Use Case**: Account management testing

6. **Limited User** (`limited.user`)

   - **Privileges**: Minimal scope access
   - **Features**: Basic OpenID only
   - **Use Case**: Scope limitation testing

7. **Security User** (`secure.user`)

   - **Privileges**: Standard access with enhanced security
   - **Features**: 2FA enabled, all verifications
   - **Use Case**: Security feature testing

8. **API User** (`api.user`)

   - **Privileges**: API access scopes
   - **Features**: API-specific permissions
   - **Use Case**: API authorization testing

9. **Guest User** (`guest.user`)

   - **Privileges**: Minimal guest access
   - **Features**: No verifications
   - **Use Case**: Guest flow testing

10. **Test User** (`test.user`)
    - **Privileges**: General testing access
    - **Features**: Mixed permissions for testing
    - **Use Case**: General purpose testing

## Environment Setup

### Prerequisites

```bash
# Source the environment configuration
source $YAO_SOURCE_ROOT/env.local.sh
```

### Core Setup Function

```go
func setupOAuthTestEnvironment(t *testing.T) (*Service, store.Store, store.Store, func()) {
    // Creates complete OAuth test environment with:
    // - Configured OAuth service with all features enabled
    // - Primary store (MongoDB preferred, Xun database fallback)
    // - Cache store (LRU cache)
    // - Pre-loaded test clients and users
    // - Cleanup function for proper teardown
}
```

### Environment Features

- **Store Management**: Automatic store selection with fallback
- **Data Isolation**: Each test gets fresh data set with unique identifiers
- **Parallel Execution Support**: Automatic unique suffix generation for concurrent tests
- **Cleanup**: Comprehensive cleanup of test data with pattern matching
- **Logging**: Detailed test logging for debugging and monitoring

## Parallel Execution Support

### Automatic Test Isolation

The testing infrastructure now includes automatic test isolation to support parallel test execution in CI/CD environments like GitHub Actions:

#### Unique Test Data Generation

- **Client IDs**: Automatically suffixed with unique identifier (e.g., `test-confidential-client-TestName-1234567890-abcd1234`)
- **User Emails**: Automatically modified to be unique (e.g., `admin@example.com` → `admin-TestName-1234567890-abcd1234@example.com`)
- **Usernames**: Automatically suffixed (e.g., `admin` → `admin-TestName-1234567890-abcd1234`)

#### Suffix Generation Strategy

The test suffix is generated using:

1. **Test Name**: Sanitized test function name
2. **Timestamp**: Millisecond precision timestamp
3. **Random Component**: 4-byte random hex string

This ensures uniqueness even when tests run simultaneously across multiple processes.

#### Enhanced Cleanup

Cleanup now includes comprehensive pattern matching for:

- User IDs with various prefixes
- Email addresses with suffixes
- Usernames with suffixes
- Client IDs with suffixes

### Concurrent Test Benefits

- **GitHub Actions**: Multiple test jobs can run in parallel without conflicts
- **Local Development**: Multiple test runs can execute simultaneously
- **CI/CD Pipelines**: Faster test execution without data collisions
- **Development Teams**: Multiple developers can run tests concurrently

## Testing Patterns

### Basic Test Structure

```go
func TestOAuthFeature(t *testing.T) {
    service, _, _, cleanup := setupOAuthTestEnvironment(t)
    defer cleanup()

    // Use pre-configured test clients and users
    // All standard OAuth flows are supported
}
```

### Parameterized Testing

```go
func TestMultipleStores(t *testing.T) {
    storeConfigs := getStoreConfigs()

    for _, config := range storeConfigs {
        t.Run(config.Name, func(t *testing.T) {
            // Test with different store backends
        })
    }
}
```

### Integration Testing

```go
func TestOAuthFlow(t *testing.T) {
    service, _, _, cleanup := setupOAuthTestEnvironment(t)
    defer cleanup()

    // Use testClients[0] for confidential client testing
    // Use testUsers[0] for admin user testing
    // Complete OAuth flows with real data
}
```

## Test Coverage

### Core OAuth Service Tests

- **Service Creation**: Configuration validation and initialization
- **Service Getters**: Provider access and configuration retrieval
- **Configuration**: Default values and validation
- **Feature Flags**: OAuth 2.1 and MCP compliance features
- **Provider Integration**: User and client provider functionality

### Configuration Tests

- **Valid Configuration**: Complete configuration testing
- **Missing Components**: Error handling for missing configuration
- **Invalid Values**: Validation of configuration parameters
- **Default Values**: Proper default value assignment

### Integration Tests

- **Client Access**: Verification of test client availability
- **User Access**: Verification of test user availability
- **Store Operations**: Multi-store compatibility testing
- **Provider Operations**: User and client provider integration

## Running Tests

### All Tests

```bash
cd $YAO_SOURCE_ROOT
go test -v ./openapi/oauth -timeout 60s
```

### Specific Test

```bash
cd $YAO_SOURCE_ROOT
go test -v ./openapi/oauth -run TestNewService -timeout 30s
```

### With Coverage

```bash
cd $YAO_SOURCE_ROOT
go test -v ./openapi/oauth -cover -timeout 60s
```

## Test Data Reference

### Quick Client Access

```go
// Get confidential client for authorization code flow
confidentialClient := testClients[0]

// Get public client for PKCE flow
publicClient := testClients[1]

// Get client credentials client
credentialsClient := testClients[2]
```

### Quick User Access

```go
// Get admin user for administrative testing
adminUser := testUsers[0]

// Get regular user for standard flow testing
regularUser := testUsers[1]

// Get user with specific features
secureUser := testUsers[6] // 2FA enabled
apiUser := testUsers[7]    // API scopes
```

## Best Practices

### Test Organization

1. **Use Standard Environment**: Always use `setupOAuthTestEnvironment()`
2. **Leverage Test Data**: Use pre-configured clients and users
3. **Proper Cleanup**: Always defer cleanup function
4. **Descriptive Names**: Use clear test and subtest names

### Data Management

1. **Data Isolation**: Each test gets fresh environment
2. **Cleanup**: Automatic cleanup prevents test pollution
3. **Logging**: Comprehensive logging for debugging
4. **Consistency**: Standard data sets ensure consistent testing

### Error Handling

1. **Proper Assertions**: Use testify for clear assertions
2. **Error Messages**: Include context in error messages
3. **Cleanup on Failure**: Cleanup runs even on test failure
4. **Detailed Logging**: Log important test steps

## Environment Variables

### Required for Full Testing

```bash
# MongoDB connection (optional, will fallback to Badger)
export MONGO_TEST_HOST=localhost
export MONGO_TEST_PORT=27017
export MONGO_TEST_USER=test
export MONGO_TEST_PASS=test
```

### Configuration Files

- **Environment Setup**: `$YAO_SOURCE_ROOT/env.local.sh`
- **Test Configuration**: Built into test environment
- **Store Configuration**: Automatic configuration management

## Troubleshooting

### Common Issues

1. **Store Connection**: Check MongoDB availability or use Xun (database) fallback
2. **Environment Setup**: Ensure `env.local.sh` is sourced
3. **Test Timeouts**: Increase timeout for slow operations
4. **Data Conflicts**: ✅ **RESOLVED** - Now automatically handled with unique test suffixes

#### Historical Issue: UNIQUE Constraint Violations (RESOLVED)

**Previous Problem**: Tests running in parallel (especially in GitHub Actions) would fail with:

```
UNIQUE constraint failed: yao_user.email
```

**Root Cause**: Multiple tests creating users with identical email addresses simultaneously.

**Solution Implemented**:

- Automatic unique suffix generation for all test data
- Enhanced cleanup with comprehensive pattern matching
- Proper test isolation for concurrent execution

**Before**: All tests used `admin@example.com`
**After**: Each test uses `admin-t1754189657379-874a8@example.com` (short, timestamp-based unique suffixes)

### Debug Logging

Tests include comprehensive logging:

- Environment initialization
- Test data creation
- Store operations
- Test execution steps

### Performance Considerations

- **MongoDB**: Preferred for full feature testing
- **Xun**: Database-backed fallback with LRU cache layer
- **Cache**: LRU cache for improved performance
- **Cleanup**: Efficient cleanup procedures

## Extending Tests

### Adding New Test Cases

1. Use `setupOAuthTestEnvironment()` as base
2. Leverage existing test data sets
3. Follow established patterns
4. Include proper cleanup

### Adding New Test Data

1. Add to `testClients` or `testUsers` arrays
2. Update `setupTestData()` function
3. Update `cleanupTestData()` function
4. Document new test data purpose

### Custom Test Environments

1. Create custom configuration based on standard
2. Use existing store and cache setup
3. Implement custom cleanup
4. Maintain test isolation

This testing infrastructure provides comprehensive coverage for OAuth 2.1 authorization server functionality with proper environment management, standardized test data, and robust cleanup procedures.
