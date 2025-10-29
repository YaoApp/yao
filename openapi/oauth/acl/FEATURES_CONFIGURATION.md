# ACL Features Configuration Guide

## Overview

This guide explains how to configure and manage feature definitions for your application. Features define what functionality is available to different roles, providing a feature flag system that allows the frontend to dynamically show or hide UI elements based on role permissions.

---

## Directory Structure

All feature configurations should be placed in the `openapi/features/` directory with the following structure:

```
openapi/features/
├── features.yml        # Role to features mapping
├── alias.yml          # Feature aliases (groups of features)
└── <domain>/          # Domain-specific feature definitions
    ├── profile.yml    # User profile features
    ├── team.yml       # Team management features
    ├── <subdomain>/   # Nested subdomain (supports unlimited depth)
    │   ├── members.yml
    │   └── <deep-subdomain>/
    │       └── settings.yml
    └── ...
```

**Organization Guidelines**:

- Group related features by domain (e.g., `user/`, `team/`, `kb/`)
- Use descriptive filenames matching the domain name
- Keep each file focused on a single domain or logical grouping
- Each directory represents a domain that can be queried separately
- **Supports nested directories** for hierarchical organization (e.g., `user/team/members.yml` → domain `user/team`)
- Nested domain names use forward slashes as separators (e.g., `user/team`, `kb/collections/advanced`)

---

## Configuration Files

### 1. Role Features Mapping (`features.yml`)

The `features.yml` file defines which features are available to each role.

#### Structure

```yaml
# Role ID to features mapping
# Features can be aliases or actual feature names

# ============ System Roles ============

# System Root - Super administrator with all features
system:root:
  - "*:*:*"

# System Admin - Platform administrator
system:admin:
  - "*:*:*"

# ============ Owner Roles (User Login) ============

# Owner Free - Free tier account owner
owner:free:
  - profile:manage
  - team:manage
  - collections:create

# Owner Pro - Professional tier account owner
owner:pro:
  - user:full

# Owner Enterprise - Enterprise tier account owner
owner:ent:
  - user:full

# ============ Team Roles (Team Login) ============

# Team Admin - Team administrator with full team management
team:admin:
  - profile:manage
  - team:manage
  - collections:create

# Team Member - Standard team member with basic features
team:member:
  - profile:manage
  - collections:create
```

#### Fields

| Field   | Type  | Required | Description                                                     |
| ------- | ----- | -------- | --------------------------------------------------------------- |
| Role ID | array | Yes      | List of features (can include aliases and actual feature names) |

#### Wildcard Support

- `*:*:*` - Grants all features (use for system administrators only)
- Future support for partial wildcards may be added

**Best Practices**:

- Use aliases for common feature groups
- Use wildcards only for system-level roles
- Keep role definitions organized by category (system, owner, team)
- Document each role's purpose with comments

---

### 2. Feature Definitions (Domain Files)

Feature definition files define specific features within a domain. Each file contains multiple feature definitions.

#### Structure

```yaml
# user/profile.yml
profile:read:
  description: "Read own profile"

profile:edit:
  description: "Edit own profile"
```

```yaml
# user/team.yml
team:edit:
  description: "Edit team information"

team:member:invite:
  description: "Invite team members"

team:member:robot:create:
  description: "Create robot team members"

team:member:robot:edit:
  description: "Edit robot team members"

team:member:remove:
  description: "Remove team members"
```

```yaml
# kb/collections.yml
collections:create:
  description: "Create knowledge base collections"
```

#### Feature Definition Fields

| Field         | Type   | Required | Default | Description                               |
| ------------- | ------ | -------- | ------- | ----------------------------------------- |
| `description` | string | Yes      | ""      | Human-readable description of the feature |

#### Feature Naming Convention

Use descriptive, colon-separated names that indicate the feature's purpose:

```
resource:action
```

or

```
resource:subresource:action
```

**Examples**:

- `profile:read` - View profile
- `profile:edit` - Edit profile
- `team:edit` - Edit team
- `team:member:invite` - Invite team members
- `collections:create` - Create collections

**Important**: Feature names are independent of domain names. The domain is determined by the **file path** (relative to `openapi/features/`, without `.yml` extension), not by the feature names defined within the file.

---

### 3. Feature Aliases (`alias.yml`)

Aliases allow you to group multiple features under a single name for simplified role assignment.

#### Structure

