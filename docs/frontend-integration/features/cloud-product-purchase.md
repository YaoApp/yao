# Cloud Products, Async Purchase, and Purchase Snapshot

**Audience:** CUI cloud computer flows and host/integration engineers.

## 1. Live products

```http
GET /v1/products?source=tao&policy=paid&status=enabled
```

`/v1/products` is a source-neutral API: `source` selects a registered product
provider. Tao is the first provider (`source=tao`); future sources can register
without changing the endpoint or response model. Yao proxies the selected
provider on each request; it does **not** read or write `__yao.products`.

The endpoint requires the token scope:

```text
product:products:read
```

Unknown/missing `source` is `400`; provider failures are surfaced as `5xx`
(or the provider's auth error) without falling back to another source.

Each item includes:

```json
{
  "id": "tao-...",
  "name": "cvm-2c2g",
  "scope": "system",
  "source": "tao",
  "region": "gz-cn",
  "resource": "cvm-2c2g",
  "spec": {"cpu_cores": 2, "memory_mb": 2048, "disk_gb": 50, "bandwidth_mbps": 1, "images": ["tao-base-1.0"]},
  "policy": "paid",
  "total_slots": 100,
  "available_slots": 100,
  "billing_units": ["HOUR", "MONTH"],
  "prices": {"HOUR": 480, "MONTH": 286800},
  "duration_tiers": [{"months": "1-2", "discount": 0.87}],
  "images": ["tao-base-1.0"],
  "health": "healthy",
  "status": "enabled"
}
```

The live Tao `id` includes `region`; do not mix it with the legacy sync
product id (`service + resource` only). Unknown sources return `400`; a
provider failure is reported as `5xx` without falling back to another source.

## 2. Purchase + provisioning task

```http
POST /v1/tao/computers
Authorization: <Yao session>
```

```json
{
  "resource": "cvm-2c2g",
  "region": "gz-cn",
  "name": "Linux build host",
  "billing_unit": "HOUR",
  "request_params": {"image": "tao-base-1.0"},
  "members": [],
  "computer": {"name": "My cloud computer", "visibility": "private"}
}
```

Response: `202 Accepted`

```json
{"task_id": "task_...", "status": "queued"}
```

The server enqueues a durable task; it performs purchase → Tao instance →
bootstrap → `tai start` in the background. CUI must not call the legacy
`GET /v1/tao/computers/{instance_id}` for provisioning.

Status:

```http
GET /v1/tao/computers/tasks/{task_id}
```

```json
{
  "task_id": "task_...",
  "status": "queued | running | succeeded | failed",
  "phase": "purchase | provision | tai | ready",
  "instance_id": "inst_...",
  "computer_id": "cmp_...",
  "error": null
}
```

A failed task is HTTP `200` with `status=failed` and the real `error`
message. `instance_id` can be present on a failed task when failure happened
after Tao created the instance; do not retry such a purchase from CUI.

## 3. Server-side dedupe

CUI never sends `Idempotency-Key`. Yao computes a dedupe key from:

```text
user_id, team_id, resource, region, image, disk_gb,
billing_unit, period_months, computer.name, visibility, members
```

- duplicate active/successful request returns the existing task;
- failed task without `instance_id` is re-queued only on an explicit duplicate
  POST (the same task ID is returned);
- failed task with `instance_id` returns `409 purchase_conflict`, carrying the
  task ID in `state`;
- deleting the Computer releases the key for a future purchase.

## 4. Purchase snapshot

`GET /v1/computers/{id}` includes the immutable facts captured when Tao
created the instance:

```json
{
  "purchase": {
    "resource": "cvm-2c2g",
    "region": "gz-cn",
    "image": "tao-base-1.0",
    "billing_unit": "HOUR",
    "period_months": null,
    "price_credits": 1,
    "price_total_credits": null,
    "expires_at": null,
    "purchased_at": "2026-11-26T10:00:00Z"
  }
}
```

`purchase` is omitted for Computers purchased before this feature or for
non-purchase computers. It never depends on the current Tao catalog; a
delisted product remains visible on the purchased Computer.

## 5. Events

The worker advances Computer status through the normal Computer service, so
existing `computer.created` / `computer.status` events invalidate list/detail.
After a WebSocket disconnect, use `GET /v1/computers/{id}/status` as a
low-frequency fallback.
