# Yao Table Module

A Go module for table data operations and management with TypeScript API support.

## Usage in TypeScript

You can use the Table module in TypeScript through the Process API. Below are examples of common operations with return type descriptions.

### Basic Operations

#### Get table settings

```typescript
/**
 * Gets the settings for a table
 * @param tableID - ID of the table
 * @returns object - Table settings
 */
const settings = Process("yao.table.Setting", "pet");
```

#### Get XGen configuration

```typescript
/**
 * Gets the XGen configuration for a table
 * @param tableID - ID of the table
 * @param data - Optional additional data
 * @returns object - XGen configuration
 */
const xgen = Process("yao.table.Xgen", "pet", {
  /* optional data */
});
```

### Table Data Operations

#### Search table records

```typescript
/**
 * Searches for records in a table
 * @param tableID - ID of the table
 * @param params - Query parameters
 * @param page - Page number
 * @param pageSize - Number of records per page
 * @returns object - Search results
 */
const results = Process(
  "yao.table.Search",
  "pet",
  {
    wheres: [{ column: "status", value: "checked" }],
    withs: { user: {} },
  },
  1,
  5
);
```

#### Get multiple records

```typescript
/**
 * Gets multiple records from a table
 * @param tableID - ID of the table
 * @param params - Query parameters
 * @returns array - Array of records
 */
const records = Process("yao.table.Get", "pet", {
  limit: 10,
  withs: { user: {} },
});
```

#### Find a record by ID

```typescript
/**
 * Finds a record by ID
 * @param tableID - ID of the table
 * @param id - Record ID
 * @returns object - Record data
 */
const record = Process("yao.table.Find", "pet", 1);
```

#### Save a record

```typescript
/**
 * Saves a record (creates new or updates existing)
 * @param tableID - ID of the table
 * @param record - Record data
 * @returns number - ID of the saved record
 */
const id = Process("yao.table.Save", "pet", {
  name: "New Pet",
  type: "cat",
  status: "checked",
  doctor_id: 1,
});
```

#### Create a new record

```typescript
/**
 * Creates a new record
 * @param tableID - ID of the table
 * @param record - Record data
 * @returns number - ID of the created record
 */
const id = Process("yao.table.Create", "pet", {
  name: "New Pet",
  type: "cat",
  status: "checked",
  doctor_id: 1,
});
```

#### Insert multiple records

```typescript
/**
 * Inserts multiple records
 * @param tableID - ID of the table
 * @param columns - Column names
 * @param values - Array of records (array of arrays)
 * @returns null
 */
Process(
  "yao.table.Insert",
  "pet",
  ["name", "type", "status", "doctor_id"],
  [
    ["Cookie", "cat", "checked", 1],
    ["Baby", "dog", "checked", 1],
    ["Poo", "others", "checked", 1],
  ]
);
```

#### Update a record

```typescript
/**
 * Updates a record by ID
 * @param tableID - ID of the table
 * @param id - Record ID
 * @param record - Record data to update
 * @returns null
 */
Process("yao.table.Update", "pet", 1, {
  name: "Updated Pet Name",
  status: "unchecked",
});
```

#### Update records with a WHERE clause

```typescript
/**
 * Updates records that match a WHERE clause
 * @param tableID - ID of the table
 * @param query - Query parameters with WHERE conditions
 * @param record - Record data to update
 * @returns null
 */
Process(
  "yao.table.UpdateWhere",
  "pet",
  { wheres: [{ column: "status", value: "checked" }] },
  { status: "unchecked" }
);
```

#### Update records by IDs

```typescript
/**
 * Updates records by IDs
 * @param tableID - ID of the table
 * @param ids - Comma-separated list of IDs
 * @param record - Record data to update
 * @returns null
 */
Process("yao.table.UpdateIn", "pet", "1,2,3", {
  status: "unchecked",
});
```

#### Delete a record

```typescript
/**
 * Deletes a record by ID
 * @param tableID - ID of the table
 * @param id - Record ID
 * @returns null
 */
Process("yao.table.Delete", "pet", 1);
```

#### Delete records with a WHERE clause

```typescript
/**
 * Deletes records that match a WHERE clause
 * @param tableID - ID of the table
 * @param query - Query parameters with WHERE conditions
 * @returns null
 */
Process("yao.table.DeleteWhere", "pet", {
  wheres: [{ column: "status", value: "checked" }],
});
```

#### Delete records by IDs

```typescript
/**
 * Deletes records by IDs
 * @param tableID - ID of the table
 * @param ids - Comma-separated list of IDs
 * @returns null
 */
Process("yao.table.DeleteIn", "pet", "1,2,3");
```