```yaml
# Feature Aliases - Groups of related features

# ============ Profile Feature Aliases ============

profile:manage:
  - profile:read
  - profile:edit

# ============ Team Feature Aliases ============

team:manage:
  - team:edit
  - team:member:invite
  - team:member:robot:create
  - team:member:robot:edit
  - team:member:remove

team:member:manage:
  - team:member:invite
  - team:member:robot:create
  - team:member:robot:edit
  - team:member:remove

# ============ Knowledge Base Feature Aliases ============

kb:manage:
  - collections:create

# ============ Combined Feature Bundles ============

user:basic:
  - profile:read
  - profile:edit

user:full:
  - profile:manage
  - team:manage
  - kb:manage

admin:full:
  - profile:manage
  - team:manage
  - kb:manage
```

#### Alias Usage

**In Role Configuration**:

```yaml
# Use aliases instead of listing individual features
owner:pro:
  - user:full # Expands to all features in user:full alias

team:admin:
  - profile:manage # Expands to profile:read + profile:edit
  - team:manage # Expands to all team management features
```

**Benefits**:

- **Simplified Management**: Change multiple features by updating one alias
- **Consistency**: Ensure roles get consistent feature sets
- **Readability**: Clear, semantic feature group names
- **Maintenance**: Easier to add/remove features from groups

**Best Practices**:

- Use hierarchical naming: `domain:level` (e.g., `user:basic`, `user:full`)
- Create aliases for common feature patterns
- Document what each alias includes
- Aliases can reference other aliases (they will be recursively expanded)

---

## Domain-Based Querying

The feature system is designed to support efficient domain-based queries, allowing the frontend to request only the features relevant to a specific section of the application.

### Available Domains

Based on directory structure:

```
openapi/features/
├── user/
│   ├── profile.yml                → domain: "user/profile"
│   └── team/
│       ├── settings.yml           → domain: "user/team/settings"
│       └── members.yml            → domain: "user/team/members"
└── kb/
    ├── collections.yml            → domain: "kb/collections"
    └── collections/
        ├── basic.yml              → domain: "kb/collections/basic"
        └── document/
            ├── meta.yml           → domain: "kb/collections/document/meta"
            └── content.yml        → domain: "kb/collections/document/content"
```

**Query examples**:

| Query Domain                     | Returns Features From                                                                                                  |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `"user"`                         | All files: `user/profile`, `user/team/settings`, `user/team/members`                                                   |
| `"user/team"`                    | All files: `user/team/settings`, `user/team/members`                                                                   |
| `"user/team/members"`            | Only file: `user/team/members` (specific file)                                                                         |
| `"user/profile"`                 | Only file: `user/profile` (specific file)                                                                              |
| `"kb"`                           | All files: `kb/collections`, `kb/collections/basic`, `kb/collections/document/meta`, `kb/collections/document/content` |
| `"kb/collections"`               | All files: `kb/collections`, `kb/collections/basic`, `kb/collections/document/meta`, `kb/collections/document/content` |
| `"kb/collections/document"`      | All files: `kb/collections/document/meta`, `kb/collections/document/content`                                           |
| `"kb/collections/document/meta"` | Only file: `kb/collections/document/meta` (specific file)                                                              |

### Query Methods

```go
// Get all features for a role
features := featureManager.Features("owner:free")
// Returns: map[string]bool{
//   "profile:read": true,
//   "profile:edit": true,
//   "team:edit": true,
//   "team:member:invite": true,
//   ...
// }

// Get features by domain (includes nested subdomains)
userFeatures := featureManager.FeaturesByDomain("owner:free", "user")
// Returns features from "user" AND all nested domains like "user/profile", "user/team/settings", etc.
// Returns: map[string]bool{
//   "profile:read": true,       // from user/profile.yml (domain: user/profile)
//   "profile:edit": true,       // from user/profile.yml (domain: user/profile)
//   "team:view": true,          // from user/team/settings.yml (domain: user/team/settings)
//   "team:edit": true,          // from user/team/settings.yml (domain: user/team/settings)
//   "team:member:invite": true, // from user/team/members.yml (domain: user/team/members)
//   ...
// }

// Get features by specific file/domain
profileFeatures := featureManager.FeaturesByDomain("owner:free", "user/profile")
// Returns only features from "user/profile.yml"
// Returns: map[string]bool{
//   "profile:read": true,
//   "profile:edit": true,
//   ...
// }

// Get features by nested domain prefix
teamFeatures := featureManager.FeaturesByDomain("owner:free", "user/team")
// Returns features from all "user/team/*" domains (hierarchical match)
// Returns: map[string]bool{
//   "team:view": true,          // from user/team/settings.yml (domain: user/team/settings)
//   "team:edit": true,          // from user/team/settings.yml (domain: user/team/settings)
//   "team:member:invite": true, // from user/team/members.yml (domain: user/team/members)
//   ...
// }

// Get all features in a specific domain (exact match only)
allProfileFeatures := featureManager.DomainFeatures("user/profile")
// Returns only features in the exact "user/profile" domain (from user/profile.yml)
// Does NOT include nested subdomains
// Returns: map[string]bool{
//   "profile:read": true,
//   "profile:edit": true,
//   ...
// }

// Get all domains
domains := featureManager.Domains()
// Returns: []string{"user/profile", "user/team/settings", "user/team/members", "kb/collections", "kb/collections/basic", "kb/collections/document/meta", "kb/collections/document/content"}
```

