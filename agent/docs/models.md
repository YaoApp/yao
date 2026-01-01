# Assistant Models

Assistants can define their own namespaced data models in the `models/` directory. These models are automatically loaded with the `agents.<assistant-id>.` prefix and use isolated database tables.

## Directory Structure

```
assistants/
└── my-assistant/
    ├── package.yao
    └── models/
        ├── order.mod.yao       # → agents.my-assistant.order
        ├── item.mod.yao        # → agents.my-assistant.item
        └── nested/
            └── log.mod.yao     # → agents.my-assistant.nested.log
```

## Model Definition

Standard Yao model definition with automatic table prefixing.

**models/order.mod.yao**

```json
{
  "name": "Order",
  "label": "Order Record",
  "description": "Customer orders",
  "table": {
    "name": "order",
    "comment": "Order records"
  },
  "columns": [
    {
      "name": "id",
      "type": "ID",
      "label": "ID",
      "primary": true
    },
    {
      "name": "order_no",
      "type": "string",
      "label": "Order Number",
      "length": 100,
      "nullable": false,
      "unique": true,
      "index": true
    },
    {
      "name": "customer_id",
      "type": "string",
      "label": "Customer ID",
      "length": 255,
      "nullable": false,
      "index": true
    },
    {
      "name": "total_amount",
      "type": "decimal",
      "label": "Total Amount",
      "precision": 15,
      "scale": 2,
      "nullable": false
    },
    {
      "name": "status",
      "type": "enum",
      "label": "Status",
      "option": ["pending", "confirmed", "shipped", "completed", "cancelled"],
      "default": "pending",
      "nullable": false,
      "index": true
    },
    {
      "name": "metadata",
      "type": "json",
      "label": "Metadata",
      "nullable": true
    }
  ],
  "relations": {
    "items": {
      "type": "hasMany",
      "model": "item",
      "key": "order_id",
      "foreign": "id"
    }
  },
  "indexes": [
    {
      "name": "idx_customer_status",
      "columns": ["customer_id", "status"],
      "type": "index"
    }
  ],
  "option": {
    "timestamps": true,
    "soft_deletes": true
  }
}
```

## Table Naming

Tables are automatically prefixed with `agents_<assistant-id>_`:

| Assistant ID | Model File             | Model ID                 | Table Name               |
| ------------ | ---------------------- | ------------------------ | ------------------------ |
| `expense`    | `models/order.mod.yao` | `agents.expense.order`   | `agents_expense_order`   |
| `tests.demo` | `models/user.mod.yao`  | `agents.tests.demo.user` | `agents_tests_demo_user` |

## Using Models

### In Hooks

```typescript
import { Process } from "@yao/runtime";

function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  // Query assistant's own model
  const orders = Process("models.agents.my-assistant.order.Paginate", {
    wheres: [{ column: "status", value: "pending" }],
    limit: 10,
  });

  return { messages };
}
```

### In MCP Tools

```json
{
  "transport": "process",
  "tools": {
    "list_orders": "models.agents.my-assistant.order.Paginate",
    "get_order": "models.agents.my-assistant.order.Find",
    "create_order": "models.agents.my-assistant.order.Create",
    "update_order": "models.agents.my-assistant.order.Update"
  }
}
```

### In Scripts

**src/orders.ts**

```typescript
import { Process } from "@yao/runtime";

export function ListPending(): any[] {
  return Process("models.agents.my-assistant.order.Get", {
    wheres: [{ column: "status", value: "pending" }],
    orders: [{ column: "created_at", option: "desc" }],
  });
}

export function CreateOrder(data: any): any {
  return Process("models.agents.my-assistant.order.Create", data);
}

export function UpdateStatus(id: number, status: string): any {
  return Process("models.agents.my-assistant.order.Update", id, { status });
}
```

## Column Types

| Type         | Description                | Options                 |
| ------------ | -------------------------- | ----------------------- |
| `ID`         | Auto-increment primary key | `primary: true`         |
| `string`     | VARCHAR                    | `length` (default: 255) |
| `text`       | TEXT                       | -                       |
| `integer`    | INT                        | -                       |
| `bigInteger` | BIGINT                     | -                       |
| `float`      | FLOAT                      | `precision`, `scale`    |
| `decimal`    | DECIMAL                    | `precision`, `scale`    |
| `boolean`    | BOOLEAN                    | -                       |
| `date`       | DATE                       | -                       |
| `datetime`   | DATETIME                   | -                       |
| `timestamp`  | TIMESTAMP                  | -                       |
| `json`       | JSON/JSONB                 | -                       |
| `enum`       | ENUM                       | `option: [...]`         |