#### Export table data to Excel

```typescript
/**
 * Exports table data to an Excel file
 * @param tableID - ID of the table
 * @param queryParam - Query parameters (optional)
 * @param chunkSize - Number of records per chunk (default: 50)
 * @returns string - Path to the exported Excel file
 */
const filePath = Process(
  "yao.table.Export",
  "pet",
  { wheres: [{ column: "status", value: "checked" }] },
  100
);
```

### Component Integration

#### Get component data

```typescript
/**
 * Gets data for a component
 * @param tableID - ID of the table
 * @param xpath - XPath to the component
 * @param method - Component method
 * @param query - Optional query parameters
 * @returns any - Component data
 */
const options = Process(
  "yao.table.Component",
  "pet",
  "fields.filter.status.edit.props.xProps",
  "remote",
  { select: ["name", "status"], limit: 10 }
);
```

#### Upload a file

```typescript
/**
 * Uploads a file for a table field
 * @param tableID - ID of the table
 * @param xpath - XPath to the field
 * @param method - Upload method
 * @param file - File data
 * @returns string - URL to the uploaded file
 */
const fileURL = Process(
  "yao.table.Upload",
  "pet",
  "fields.table.image.edit.props",
  "api",
  fileData
);
```

#### Download a file

```typescript
/**
 * Downloads a file
 * @param tableID - ID of the table
 * @param field - Field name
 * @param file - File path
 * @param token - JWT token
 * @param isAppRoot - Is app root (optional, default: 0)
 * @returns object - File content and type
 */
const fileContent = Process(
  "yao.table.Download",
  "pet",
  "image",
  "/path/to/file.jpg",
  "JWT_TOKEN_STRING",
  0
);
```

### DSL Operations

#### Check if a table exists

```typescript
/**
 * Checks if a table exists
 * @param tableID - ID of the table
 * @returns boolean - true if the table exists, false otherwise
 */
const exists = Process("yao.table.Exists", "pet");
```

#### Read table DSL

```typescript
/**
 * Reads a table's DSL
 * @param tableID - ID of the table
 * @returns string - Table DSL as a string
 */
const dsl = Process("yao.table.Read", "pet");
```

#### List all tables

```typescript
/**
 * Lists all loaded tables
 * @returns object - Map of table ID to table DSL
 */
const tables = Process("yao.table.List");
```

#### Get table DSL

```typescript
/**
 * Gets a table's DSL object
 * @param tableID - ID of the table
 * @returns object - Table DSL object
 */
const tableDSL = Process("yao.table.DSL", "pet");
```

#### Load a table

```typescript
/**
 * Loads a table from a file or source
 * @param tableID - ID of the table (when loading from source)
 * @param file - File path (when loading from file)
 * @param source - Source JSON/YAML (optional, when loading from source)
 * @returns any - Result of the load operation
 */
// Load from file
const result = Process("yao.table.Load", "tables/pet.tab.yao");

// Load from source
Process(
  "yao.table.Load",
  "dynamic.pet",
  "/tables/dynamic/pet.tab.yao",
  `{
  "name": "Pet Admin",
  "action": {
    "bind": { "model": "pet" }
  }
}`
);
```

#### Reload a table

```typescript
/**
 * Reloads a table
 * @param tableID - ID of the table
 * @returns null
 */
Process("yao.table.Reload", "pet");
```

#### Unload a table

```typescript
/**
 * Unloads a table
 * @param tableID - ID of the table
 * @returns null
 */
Process("yao.table.Unload", "pet");
```

## Complete Workflow Example

```typescript
// Get table settings
const settings = Process("yao.table.Setting", "pet");

// Search for records
const results = Process(
  "yao.table.Search",
  "pet",
  {
    wheres: [{ column: "status", value: "checked" }],
  },
  1,
  10
);

// Create a new record
const id = Process("yao.table.Create", "pet", {
  name: "New Pet",
  type: "cat",
  status: "checked",
  doctor_id: 1,
});

// Update the record
Process("yao.table.Update", "pet", id, {
  name: "Updated Pet Name",
});

// Find the record
const record = Process("yao.table.Find", "pet", id);

// Delete the record
Process("yao.table.Delete", "pet", id);

// Export data to Excel
const filePath = Process("yao.table.Export", "pet", null, 100);
```

## Notes

- The table module provides a comprehensive set of operations for managing table data.
- Component integration allows you to interact with form components, including upload and download functionality.
- DSL operations provide the ability to dynamically load, reload, and unload tables at runtime.
