# Yao Utils Module

A Go module for utility functions with TypeScript API support.

## Usage in TypeScript

You can use the Utils module in TypeScript through the Process API. Below are examples of common operations with return type descriptions.

### String Operations

#### Concatenate strings

```typescript
/**
 * Joins an array of values with a separator
 * @param values - Array of values to join
 * @param separator - String separator
 * @returns string - Joined string
 */
const joined = Process("utils.str.Join", ["Hello", "World"], " ");
// Returns: "Hello World"
```

#### Join file paths

```typescript
/**
 * Joins path segments into a single path
 * @param ...paths - Path segments to join
 * @returns string - Joined path
 */
const path = Process("utils.str.JoinPath", "path", "to", "file.txt");
// Returns: "path/to/file.txt"
```

#### Generate UUID

```typescript
/**
 * Generates a UUID string
 * @returns string - UUID string
 */
const uuid = Process("utils.str.UUID");
// Returns: "550e8400-e29b-41d4-a716-446655440000" (example)
```

#### Convert Chinese to Pinyin

```typescript
/**
 * Converts Chinese characters to Pinyin
 * @param text - Chinese text to convert
 * @param options - Optional configuration
 * @returns string - Pinyin text
 */
const pinyin = Process("utils.str.Pinyin", "你好");
// Returns: "ni hao"

// With tone marks
const pinyinWithTone = Process("utils.str.Pinyin", "你好", {
  tone: true,
  separator: "-",
});
// Returns: "nǐ-hǎo"

// With tone numbers
const pinyinWithToneNumbers = Process("utils.str.Pinyin", "你好", {
  tone: "number",
  separator: "-",
});
// Returns: "ni3-hao3"

// With multiple pronunciations for characters (heteronym mode)
const pinyinWithHeteronym = Process("utils.str.Pinyin", "中国", {
  heteronym: true,
  tone: true,
});
// Returns: "zhōng|zhòng guó"
```

#### Convert hex to string

```typescript
/**
 * Converts a hexadecimal string to a regular string
 * @param hex - Hexadecimal string
 * @returns string - Decoded string
 */
const text = Process("utils.str.Hex", "48656c6c6f20576f726c64");
// Returns: "Hello World"
```

### Date and Time

#### Get current timestamp

```typescript
/**
 * Gets the current Unix timestamp (seconds)
 * @returns number - Unix timestamp
 */
const timestamp = Process("utils.now.Timestamp");
// Returns: 1625097600 (example)
```

#### Get current timestamp in milliseconds

```typescript
/**
 * Gets the current Unix timestamp in milliseconds
 * @returns number - Unix timestamp in milliseconds
 */
const timestampMs = Process("utils.now.Timestampms");
// Returns: 1625097600000 (example)
```

#### Get current date

```typescript
/**
 * Gets the current date in YYYY-MM-DD format
 * @returns string - Current date
 */
const date = Process("utils.now.Date");
// Returns: "2023-07-01"
```

#### Get current time

```typescript
/**
 * Gets the current time in HH:MM:SS format
 * @returns string - Current time
 */
const time = Process("utils.now.Time");
// Returns: "12:34:56"
```

#### Get current date and time

```typescript
/**
 * Gets the current date and time in YYYY-MM-DD HH:MM:SS format
 * @returns string - Current date and time
 */
const dateTime = Process("utils.now.DateTime");
// Returns: "2023-07-01 12:34:56"
```

#### Sleep for a duration

```typescript
/**
 * Pauses execution for a specified time
 * @param milliseconds - Time to sleep in milliseconds
 * @returns null
 */
Process("utils.time.Sleep", 1000);
// Sleeps for 1 second
```

### Error Handling

#### Throw forbidden error

```typescript
/**
 * Throws a 403 Forbidden error
 * @param message - Optional error message
 * @throws Exception with code 403
 */
Process("utils.throw.Forbidden", "Access denied");
```

#### Throw unauthorized error

```typescript
/**
 * Throws a 401 Unauthorized error
 * @param message - Optional error message
 * @throws Exception with code 401
 */
Process("utils.throw.Unauthorized", "Authentication required");
```

#### Throw not found error

```typescript
/**
 * Throws a 404 Not Found error
 * @param message - Optional error message
 * @throws Exception with code 404
 */
Process("utils.throw.NotFound", "Resource not found");
```

#### Throw bad request error

```typescript
/**
 * Throws a 400 Bad Request error
 * @param message - Optional error message
 * @throws Exception with code 400
 */
Process("utils.throw.BadRequest", "Invalid parameters");
```

