# Yao Excel Module

A Go module for manipulating Excel files with TypeScript API support.

## IMPORTANT: Always Close Resources

**IMPORTANT**: Always make sure to close Excel file handles using `excel.close` when done to prevent memory leaks and file locking issues. Failing to close handles may cause file corruption or application errors.

## Quick Example

Here's a simple but complete example showing proper resource management:

```typescript
// Open an Excel file
const h = Process("excel.Open", "data.xlsx", true);

// Perform operations
const sheets = Process("excel.Sheets", h);
Process("excel.write.Cell", h, sheets[0], "A1", "Hello World");
Process("excel.Save", h);

// IMPORTANT: Always close the handle when done
Process("excel.Close", h);
```

## Usage in TypeScript

You can use the Excel module in TypeScript through the Process API. Below are examples of common operations with return type descriptions.

### Basic Operations

#### Open an Excel file

```typescript
/**
 * Opens an Excel file
 * @param path - Path to the Excel file
 * @param writable - Whether to open in writable mode (true) or read-only mode (false)
 * @returns string - Handle ID used for subsequent operations
 */
const h: string = Process("excel.Open", "file.xlsx", true);

// Open in read-only mode (false parameter or not passed)
const hRead: string = Process("excel.Open", "file.xlsx", false);
// or simply
const h2: string = Process("excel.Open", "file.xlsx");

// IMPORTANT: Don't forget to close the handle when done
// Process("excel.Close", h);
```

### Sheet Operations

#### Create a new sheet

```typescript
/**
 * Creates a new sheet in the workbook
 * @param handle - Handle ID from excel.open
 * @param name - Name for the new sheet
 * @returns number - Index of the new sheet
 */
const idx: number = Process("excel.sheet.create", h, "NewSheet");
```

#### List all sheets

```typescript
/**
 * Lists all sheets in the workbook
 * @param handle - Handle ID from excel.open
 * @returns string[] - Array of sheet names
 */
const sheets: string[] = Process("excel.sheet.list", h);
// Example output: ["Sheet1", "Sheet2", "NewSheet"]
```

#### Read sheet data

```typescript
/**
 * Reads all data from a sheet
 * @param handle - Handle ID from excel.open
 * @param name - Sheet name
 * @returns any[][] - Two-dimensional array of cell values
 */
const data: any[][] = Process("excel.sheet.read", h, "Sheet1");
```

#### Update sheet data

```typescript
/**
 * Updates data in a sheet. Creates the sheet if it doesn't exist.
 * @param handle - Handle ID from excel.open
 * @param name - Sheet name
 * @param data - Two-dimensional array of values to write
 * @returns null
 */
const data = [
  ["Header1", "Header2", "Header3"],
  [1, "Data1", true],
  [2, "Data2", false],
];
Process("excel.sheet.update", h, "Sheet1", data);
```

#### Copy a sheet

```typescript
/**
 * Copies a sheet with all its content and formatting
 * @param handle - Handle ID from excel.open
 * @param source - Source sheet name
 * @param target - Target sheet name (must not exist)
 * @returns null
 */
Process("excel.sheet.copy", h, "Sheet1", "Sheet1Copy");
```

#### Delete a sheet

```typescript
/**
 * Deletes a sheet from the workbook
 * @param handle - Handle ID from excel.open
 * @param name - Sheet name to delete
 * @returns null
 */
Process("excel.sheet.delete", h, "Sheet1Copy");
```

#### Check if a sheet exists

```typescript
/**
 * Checks if a sheet exists in the workbook
 * @param handle - Handle ID from excel.open
 * @param name - Sheet name to check
 * @returns boolean - true if sheet exists, false otherwise
 */
const exists: boolean = Process("excel.sheet.exists", h, "Sheet1");
```

#### Read sheet rows with pagination

```typescript
/**
 * Reads rows from a sheet with pagination support
 * @param handle - Handle ID from excel.open
 * @param name - Sheet name
 * @param start - Starting row index (0-based)
 * @param size - Number of rows to read
 * @returns string[][] - Two-dimensional array of cell values
 */
const rows: string[][] = Process("excel.sheet.rows", h, "Sheet1", 0, 10); // Read first 10 rows
```

#### Get sheet dimensions

