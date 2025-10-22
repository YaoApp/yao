# ACL Scopes Configuration Guide

## Overview

This guide explains how to configure and manage ACL (Access Control List) scopes for your OAuth-protected APIs. Scopes define what resources and actions are accessible to different users, teams, and clients.

---

## Directory Structure

All scope configurations should be placed in the `openapi/scopes/` directory with the following structure:

```
openapi/scopes/
├── scopes.yml           # Global configuration and default policies
├── alias.yml            # Scope aliases for simplified permission management
└── <resource>/          # Resource-specific scope definitions
    ├── collections.yml  # Collections resource scopes
    ├── documents.yml    # Documents resource scopes
    └── ...
```

**Organization Guidelines**:

- Group related scopes by resource (e.g., `kb/`, `user/`, `job/`, `file/`)
- Use descriptive filenames matching the resource name
- Keep each file focused on a single resource or logical grouping

---

## Configuration Files

### 1. Global Configuration (`scopes.yml`)

The `scopes.yml` file defines global ACL behavior, public endpoints, and default rules.

#### Structure

```yaml
# Default action for unmatched API endpoints
default: deny # Options: "deny" or "allow"

# Public endpoints (accessible without authentication)
public:
  - GET /user/entry
  - GET /user/entry/captcha
  - POST /user/entry/verify
  - GET /user/teams/invitations/:invitation_id

# Default endpoint rules (can be overridden by specific scopes)
endpoints:
  # Read operations allowed for authenticated users
  - GET /kb/* allow
  - GET /kb/collections allow

  # Write operations require specific scopes
  - POST /kb/* deny
  - PUT /kb/* deny
  - DELETE /kb/* deny
```

#### Fields

| Field       | Type   | Required | Description                                                   |
| ----------- | ------ | -------- | ------------------------------------------------------------- |
| `default`   | string | Yes      | Default policy for unmatched endpoints: `"allow"` or `"deny"` |
| `public`    | array  | No       | List of public endpoints (no authentication required)         |
| `endpoints` | array  | No       | Default endpoint rules (see Endpoint Rules below)             |

#### Endpoint Rules Format

Each endpoint rule can be specified as:

**Simple String Format** (recommended):

```yaml
- GET /api/users allow
- POST /api/users deny
- DELETE /api/users/* deny
```

**Struct Format**:

```yaml
- method: GET
  path: /api/users
  action: allow
```

**Path Patterns**:

- **Exact path**: `/kb/collections` - matches exactly
- **Parameter path**: `/kb/collections/:collectionID` - matches with parameters
- **Wildcard path**: `/kb/*` - matches all paths under `/kb/`

**Best Practices**:

- Set `default: deny` for security (deny by default, allow explicitly)
- List public endpoints explicitly (login, registration, health checks)
- Use wildcards for broad policies, then override with specific scopes
- Order matters: more specific rules should come after general ones

---

### 2. Scope Definitions (Resource Files)

Scope definition files define specific permissions for resources. Each file contains multiple scope definitions.

#### Structure

```yaml
# Scope naming convention: resource:action:level
collections:read:all:
  description: "Read knowledge base for all users"
  endpoints:
    - GET /kb/collections
    - GET /kb/collections/:collectionID
    - GET /kb/collections/:collectionID/exists

collections:read:own:
  owner: true     # Only show collections owned by current user
  creator: true   # Only show collections created by current user
  description: "Read knowledge base for own collections"
  endpoints:
    - GET /kb/collections/own
    - GET /kb/collections/own/:collectionID
    - GET /kb/collections/own/:collectionID/exists

collections:write:own:
  owner: true
  editor: true    # Only allow editing by last editor
  description: "Write knowledge base for own collections"
  endpoints:
    - POST /kb/collections/own
    - PUT /kb/collections/own/:collectionID
    - DELETE /kb/collections/own/:collectionID

collections:read:team:
  team: true      # Only show team collections
  description: "Read knowledge base for team collections"
  endpoints:
    - GET /kb/collections/team
    - GET /kb/collections/team/:collectionID

collections:read:department:
  extra:          # Custom constraints
    department_only: true
    region: "us-west"
  description: "Read collections for department in specific region"
  endpoints:
    - GET /kb/collections/department
    - GET /kb/collections/department/:collectionID
```

