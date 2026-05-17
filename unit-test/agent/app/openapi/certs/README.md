# OAuth Certificate Files Documentation

## Overview

This directory contains certificate files used for OAuth 2.1 authentication and authorization. Each file serves a specific purpose in the OAuth security infrastructure.

## Configuration Usage

### Path Configuration in `openapi.yao`

The certificate files in this directory are referenced in the OAuth configuration using **relative paths**. The OpenAPI service automatically resolves these paths relative to the `{YAO_ROOT}/openapi/certs/` directory.

**Configuration Example:**

```json
{
  "oauth": {
    "signing": {
      "signing_cert_path": "signing-cert.pem", // → {YAO_ROOT}/openapi/certs/signing-cert.pem
      "signing_key_path": "signing-key.pem", // → {YAO_ROOT}/openapi/certs/signing-key.pem
      "mtls_client_ca_cert_path": "mtls-client-ca.pem" // → {YAO_ROOT}/openapi/certs/mtls-client-ca.pem
    }
  }
}
```

### Directory Structure and Path Resolution

```
{YAO_ROOT}/                                    # Application root directory
├── openapi/
    ├── openapi.yao                           # Configuration file
    └── certs/                                # Certificate directory (this directory)
        ├── signing-cert.pem                  # Referenced as: "signing-cert.pem"
        ├── signing-key.pem                   # Referenced as: "signing-key.pem"
        ├── mtls-client-ca.pem                # Referenced as: "mtls-client-ca.pem"
        ├── mtls-client-ca-key-...            # CA private key (testing only)
        └── ssl/                              # Optional: Subdirectories supported
            └── production-cert.pem           # Referenced as: "ssl/production-cert.pem"
```

### Path Resolution Benefits

- 📁 **Clean Configuration**: Simple filenames in configuration files
- 🔧 **Environment Independence**: Same config works across deployments
- 🛡️ **Security**: Certificates contained in dedicated directory
- 📦 **Portability**: Easy to package and deploy with application

### Advanced Path Options

1. **Simple Filenames** (Recommended):

   ```json
   "signing_cert_path": "signing-cert.pem"
   ```

2. **Subdirectory Organization**:

   ```json
   "signing_cert_path": "production/signing-cert.pem"
   ```

3. **Absolute Paths** (Not recommended for portability):
   ```json
   "signing_cert_path": "/etc/ssl/certs/oauth-signing.pem"
   ```

## Certificate Files

### 1. Token Signing Certificates

#### `signing-key.pem` 🔐

- **Purpose**: Private key for signing OAuth access tokens
- **Usage**: OAuth server uses this to sign tokens
- **Security**: ⚠️ **HIGHLY SENSITIVE** - Keep secure and never expose
- **Permissions**: 600 (read-write owner only)

#### `signing-cert.pem` 📜

- **Purpose**: Public certificate for verifying tokens
- **Usage**: Clients and resource servers use this to verify token authenticity
- **Security**: Can be shared publicly
- **Permissions**: 644 (readable by all)

### 2. mTLS Client Authentication Certificates

#### `mtls-client-ca.pem` 🏛️

- **Purpose**: CA certificate for validating client certificates
- **Usage**: OAuth server uses this to verify client TLS certificates
- **Security**: Can be shared with clients who need to validate the CA chain
- **Permissions**: 644 (readable by all)

#### `mtls-client-ca-key-TESTING-ONLY-DO-NOT-USE-IN-PRODUCTION.pem` ⚠️

- **Purpose**: CA private key for issuing client certificates
- **Usage**: Sign client certificates (TESTING ONLY)
- **Security**: 🚨 **EXTREMELY SENSITIVE** - Never use in production
- **Permissions**: 600 (read-write owner only)

## Security Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    OAuth Security Flow                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. mTLS Handshake                                              │
│     Client ──[Client Certificate]──> OAuth Server              │
│     OAuth Server validates using mtls-client-ca.pem            │
│                                                                 │
│  2. Token Issuance                                              │
│     OAuth Server signs tokens using signing-key.pem            │
│     Client verifies tokens using signing-cert.pem              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Usage Scenarios

### High-Security Environments

- **Financial APIs**: Banking and payment systems
- **Healthcare**: Medical data exchange (HIPAA compliance)
- **Government**: Classified information systems
- **Enterprise**: Internal microservices communication

### Client Certificate Workflow

1. **Generate client private key**
2. **Create Certificate Signing Request (CSR)**
3. **Sign CSR with CA private key** (using the TESTING-ONLY file)
4. **Deploy client certificate** to client application
5. **OAuth server validates** client certificate against CA