```typescript
/**
 * Gets the dimensions (number of rows and columns) of a sheet
 * @param handle - Handle ID from excel.open
 * @param name - Sheet name
 * @returns {rows: number, cols: number} - Object containing row and column counts
 */
const dim: { rows: number; cols: number } = Process(
  "excel.sheet.dimension",
  h,
  "Sheet1"
);
console.log(`Sheet has ${dim.rows} rows and ${dim.cols} columns`);
```

### Example: Sheet Operations Workflow

```typescript
// Open Excel file in writable mode
const h: string = Process("excel.Open", "file.xlsx", true);

// Create a new sheet
const idx: number = Process("excel.sheet.create", h, "DataSheet");

// Write some data to the new sheet
const data = [
  ["Name", "Age", "Active"],
  ["John", 30, true],
  ["Jane", 25, false],
];
Process("excel.sheet.update", h, "DataSheet", data);

// Make a backup copy of the sheet
Process("excel.sheet.copy", h, "DataSheet", "DataSheet_Backup");

// List all sheets to verify
const sheets: string[] = Process("excel.sheet.list", h);
console.log("Available sheets:", sheets);

// Read data from the backup sheet
const backupData: any[][] = Process("excel.sheet.read", h, "DataSheet_Backup");
console.log("Backup data:", backupData);

// Delete the backup sheet when no longer needed
Process("excel.sheet.delete", h, "DataSheet_Backup");

// Save changes
Process("excel.Save", h);

// IMPORTANT: Always close the handle when done
Process("excel.Close", h);
```

#### Get all sheets in the workbook

```typescript
/**
 * Gets all sheet names in the workbook
 * @param handle - Handle ID from excel.open
 * @returns string[] - Array of sheet names
 */
const sheets: string[] = Process("excel.Sheets", h);
// Example output: ["Sheet1", "Sheet2"]
```

#### Close a file

```typescript
/**
 * Closes an Excel file
 * @param handle - Handle ID from excel.open
 * @returns null
 */
Process("excel.Close", h);
```

#### Save changes to file

```typescript
/**
 * Saves changes to the Excel file
 * @param handle - Handle ID from excel.open
 * @returns null
 */
Process("excel.Save", h);
```

### Reading Data

#### Read a cell's value

```typescript
/**
 * Reads a cell's value
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param cell - Cell reference (e.g. "A1")
 * @returns string - Cell value
 */
const value: string = Process("excel.read.Cell", h, "SheetName", "A1");
```

#### Read all rows

```typescript
/**
 * Reads all rows in a sheet
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @returns string[][] - Two-dimensional array of cell values
 */
const rows: string[][] = Process("excel.read.Row", h, "SheetName");
```

#### Read all columns

```typescript
/**
 * Reads all columns in a sheet
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @returns string[][] - Two-dimensional array of cell values
 */
const columns: string[][] = Process("excel.read.Column", h, "SheetName");
```

### Writing Data

#### Write to a cell

```typescript
/**
 * Writes a value to a cell
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param cell - Cell reference (e.g. "A1")
 * @param value - Value to write (string, number, boolean, etc.)
 * @returns null
 */
Process("excel.write.Cell", h, "SheetName", "A1", "Hello World");
// Can write different types of values
Process("excel.write.Cell", h, "SheetName", "A2", 123.45);
Process("excel.write.Cell", h, "SheetName", "A3", true);
```

#### Write a row

```typescript
/**
 * Writes values to a row starting at the specified cell
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param startCell - Starting cell reference (e.g. "A1")
 * @param values - Array of values to write
 * @returns null
 */
Process("excel.write.Row", h, "SheetName", "A1", ["Cell1", "Cell2", "Cell3"]);
```

#### Write a column

```typescript
/**
 * Writes values to a column starting at the specified cell
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param startCell - Starting cell reference (e.g. "A1")
 * @param values - Array of values to write
 * @returns null
 */
Process("excel.write.Column", h, "SheetName", "A1", ["Row1", "Row2", "Row3"]);
```

#### Write multiple rows

```typescript
/**
 * Writes a two-dimensional array of values starting at the specified cell
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param startCell - Starting cell reference (e.g. "A1")
 * @param values - Two-dimensional array of values to write
 * @returns null
 */
Process("excel.write.All", h, "SheetName", "A1", [
  ["Row1Cell1", "Row1Cell2", "Row1Cell3"],
  ["Row2Cell1", "Row2Cell2", "Row2Cell3"],
]);
```

### Formatting and Styling

#### Set cell style