#### Scope Definition Fields

| Field         | Type   | Required | Default | Description                                                                        |
| ------------- | ------ | -------- | ------- | ---------------------------------------------------------------------------------- |
| `description` | string | No       | ""      | Human-readable description of the scope                                            |
| `owner`       | bool   | No       | false   | If `true`, data access is restricted to owner only (sets `OwnerOnly` constraint)   |
| `creator`     | bool   | No       | false   | If `true`, data access is restricted to creator only (sets `CreatorOnly` constraint) |
| `editor`      | bool   | No       | false   | If `true`, data access is restricted to editor only (sets `EditorOnly` constraint)   |
| `team`        | bool   | No       | false   | If `true`, data access is restricted to team only (sets `TeamOnly` constraint)     |
| `extra`       | map    | No       | {}      | User-defined custom constraints (key-value pairs)                                  |
| `endpoints`   | array  | Yes      | -       | List of API endpoints this scope grants access to                                  |

#### Endpoint Format

Each endpoint in the `endpoints` array should be formatted as:

```
METHOD /path
```

**Examples**:

```yaml
endpoints:
  - GET /kb/collections
  - GET /kb/collections/:collectionID
  - POST /kb/collections/own
  - PUT /kb/collections/:collectionID
  - DELETE /kb/collections/own/:collectionID
```

**Supported HTTP Methods**:

- `GET` - Read operations
- `POST` - Create operations
- `PUT` - Update operations
- `DELETE` - Delete operations
- `PATCH` - Partial update operations

**Path Parameters**:

- Use `:paramName` syntax for path parameters (e.g., `:collectionID`, `:userID`)
- Parameter names should be descriptive and consistent

---

### 3. Scope Aliases (`alias.yml`)

Aliases allow you to group multiple scopes under a single name for simplified permission management.

#### Structure

```yaml
# Alias naming: category:level
user:auth:
  - entry:access:public
  - entry:register:authenticated
  - entry:logout:own

kb:read:
  - collections:read:all
  - documents:read:all
  - search:read:all
  - hits:read:all

kb:own:
  - collections:read:own
  - collections:write:own
  - collections:delete:own
  - documents:read:own
  - documents:write:own
  - documents:delete:own

kb:admin:
  - collections:read:all
  - collections:write:all
  - collections:delete:all
  - documents:read:all
  - documents:write:all
  - documents:delete:all
  - search:read:all
  - graphs:read:all

# System root permission - absolute highest privilege
system:root:
  - "*:*:*"
```

#### Alias Usage

**In Role Configuration**:

```go
// Assign aliases to roles instead of individual scopes
role := &Role{
    ID: "kb-viewer",
    AllowedScopes: []string{
        "kb:read",      // Expands to all KB read scopes
        "user:auth",    // Expands to all auth scopes
    },
}
```

**Benefits**:

- **Simplified Management**: Change multiple scopes by updating one alias
- **Consistency**: Ensure users get consistent permission sets
- **Readability**: Clear, semantic permission names
- **Maintenance**: Easier to add/remove scopes from permission groups

**Best Practices**:

- Use hierarchical naming: `resource:level` (e.g., `kb:read`, `kb:own`, `kb:admin`)
- Create aliases for common permission patterns
- Document what each alias includes
- Use wildcards (`*:*:*`) sparingly and only for system-level access

---

## Scope Naming Convention

Follow a consistent three-part naming convention for scopes:

```
resource:action:level
```

### Components

1. **Resource** (noun): The resource being accessed

   - Examples: `collections`, `documents`, `profile`, `jobs`, `files`
   - Should be plural for collections, singular for single resources

2. **Action** (verb): The operation being performed

   - `read` - View/retrieve data (GET)
   - `write` - Create/update data (POST, PUT, PATCH)
   - `delete` - Remove data (DELETE)
   - `control` - Special operations (start, stop, pause)
   - `access` - Generic access without CRUD semantics

3. **Level** (scope): The access level or data visibility
   - `all` - Full access to all resources
   - `own` - Access only to user's own resources
   - `team` - Access to team resources
   - `public` - Public/unauthenticated access
   - `authenticated` - Basic authenticated access