## Production Security Best Practices

### 🚨 Critical Security Warnings

1. **Never store CA private keys in application directories**
2. **Use Hardware Security Modules (HSM) for production**
3. **Implement certificate rotation policies**
4. **Monitor certificate expiration dates**
5. **Use separate CAs for different environments**

### Recommended Production Setup

```
Production Environment:
├── Certificate Authority (External/HSM)
│   ├── Root CA (Hardware-protected)
│   └── Intermediate CA (Hardware-protected)
├── OAuth Server
│   ├── signing-cert.pem (Public)
│   ├── signing-key.pem (Protected)
│   └── mtls-client-ca.pem (Public)
└── Client Applications
    ├── client-cert.pem (Public)
    └── client-key.pem (Protected)
```

## Certificate Validation

### Verify Certificate Integrity

```bash
# Check certificate details
openssl x509 -in signing-cert.pem -text -noout

# Verify certificate chain
openssl verify -CAfile mtls-client-ca.pem client-cert.pem

# Check certificate expiration
openssl x509 -in signing-cert.pem -noout -dates
```

## Environment-Specific Considerations

### Development Environment ✅

- Use self-signed certificates (like these)
- Store certificates in application directory
- Simple certificate management

### Testing Environment ⚠️

- Use dedicated test CA
- Implement certificate rotation testing
- Mirror production certificate structure

### Production Environment 🏭

- Use commercial or enterprise CA
- Hardware-protected private keys
- Automated certificate management
- Comprehensive monitoring and alerting

## Certificate Rotation

### Recommended Rotation Schedule

- **Token Signing Certificates**: Every 6 months
- **mTLS Client Certificates**: Every 12 months
- **CA Certificates**: Every 2-3 years

### Rotation Process

1. Generate new certificates
2. Update OAuth server configuration
3. Distribute new public certificates
4. Test thoroughly
5. Update client applications
6. Revoke old certificates

## Troubleshooting

### Common Issues

#### Certificate File Issues

- **Certificate expired**: Check expiration dates using `openssl x509 -in cert.pem -noout -dates`
- **Certificate chain invalid**: Verify CA chain with `openssl verify -CAfile ca.pem cert.pem`
- **Permission denied**: Check file permissions (private keys should be 600)
- **Certificate mismatch**: Ensure correct certificate-key pairs

#### Configuration Path Issues

- **File not found**:

  - ✅ Verify certificate files exist in `{YAO_ROOT}/openapi/certs/` directory
  - ✅ Check relative path spelling in `openapi.yao` configuration
  - ✅ Use forward slashes `/` for subdirectories (not backslashes `\`)

- **Invalid certificate path**:

  ```bash
  # Check if file exists
  ls -la {YAO_ROOT}/openapi/certs/signing-cert.pem

  # Verify configuration path
  cat openapi.yao | grep signing_cert_path
  ```

- **Permission issues**:
  ```bash
  # Fix certificate file permissions
  chmod 644 signing-cert.pem          # Public certificates
  chmod 600 signing-key.pem           # Private keys
  chmod 600 mtls-client-ca-key-*.pem  # CA private keys
  ```

#### Path Resolution Debugging

1. **Verify your application root**:

   ```bash
   echo $YAO_ROOT  # Should show your application root path
   ```

2. **Check resolved paths**:

   ```bash
   # Expected resolution for "signing-cert.pem"
   ls -la $YAO_ROOT/openapi/certs/signing-cert.pem
   ```

3. **Test configuration**:
   ```bash
   # Validate openapi.yao syntax
   yao inspect openapi
   ```

## Compliance and Standards

This implementation follows:

- **RFC 6749**: OAuth 2.0 Authorization Framework
- **RFC 8705**: OAuth 2.0 Mutual-TLS Client Authentication
- **RFC 7517**: JSON Web Key (JWK)
- **RFC 7519**: JSON Web Token (JWT)

## Support

### Documentation Resources

- **Configuration Guide**: See `../README.md` for complete OAuth configuration documentation
- **Path Configuration**: Detailed path resolution rules and examples in main README
- **Duration Format**: Human-readable time format guidelines for certificate rotation

### Getting Help

- **Certificate Management**: Consult your security team or system administrator
- **OAuth Implementation**: Refer to OAuth 2.1 specification (RFC 6749, RFC 8705)
- **Configuration Issues**: Check troubleshooting section above for common problems
- **Path Resolution**: Review path configuration examples in main OpenAPI documentation

---

**Last Updated**: 2025-01-18  
**Version**: 1.1 - Added configuration path documentation  
**Environment**: Development/Testing Only