#### Throw internal error

```typescript
/**
 * Throws a 500 Internal Error
 * @param message - Optional error message
 * @throws Exception with code 500
 */
Process("utils.throw.InternalError", "Something went wrong");
```

#### Throw custom exception

```typescript
/**
 * Throws a custom exception with specified message and code
 * @param message - Error message
 * @param code - Error code
 * @throws Exception with specified code
 */
Process("utils.throw.Exception", "Payment required", 402);
```

### URL Handling

#### Parse query string

```typescript
/**
 * Parses a URL query string into a map
 * @param queryString - URL query string
 * @returns object - Map of query parameters
 */
const query = Process("utils.url.ParseQuery", "name=John&age=30");
// Returns: { name: ["John"], age: ["30"] }
```

#### Parse URL

```typescript
/**
 * Parses a URL into its components
 * @param url - URL to parse
 * @returns object - URL components
 */
const urlParts = Process(
  "utils.url.ParseURL",
  "https://example.com:8080/path?q=search"
);
// Returns: {
//   scheme: "https",
//   host: "example.com:8080",
//   domain: "example.com",
//   path: "/path",
//   port: "8080",
//   query: { q: ["search"] },
//   url: "https://example.com:8080/path?q=search"
// }
```

#### Convert to query parameters

```typescript
/**
 * Converts various data types to query parameters
 * @param data - Data to convert (map, url.Values, etc.)
 * @returns string - Query parameter string
 */
const params = Process("utils.url.QueryParam", {
  name: "John",
  tags: ["dev", "admin"],
});
// Returns: "name=John&tags=dev&tags=admin"
```

### Formatting and Output

#### Print formatted string

```typescript
/**
 * Prints a formatted string to stdout
 * @param format - Format string
 * @param ...args - Arguments for format
 * @returns null
 */
Process("utils.fmt.Printf", "Hello, %s!", "World");
// Prints: Hello, World!
```

#### Print colored string

```typescript
/**
 * Prints a colored formatted string to stdout
 * @param color - Color name (red, green, blue, etc.)
 * @param format - Format string
 * @param ...args - Arguments for format
 * @returns null
 */
Process("utils.fmt.ColorPrintf", "green", "Success: %s", "Operation completed");
// Prints: Success: Operation completed (in green)

// Available colors:
// red, green, yellow, blue, magenta, cyan, white, black
// hired, higreen, hiyellow, hiblue, himagenta, hicyan, hiwhite, hiblack
```

### Tree Operations

#### Flatten tree to array

```typescript
/**
 * Flattens a hierarchical tree structure to a flat array
 * @param tree - Tree structure (array of nodes with children)
 * @param options - Optional configuration
 * @returns array - Flattened array
 */
const flat = Process("utils.tree.Flatten", [
  {
    id: 1,
    name: "Parent",
    children: [
      { id: 2, name: "Child 1" },
      { id: 3, name: "Child 2" },
    ],
  },
]);
// Returns: [
//   { id: 1, name: "Parent", parent: null },
//   { id: 2, name: "Child 1", parent: 1 },
//   { id: 3, name: "Child 2", parent: 1 }
// ]

// With custom options
const customFlat = Process(
  "utils.tree.Flatten",
  [{ uid: 1, title: "Parent", items: [{ uid: 2, title: "Child" }] }],
  { primary: "uid", children: "items", parent: "parentId" }
);
// Returns: [
//   { uid: 1, title: "Parent", parentId: null },
//   { uid: 2, title: "Child", parentId: 1 }
// ]
```

### JSON Operations

#### Validate JSON structure

```typescript
/**
 * Validates a JSON structure against rules
 * @param data - JSON data to validate
 * @param rules - Validation rules
 * @returns boolean - True if valid, false otherwise
 */
const isValid = Process("utils.json.Validate", { name: "John", age: 30 }, [
  { haskey: "name" },
  { haskey: "age" },
]);
// Returns: true
```

### Flow Control

#### Conditional processing (IF)

```typescript
/**
 * Conditionally executes a process based on conditions
 * @param conditions - Array of condition objects
 * @returns any - Result of the executed process
 */
const result = Process(
  "utils.flow.IF",
  {
    when: [{ operator: "eq", value: 1, field: "status" }],
    process: "scripts.test.active",
    args: ["User is active"],
  },
  {
    when: [{ operator: "eq", value: 0, field: "status" }],
    process: "scripts.test.inactive",
    args: ["User is inactive"],
  }
);
```