### Examples

| Scope                    | Description                   |
| ------------------------ | ----------------------------- |
| `collections:read:all`   | Read all collections          |
| `collections:read:own`   | Read only own collections     |
| `collections:read:team`  | Read team collections         |
| `collections:write:own`  | Create/update own collections |
| `collections:delete:own` | Delete own collections        |
| `documents:write:all`    | Create/update any document    |
| `documents:delete:team`  | Delete team documents         |
| `profile:read:own`       | Read own profile              |
| `jobs:control:own`       | Control (start/stop) own jobs |
| `search:read:all`        | Search across all resources   |

---

## Data Access Constraints

Data access constraints control how API handlers should filter data based on ownership.

### Owner-Only Access (`owner: true`)

When `owner: true` is set, the scope grants access only to resources owned by the current user.

```yaml
collections:read:own:
  owner: true
  description: "Read knowledge base for own collections"
  endpoints:
    - GET /kb/collections/own
    - GET /kb/collections/own/:collectionID
```

**API Implementation**:

```go
func GetCollections(c *gin.Context) {
    authInfo := authorized.GetInfo(c)

    query := db.Query("SELECT * FROM collections")

    // Apply owner constraint
    if authInfo.Constraints.OwnerOnly {
        query = query.Where("user_id = ?", authInfo.UserID)
    }

    collections, _ := query.Get()
    c.JSON(200, collections)
}
```

### Team-Only Access (`team: true`)

When `team: true` is set, the scope grants access only to resources owned by the current team.

```yaml
collections:read:team:
  team: true
  description: "Read knowledge base for team collections"
  endpoints:
    - GET /kb/collections/team
    - GET /kb/collections/team/:collectionID
```

**API Implementation**:

```go
func GetCollections(c *gin.Context) {
    authInfo := authorized.GetInfo(c)

    query := db.Query("SELECT * FROM collections")

    // Apply team constraint
    if authInfo.Constraints.TeamOnly {
        query = query.Where("team_id = ?", authInfo.TeamID)
    }

    collections, _ := query.Get()
    c.JSON(200, collections)
}
```

### Combined Constraints

Both constraints can be applied:

```yaml
documents:read:own:
  owner: true
  team: true # Can be used together
  description: "Read own documents within team context"
  endpoints:
    - GET /kb/documents/own
```

**API Implementation**:

```go
func GetDocuments(c *gin.Context) {
    authInfo := authorized.GetInfo(c)

    query := db.Query("SELECT * FROM documents")

    // Apply constraints (OwnerOnly is more restrictive)
    if authInfo.Constraints.OwnerOnly {
        query = query.Where("user_id = ?", authInfo.UserID)
    } else if authInfo.Constraints.TeamOnly {
        query = query.Where("team_id = ?", authInfo.TeamID)
    }

    documents, _ := query.Get()
    c.JSON(200, documents)
}
```

---

## Complete Example

Let's create a complete scope configuration for a blog system.

### Directory Structure

```
openapi/scopes/
├── scopes.yml
├── alias.yml
└── blog/
    ├── posts.yml
    ├── comments.yml
    └── categories.yml
```

### `scopes.yml`

```yaml
default: deny

public:
  - GET /blog/posts
  - GET /blog/posts/:postID
  - GET /blog/categories

endpoints:
  # Read operations allowed for authenticated users
  - GET /blog/* allow

  # Write operations require specific scopes
  - POST /blog/* deny
  - PUT /blog/* deny
  - DELETE /blog/* deny
```

### `blog/posts.yml`

```yaml
posts:read:all:
  description: "Read all blog posts"
  endpoints:
    - GET /blog/posts
    - GET /blog/posts/:postID

posts:read:own:
  owner: true
  description: "Read own blog posts"
  endpoints:
    - GET /blog/posts/own
    - GET /blog/posts/own/:postID

posts:write:own:
  owner: true
  description: "Create and update own blog posts"
  endpoints:
    - POST /blog/posts
    - PUT /blog/posts/:postID
    - PATCH /blog/posts/:postID

posts:delete:own:
  owner: true
  description: "Delete own blog posts"
  endpoints:
    - DELETE /blog/posts/:postID

posts:write:all:
  description: "Create and update any blog post (admin)"
  endpoints:
    - POST /blog/posts/admin
    - PUT /blog/posts/admin/:postID

posts:delete:all:
  description: "Delete any blog post (admin)"
  endpoints:
    - DELETE /blog/posts/admin/:postID
```