### Backend API Integration

Create API endpoints that use the package-level functions to return features:

```go
// In your API router setup
func SetupFeatureRoutes(router *gin.Engine) {
    api := router.Group("/api/v1/features")
    {
        // Get all features for current user
        api.GET("/", func(c *gin.Context) {
            features, err := acl.GetFeatures(c)
            if err != nil {
                c.JSON(500, gin.H{"error": err.Error()})
                return
            }
            c.JSON(200, gin.H{"features": features})
        })

        // Get features by domain
        api.GET("/:domain", func(c *gin.Context) {
            domain := c.Param("domain")
            features, err := acl.GetFeaturesByDomain(c, domain)
            if err != nil {
                c.JSON(500, gin.H{"error": err.Error()})
                return
            }
            c.JSON(200, gin.H{"features": features})
        })
    }
}
```

### Frontend Usage

The frontend can query features from the API and dynamically show/hide UI elements:

```javascript
// Fetch all features for current user (from API endpoint using acl.GetFeatures)
const response = await fetch("/api/v1/features", {
  headers: {
    Authorization: `Bearer ${token}`,
  },
});
const { features } = await response.json();

// Check if feature is available (O(1) lookup)
if (features["profile:edit"]) {
  // Show edit profile button
  showEditButton();
}

if (features["team:member:invite"]) {
  // Show invite members button
  showInviteButton();
}

// Query features by domain for specific page
// This will include all nested subdomains automatically
const userResponse = await fetch("/api/v1/features/user", {
  headers: {
    Authorization: `Bearer ${token}`,
  },
});
const { features: userFeatures } = await userResponse.json();
// userFeatures includes: user/profile, user/team/settings, user/team/members, etc.

// Query specific file domain
const profileResponse = await fetch("/api/v1/features/user/profile", {
  headers: {
    Authorization: `Bearer ${token}`,
  },
});
const { features: profileFeatures } = await profileResponse.json();
// profileFeatures only includes features from user/profile.yml

// Query nested domain prefix
const teamResponse = await fetch("/api/v1/features/user/team", {
  headers: {
    Authorization: `Bearer ${token}`,
  },
});
const { features: teamFeatures } = await teamResponse.json();
// teamFeatures includes: user/team/settings, user/team/members, etc.

// Render UI based on available features
renderTeamManagementUI(teamFeatures);

// React example with hooks
function UserProfilePage() {
  const [features, setFeatures] = useState({});

  useEffect(() => {
    async function loadFeatures() {
      const response = await fetch("/api/v1/features/user/profile");
      const { features } = await response.json();
      setFeatures(features);
    }
    loadFeatures();
  }, []);

  return (
    <div>
      {features["profile:edit"] && (
        <button onClick={handleEdit}>Edit Profile</button>
      )}
      {features["profile:delete"] && (
        <button onClick={handleDelete}>Delete Account</button>
      )}
    </div>
  );
}
```

---

## Complete Example

Let's create a complete feature configuration for a collaboration platform.

### Directory Structure