## Column Options

| Option      | Type      | Description       |
| ----------- | --------- | ----------------- |
| `nullable`  | `boolean` | Allow NULL values |
| `default`   | `any`     | Default value     |
| `unique`    | `boolean` | Unique constraint |
| `index`     | `boolean` | Create index      |
| `primary`   | `boolean` | Primary key       |
| `length`    | `integer` | String length     |
| `precision` | `integer` | Decimal precision |
| `scale`     | `integer` | Decimal scale     |
| `comment`   | `string`  | Column comment    |

## Relations

```json
{
  "relations": {
    "items": {
      "type": "hasMany",
      "model": "item",
      "key": "order_id",
      "foreign": "id"
    },
    "customer": {
      "type": "hasOne",
      "model": "customer",
      "key": "id",
      "foreign": "customer_id"
    }
  }
}
```

| Type             | Description                   |
| ---------------- | ----------------------------- |
| `hasOne`         | One-to-one relationship       |
| `hasMany`        | One-to-many relationship      |
| `hasOneThrough`  | Has one through intermediate  |
| `hasManyThrough` | Has many through intermediate |

## Model Options

```json
{
  "option": {
    "timestamps": true,
    "soft_deletes": true,
    "permission": true
  }
}
```

| Option         | Description                            |
| -------------- | -------------------------------------- |
| `timestamps`   | Add `created_at`, `updated_at` columns |
| `soft_deletes` | Add `deleted_at` for soft delete       |
| `permission`   | Enable permission checks               |

## Process Reference

Common model processes:

| Process       | Description      | Arguments                   |
| ------------- | ---------------- | --------------------------- |
| `Find`        | Get by ID        | `id`, `query?`              |
| `Get`         | Get records      | `query`                     |
| `Paginate`    | Paginated list   | `query`, `page`, `pagesize` |
| `Create`      | Create record    | `data`                      |
| `Update`      | Update record    | `id`, `data`                |
| `Save`        | Create or update | `data`                      |
| `Delete`      | Delete record    | `id`                        |
| `Destroy`     | Hard delete      | `id`                        |
| `Insert`      | Batch insert     | `columns`, `rows`           |
| `UpdateWhere` | Batch update     | `query`, `data`             |
| `DeleteWhere` | Batch delete     | `query`                     |

## Migration

Models are automatically migrated when Yao starts. The migration:

1. Creates tables if not exist
2. Adds new columns
3. Creates indexes
4. Does NOT drop columns (safe migration)

To force schema sync:

```bash
yao migrate --reset  # Warning: drops and recreates tables
```

## Example: Complete Assistant with Models

**assistants/inventory/package.yao**

```json
{
  "name": "Inventory Assistant",
  "connector": "gpt-4o",
  "mcp": {
    "servers": [{ "server_id": "inventory" }]
  }
}
```

**assistants/inventory/models/product.mod.yao**

```json
{
  "name": "Product",
  "table": { "name": "product" },
  "columns": [
    { "name": "id", "type": "ID", "primary": true },
    { "name": "sku", "type": "string", "length": 50, "unique": true },
    { "name": "name", "type": "string", "length": 200 },
    { "name": "quantity", "type": "integer", "default": 0 },
    { "name": "price", "type": "decimal", "precision": 10, "scale": 2 }
  ],
  "option": { "timestamps": true }
}
```

**assistants/inventory/mcps/inventory.mcp.yao**

```json
{
  "label": "Inventory",
  "transport": "process",
  "tools": {
    "list_products": "models.agents.inventory.product.Paginate",
    "get_product": "models.agents.inventory.product.Find",
    "update_stock": "agents.inventory.stock.Update"
  }
}
```

**assistants/inventory/src/stock.ts**

```typescript
import { Process } from "@yao/runtime";

export function Update(args: { sku: string; quantity: number }): any {
  const product = Process("models.agents.inventory.product.Get", {
    wheres: [{ column: "sku", value: args.sku }],
    limit: 1,
  });

  if (!product || product.length === 0) {
    throw new Error(`Product not found: ${args.sku}`);
  }

  return Process("models.agents.inventory.product.Update", product[0].id, {
    quantity: args.quantity,
  });
}
```