#### Case statement

```typescript
/**
 * Executes the first matching case based on conditions
 * @param ...cases - Case objects with conditions
 * @returns any - Result of the executed process
 */
const result = Process(
  "utils.flow.Case",
  {
    when: [{ operator: "eq", value: "admin", field: "role" }],
    process: "scripts.user.adminPanel",
    args: [],
  },
  {
    when: [{ operator: "eq", value: "user", field: "role" }],
    process: "scripts.user.userDashboard",
    args: [],
  }
);
```

#### For loop

```typescript
/**
 * Executes a process multiple times in a loop
 * @param from - Starting index (inclusive)
 * @param to - Ending index (exclusive)
 * @param processConfig - Process configuration
 * @returns null
 */
Process("utils.flow.For", 0, 5, {
  process: "scripts.test.log",
  args: ["Loop index: ::value"],
});
// Calls scripts.test.log 5 times with indexes 0-4
```

#### Each loop (iterate over array or map)

```typescript
/**
 * Iterates over an array or map and executes a process for each item
 * @param data - Array or map to iterate over
 * @param processConfig - Process configuration
 * @returns null
 */
Process("utils.flow.Each", ["apple", "banana", "orange"], {
  process: "scripts.test.log",
  args: ["Item: ::value at index ::key"],
});

// Also works with objects
Process(
  "utils.flow.Each",
  { name: "John", age: 30 },
  {
    process: "scripts.test.log",
    args: ["::key = ::value"],
  }
);
```

#### Return value

```typescript
/**
 * Returns values as-is, useful for terminating process chains
 * @param ...values - Values to return
 * @returns any - The provided values
 */
const result = Process("utils.flow.Return", "Done", { status: "success" });
// Returns: ["Done", { status: "success" }]
```

#### Throw error

```typescript
/**
 * Throws a custom error with message and code
 * @param message - Error message
 * @param code - Error code
 * @throws Exception with the specified code
 */
Process("utils.flow.Throw", "Operation failed", 500);
```

### Environment Variables

#### Get environment variable

```typescript
/**
 * Gets the value of an environment variable
 * @param name - Environment variable name
 * @returns string - Environment variable value
 */
const dbHost = Process("utils.env.Get", "DB_HOST");
```

#### Set environment variable

```typescript
/**
 * Sets the value of an environment variable
 * @param name - Environment variable name
 * @param value - Environment variable value
 * @returns null
 */
Process("utils.env.Set", "APP_MODE", "production");
```

#### Get multiple environment variables

```typescript
/**
 * Gets multiple environment variables
 * @param ...names - Environment variable names
 * @returns object - Map of environment variables
 */
const config = Process("utils.env.GetMany", "DB_HOST", "DB_PORT", "DB_USER");
// Returns: { "DB_HOST": "localhost", "DB_PORT": "5432", "DB_USER": "postgres" }
```

#### Set multiple environment variables

```typescript
/**
 * Sets multiple environment variables
 * @param variables - Map of environment variables
 * @returns null
 */
Process("utils.env.SetMany", {
  API_KEY: "abc123",
  API_SECRET: "xyz789",
  API_URL: "https://api.example.com",
});
```

### Authentication

#### Generate JWT token

```typescript
/**
 * Generates a JWT token
 * @param id - User ID or subject identifier
 * @param data - Data to include in the token
 * @param options - JWT options (optional)
 * @returns object - JWT token and expiration
 */
const token = Process(
  "utils.jwt.Make",
  1,
  { name: "John", role: "admin" },
  {
    timeout: 3600,
    subject: "Authentication",
    issuer: "YaoApp",
  }
);
// Returns: {
//   token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
//   expires_at: 1625097600
// }
```

#### Verify JWT token

```typescript
/**
 * Verifies a JWT token
 * @param token - JWT token to verify
 * @returns object - Token claims
 * @throws Exception if token is invalid
 */
const claims = Process(
  "utils.jwt.Verify",
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
);
// Returns: { id: 1, sid: "session_id", data: { name: "John", role: "admin" }, ... }
```

#### Verify password

```typescript
/**
 * Verifies a password against a hash
 * @param password - Plain text password
 * @param hash - Bcrypt hash to compare against
 * @returns boolean - True if password matches
 * @throws Exception if password is invalid
 */
const isValid = Process("utils.pwd.Verify", "mypassword", "$2a$10$...");
// Returns: true
// Throws exception if invalid
```

### Captcha