### `blog/comments.yml`

```yaml
comments:read:all:
  description: "Read all comments"
  endpoints:
    - GET /blog/posts/:postID/comments
    - GET /blog/comments/:commentID

comments:write:own:
  owner: true
  description: "Write own comments"
  endpoints:
    - POST /blog/posts/:postID/comments
    - PUT /blog/comments/:commentID

comments:delete:own:
  owner: true
  description: "Delete own comments"
  endpoints:
    - DELETE /blog/comments/:commentID

comments:delete:all:
  description: "Delete any comment (moderator)"
  endpoints:
    - DELETE /blog/comments/admin/:commentID
```

### `alias.yml`

```yaml
# Blog reader - can read all posts and comments
blog:reader:
  - posts:read:all
  - comments:read:all

# Blog author - can manage own posts and comments
blog:author:
  - posts:read:all
  - posts:write:own
  - posts:delete:own
  - comments:read:all
  - comments:write:own
  - comments:delete:own

# Blog moderator - can manage all comments
blog:moderator:
  - posts:read:all
  - comments:read:all
  - comments:delete:all

# Blog admin - full access to all blog features
blog:admin:
  - posts:read:all
  - posts:write:all
  - posts:delete:all
  - comments:read:all
  - comments:write:own
  - comments:delete:all
```

---

## Wildcard Scopes

Wildcard scopes allow flexible permission matching using `*` as a placeholder.

### Syntax

```yaml
system:root:
  - "*:*:*" # Matches everything

blog:admin:
  - "posts:*:*" # Matches all post operations at all levels
  - "comments:*:*" # Matches all comment operations at all levels

kb:read:
  - "collections:read:*" # Matches collections:read:all, collections:read:own, etc.
  - "documents:read:*" # Matches documents:read:all, documents:read:own, etc.
```

### Matching Rules

1. **Full wildcard** (`*:*:*`): Matches any scope
2. **Resource wildcard** (`posts:*:*`): Matches any action and level for the resource
3. **Action wildcard** (`posts:read:*`): Matches any level for the resource and action
4. **No partial wildcards**: `post*:read:all` is NOT supported

### Use Cases

- **System root access**: `*:*:*` for system administrators
- **Resource administrators**: `resource:*:*` for resource-level admins
- **Grouped permissions**: `resource:action:*` for action-level permissions

### Security Considerations

- Use wildcards sparingly
- Prefer explicit scope lists for most roles
- Reserve `*:*:*` for system-level accounts only
- Document wildcard usage clearly
- Consider restricted scopes to block specific actions even with wildcards

---

## Best Practices

### 1. Scope Design

✅ **DO**:

- Use consistent naming conventions
- Group related scopes in the same file
- Provide clear descriptions for each scope
- Design scopes around resources and actions, not UI features
- Keep scopes granular but not too fine-grained

❌ **DON'T**:

- Mix different resources in one scope file
- Create scopes for every single endpoint
- Use vague or inconsistent naming
- Duplicate endpoint definitions across scopes

### 2. Permission Levels

Create a clear hierarchy of permission levels:

1. **Public** (`public`): No authentication required
2. **Authenticated** (`authenticated`): Basic logged-in access
3. **Owner** (`own`): User's own resources
4. **Team** (`team`): Team's resources
5. **All** (`all`): All resources (admin level)

### 3. Aliases

✅ **DO**:

- Create aliases for common user roles (viewer, editor, admin)
- Use aliases to group related scopes
- Document what each alias grants
- Keep alias names intuitive

❌ **DON'T**:

- Create single-scope aliases (use the scope directly)
- Nest aliases (aliases should reference scopes, not other aliases)
- Use ambiguous alias names

### 4. Data Constraints

✅ **DO**:

- Set `owner: true` for personal resource scopes
- Set `team: true` for team resource scopes
- Implement constraint checks in ALL relevant API handlers
- Return appropriate errors when constraints are violated

❌ **DON'T**:

- Rely solely on URL paths (`/own`, `/team`) for access control
- Skip constraint validation in database queries
- Assume constraints are enforced automatically

### 5. Endpoint Definitions

✅ **DO**:

- List all related endpoints for a scope
- Use consistent parameter naming (`:id`, `:userID`, `:collectionID`)
- Include all HTTP methods the scope covers
- Group similar endpoints together

❌ **DON'T**:

- Define the same endpoint in multiple scopes (unless intentional)
- Use inconsistent path formats
- Forget to include related endpoints

### 6. Testing

- Test each scope definition with real requests
- Verify data constraints are enforced correctly
- Test wildcard matching behavior
- Ensure public endpoints are accessible without auth
- Validate that denied endpoints return proper errors

### 7. Documentation

- Comment complex scope definitions
- Document the purpose of each alias
- Maintain a scope reference for developers
- Update documentation when scopes change
- Provide examples of scope usage in roles

---

## Troubleshooting

### Common Issues

**Issue**: Endpoint not accessible even with correct scope

**Solution**:

- Check if endpoint is in `scopes.yml` default deny list
- Verify scope name matches exactly (case-sensitive)
- Ensure endpoint path matches (check for typos, extra slashes)
- Verify HTTP method matches

---

**Issue**: Data constraint not working

**Solution**:

- Confirm `owner: true` or `team: true` is set in scope definition
- Check if API handler reads `authInfo.Constraints`
- Verify database query applies constraint filters
- Ensure `authInfo.UserID` or `authInfo.TeamID` is populated

---

**Issue**: Wildcard scope not matching

**Solution**:

- Verify wildcard syntax (`*` in correct position)
- Check scope name format (must be `part1:part2:part3`)
- Ensure no typos in scope name parts
- Remember: wildcards only work with colon-separated scopes

---

**Issue**: Changes not taking effect

**Solution**:

- Restart the application to reload scope configurations
- Clear role cache: `role.RoleManager.ClearCache()`
- Verify YAML syntax is correct (use YAML validator)
- Check file is in correct directory

---

## Reference

### Related Files

- **[types.go](./types.go)**: Scope configuration structures
- **[scope.go](./scope.go)**: Scope matching and validation logic
- **[README.md](./README.md)**: ACL enforcement logic
- **[DESIGN.md](./DESIGN.md)**: Overall ACL system design

### Related Concepts

- **OAuth 2.1 Scopes**: Standard OAuth scope mechanism
- **RBAC**: Role-Based Access Control
- **Data Constraints**: Fine-grained data access control
- **Endpoint Matching**: Path pattern matching algorithm

---

## Migration Guide

### From Legacy Permissions

If migrating from a legacy permission system:

1. **Map old permissions to scopes**:

   ```
   can_read_posts → posts:read:all
   can_edit_own_posts → posts:write:own
   can_delete_any_post → posts:delete:all
   ```

2. **Create scope definitions** for each permission

3. **Define aliases** for existing roles:

   ```yaml
   role:editor:
     - posts:read:all
     - posts:write:own
     - posts:delete:own
   ```

4. **Update API handlers** to check constraints

5. **Migrate role assignments** to use new scopes/aliases

6. **Test thoroughly** before deploying

### Version Compatibility

- **v1.0**: Basic scope checking
- **v1.1**: Data constraints (`owner`, `team`)
- **v1.2**: Wildcard scopes, restricted scopes

---

## Summary

Key points to remember:

1. **Three main files**: `scopes.yml` (global), `alias.yml` (aliases), resource files (scopes)
2. **Naming convention**: `resource:action:level`
3. **Data constraints**: Use `owner: true` and `team: true` for data filtering
4. **Aliases**: Group scopes for easier role management
5. **Wildcards**: Use `*` for flexible matching, but sparingly
6. **Testing**: Always test scope configurations thoroughly

For more details, refer to:

- [README.md](./README.md) - Enforcement logic
- [DESIGN.md](./DESIGN.md) - System architecture