```typescript
/**
 * Sets a cell's style
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param cell - Cell reference (e.g. "A1")
 * @param styleID - Style ID
 * @returns null
 */
Process("excel.set.Style", h, "SheetName", "A1", 1);
```

#### Style ID Constants

When using `excel.set.Style`, you need to provide a style ID. The following style IDs are supported:

```typescript
// Border styles
const BORDER_NONE = 0; // No border
const BORDER_CONTINUOUS = 1; // Continuous border (thin)
const BORDER_CONTINUOUS_2 = 2; // Continuous border (medium)
const BORDER_DASH = 3; // Dashed border
const BORDER_DOT = 4; // Dotted border
const BORDER_CONTINUOUS_3 = 5; // Continuous border (thick)
const BORDER_DOUBLE = 6; // Double line border
const BORDER_CONTINUOUS_0 = 7; // Continuous border (hair)
const BORDER_DASH_2 = 8; // Dashed border (medium)
const BORDER_DASH_DOT = 9; // Dash-dot border
const BORDER_DASH_DOT_2 = 10; // Dash-dot border (medium)
const BORDER_DASH_DOT_DOT = 11; // Dash-dot-dot border
const BORDER_DASH_DOT_DOT_2 = 12; // Dash-dot-dot border (medium)
const BORDER_SLANT_DASH_DOT = 13; // Slanted dash-dot border

// Fill patterns
const FILL_NONE = 0; // No fill
const FILL_SOLID = 1; // Solid fill
const FILL_MEDIUM_GRAY = 2; // Medium gray fill
const FILL_DARK_GRAY = 3; // Dark gray fill
const FILL_LIGHT_GRAY = 4; // Light gray fill
const FILL_DARK_HORIZONTAL = 5; // Dark horizontal line pattern
const FILL_DARK_VERTICAL = 6; // Dark vertical line pattern
const FILL_DARK_DOWN = 7; // Dark diagonal down pattern
const FILL_DARK_UP = 8; // Dark diagonal up pattern
const FILL_DARK_GRID = 9; // Dark grid pattern
const FILL_DARK_TRELLIS = 10; // Dark trellis pattern
const FILL_LIGHT_HORIZONTAL = 11; // Light horizontal line pattern
const FILL_LIGHT_VERTICAL = 12; // Light vertical line pattern
const FILL_LIGHT_DOWN = 13; // Light diagonal down pattern
const FILL_LIGHT_UP = 14; // Light diagonal up pattern
const FILL_LIGHT_GRID = 15; // Light grid pattern
const FILL_LIGHT_TRELLIS = 16; // Light trellis pattern
const FILL_GRAY_125 = 17; // 12.5% gray fill
const FILL_GRAY_0625 = 18; // 6.25% gray fill
```

Example of creating a custom style with borders and fill:

```typescript
// Create style with thick border and light gray fill
const styleID = 1; // This would typically be a custom style ID created via the NewStyle API

// Apply the style to cell A1
Process("excel.set.Style", h, "SheetName", "A1", styleID);
```

Note: The excelize library supports creating custom styles through the `NewStyle` function. Currently, in the Yao Excel module, only predefined style IDs are supported. For more complex styling needs, consider creating a custom style in the future versions of the API.

#### Set row height

```typescript
/**
 * Sets a row's height
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param row - Row number
 * @param height - Height in points
 * @returns null
 */
Process("excel.set.RowHeight", h, "SheetName", 1, 30); // Set row 1 to 30 pts height
```

#### Set column width

```typescript
/**
 * Sets column width for a range of columns
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param startCol - Starting column letter
 * @param endCol - Ending column letter
 * @param width - Width in points
 * @returns null
 */
Process("excel.set.ColumnWidth", h, "SheetName", "A", "B", 20);
```

#### Merge cells

```typescript
/**
 * Merges cells in a range
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param startCell - Starting cell reference (e.g. "A1")
 * @param endCell - Ending cell reference (e.g. "B2")
 * @returns null
 */
Process("excel.set.MergeCell", h, "SheetName", "A1", "B2");
```

#### Unmerge cells

```typescript
/**
 * Unmerges previously merged cells
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param startCell - Starting cell reference (e.g. "A1")
 * @param endCell - Ending cell reference (e.g. "B2")
 * @returns null
 */
Process("excel.set.UnmergeCell", h, "SheetName", "A1", "B2");
```

#### Set a formula

