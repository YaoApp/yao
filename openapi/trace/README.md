# Trace API

The Trace API provides endpoints to monitor, retrieve, and stream execution traces.

**Base URL**: `/v1/trace`  
**Auth**: Bearer Token (OAuth2)

## Endpoints

| Method | Endpoint                           | Description                                     |
| :----- | :--------------------------------- | :---------------------------------------------- |
| `GET`  | `/traces/:traceID/events`          | Stream trace events (SSE) or get event history. |
| `GET`  | `/traces/:traceID/info`            | Get trace metadata (status, user, time).        |
| `GET`  | `/traces/:traceID/nodes`           | List all execution nodes.                       |
| `GET`  | `/traces/:traceID/nodes/:nodeID`   | Get details for a specific node.                |
| `GET`  | `/traces/:traceID/logs`            | List all logs.                                  |
| `GET`  | `/traces/:traceID/logs/:nodeID`    | List logs for a specific node.                  |
| `GET`  | `/traces/:traceID/spaces`          | List memory spaces (metadata only).             |
| `GET`  | `/traces/:traceID/spaces/:spaceID` | Get space details (includes KV data).           |

---

## Events (SSE)

**Endpoint**: `/traces/:traceID/events?stream=true`  
**Format**: Server-Sent Events (SSE)  
**Terminator**: `data: [DONE]`

### Event Envelope

Each event starts with `event: <type>` followed by `data: <json>`.

```
event: node_start

data: {
  "type": "node_start",
  "trace_id": "...",
  "node_id": "...",
  "space_id": "",
  "timestamp": 1763633999330,
  "data": { ... }
}
```

| Field       | Type   | Description                                           |
| :---------- | :----- | :---------------------------------------------------- |
| `type`      | String | Event type (e.g., `init`, `node_start`, `log_added`). |
| `trace_id`  | String | Unique trace identifier.                              |
| `node_id`   | String | (Optional) Associated node ID.                        |
| `space_id`  | String | (Optional) Associated space ID.                       |
| `timestamp` | Int64  | Event time in milliseconds (Unix epoch).              |
| `data`      | Object | Event payload, structure varies by `type`.            |

### Event Payloads

#### 1. `init`

Trace initialization.

| Field        | Type   | Description                             |
| :----------- | :----- | :-------------------------------------- |
| `trace_id`   | String | Trace ID.                               |
| `agent_name` | String | (Optional) Name of the agent/assistant. |
| `root_node`  | Object | (Optional) Preview of the root node.    |

**Example**:

```json
{
  "type": "init",
  "trace_id": "20251120633999366550",
  "timestamp": 1763633999329,
  "data": {
    "trace_id": "20251120633999366550"
  }
}
```

#### 2. `node_start`

A node execution has started.