```
openapi/features/
├── features.yml
├── alias.yml
├── user/
│   ├── profile.yml                    # domain: user/profile
│   └── team/
│       ├── settings.yml               # domain: user/team/settings
│       └── members.yml                # domain: user/team/members
├── kb/
│   ├── collections.yml                # domain: kb/collections
│   └── collections/
│       ├── basic.yml                  # domain: kb/collections/basic
│       ├── advanced.yml               # domain: kb/collections/advanced
│       └── document/
│           ├── meta.yml               # domain: kb/collections/document/meta
│           └── content.yml            # domain: kb/collections/document/content
└── project/
    ├── boards.yml                     # domain: project/boards
    └── tasks.yml                      # domain: project/tasks
```

### `features.yml`

```yaml
# System Roles
system:root:
  - "*:*:*"

system:admin:
  - "*:*:*"

# Free Tier
owner:free:
  - user:basic
  - project:viewer

# Pro Tier
owner:pro:
  - user:full
  - project:editor

# Enterprise Tier
owner:ent:
  - user:full
  - project:admin

# Team Roles
team:admin:
  - user:full
  - project:admin

team:member:
  - user:basic
  - project:editor
```

### `user/profile.yml`

```yaml
profile:read:
  description: "View own profile information"

profile:edit:
  description: "Edit own profile information"

profile:export:
  description: "Export profile data"

profile:delete:
  description: "Delete own account"
```

### `user/team/settings.yml` (domain: `user/team/settings`)

```yaml
team:view:
  description: "View team information"

team:edit:
  description: "Edit team settings"

team:billing:view:
  description: "View team billing information"

team:billing:edit:
  description: "Manage team billing and subscriptions"
```

### `user/team/members.yml` (domain: `user/team/members`)

```yaml
team:member:invite:
  description: "Invite new team members"

team:member:remove:
  description: "Remove team members"
```

### `project/boards.yml`

```yaml
boards:view:
  description: "View project boards"

boards:create:
  description: "Create new project boards"

boards:edit:
  description: "Edit project boards"

boards:delete:
  description: "Delete project boards"

boards:share:
  description: "Share boards with others"
```

### `project/tasks.yml`

```yaml
tasks:view:
  description: "View tasks"

tasks:create:
  description: "Create new tasks"

tasks:edit:
  description: "Edit tasks"

tasks:delete:
  description: "Delete tasks"

tasks:assign:
  description: "Assign tasks to team members"

tasks:comment:
  description: "Comment on tasks"
```

### `alias.yml`

```yaml
# User Aliases
user:basic:
  - profile:read
  - profile:edit
  - team:view

user:full:
  - profile:read
  - profile:edit
  - profile:export
  - team:view
  - team:edit
  - team:member:invite
  - team:member:remove

user:admin:
  - user:full
  - profile:delete
  - team:billing:view
  - team:billing:edit

# Project Aliases
project:viewer:
  - boards:view
  - tasks:view

project:editor:
  - boards:view
  - boards:create
  - boards:edit
  - tasks:view
  - tasks:create
  - tasks:edit
  - tasks:comment

project:admin:
  - boards:view
  - boards:create
  - boards:edit
  - boards:delete
  - boards:share
  - tasks:view
  - tasks:create
  - tasks:edit
  - tasks:delete
  - tasks:assign
  - tasks:comment
```

---

## Best Practices

### 1. Feature Design

✅ **DO**:

- Use consistent naming conventions across domains
- Group related features in the same domain/file
- Provide clear descriptions for each feature
- Design features around UI functionality, not just API endpoints
- Keep features granular but not too fine-grained

❌ **DON'T**:

- Mix different domains in one feature file
- Create features for every single button (unless the feature genuinely represents a single, critical action)
- Use vague or inconsistent naming
- Duplicate feature definitions across files

### 2. Domain Organization

✅ **DO**:

- Create domains based on application sections (user, team, project, etc.)
- Keep domain names short and meaningful
- Organize features hierarchically within domains
- Use nested directories for logical grouping (e.g., `user/team/`, `kb/collections/document/`)
- Use domains to enable lazy-loading of features
- Leverage hierarchical querying: query parent domain to get all child features
- Unlimited nesting depth is supported for complex structures

❌ **DON'T**:

- Create too many small domains (consolidate related features)
- Use generic domain names like "misc" or "other"
- Mix unrelated features in one domain
- Create unnecessarily deep nesting (keep it reasonable, typically 2-4 levels)

### 3. Aliases

✅ **DO**:

- Create aliases for user tiers (basic, pro, enterprise)
- Create aliases for common roles (viewer, editor, admin)
- Use aliases to group related features
- Document what each alias grants

❌ **DON'T**:

- Create single-feature aliases (use the feature directly)
- Create aliases that are too broad
- Use ambiguous alias names

### 4. Role Assignment

✅ **DO**:

- Use aliases for most role definitions
- Use wildcards (`*:*:*`) only for system roles
- Organize roles by category (system, owner, team)
- Document the purpose of each role

❌ **DON'T**:

- List dozens of individual features per role
- Grant `*:*:*` to non-system roles
- Create too many role variations

### 5. Backend API Integration

✅ **DO**:

- Use `acl.GetFeatures(c)` and `acl.GetFeaturesByDomain(c, domain)` in your API handlers
- Let the context functions automatically determine user vs team member context
- Return features as JSON with `map[string]bool` format
- Handle errors gracefully and return appropriate HTTP status codes
- Use domain-based queries to reduce payload size

❌ **DON'T**:

- Manually extract `__user_id` and `__team_id` from context (use the helper functions)
- Return features as arrays (use map for O(1) lookups on frontend)
- Expose internal role IDs to the frontend
- Query all features when only one domain is needed

### 6. Frontend Integration

✅ **DO**:

- Query features by domain for better performance
- Cache feature results on the frontend
- Use feature flags to show/hide UI elements
- Provide fallbacks for missing features
- Use the API endpoints that leverage `acl.GetFeatures` and `acl.GetFeaturesByDomain`

❌ **DON'T**:

- Query all features when only one domain is needed
- Make feature queries for every component render
- Assume a feature exists without checking
- Store features in insecure locations (use memory/session storage)

### 7. Testing

- Test each role's feature set
- Verify alias expansion works correctly
- Test wildcard matching behavior
- Ensure features query by domain returns correct results
- Validate frontend properly hides/shows UI based on features
- Test `GetFeatures` and `GetFeaturesByDomain` with mock gin.Context
- Verify user vs team member context detection works correctly

### 8. Documentation

- Comment complex feature groupings
- Document the purpose of each alias
- Maintain a feature reference for developers
- Update documentation when features change
- Provide examples of feature usage in frontend

---

## Troubleshooting

### Common Issues

**Issue**: Role has no features

**Solution**:

- Verify role ID matches exactly in `features.yml` (case-sensitive)
- Check if aliases are defined in `alias.yml`
- Ensure feature files exist in domain directories
- Restart application to reload configurations

---

**Issue**: Alias not expanding

**Solution**:

- Check alias definition in `alias.yml`
- Verify feature names in alias are correct
- Check for circular alias references
- Ensure YAML syntax is valid

---

**Issue**: Domain query returns empty

**Solution**:

- Verify domain directory exists
- Check feature files in domain directory are valid YAML
- Ensure features are properly defined with descriptions
- Check that role includes features from that domain

---

**Issue**: Changes not taking effect

**Solution**:

- Restart the application to reload feature configurations
- Verify YAML syntax is correct (use YAML validator)
- Check file is in correct directory
- Clear frontend cache

---

## API Reference

### Package-Level Functions (Gin Context Integration)

These functions automatically extract user/team information from `gin.Context` and return features for the current user or team member:

```go
// GetFeatures returns all features for the current user/team member from gin context
// Automatically determines whether to use user or team member lookup based on context
// Returns a map for O(1) feature lookup: feature_name -> true
//
// Usage in gin handler:
//   func MyHandler(c *gin.Context) {
//       features, err := acl.GetFeatures(c)
//       if err != nil {
//           c.JSON(500, gin.H{"error": err.Error()})
//           return
//       }
//       c.JSON(200, features)
//   }
GetFeatures(c *gin.Context) (map[string]bool, error)

// GetFeaturesByDomain returns features filtered by domain from gin context
// Automatically determines whether to use user or team member lookup based on context
// Supports hierarchical matching: "user" includes "user/profile", "user/team", etc.
// Returns a map for O(1) feature lookup: feature_name -> true
//
// Usage in gin handler:
//   func MyUserPageHandler(c *gin.Context) {
//       features, err := acl.GetFeaturesByDomain(c, "user")
//       if err != nil {
//           c.JSON(500, gin.H{"error": err.Error()})
//           return
//       }
//       c.JSON(200, features)
//   }
GetFeaturesByDomain(c *gin.Context, domain string) (map[string]bool, error)
```

**Context Requirements**:

- `__user_id` (string): Required - The current user's ID
- `__team_id` (string): Optional - If present, queries team member role; otherwise queries user role