```typescript
/**
 * Sets a formula in a cell
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param cell - Cell reference (e.g. "C1")
 * @param formula - Excel formula without the leading equals sign
 * @returns null
 */
Process("excel.set.Formula", h, "SheetName", "C1", "SUM(A1:B1)");
```

#### Add a hyperlink

```typescript
/**
 * Adds a hyperlink to a cell
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @param cell - Cell reference (e.g. "A1")
 * @param url - URL for the hyperlink
 * @param text - Display text for the hyperlink
 * @returns null
 */
Process(
  "excel.set.Link",
  h,
  "SheetName",
  "A1",
  "https://example.com",
  "Visit Example"
);
```

### Iterating Through Data

#### Row Iterator

```typescript
/**
 * Opens a row iterator
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @returns string - Row iterator ID
 */
const rid: string = Process("excel.each.OpenRow", h, "SheetName");

/**
 * Gets the next row from the iterator
 * @param rowID - Row iterator ID from excel.each.openrow
 * @returns string[] | null - Array of cell values or null if no more rows
 */
let row: string[] | null;
while ((row = Process("excel.each.NextRow", rid)) !== null) {
  // Process the row
  console.log(row);
}

/**
 * IMPORTANT: Always close the row iterator when done
 * @param rowID - Row iterator ID from excel.each.openrow
 * @returns null
 */
Process("excel.each.CloseRow", rid);
```

#### Column Iterator

```typescript
/**
 * Opens a column iterator
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @returns string - Column iterator ID
 */
const cid: string = Process("excel.each.OpenColumn", h, "SheetName");

/**
 * Gets the next column from the iterator
 * @param colID - Column iterator ID from excel.each.opencolumn
 * @returns string[] | null - Array of cell values or null if no more columns
 */
let col: string[] | null;
while ((col = Process("excel.each.NextColumn", cid)) !== null) {
  // Process the column
  console.log(col);
}

/**
 * IMPORTANT: Always close the column iterator when done
 * @param colID - Column iterator ID from excel.each.opencolumn
 * @returns null
 */
Process("excel.each.CloseColumn", cid);
```

### Utility Functions

#### Convert between column names and indices

```typescript
/**
 * Converts a column name to a column number
 * @param colName - Column name (e.g. "A", "AB")
 * @returns number - Column number (1-based)
 */
const colNum: number = Process("excel.convert.ColumnNameToNumber", "AK"); // Returns 37

/**
 * Converts a column number to a column name
 * @param colNum - Column number (1-based)
 * @returns string - Column name
 */
const colName: string = Process("excel.convert.ColumnNumberToName", 37); // Returns "AK"
```

#### Convert between cell references and coordinates

```typescript
/**
 * Converts a cell reference to coordinates
 * @param cell - Cell reference (e.g. "A1")
 * @returns number[] - Array with [columnNumber, rowNumber] (1-based)
 */
const coords: number[] = Process("excel.convert.CellNameToCoordinates", "A1"); // Returns [1, 1]

/**
 * Converts coordinates to a cell reference
 * @param col - Column number (1-based)
 * @param row - Row number (1-based)
 * @returns string - Cell reference
 */
const cellName: string = Process("excel.convert.CoordinatesToCellName", 1, 1); // Returns "A1"
```

## Complete Workflow Example

```typescript
// Open Excel file in writable mode
const h: string = Process("excel.Open", "file.xlsx", true);

// Get available sheets
const sheets: string[] = Process("excel.Sheets", h);
const sheetName: string = sheets[0];

// Read some data
const value: string = Process("excel.read.Cell", h, sheetName, "A1");
console.log("Cell A1 contains:", value);

// Write data
Process("excel.write.Cell", h, sheetName, "B1", "New Value");
Process("excel.write.Row", h, sheetName, "A2", ["Data1", "Data2", "Data3"]);

// Add a formula
Process("excel.set.Formula", h, sheetName, "D1", "SUM(A1:C1)");

// Format cells
Process("excel.set.RowHeight", h, sheetName, 1, 30);
Process("excel.set.ColumnWidth", h, sheetName, "A", "D", 15);

// Save changes
Process("excel.Save", h);

// IMPORTANT: Always close the handle when done
Process("excel.Close", h);
```

## Notes

- Always make sure to close open file handles using `excel.close` when done to prevent resource leaks and file locking issues.
- Remember to save changes with `excel.save` before closing to ensure all modifications are persisted.
- For performance reasons, try to batch operations where possible instead of making many small changes.