| Field   | Type   | Description                                                            |
| :------ | :----- | :--------------------------------------------------------------------- |
| `node`  | Object | Full node structure (see [Node Structure](#node-structure-in-events)). |
| `nodes` | Array  | (Optional) List of nodes if starting in parallel.                      |

**Example**:

```json
{
  "type": "node_start",
  "trace_id": "20251120633999366550",
  "node_id": "701dybnkuw6a",
  "timestamp": 1763633999330,
  "data": {
    "node": {
      "id": "701dybnkuw6a",
      "parent_id": "",
      "label": "AI Assistant",
      "icon": "assistant",
      "description": "AI Assistant is processing the request",
      "status": "running",
      "input": [{ "role": "user", "content": "Hello there" }],
      "created_at": 1763633999330,
      "start_time": 1763633999330
    }
  }
}
```

#### 3. `node_complete`

Node execution completed successfully.

| Field      | Type   | Description                         |
| :--------- | :----- | :---------------------------------- |
| `node_id`  | String | ID of the completed node.           |
| `status`   | String | Always `"success"`.                 |
| `duration` | Int64  | Execution duration in milliseconds. |
| `end_time` | Int64  | Completion timestamp (ms).          |
| `output`   | Any    | Node execution result.              |

**Example**:

```json
{
  "type": "node_complete",
  "node_id": "ee8e6nendxjx",
  "timestamp": 1763634001537,
  "data": {
    "node_id": "ee8e6nendxjx",
    "status": "success",
    "duration": 2206,
    "end_time": 1763634001537,
    "output": {
      "content": "Hello! How can I assist you today?",
      "role": "assistant"
    }
  }
}
```

#### 4. `node_failed`

Node execution failed with error.

| Field      | Type   | Description                         |
| :--------- | :----- | :---------------------------------- |
| `node_id`  | String | ID of the failed node.              |
| `status`   | String | Always `"failed"`.                  |
| `duration` | Int64  | Execution duration in milliseconds. |
| `end_time` | Int64  | Failure timestamp (ms).             |
| `error`    | String | Error message.                      |

#### 5. `log_added`

New log entry added.

| Field       | Type   | Description                                  |
| :---------- | :----- | :------------------------------------------- |
| `Level`     | String | Log level: `info`, `debug`, `warn`, `error`. |
| `Message`   | String | Log message text.                            |
| `Data`      | Array  | Array of structured log data objects.        |
| `NodeID`    | String | Associated node ID.                          |
| `Timestamp` | Int64  | Log timestamp (ms).                          |

**Example**:

```json
{
  "type": "log_added",
  "node_id": "ee8e6nendxjx",
  "timestamp": 1763633999331,
  "data": {
    "Level": "debug",
    "Message": "OpenAI Stream: Starting stream request",
    "Data": [{ "message_count": 1 }],
    "NodeID": "ee8e6nendxjx",
    "Timestamp": 1763633999331
  }
}
```

#### 6. `space_created`

Memory space created.

**Data**: Full `TraceSpace` object (see [Space Object](#space-object)).

#### 7. `space_deleted`

Memory space deleted.

| Field      | Type   | Description              |
| :--------- | :----- | :----------------------- |
| `space_id` | String | ID of the deleted space. |

#### 8. `memory_add` / `memory_update`

Key-value added or updated in a space.

| Field  | Type   | Description            |
| :----- | :----- | :--------------------- |
| `type` | String | Space ID.              |
| `item` | Object | Memory item structure. |

**MemoryItem Structure**:

| Field        | Type   | Description                         |
| :----------- | :----- | :---------------------------------- |
| `id`         | String | Key name.                           |
| `type`       | String | Space ID (same as parent `type`).   |
| `title`      | String | (Optional) Display title.           |
| `content`    | Any    | The value stored.                   |
| `timestamp`  | Int64  | Operation timestamp (ms).           |
| `importance` | String | (Optional) `high`, `medium`, `low`. |

#### 9. `memory_delete`

Key-value deleted from a space.

| Field      | Type   | Description                            |
| :--------- | :----- | :------------------------------------- |
| `space_id` | String | Space ID.                              |
| `key`      | String | (Optional) Deleted key name.           |
| `cleared`  | Bool   | (Optional) `true` if all keys cleared. |

#### 10. `complete`

Trace execution finished.

| Field            | Type   | Description                                       |
| :--------------- | :----- | :------------------------------------------------ |
| `trace_id`       | String | Trace ID.                                         |
| `status`         | String | Final status: `completed`, `failed`, `cancelled`. |
| `total_duration` | Int64  | Total execution time in milliseconds.             |

**Example**:

```json
{
  "type": "complete",
  "timestamp": 1763634001540,
  "data": {
    "trace_id": "20251120633999366550",
    "status": "completed",
    "total_duration": 2210
  }
}
```

---

## Resource Structures

### Node Structure (in Events and API)

When a node appears in events or API responses, it includes:

| Field         | Type   | Description                                             |
| :------------ | :----- | :------------------------------------------------------ |
| `id`          | String | Unique node ID.                                         |
| `parent_id`   | String | ID of the parent node (empty for root).                 |
| `children`    | Array  | List of child node objects (usually empty in events).   |
| `label`       | String | Human-readable name.                                    |
| `icon`        | String | UI icon identifier.                                     |
| `description` | String | Detailed description.                                   |
| `status`      | String | `pending`, `running`, `completed`, `failed`, `skipped`. |
| `input`       | Any    | Input arguments.                                        |
| `output`      | Any    | Execution result (null when starting).                  |
| `metadata`    | Map    | Custom metadata (e.g., `{"node_order": 1}`).            |
| `created_at`  | Int64  | Timestamp (ms).                                         |
| `start_time`  | Int64  | Timestamp (ms).                                         |
| `end_time`    | Int64  | Timestamp (ms), 0 if not finished.                      |
| `updated_at`  | Int64  | Timestamp (ms).                                         |

### Space Object

Represents a memory context/container.

| Field         | Type   | Description                               |
| :------------ | :----- | :---------------------------------------- |
| `id`          | String | Unique space ID.                          |
| `label`       | String | Human-readable name.                      |
| `icon`        | String | UI icon identifier.                       |
| `description` | String | Purpose of the space.                     |
| `ttl`         | Int64  | Time-to-live in seconds (0 = infinite).   |
| `metadata`    | Map    | Custom metadata.                          |
| `data`        | Map    | (Detail API only) Key-value pairs stored. |
| `created_at`  | Int64  | Timestamp (ms).                           |
| `updated_at`  | Int64  | Timestamp (ms).                           |
