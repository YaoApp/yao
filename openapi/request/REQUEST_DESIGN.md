# OpenAPI Request Design

This document describes the design for global request tracking, billing, rate limiting, and auditing in the YAO OpenAPI layer.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Storage Strategy](#storage-strategy)
- [Data Model](#data-model)
- [Middleware Design](#middleware-design)
- [Rate Limiting](#rate-limiting)
- [Billing Integration](#billing-integration)
- [API Interface](#api-interface)
- [Integration with Services](#integration-with-services)

## Overview

The Request module provides a unified layer for:

1. **Request Tracking** - Record all API requests with unique IDs
2. **Billing** - Track token usage and API calls for billing
3. **Rate Limiting** - Enforce request limits per user/team
4. **Auditing** - Provide audit trail for compliance

### Design Goals

| Goal                | Solution                                         |
| ------------------- | ------------------------------------------------ |
| Unified tracking    | Single middleware for all API endpoints          |
| Accurate billing    | Token usage updated by services after completion |
| Flexible rate limit | Configurable limits per user/team/endpoint       |
| Low overhead        | KV for real-time, SQL for archive                |

### Scope

| In Scope                 | Out of Scope                   |
| ------------------------ | ------------------------------ |
| All `/api/*` endpoints   | Static file serving            |
| Token usage tracking     | Detailed request/response logs |
| Rate limiting            | Request body storage           |
| Request duration metrics | Response caching               |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     HTTP Request                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     OAuth Guard                              │
│  - Token validation                                          │
│  - Set AuthorizedInfo in context                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Request Middleware                          │
│  - Generate request_id                                       │
│  - KV: Rate limit check                                      │
│  - KV: Quota check                                           │
│  - KV: Request status tracking                               │
│  - Async: Archive to SQL                                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Service Handlers                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │  Agent  │  │   KB    │  │   LLM   │  │  File   │  ...   │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘        │
│       │                                                      │
│       └── Update token usage via request_id                 │
└─────────────────────────────────────────────────────────────┘
```

## Storage Strategy

### Two-Layer Storage

| Layer          | Storage  | Purpose                      | TTL       |
| -------------- | -------- | ---------------------------- | --------- |
| **Real-time**  | KV/Redis | Rate limiting, quota, status | 1h - 7d   |
| **Persistent** | SQL      | Archive, billing, audit      | Permanent |

### Why Hybrid?

| Scenario          | KV (Redis)        | SQL              |
| ----------------- | ----------------- | ---------------- |
| Rate limit check  | ⚡ < 1ms          | ❌ Too slow      |
| Quota check       | ⚡ < 1ms          | ❌ Too slow      |
| Request status    | ⚡ Fast update    | ❌ Too slow      |
| Billing report    | ❌ No aggregation | ✅ SUM/GROUP BY  |
| Audit query       | ❌ No persistence | ✅ Full history  |
| Complex filtering | ❌ Key-only       | ✅ WHERE clauses |

### KV Keys Design

```
# Rate Limiting (TTL: 60s)
ratelimit:user:{user_id}:{service}     → count
ratelimit:team:{team_id}:{service}     → count
ratelimit:ip:{ip}                      → count

# Request Status (TTL: 1h)
request:{request_id}                   → {status, service, created_at, ...}

# Token Usage - Daily (TTL: 7d)
tokens:user:{user_id}:{YYYY-MM-DD}     → {input, output, total}
tokens:team:{team_id}:{YYYY-MM-DD}     → {input, output, total}

# Quota (TTL: 24h for daily, 30d for monthly)
quota:user:{user_id}:daily             → remaining_tokens
quota:team:{team_id}:monthly           → remaining_tokens
```

### Data Flow

```
Request arrives
    │
    ├── 1. KV: Rate limit check
    │       INCR ratelimit:user:{id}:{service}
    │       if > limit → 429 Too Many Requests
    │
    ├── 2. KV: Quota check
    │       GET quota:user:{id}:daily
    │       if <= 0 → 429 Quota Exceeded
    │
    ├── 3. KV: Record request status
    │       SET request:{id} {status: "running", ...} EX 3600
    │
    ├── 4. Execute request...
    │
    ├── 5. KV: Update tokens
    │       HINCRBY tokens:user:{id}:{date} input {n}
    │       HINCRBY tokens:user:{id}:{date} output {n}
    │       DECRBY quota:user:{id}:daily {total}
    │
    ├── 6. KV: Update request status
    │       SET request:{id} {status: "completed", duration_ms: ...}
    │
    └── 7. Async: Archive to SQL
            INSERT INTO openapi_request ...

```

## Data Model

### KV Data Structures

#### Rate Limit Counter

```go
// Key: ratelimit:{type}:{id}:{service}
// Value: integer count
// TTL: 60 seconds (sliding window)

type RateLimitKey struct {
    Type    string // "user", "team", "ip"
    ID      string // user_id, team_id, or IP
    Service string // "agent", "kb", "llm", etc.
}

func (k RateLimitKey) String() string {
    return fmt.Sprintf("ratelimit:%s:%s:%s", k.Type, k.ID, k.Service)
}
```

#### Request Status

```go
// Key: request:{request_id}
// Value: JSON object
// TTL: 1 hour

type RequestStatus struct {
    RequestID   string    `json:"request_id"`
    UserID      string    `json:"user_id"`
    TeamID      string    `json:"team_id,omitempty"`
    Service     string    `json:"service"`
    ResourceID  string    `json:"resource_id,omitempty"`
    Status      string    `json:"status"` // running, completed, failed
    CreatedAt   time.Time `json:"created_at"`
    CompletedAt time.Time `json:"completed_at,omitempty"`
    DurationMs  int64     `json:"duration_ms,omitempty"`
    Error       string    `json:"error,omitempty"`
}
```

#### Token Usage (Daily)

```go
// Key: tokens:{type}:{id}:{date}
// Value: Hash {input, output, total}
// TTL: 7 days

type TokenUsage struct {
    Input  int64 `json:"input"`
    Output int64 `json:"output"`
    Total  int64 `json:"total"`
}
```

#### Quota

```go
// Key: quota:{type}:{id}:{period}
// Value: remaining tokens (integer)
// TTL: 24h (daily) or 30d (monthly)

type QuotaKey struct {
    Type   string // "user", "team"
    ID     string
    Period string // "daily", "monthly"
}
```

### SQL Table (Archive)

**Table Name:** `openapi_request`

**Purpose:** Long-term storage for billing reports, audit logs, and analytics.

| Column          | Type        | Nullable | Index  | Description                                      |
| --------------- | ----------- | -------- | ------ | ------------------------------------------------ |
| `id`            | ID          | No       | PK     | Auto-increment primary key                       |
| `request_id`    | string(64)  | No       | Unique | Unique request identifier                        |
| `user_id`       | string(200) | No       | Yes    | User ID from auth                                |
| `team_id`       | string(200) | Yes      | Yes    | Team ID from auth                                |
| `session_id`    | string(200) | Yes      | Yes    | Session ID                                       |
| `endpoint`      | string(200) | No       | Yes    | API endpoint path                                |
| `method`        | string(10)  | No       | -      | HTTP method (GET, POST, etc.)                    |
| `service`       | string(50)  | No       | Yes    | Service type: `agent`, `kb`, `llm`, `file`, etc. |
| `resource_id`   | string(200) | Yes      | Yes    | Resource ID (assistant_id, collection_id, etc.)  |
| `status`        | enum        | No       | Yes    | `pending`, `running`, `completed`, `failed`      |
| `status_code`   | integer     | Yes      | -      | HTTP response status code                        |
| `referer`       | string(50)  | Yes      | -      | Request source (api, jssdk, agent, etc.)         |
| `client_type`   | string(50)  | Yes      | -      | Client type (web, ios, android, etc.)            |
| `client_ip`     | string(50)  | Yes      | Yes    | Client IP address                                |
| `input_tokens`  | integer     | Yes      | -      | Input token count (LLM calls)                    |
| `output_tokens` | integer     | Yes      | -      | Output token count (LLM calls)                   |
| `total_tokens`  | integer     | Yes      | Yes    | Total token count                                |
| `duration_ms`   | integer     | Yes      | Yes    | Request duration in milliseconds                 |
| `error`         | text        | Yes      | -      | Error message if failed                          |
| `metadata`      | json        | Yes      | -      | Additional metadata                              |
| `created_at`    | timestamp   | No       | Yes    | Request start time                               |
| `completed_at`  | timestamp   | Yes      | Yes    | Request completion time                          |

**Indexes:**

| Name               | Columns                                 | Type  | Purpose                  |
| ------------------ | --------------------------------------- | ----- | ------------------------ |
| `idx_req_user`     | `user_id`, `created_at`                 | index | User request history     |
| `idx_req_team`     | `team_id`, `created_at`                 | index | Team request history     |
| `idx_req_endpoint` | `endpoint`, `created_at`                | index | Endpoint analytics       |
| `idx_req_service`  | `service`, `created_at`                 | index | Service analytics        |
| `idx_req_status`   | `status`                                | index | Find incomplete requests |
| `idx_req_billing`  | `team_id`, `created_at`, `total_tokens` | index | Billing queries          |
| `idx_req_ip`       | `client_ip`, `created_at`               | index | IP-based rate limiting   |

### Service Types

| Service | Description          | Resource ID Example |
| ------- | -------------------- | ------------------- |
| `agent` | Chat/Agent API       | `assistant_id`      |
| `kb`    | Knowledge Base API   | `collection_id`     |
| `llm`   | Direct LLM API       | `connector_id`      |
| `file`  | File upload/download | `file_id`           |
| `user`  | User management      | `user_id`           |
| `team`  | Team management      | `team_id`           |
| `mcp`   | MCP server calls     | `server_id`         |

### Status Values

| Status      | Description                    | Set By     |
| ----------- | ------------------------------ | ---------- |
| `pending`   | Request received, not started  | Middleware |
| `running`   | Request being processed        | Middleware |
| `completed` | Request completed successfully | Middleware |
| `failed`    | Request failed with error      | Middleware |

## Middleware Design

### Request Flow

```
Request arrives
    │
    ├── 1. Generate request_id (uuid or nanoid)
    │
    ├── 2. Set request_id in context and response header
    │       c.Set("request_id", requestID)
    │       c.Header("X-Request-ID", requestID)
    │
    ├── 3. Get auth info from context (set by OAuth Guard)
    │       authInfo := authorized.GetInfo(c)
    │
    ├── 4. Detect service type from endpoint
    │       service := detectService(c.FullPath())
    │
    ├── 5. Create request record (async)
    │       status = "running"
    │
    ├── 6. Check rate limits
    │       if exceeded → return 429, update status = "failed"
    │
    ├── 7. Execute handler
    │       c.Next()
    │
    └── 8. Update request record (async)
            status = "completed" or "failed"
            duration_ms = time.Since(start)
            status_code = c.Writer.Status()
```

### Implementation

```go
package request

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/yaoapp/yao/openapi/oauth/authorized"
)

// Middleware creates the request tracking middleware
func Middleware(kv KVStore, sql SQLStore) gin.HandlerFunc {
    return func(c *gin.Context) {
        startTime := time.Now()

        // 1. Generate request ID
        requestID := generateRequestID()
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)

        // 2. Get auth info
        authInfo := authorized.GetInfo(c)

        // 3. Detect service and resource
        service := detectService(c.FullPath())
        resourceID := extractResourceID(c, service)

        // 4. KV: Check rate limits (synchronous, must be fast)
        if err := checkRateLimit(kv, authInfo, service, c.ClientIP()); err != nil {
            c.AbortWithStatusJSON(429, gin.H{
                "error":   "rate_limit_exceeded",
                "message": err.Error(),
            })
            return
        }

        // 5. KV: Check quota (synchronous)
        if err := checkQuota(kv, authInfo); err != nil {
            c.AbortWithStatusJSON(429, gin.H{
                "error":   "quota_exceeded",
                "message": err.Error(),
            })
            return
        }

        // 6. KV: Record request status
        reqStatus := &RequestStatus{
            RequestID:  requestID,
            UserID:     authInfo.UserID,
            TeamID:     authInfo.TeamID,
            Service:    service,
            ResourceID: resourceID,
            Status:     "running",
            CreatedAt:  startTime,
        }
        kv.SetRequestStatus(requestID, reqStatus, time.Hour)

        // 7. Execute handler
        c.Next()

        // 8. KV: Update request status
        reqStatus.Status = "completed"
        reqStatus.CompletedAt = time.Now()
        reqStatus.DurationMs = time.Since(startTime).Milliseconds()
        if errMsg := getErrorFromContext(c); errMsg != "" {
            reqStatus.Status = "failed"
            reqStatus.Error = errMsg
        }
        kv.SetRequestStatus(requestID, reqStatus, time.Hour)

        // 9. Async: Archive to SQL
        go func() {
            sql.Archive(&Request{
                RequestID:   requestID,
                UserID:      authInfo.UserID,
                TeamID:      authInfo.TeamID,
                SessionID:   authInfo.SessionID,
                Endpoint:    c.FullPath(),
                Method:      c.Request.Method,
                Service:     service,
                ResourceID:  resourceID,
                Status:      reqStatus.Status,
                StatusCode:  c.Writer.Status(),
                Referer:     c.GetHeader("X-Yao-Referer"),
                ClientType:  getClientType(c.GetHeader("User-Agent")),
                ClientIP:    c.ClientIP(),
                DurationMs:  reqStatus.DurationMs,
                Error:       reqStatus.Error,
                CreatedAt:   startTime,
                CompletedAt: &reqStatus.CompletedAt,
            })
        }()
    }
}

// detectService determines the service type from endpoint
func detectService(endpoint string) string {
    switch {
    case strings.HasPrefix(endpoint, "/api/chat"):
        return ServiceAgent
    case strings.HasPrefix(endpoint, "/api/agent"):
        return ServiceAgent
    case strings.HasPrefix(endpoint, "/api/kb"):
        return ServiceKB
    case strings.HasPrefix(endpoint, "/api/llm"):
        return ServiceLLM
    case strings.HasPrefix(endpoint, "/api/file"):
        return ServiceFile
    case strings.HasPrefix(endpoint, "/api/user"):
        return ServiceUser
    case strings.HasPrefix(endpoint, "/api/team"):
        return ServiceTeam
    case strings.HasPrefix(endpoint, "/api/mcp"):
        return ServiceMCP
    default:
        return ServiceOther
    }
}
```

## Rate Limiting

### Configuration

```yaml
# openapi.yml
rate_limit:
  enabled: true

  # Default limits (requests per minute)
  default:
    per_user: 60
    per_team: 300
    per_ip: 100

  # Service-specific limits
  services:
    agent:
      per_user: 30
      per_team: 150
    llm:
      per_user: 20
      per_team: 100
    kb:
      per_user: 60
      per_team: 300

  # Token limits (per day)
  tokens:
    per_user: 100000
    per_team: 1000000

# quota configuration
quota:
  enabled: true

  # Default quotas
  default:
    user_daily: 100000 # tokens per day
    team_monthly: 10000000 # tokens per month


  # Can be overridden per user/team in database
```

### Rate Limit Check (KV-based)

```go
func checkRateLimit(kv KVStore, authInfo *types.AuthorizedInfo, service, clientIP string) error {
    config := GetRateLimitConfig()
    if !config.Enabled {
        return nil
    }

    // 1. Check per-user limit (INCR with TTL)
    userKey := fmt.Sprintf("ratelimit:user:%s:%s", authInfo.UserID, service)
    userCount, _ := kv.Incr(userKey, 60*time.Second) // TTL 60s
    if userCount > int64(config.GetUserLimit(service)) {
        return fmt.Errorf("user rate limit exceeded: %d requests per minute", config.GetUserLimit(service))
    }

    // 2. Check per-team limit
    if authInfo.TeamID != "" {
        teamKey := fmt.Sprintf("ratelimit:team:%s:%s", authInfo.TeamID, service)
        teamCount, _ := kv.Incr(teamKey, 60*time.Second)
        if teamCount > int64(config.GetTeamLimit(service)) {
            return fmt.Errorf("team rate limit exceeded")
        }
    }

    // 3. Check per-IP limit
    ipKey := fmt.Sprintf("ratelimit:ip:%s", clientIP)
    ipCount, _ := kv.Incr(ipKey, 60*time.Second)
    if ipCount > int64(config.GetIPLimit()) {
        return fmt.Errorf("IP rate limit exceeded")
    }

    return nil
}
```

### Quota Check (KV-based)

```go
func checkQuota(kv KVStore, authInfo *types.AuthorizedInfo) error {
    config := GetQuotaConfig()
    if !config.Enabled {
        return nil
    }

    // Check user daily quota
    userQuotaKey := fmt.Sprintf("quota:user:%s:daily", authInfo.UserID)
    remaining, exists := kv.Get(userQuotaKey)

    if !exists {
        // Initialize quota for the day
        limit := config.GetUserDailyLimit(authInfo.UserID)
        kv.Set(userQuotaKey, limit, 24*time.Hour)
        remaining = limit
    }

    if remaining <= 0 {
        return fmt.Errorf("daily token quota exceeded")
    }

    // Check team monthly quota if applicable
    if authInfo.TeamID != "" {
        teamQuotaKey := fmt.Sprintf("quota:team:%s:monthly", authInfo.TeamID)
        teamRemaining, exists := kv.Get(teamQuotaKey)

        if !exists {
            limit := config.GetTeamMonthlyLimit(authInfo.TeamID)
            kv.Set(teamQuotaKey, limit, 30*24*time.Hour)
            teamRemaining = limit
        }

        if teamRemaining <= 0 {
            return fmt.Errorf("team monthly token quota exceeded")
        }
    }

    return nil
}
```

## Billing Integration

### Token Usage Update

Services update token usage after completion. This updates both KV (real-time) and SQL (archive).

```go
// Called by Agent/LLM services after completion
func UpdateTokenUsage(kv KVStore, sql SQLStore, requestID string, userID, teamID string, input, output int) error {
    total := input + output
    date := time.Now().Format("2006-01-02")

    // 1. KV: Update daily token usage
    userTokenKey := fmt.Sprintf("tokens:user:%s:%s", userID, date)
    kv.HIncrBy(userTokenKey, "input", int64(input))
    kv.HIncrBy(userTokenKey, "output", int64(output))
    kv.HIncrBy(userTokenKey, "total", int64(total))
    kv.Expire(userTokenKey, 7*24*time.Hour) // Keep for 7 days

    if teamID != "" {
        teamTokenKey := fmt.Sprintf("tokens:team:%s:%s", teamID, date)
        kv.HIncrBy(teamTokenKey, "input", int64(input))
        kv.HIncrBy(teamTokenKey, "output", int64(output))
        kv.HIncrBy(teamTokenKey, "total", int64(total))
        kv.Expire(teamTokenKey, 7*24*time.Hour)
    }

    // 2. KV: Deduct from quota
    userQuotaKey := fmt.Sprintf("quota:user:%s:daily", userID)
    kv.DecrBy(userQuotaKey, int64(total))

    if teamID != "" {
        teamQuotaKey := fmt.Sprintf("quota:team:%s:monthly", teamID)
        kv.DecrBy(teamQuotaKey, int64(total))
    }

    // 3. SQL: Update request record (async)
    go sql.UpdateTokens(requestID, input, output)

    return nil
}
```

### Billing Queries

```sql
-- Daily token usage by team
SELECT
    DATE(created_at) as date,
    team_id,
    service,
    SUM(total_tokens) as tokens,
    COUNT(*) as requests
FROM openapi_request
WHERE team_id = ?
  AND created_at >= ? AND created_at < ?
  AND status = 'completed'
GROUP BY DATE(created_at), team_id, service

-- Monthly billing summary
SELECT
    team_id,
    service,
    SUM(total_tokens) as total_tokens,
    SUM(input_tokens) as input_tokens,
    SUM(output_tokens) as output_tokens,
    COUNT(*) as request_count,
    AVG(duration_ms) as avg_duration
FROM openapi_request
WHERE created_at >= ? AND created_at < ?
  AND status = 'completed'
GROUP BY team_id, service

-- User quota check
SELECT SUM(total_tokens) as used
FROM openapi_request
WHERE user_id = ?
  AND created_at >= CURDATE()
  AND status = 'completed'
```

## API Interface

### KV Store Interface

```go
// KVStore defines the KV storage interface for real-time operations
type KVStore interface {
    // Basic operations
    Get(key string) (int64, bool)
    Set(key string, value int64, ttl time.Duration) error
    Incr(key string, ttl time.Duration) (int64, error)
    DecrBy(key string, delta int64) (int64, error)
    Expire(key string, ttl time.Duration) error
    Del(key string) error

    // Hash operations (for token usage)
    HGet(key, field string) (int64, error)
    HSet(key, field string, value int64) error
    HIncrBy(key, field string, delta int64) (int64, error)
    HGetAll(key string) (map[string]int64, error)

    // Request status (JSON)
    SetRequestStatus(requestID string, status *RequestStatus, ttl time.Duration) error
    GetRequestStatus(requestID string) (*RequestStatus, error)
}
```

### SQL Store Interface

```go
// SQLStore defines the SQL storage interface for archiving and analytics
type SQLStore interface {
    // Archive stores a completed request
    Archive(req *Request) error

    // UpdateTokens updates token usage for a request
    UpdateTokens(requestID string, input, output int) error

    // Get retrieves a request by ID
    Get(requestID string) (*Request, error)

    // List lists requests with filters
    List(filter *RequestFilter) (*RequestList, error)

    // GetUsage gets usage statistics
    GetUsage(filter *UsageFilter) (*UsageStats, error)
}
```

### Data Structures

```go
// Request represents an API request record
type Request struct {
    RequestID    string                 `json:"request_id"`
    UserID       string                 `json:"user_id"`
    TeamID       string                 `json:"team_id,omitempty"`
    SessionID    string                 `json:"session_id,omitempty"`
    Endpoint     string                 `json:"endpoint"`
    Method       string                 `json:"method"`
    Service      string                 `json:"service"`
    ResourceID   string                 `json:"resource_id,omitempty"`
    Status       Status                 `json:"status"`
    StatusCode   int                    `json:"status_code,omitempty"`
    Referer      string                 `json:"referer,omitempty"`
    ClientType   string                 `json:"client_type,omitempty"`
    ClientIP     string                 `json:"client_ip,omitempty"`
    InputTokens  int                    `json:"input_tokens,omitempty"`
    OutputTokens int                    `json:"output_tokens,omitempty"`
    TotalTokens  int                    `json:"total_tokens,omitempty"`
    DurationMs   int64                  `json:"duration_ms,omitempty"`
    Error        string                 `json:"error,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt    time.Time              `json:"created_at"`
    CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// CompletionInfo contains info for completing a request
type CompletionInfo struct {
    StatusCode int
    DurationMs int64
    Error      string
}

// RequestFilter for listing requests
type RequestFilter struct {
    UserID    string    `json:"user_id,omitempty"`
    TeamID    string    `json:"team_id,omitempty"`
    Service   string    `json:"service,omitempty"`
    Status    Status    `json:"status,omitempty"`
    StartTime time.Time `json:"start_time,omitempty"`
    EndTime   time.Time `json:"end_time,omitempty"`
    Page      int       `json:"page,omitempty"`
    PageSize  int       `json:"pagesize,omitempty"`
}

// UsageFilter for usage statistics
type UsageFilter struct {
    UserID    string    `json:"user_id,omitempty"`
    TeamID    string    `json:"team_id,omitempty"`
    Service   string    `json:"service,omitempty"`
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time"`
    GroupBy   string    `json:"group_by,omitempty"` // day, week, month
}

// UsageStats contains usage statistics
type UsageStats struct {
    TotalRequests  int64          `json:"total_requests"`
    TotalTokens    int64          `json:"total_tokens"`
    InputTokens    int64          `json:"input_tokens"`
    OutputTokens   int64          `json:"output_tokens"`
    AvgDurationMs  float64        `json:"avg_duration_ms"`
    ByService      map[string]int64 `json:"by_service,omitempty"`
    ByDay          []DailyUsage   `json:"by_day,omitempty"`
}

type DailyUsage struct {
    Date     string `json:"date"`
    Requests int64  `json:"requests"`
    Tokens   int64  `json:"tokens"`
}
```

## Integration with Services

### Agent Service

```go
// agent/context/openapi.go
func GetCompletionRequest(c *gin.Context, cache store.Store) (*CompletionRequest, *Context, *Options, error) {
    // Get request ID from middleware
    requestID := c.GetString("request_id")

    // Create context with request ID
    ctx := New(c.Request.Context(), authInfo, chatID)
    ctx.RequestID = requestID  // Use global request_id

    // ...
}

// agent/assistant/agent.go
func (ast *Assistant) Stream(ctx, inputMessages, options) {
    defer func() {
        // Update token usage in global request record
        if ctx.RequestID != "" && completionResponse != nil && completionResponse.Usage != nil {
            request.UpdateTokenUsage(
                ctx.RequestID,
                completionResponse.Usage.PromptTokens,
                completionResponse.Usage.CompletionTokens,
            )
        }
    }()

    // ...
}
```

### KB Service

```go
// kb/api/search.go
func (api *API) Search(c *gin.Context) {
    requestID := c.GetString("request_id")

    // Perform search...

    // Update metadata if needed
    if requestID != "" {
        request.UpdateMetadata(requestID, map[string]interface{}{
            "results_count": len(results),
            "collection_id": collectionID,
        })
    }
}
```

### Middleware Registration

```go
// openapi/openapi.go
func (s *OpenAPI) RegisterRoutes(r *gin.Engine) {
    api := r.Group("/api")

    // 1. OAuth Guard (authentication)
    api.Use(oauth.Guard)

    // 2. Request Middleware (tracking, rate limiting)
    api.Use(request.Middleware(requestStore))

    // 3. Service routes
    s.registerAgentRoutes(api)
    s.registerKBRoutes(api)
    s.registerLLMRoutes(api)
    // ...
}
```

## Summary

### Components

| Component    | Location                        | Responsibility               |
| ------------ | ------------------------------- | ---------------------------- |
| KV Store     | `openapi/request/kv.go`         | Real-time: rate limit, quota |
| SQL Store    | `openapi/request/sql.go`        | Archive: billing, audit      |
| Middleware   | `openapi/request/middleware.go` | Track requests, orchestrate  |
| Rate Limiter | `openapi/request/ratelimit.go`  | Enforce rate limits          |
| Types        | `openapi/request/types.go`      | Data structures              |

### Storage Comparison

| Operation        | KV (Redis)     | SQL (Archive) |
| ---------------- | -------------- | ------------- |
| Rate limit check | ✅ Synchronous | ❌ Not used   |
| Quota check      | ✅ Synchronous | ❌ Not used   |
| Request status   | ✅ Synchronous | ❌ Not used   |
| Token update     | ✅ Synchronous | ✅ Async      |
| Billing report   | ❌ Not used    | ✅ Query      |
| Audit log        | ❌ Not used    | ✅ Query      |

### Key Points

1. **Two-layer storage**: KV for real-time, SQL for archive
2. **KV operations are synchronous**: Rate limit and quota checks must be fast
3. **SQL writes are async**: Archive happens in background goroutine
4. **Services update tokens via `request_id`**: Updates both KV and SQL
5. **KV data has TTL**: Auto-expires to prevent memory bloat
6. **SQL data is permanent**: For billing and compliance
