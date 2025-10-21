# ACL System Design Document

## I. Design Goals

1. **High Performance**: Permission checks should be very fast (O(1) or O(log n) level)
2. **Concurrency Safe**: Support multi-threaded concurrent reads and safe dynamic updates
3. **Flexible Configuration**: Support multi-level permission configuration (global, alias, specific scopes)
4. **Path Matching**: Support exact match, parameter match (:id), and wildcard match (\*)

## II. Data Structure Design

### 2.1 Configuration Layer

Raw data loaded from configuration files:

```
GlobalConfig (scopes.yml)
├── default: "allow" | "deny"          # Default policy
├── public: []string                    # Public endpoints (no authentication required)
└── endpoints: []EndpointRule           # Default endpoint rules

AliasConfig (alias.yml)
└── map[string][]string                 # Alias -> scopes list

ScopeDefinition (kb/*.yml, job/*.yml...)
├── name: string                        # Scope name
├── description: string                 # Description
├── owner: bool                         # Owner only
├── team: bool                          # Team only
└── endpoints: []string                 # Endpoint list
```

### 2.2 Runtime Layer

Optimized index structures for fast queries:

```
ScopeManager
├── mu: sync.RWMutex                    # Read-write lock (supports concurrency)
├── defaultAction: string               # Default policy
├── publicPaths: map[string]struct{}    # Public path set - O(1) lookup
├── endpointIndex: map[string]*PathMatcher  # method -> path matcher
├── scopeIndex: map[string]*Scope       # scope_name -> Scope details
└── aliasIndex: map[string][]string     # alias -> expanded scopes
```

### 2.3 Path Matcher

Organize path rules by priority:

```
PathMatcher (per HTTP method)
├── exactPaths: map[string]*EndpointInfo
│   └── "/kb/collections" -> EndpointInfo       # Exact match (priority 1)
│
├── paramPaths: map[string]*EndpointInfo
│   └── "/kb/collections/:id" -> EndpointInfo   # Parameter match (priority 2)
│
└── wildcardPaths: []*WildcardPath
    ├── "/kb/collections/*" -> EndpointInfo     # Longer prefix first
    └── "/kb/*" -> EndpointInfo                 # Wildcard match (priority 3)
```

**Matching Logic**:

1. Check exactPaths first (O(1) map lookup)
2. Then check paramPaths (O(1) map lookup, requires path normalization)
3. Finally iterate wildcardPaths (sorted by prefix length, longer first)

### 2.4 Endpoint Info

Store access control policy for each endpoint:

```
EndpointInfo
├── Method: string                      # HTTP method
├── Path: string                        # Path pattern
├── Policy: EndpointPolicy              # allow / deny / require-scopes
├── RequiredScopes: []string            # Required scopes (OR relationship)
├── OwnerOnly: bool                     # Owner only
└── TeamOnly: bool                      # Team only
```

## III. Permission Check Flow

```
Check(method, path, scopes)
  │
  ├─1. Check if public path (O(1))
  │   └─→ Yes: Allow access
  │
  ├─2. Get PathMatcher by method (O(1))
  │   └─→ Not found: Use default policy
  │
  ├─3. Path matching (by priority)
  │   ├─ 3.1 Exact match (O(1))
  │   ├─ 3.2 Parameter match (O(1))
  │   └─ 3.3 Wildcard match (O(n), n is small)
  │
  ├─4. Apply policy based on match result
  │   ├─ PolicyAllow: Allow access
  │   ├─ PolicyDeny: Deny access
  │   └─ PolicyRequireScopes:
  │       │
  │       ├─ 4.1 Expand aliases (if any)
  │       ├─ 4.2 Check if user has any required scope (OR relationship)
  │       ├─ 4.3 Check resource constraints (owner/team)
  │       └─ 4.4 Return decision result
  │
  └─5. Return AccessDecision (with detailed information)
```

## IV. Performance Optimizations

### 4.1 Index Optimization

- **Method Grouping**: Independent indexes for different HTTP methods, reducing search space
- **Multi-layer Matching**: Exact > Parameter > Wildcard, fast location
- **Map Lookup**: O(1) time complexity