#### Generate captcha

```typescript
/**
 * Generates a captcha
 * @param options - Captcha options
 * @returns object - Captcha ID and image/audio content
 */
const captcha = Process("utils.captcha.Make", {
  width: 240,
  height: 80,
  length: 6,
  type: "image", // or "audio"
  lang: "en",
});
// Returns: {
//   id: "captcha_id",
//   content: "data:image/png;base64,..." // base64 encoded image or audio
// }
```

#### Verify captcha

```typescript
/**
 * Verifies a captcha code
 * @param id - Captcha ID
 * @param code - User input code
 * @returns boolean - True if captcha is valid
 * @throws Exception if captcha is invalid
 */
const isValid = Process("utils.captcha.Verify", "captcha_id", "123456");
// Returns: true
// Throws exception if invalid
```

### Array Operations

#### Get array values by column name

```typescript
/**
 * Extracts values from a specific column in an array of records
 * @param records - Array of records
 * @param column - Column name to extract
 * @returns array - Extracted values
 */
const ids = Process(
  "utils.arr.Column",
  [
    { id: 1, name: "John" },
    { id: 2, name: "Jane" },
    { id: 3, name: "Bob" },
  ],
  "id"
);
// Returns: [1, 2, 3]
```

#### Keep only specific columns

```typescript
/**
 * Keeps only specified columns in an array of records
 * @param records - Array of records
 * @param columns - Columns to keep
 * @returns array - Records with only specified columns
 */
const simplified = Process(
  "utils.arr.Keep",
  [
    { id: 1, name: "John", email: "john@example.com", role: "admin" },
    { id: 2, name: "Jane", email: "jane@example.com", role: "user" },
  ],
  ["id", "name"]
);
// Returns: [
//   { id: 1, name: "John" },
//   { id: 2, name: "Jane" }
// ]
```

#### Pluck values from records

```typescript
/**
 * Transforms an array of records based on specified columns
 * @param columns - Columns to include
 * @param data - Input data
 * @returns array - Transformed data
 */
const users = Process(
  "utils.arr.Pluck",
  ["id", "full_name"],
  [
    { id: 1, first_name: "John", last_name: "Doe" },
    { id: 2, first_name: "Jane", last_name: "Smith" },
  ]
);
// Can transform data based on column mapping
```

#### Split records into columns and values

```typescript
/**
 * Splits records into column names and value arrays
 * @param records - Array of records
 * @returns object - Contains columns array and values matrix
 */
const split = Process("utils.arr.Split", [
  { id: 1, name: "John", age: 30 },
  { id: 2, name: "Jane", age: 25 },
]);
// Returns: {
//   columns: ["id", "name", "age"],
//   values: [
//     [1, "John", 30],
//     [2, "Jane", 25]
//   ]
// }
```

#### Get array indexes

```typescript
/**
 * Gets the indexes of an array
 * @param array - Input array
 * @returns array - Array indexes
 */
const indexes = Process("utils.arr.Indexes", ["apple", "banana", "orange"]);
// Returns: [0, 1, 2]
```

#### Convert array to tree structure

```typescript
/**
 * Converts a flat array to a tree structure
 * @param records - Array of records
 * @param options - Tree configuration
 * @returns array - Tree structure
 */
const tree = Process(
  "utils.arr.Tree",
  [
    { id: 1, parent_id: null, name: "Parent" },
    { id: 2, parent_id: 1, name: "Child 1" },
    { id: 3, parent_id: 1, name: "Child 2" },
  ],
  {
    parent: "parent_id",
    empty: null,
    children: "children",
    id: "id",
  }
);
// Returns hierarchical tree structure
```

#### Remove duplicate values

```typescript
/**
 * Removes duplicate values from an array
 * @param array - Input array
 * @returns array - Array with unique values
 */
const unique = Process("utils.arr.Unique", [1, 2, 2, 3, 3, 3, 4]);
// Returns: [1, 2, 3, 4]
```

#### Get item by index

```typescript
/**
 * Gets an item from an array by index
 * @param array - Input array
 * @param index - Array index
 * @returns any - Item at the specified index
 */
const item = Process("utils.arr.Get", ["apple", "banana", "orange"], 1);
// Returns: "banana"
```

#### Set values in array of maps