**Behavior**:

1. If `__team_id` is present in context → queries member role using `RoleManager.GetMemberRole(teamID, userID)`
2. If `__team_id` is not present → queries user role using `RoleManager.GetUserRole(userID)`
3. Returns empty map if ACL is disabled or role manager is not initialized
4. Returns error if context values are invalid

**Example Integration**:

```go
package api

import (
    "github.com/gin-gonic/gin"
    "github.com/yaoapp/yao/openapi/oauth/acl"
)

// GetUserFeatures returns all features for current user
func GetUserFeatures(c *gin.Context) {
    features, err := acl.GetFeatures(c)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"features": features})
}

// GetUserPageFeatures returns features for user management page
func GetUserPageFeatures(c *gin.Context) {
    features, err := acl.GetFeaturesByDomain(c, "user")
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"features": features})
}

// GetKBPageFeatures returns features for knowledge base page
func GetKBPageFeatures(c *gin.Context) {
    features, err := acl.GetFeaturesByDomain(c, "kb")
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"features": features})
}
```

---

### FeatureManager Methods

These methods require explicit role ID and are used internally or for advanced use cases:

```go
// Features returns all features for a role (expanded)
Features(roleID string) map[string]bool

// FeaturesByDomain returns features filtered by domain
// Supports hierarchical matching: "user" includes "user/team", "user/profile", etc.
FeaturesByDomain(roleID, domain string) map[string]bool

// DomainFeatures returns all features in a specific domain (exact match only)
// Does NOT include nested subdomains
DomainFeatures(domain string) map[string]bool

// Domains returns all available domains (including nested domains)
Domains() []string

// Definition returns detailed info about a feature
Definition(featureName string) *FeatureDefinition

// Reload reloads feature configurations
Reload() error
```

**Convenience Methods (with role resolution)**:

```go
// FeaturesForUser returns all features for a user by user ID
FeaturesForUser(ctx context.Context, userID string) (map[string]bool, error)

// FeaturesForUserByDomain returns features for a user filtered by domain
FeaturesForUserByDomain(ctx context.Context, userID, domain string) (map[string]bool, error)

// FeaturesForTeamUser returns all features for a team member
FeaturesForTeamUser(ctx context.Context, teamID, userID string) (map[string]bool, error)

// FeaturesForTeamUserByDomain returns features for a team member filtered by domain
FeaturesForTeamUserByDomain(ctx context.Context, teamID, userID, domain string) (map[string]bool, error)
```

---

## Integration with Scopes

Features and Scopes work together but serve different purposes:

- **Features**: Control UI visibility and functionality (frontend)
- **Scopes**: Control API access and permissions (backend)

A user might have the feature `team:member:invite` (showing the invite button) and also need the scope `teams:invitations:write` (actually sending invitations).

**Example**:

```yaml
# features.yml
owner:free:
  - team:member:invite # Shows invite button

# scopes/alias.yml
role:owner:free:
  - teams:invitations:write # Allows API calls
```

---

## Summary

Key points to remember:

1. **Three main files**: `features.yml` (roles), `alias.yml` (aliases), domain files (features)
2. **Naming convention**: Use descriptive, colon-separated names
3. **Domain organization**: Group features by application section, supports multi-level nesting
4. **Aliases**: Group features for easier role management
5. **Wildcards**: Use `*:*:*` for system roles only
6. **Return type**: All queries return `map[string]bool` for efficient lookups
7. **Backend API**: Use `acl.GetFeatures(c)` and `acl.GetFeaturesByDomain(c, domain)` in handlers
8. **Context detection**: Automatically handles user vs team member based on `__team_id` presence
9. **Frontend**: Query by domain for better performance, cache results
10. **Hierarchical queries**: Querying parent domain includes all nested subdomains

**Quick Start for Backend Integration**:

```go
// In your API handler
func GetFeatures(c *gin.Context) {
    features, err := acl.GetFeatures(c)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"features": features})
}

func GetDomainFeatures(c *gin.Context) {
    domain := c.Param("domain")
    features, err := acl.GetFeaturesByDomain(c, domain)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"features": features})
}
```

For more details, refer to:

- [SCOPES_CONFIGURATION.md](./SCOPES_CONFIGURATION.md) - Scope permissions
- [README.md](./README.md) - ACL enforcement logic
- [DESIGN.md](./DESIGN.md) - System architecture
