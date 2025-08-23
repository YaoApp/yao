# DSL Management API

This document describes the RESTful API for managing Yao DSL resources (models, connectors, MCP clients, etc.).

## Base URL

All endpoints are prefixed with the configured base URL followed by `/dsl` (e.g., `/v1/dsl`).

## Authentication

All endpoints require OAuth authentication via the configured OAuth provider.

## DSL Types

Supported DSL types:

- `model` - Database models
- `connector` - External service connectors
- `mcp-client` - MCP client configurations
- `api` - HTTP API definitions

## Endpoints

### Information Endpoints

#### Inspect DSL

Get detailed information about a DSL resource.

```
GET /inspect/{type}/{id}
```

**Parameters:**

- `type` (path): DSL type
- `id` (path): DSL identifier

**Example:**

```bash
curl -X GET "/v1/dsl/inspect/model/user" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
{
  "id": "user",
  "type": "model",
  "label": "User Model",
  "description": "User management model",
  "tags": ["auth", "user"],
  "path": "models/user.mod.yao",
  "store": "file",
  "status": "loaded",
  "readonly": false,
  "builtin": false,
  "mtime": "2024-01-15T10:30:00Z",
  "ctime": "2024-01-10T09:00:00Z"
}
```

#### Get DSL Source Code

Retrieve the source code of a DSL resource.

```
GET /source/{type}/{id}
```

**Example:**

```bash
curl -X GET "/v1/dsl/source/model/user" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
{
  "source": "{\n  \"name\": \"user\",\n  \"table\": {\n    \"name\": \"users\"\n  },\n  \"columns\": [...]\n}"
}
```

#### Get DSL File Path

Get the file system path for a DSL resource.

```
GET /path/{type}/{id}
```

**Response:**

```json
{
  "path": "models/user.mod.yao"
}
```

#### List DSLs

List DSL resources with optional filtering.

```
GET /list/{type}?sort={sort}&order={order}&store={store}&source={source}&tags={tags}&pattern={pattern}
```

**Query Parameters:**

- `sort` (optional): Sort field
- `order` (optional): Sort order ("asc" or "desc")
- `store` (optional): Storage type filter ("db" or "file")
- `source` (optional): Include source code in response (true/false)
- `tags` (optional): Comma-separated list of tags to filter by
- `pattern` (optional): File name pattern matching

**Example:**

```bash
curl -X GET "/v1/dsl/list/model?store=file&tags=user,auth" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
[
  {
    "id": "user",
    "type": "model",
    "label": "User Model",
    "description": "User management model",
    "tags": ["auth", "user"],
    "path": "models/user.mod.yao",
    "store": "file",
    "status": "loaded"
  }
]
```

#### Check DSL Existence

Check if a DSL resource exists.

```
GET /exists/{type}/{id}
```

**Response:**

```json
{
  "exists": true
}
```

### CRUD Operations

#### Create DSL

Create a new DSL resource.

```
POST /create/{type}
```

**Request Body:**

```json
{
  "id": "test_user",
  "source": "{\n  \"name\": \"test_user\",\n  \"table\": {\n    \"name\": \"test_users\",\n    \"comment\": \"Test User\"\n  },\n  \"columns\": [\n    { \"name\": \"id\", \"type\": \"ID\" },\n    { \"name\": \"name\", \"type\": \"string\", \"length\": 80 }\n  ]\n}",
  "store": "file"
}
```

**Example:**

```bash
curl -X POST "/v1/dsl/create/model" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test_user",
    "source": "{ \"name\": \"test_user\", \"table\": { \"name\": \"test_users\" }, \"columns\": [{ \"name\": \"id\", \"type\": \"ID\" }] }",
    "store": "file"
  }'
```

**Response:**

```json
{
  "message": "DSL created successfully"
}
```

#### Update DSL

Update an existing DSL resource.

```
PUT /update/{type}
```

**Request Body:**

```json
{
  "id": "test_user",
  "source": "{\n  \"name\": \"test_user\",\n  \"table\": {\n    \"name\": \"test_users\",\n    \"comment\": \"Updated Test User\"\n  },\n  \"columns\": [\n    { \"name\": \"id\", \"type\": \"ID\" },\n    { \"name\": \"name\", \"type\": \"string\", \"length\": 100 }\n  ]\n}"
}
```