```typescript
/**
 * Sets a value for a specific key in all maps in an array
 * @param array - Array of maps
 * @param key - Key to set
 * @param value - Value to set
 * @returns array - Updated array
 */
const updated = Process(
  "utils.arr.MapSet",
  [
    { id: 1, name: "John" },
    { id: 2, name: "Jane" },
  ],
  "active",
  true
);
// Returns: [
//   { id: 1, name: "John", active: true },
//   { id: 2, name: "Jane", active: true }
// ]
```

### Map Operations

#### Get a value from a map

```typescript
/**
 * Gets a value from a map by key
 * @param map - Input map
 * @param key - Key to retrieve
 * @returns any - Value associated with the key
 */
const name = Process("utils.map.Get", { id: 1, name: "John", age: 30 }, "name");
// Returns: "John"
```

#### Set a value in a map

```typescript
/**
 * Sets a value in a map
 * @param map - Input map
 * @param key - Key to set
 * @param value - Value to set
 * @returns object - Updated map
 */
const updated = Process("utils.map.Set", { name: "John" }, "age", 30);
// Returns: { name: "John", age: 30 }
```

#### Delete a key from a map

```typescript
/**
 * Deletes a key from a map
 * @param map - Input map
 * @param key - Key to delete
 * @returns object - Updated map
 */
const smaller = Process(
  "utils.map.Del",
  { id: 1, name: "John", temp: "xyz" },
  "temp"
);
// Returns: { id: 1, name: "John" }
```

#### Delete multiple keys from a map

```typescript
/**
 * Deletes multiple keys from a map
 * @param map - Input map
 * @param ...keys - Keys to delete
 * @returns object - Updated map
 */
const filtered = Process(
  "utils.map.DelMany",
  { id: 1, name: "John", password: "secret", token: "xyz" },
  "password",
  "token"
);
// Returns: { id: 1, name: "John" }
```

#### Get all keys from a map

```typescript
/**
 * Gets all keys from a map
 * @param map - Input map
 * @returns array - Array of keys
 */
const keys = Process("utils.map.Keys", { id: 1, name: "John", age: 30 });
// Returns: ["id", "name", "age"]
```

#### Get all values from a map

```typescript
/**
 * Gets all values from a map
 * @param map - Input map
 * @returns array - Array of values
 */
const values = Process("utils.map.Values", { id: 1, name: "John", age: 30 });
// Returns: [1, "John", 30]
```

#### Convert map to array

```typescript
/**
 * Converts a map to an array of key-value pairs
 * @param map - Input map
 * @returns array - Array of key-value objects
 */
const array = Process("utils.map.Array", { id: 1, name: "John" });
// Returns: [
//   { key: "id", value: 1 },
//   { key: "name", value: "John" }
// ]
```

## Complete Workflow Example

```typescript
// Generate a UUID
const id = Process("utils.str.UUID");

// Get current timestamp
const timestamp = Process("utils.now.Timestamp");

// Create a path
const path = Process("utils.str.JoinPath", "data", id, "file.txt");

// Print colored info
Process("utils.fmt.ColorPrintf", "blue", "Processing request with ID: %s", id);

// Parse URL parameters
const url = "https://example.com/api?token=123&id=" + id;
const parsedUrl = Process("utils.url.ParseURL", url);

// Handle errors conditionally
if (!parsedUrl.query.token) {
  Process("utils.throw.Unauthorized", "Missing token");
}

// Generate a JWT token
const token = Process(
  "utils.jwt.Make",
  1,
  { id: id, timestamp: timestamp },
  {
    timeout: 3600,
    subject: "API Access",
  }
);

// Use flow control for conditional processing
Process(
  "utils.flow.Case",
  {
    when: [{ operator: "gt", value: 0, field: "status" }],
    process: "utils.flow.Return",
    args: [{ token: token.token, path: path }],
  },
  {
    when: [{ operator: "eq", value: 0, field: "status" }],
    process: "utils.throw.BadRequest",
    args: ["Invalid status"],
  }
);

// Get current date and time
const now = Process("utils.now.DateTime");

// Print a success message
Process("utils.fmt.ColorPrintf", "green", "Operation completed at %s", now);
```

## Notes

- The utils module provides a variety of helper functions for common operations.
- String utilities help with text manipulation, path joining, and UUID generation.
- Date/time functions provide current date and time in various formats.
- Error handling functions provide standardized HTTP error responses.
- URL functions help parse and manipulate URL strings and query parameters.
- Formatting functions allow console output with color support.
- Flow control functions provide conditional execution and iteration capabilities.
- Environment functions allow reading and writing environment variables.
- Authentication functions handle JWT tokens and password verification.
- Array and map functions provide powerful data manipulation capabilities.
