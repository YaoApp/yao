# Yao Excel Module

A Go module for manipulating Excel files with TypeScript API support.

## IMPORTANT: Always Close Resources

**IMPORTANT**: Always make sure to close Excel file handles using `excel.close` when done to prevent memory leaks and file locking issues. Failing to close handles may cause file corruption or application errors.

## Quick Example

Here's a simple but complete example showing proper resource management:

```typescript
// Open an Excel file
const h = Process("excel.open", "data.xlsx", true);

// Perform operations
const sheets = Process("excel.sheets", h);
Process("excel.write.cell", h, sheets[0], "A1", "Hello World");
Process("excel.save", h);

// IMPORTANT: Always close the handle when done
Process("excel.close", h);
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
const h: string = Process("excel.open", "file.xlsx", true);

// Open in read-only mode (false parameter or not passed)
const hRead: string = Process("excel.open", "file.xlsx", false);
// or simply
const h2: string = Process("excel.open", "file.xlsx");

// IMPORTANT: Don't forget to close the handle when done
// Process("excel.close", h);
```

#### Get all sheets in the workbook

```typescript
/**
 * Gets all sheet names in the workbook
 * @param handle - Handle ID from excel.open
 * @returns string[] - Array of sheet names
 */
const sheets: string[] = Process("excel.sheets", h);
// Example output: ["Sheet1", "Sheet2"]
```

#### Close a file

```typescript
/**
 * Closes an Excel file
 * @param handle - Handle ID from excel.open
 * @returns null
 */
Process("excel.close", h);
```

#### Save changes to file

```typescript
/**
 * Saves changes to the Excel file
 * @param handle - Handle ID from excel.open
 * @returns null
 */
Process("excel.save", h);
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
const value: string = Process("excel.read.cell", h, "SheetName", "A1");
```

#### Read all rows

```typescript
/**
 * Reads all rows in a sheet
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @returns string[][] - Two-dimensional array of cell values
 */
const rows: string[][] = Process("excel.read.row", h, "SheetName");
```

#### Read all columns

```typescript
/**
 * Reads all columns in a sheet
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @returns string[][] - Two-dimensional array of cell values
 */
const columns: string[][] = Process("excel.read.column", h, "SheetName");
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
Process("excel.write.cell", h, "SheetName", "A1", "Hello World");
// Can write different types of values
Process("excel.write.cell", h, "SheetName", "A2", 123.45);
Process("excel.write.cell", h, "SheetName", "A3", true);
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
Process("excel.write.row", h, "SheetName", "A1", ["Cell1", "Cell2", "Cell3"]);
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
Process("excel.write.column", h, "SheetName", "A1", ["Row1", "Row2", "Row3"]);
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
Process("excel.write.all", h, "SheetName", "A1", [
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
Process("excel.set.style", h, "SheetName", "A1", 1);
```

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
Process("excel.set.rowheight", h, "SheetName", 1, 30); // Set row 1 to 30 pts height
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
Process("excel.set.columnwidth", h, "SheetName", "A", "B", 20);
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
Process("excel.set.mergecell", h, "SheetName", "A1", "B2");
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
Process("excel.set.unmergecell", h, "SheetName", "A1", "B2");
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
Process("excel.set.formula", h, "SheetName", "C1", "SUM(A1:B1)");
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
  "excel.set.link",
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
const rid: string = Process("excel.each.openrow", h, "SheetName");

/**
 * Gets the next row from the iterator
 * @param rowID - Row iterator ID from excel.each.openrow
 * @returns string[] | null - Array of cell values or null if no more rows
 */
let row: string[] | null;
while ((row = Process("excel.each.nextrow", rid)) !== null) {
  // Process the row
  console.log(row);
}

/**
 * IMPORTANT: Always close the row iterator when done
 * @param rowID - Row iterator ID from excel.each.openrow
 * @returns null
 */
Process("excel.each.closerow", rid);
```

#### Column Iterator

```typescript
/**
 * Opens a column iterator
 * @param handle - Handle ID from excel.open
 * @param sheet - Sheet name
 * @returns string - Column iterator ID
 */
const cid: string = Process("excel.each.opencolumn", h, "SheetName");

/**
 * Gets the next column from the iterator
 * @param colID - Column iterator ID from excel.each.opencolumn
 * @returns string[] | null - Array of cell values or null if no more columns
 */
let col: string[] | null;
while ((col = Process("excel.each.nextcolumn", cid)) !== null) {
  // Process the column
  console.log(col);
}

/**
 * IMPORTANT: Always close the column iterator when done
 * @param colID - Column iterator ID from excel.each.opencolumn
 * @returns null
 */
Process("excel.each.closecolumn", cid);
```

### Utility Functions

#### Convert between column names and indices

```typescript
/**
 * Converts a column name to a column number
 * @param colName - Column name (e.g. "A", "AB")
 * @returns number - Column number (1-based)
 */
const colNum: number = Process("excel.convert.columnnametonumber", "AK"); // Returns 37

/**
 * Converts a column number to a column name
 * @param colNum - Column number (1-based)
 * @returns string - Column name
 */
const colName: string = Process("excel.convert.columnnumbertoname", 37); // Returns "AK"
```

#### Convert between cell references and coordinates

```typescript
/**
 * Converts a cell reference to coordinates
 * @param cell - Cell reference (e.g. "A1")
 * @returns number[] - Array with [columnNumber, rowNumber] (1-based)
 */
const coords: number[] = Process("excel.convert.cellnametocoordinates", "A1"); // Returns [1, 1]

/**
 * Converts coordinates to a cell reference
 * @param col - Column number (1-based)
 * @param row - Row number (1-based)
 * @returns string - Cell reference
 */
const cellName: string = Process("excel.convert.coordinatestocellname", 1, 1); // Returns "A1"
```

## Complete Workflow Example

```typescript
// Open Excel file in writable mode
const h: string = Process("excel.open", "file.xlsx", true);

// Get available sheets
const sheets: string[] = Process("excel.sheets", h);
const sheetName: string = sheets[0];

// Read some data
const value: string = Process("excel.read.cell", h, sheetName, "A1");
console.log("Cell A1 contains:", value);

// Write data
Process("excel.write.cell", h, sheetName, "B1", "New Value");
Process("excel.write.row", h, sheetName, "A2", ["Data1", "Data2", "Data3"]);

// Add a formula
Process("excel.set.formula", h, sheetName, "D1", "SUM(A1:C1)");

// Format cells
Process("excel.set.rowheight", h, sheetName, 1, 30);
Process("excel.set.columnwidth", h, sheetName, "A", "D", 15);

// Save changes
Process("excel.save", h);

// IMPORTANT: Always close the handle when done
Process("excel.close", h);
```

## Notes

- Always make sure to close open file handles using `excel.close` when done to prevent resource leaks and file locking issues.
- Remember to save changes with `excel.save` before closing to ensure all modifications are persisted.
- For performance reasons, try to batch operations where possible instead of making many small changes.