### 4.2 Concurrency Optimization

- **Read-Write Lock**: Use `sync.RWMutex` for read-heavy scenarios
- **Non-blocking Reads**: Multiple goroutines can read concurrently
- **Safe Writes**: Acquire write lock when updating configuration

### 4.3 Cache Optimization (Optional, future implementation)

- Can cache recent permission check results
- Use LRU cache to avoid repeated calculations

### 4.4 Path Normalization

- Pre-process path patterns, extract parameter positions
- Sort wildcard paths by prefix length to avoid redundant matching

## V. Configuration Loading Flow

```
Load(config *Config)
  │
  ├─1. Load scopes.yml (global configuration)
  │
  ├─2. Load alias.yml (alias configuration)
  │
  ├─3. Recursively scan subdirectories (kb/, job/, user/, file/)
  │   └─→ Load all *.yml files, parse ScopeDefinition
  │
  ├─4. Build runtime indexes
  │   ├─ 4.1 Process global endpoints rules
  │   ├─ 4.2 Process endpoints for each ScopeDefinition
  │   ├─ 4.3 Build PathMatcher indexes
  │   └─ 4.4 Build scopeIndex and aliasIndex
  │
  ├─5. Set global variable acl.Global
  │
  └─6. Return ScopeManager
```

## VI. Usage Examples

### 6.1 Permission Check

```go
// Parse user information from token
userScopes := []string{"kb:read", "file:own"}
userID := "user123"
teamID := "team456"

// Build access request
request := &AccessRequest{
    Method: "GET",
    Path:   "/kb/collections/abc123",
    Scopes: userScopes,
    UserID: userID,
    TeamID: teamID,
}

// Execute permission check
    decision := acl.Global.Scope.Check(request)

if decision.Allowed {
    // Allow access
} else {
    // Deny access: decision.Reason
    // Missing permissions: decision.MissingScopes
}
```

### 6.2 Gin Middleware Integration

```go
func (acl *ACL) Enforce(c *gin.Context) (bool, error) {
    // Get user information from context
    userScopes := getUserScopes(c)
    userID := getUserID(c)
    teamID := getTeamID(c)

    // Build request
    request := &AccessRequest{
        Method: c.Request.Method,
        Path:   c.Request.URL.Path,
        Scopes: userScopes,
        UserID: userID,
        TeamID: teamID,
    }

    // Check permission
    decision := acl.Scope.Check(request)

    if !decision.Allowed {
        c.JSON(403, gin.H{
            "error": "Access denied",
            "reason": decision.Reason,
            "missing_scopes": decision.MissingScopes,
        })
        return false, nil
    }

    return true, nil
}
```

## VII. Key Issues Handling

### 7.1 Alias Expansion

- Aliases can contain aliases (recursive)
- Need to detect circular references
- Expand and cache during pre-loading

### 7.2 Path Parameter Matching

- `/kb/collections/:id` should match `/kb/collections/abc123`
- Use path normalization: extract `/kb/collections/` prefix, mark parameter positions
- Verify segment count matches during matching

### 7.3 Wildcard Matching

- `/kb/*` should match `/kb/collections` and `/kb/collections/abc123`
- Sort by prefix length: `/kb/collections/*` takes priority over `/kb/*`
- Avoid greedy matching

### 7.4 Concurrent Updates

- Use RWMutex to protect all index structures
- On update: Lock() -> rebuild indexes -> Unlock()
- On read: RLock() -> query -> RUnlock()

## VIII. Future Extensions

### 8.1 Dynamic Updates

- Provide `Reload()` method to reload configuration
- Provide `Update(scope)` method to dynamically add/modify scopes
- Hot updates should not affect ongoing requests

### 8.2 Audit Logging

- Record all permission check results
- Facilitate debugging and security auditing

### 8.3 Performance Monitoring

- Record permission check duration
- Monitor cache hit rate
- Identify performance bottlenecks

### 8.4 More Complex Policies

- AND relationships: require multiple scopes simultaneously
- Conditional expressions: dynamic permissions based on request parameters
- Time restrictions: certain permissions only valid during specific time periods