**Response:**

```json
{
  "message": "DSL updated successfully"
}
```

#### Delete DSL

Delete a DSL resource.

```
DELETE /delete/{type}/{id}
```

**Example:**

```bash
curl -X DELETE "/v1/dsl/delete/model/test_user" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
{
  "message": "DSL deleted successfully"
}
```

### Load Management

#### Load DSL

Load a DSL resource into memory.

```
POST /load/{type}
```

**Request Body:**

```json
{
  "id": "user",
  "source": "{ ... }",
  "store": "file"
}
```

**Response:**

```json
{
  "message": "DSL loaded successfully"
}
```

#### Unload DSL

Unload a DSL resource from memory.

```
POST /unload/{type}
```

**Request Body:**

```json
{
  "id": "user",
  "store": "file"
}
```

**Response:**

```json
{
  "message": "DSL unloaded successfully"
}
```

#### Reload DSL

Reload a DSL resource (unload then load).

```
POST /reload/{type}
```

**Request Body:**

```json
{
  "id": "user",
  "source": "{ ... }",
  "store": "file"
}
```

**Response:**

```json
{
  "message": "DSL reloaded successfully"
}
```

### Execution and Validation

#### Execute DSL Method

Execute a method on a loaded DSL resource.

```
POST /execute/{type}/{id}/{method}
```

**Request Body:**

```json
{
  "args": ["arg1", "arg2"]
}
```

**Example:**

```bash
curl -X POST "/v1/dsl/execute/model/user/find" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "args": [1, {"select": ["id", "name"]}]
  }'
```

**Response:**

```json
{
  "result": {
    "id": 1,
    "name": "John Doe"
  }
}
```

#### Validate DSL Source

Validate DSL source code syntax.

```
POST /validate/{type}
```

**Request Body:**

```json
{
  "source": "{\n  \"name\": \"user\",\n  \"table\": {\n    \"name\": \"users\"\n  }\n}"
}
```

**Response:**

```json
{
  "valid": true,
  "messages": []
}
```

Or if there are validation errors:

```json
{
  "valid": false,
  "messages": [
    {
      "file": "",
      "line": 5,
      "column": 10,
      "message": "Missing required field 'columns'",
      "severity": "error"
    }
  ]
}
```

## Error Responses

All endpoints return appropriate HTTP status codes and error messages:

```json
{
  "error": "DSL ID is required"
}
```

Common HTTP status codes:

- `200` - Success
- `201` - Created
- `400` - Bad Request (invalid parameters)
- `404` - Not Found
- `500` - Internal Server Error

## Example Workflows

### Creating a New Model

1. **Validate the source first:**

```bash
curl -X POST "/v1/dsl/validate/model" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "{ \"name\": \"product\", \"table\": { \"name\": \"products\" }, \"columns\": [{ \"name\": \"id\", \"type\": \"ID\" }] }"
  }'
```

2. **Create the model:**

```bash
curl -X POST "/v1/dsl/create/model" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "product",
    "source": "{ \"name\": \"product\", \"table\": { \"name\": \"products\" }, \"columns\": [{ \"name\": \"id\", \"type\": \"ID\" }] }",
    "store": "file"
  }'
```

3. **Verify it was created:**

```bash
curl -X GET "/v1/dsl/inspect/model/product" \
  -H "Authorization: Bearer {token}"
```

### Updating and Reloading a DSL

1. **Update the DSL:**

```bash
curl -X PUT "/v1/dsl/update/model" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "product",
    "source": "{ \"name\": \"product\", \"label\": \"Product Model\", \"table\": { \"name\": \"products\" }, \"columns\": [{ \"name\": \"id\", \"type\": \"ID\" }, { \"name\": \"name\", \"type\": \"string\" }] }"
  }'
```

2. **Reload to apply changes:**

```bash
curl -X POST "/v1/dsl/reload/model" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "product",
    "store": "file"
  }'
```

This API provides comprehensive DSL management capabilities that align with the test cases and interface definitions in the codebase.
