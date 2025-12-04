# Content Processing Package

This package handles content transformation for multimodal messages in agent conversations. It is called **BEFORE** sending messages to the LLM and converts extended content types into standard LLM-compatible formats.

## ⚠️ Critical Design Principle

**Input**: Messages with extended content types (`file`, `data`, etc.)  
**Output**: Messages with ONLY standard LLM-compatible types (`text`, `image_url`, `input_audio`)

The LLM should NEVER receive `type="file"` or `type="data"` content parts. These MUST be converted to `text` (or `image_url` for images if model supports vision).

## Architecture

```
Vision (main entry)
    ↓
Initialize processedFiles cache (map[fileID]text)
    ↓
processMessage (for each message)
    ↓
processContentPart (for each content part)
    ↓
Is uploader wrapper?
    ├── Yes → Check cache
    │   ├── In cache? → Use cached text ✅
    │   └── Not in cache → Try GetText(fileID) preview
    │       ├── Has preview? → Use preview + cache ✅
    │       └── No preview → Proceed to full processing ↓
    └── No (HTTP/other) → Proceed to full processing ↓
    ↓
├── Fetch content (if needed)
│   ├── HTTP URL
│   └── Uploader Wrapper (__uploader://fileid)
    ↓
├── Determine Processing Strategy
│   ├── Model supports? → Format for model
│   └── Model doesn't support? → Use agent/MCP
    ↓
ProcessorRegistry
    ↓
├── ImageProcessor
├── AudioProcessor
├── PDFProcessor
├── WordProcessor
├── ExcelProcessor
└── TextProcessor
    ↓
Cache result (if uploader wrapper)
```

## Content Type Transformation

### Input → Output Mapping

| Input Type | Model Supports? | Output Type | Processing |
|------------|-----------------|-------------|------------|
| `text` | - | `text` | Pass through |
| `image_url` | ✅ Yes | `image_url` | Convert format if needed (base64/URL) |
| `image_url` | ❌ No | `text` | Use vision agent/MCP to describe |
| `input_audio` | ✅ Yes | `input_audio` | Keep as audio |
| `input_audio` | ❌ No | `text` | Transcribe using audio agent/MCP |
| `file` (image) | ✅ Yes | `image_url` | Same as image_url processing |
| `file` (image) | ❌ No | `text` | Use vision tool to describe |
| `file` (document) | - | `text` | Extract text from PDF/Word/Excel/etc |
| `data` | - | `text` | Fetch and format data sources |

### 1. Images and Audio

**If model supports (vision/audio capability):**

- Keep as multimodal content:
  - `image_url`: Convert to appropriate format (OpenAI URL vs Claude base64)
  - `input_audio`: Convert to base64 format

**If model doesn't support:**

- Convert to text:
  - Use agent/MCP specified in `uses.Vision` or `uses.Audio`
  - Extract text description or transcription
  - Return as `type="text"` content

**HTTP URLs:**

- Fetch content first
- Then process the same way as above

### 2. Files (type="file")

**Critical**: All `type="file"` content MUST be converted to `text` or `image_url` (if image and model supports).

**Processing Steps:**

1. **Fetch file content**:
   - Uploader wrapper: `__uploader://fileid` → Parse and fetch from attachment manager
   - HTTP URL: Download from URL

2. **Detect file type** from content-type and magic bytes

3. **Process based on file type**:

| File Type    | Output Type | Processing Method                                                                                        |
| ------------ | ----------- | -------------------------------------------------------------------------------------------------------- |
| **Image**    | `image_url` or `text` | If model supports vision → `image_url`<br>If not → use vision tool → `text` |
| **PDF**      | `text` | If `uses.Vision` supports PDF → use vision tool<br>Otherwise → extract text directly |
| **Word**     | `text` | Extract text using Word document parser |
| **Excel**    | `text` | Extract and format as readable table/CSV |
| **PPT**      | `text` | Extract text and slide content |
| **CSV**      | `text` | Format as readable table |
| **Text**     | `text` | Read directly (with encoding detection) |
| **JSON/XML** | `text` | Pretty print for readability |

### 3. Data Sources (type="data")

**Critical**: All `type="data"` content MUST be converted to `text`.

**Processing Steps:**

1. **Parse DataContent.Sources** array
2. **Fetch data** from each source:
   - `model`: Query data model
   - `kb_collection`: Search knowledge base collection
   - `kb_document`: Get document content
   - `table`: Query database table
   - `api`: Call API endpoint
   - `mcp_resource`: Fetch MCP resource
3. **Format as readable text**:
   - Tables: Format as markdown tables or CSV
   - Documents: Include title and content
   - JSON: Pretty print
4. **Return as** `type="text"` content

## Components

### Core Files

- **content.go** - Main entry point (`Vision` function)
- **types.go** - Type definitions and constants
- **interfaces.go** - Interface definitions

### Fetching

- **fetch.go** - Fetch content from HTTP or uploader

### Processors

- **processor.go** - Processor registry and routing
- **image.go** - Image processing
- **audio.go** - Audio processing
- **pdf.go** - PDF document processing
- **word.go** - Word document processing
- **excel.go** - Excel spreadsheet processing
- **text.go** - Plain text and CSV processing

## Frontend Message Format

The frontend (InputArea) sends messages in the following format:

### Image Attachments
```json
{
  "type": "image_url",
  "image_url": {
    "url": "__yao.attachment://file_id",
    "detail": "auto"
  }
}
```

### File Attachments
```json
{
  "type": "file",
  "file": {
    "url": "__yao.attachment://file_id",
    "filename": "document.pdf"
  }
}
```

The `url` field contains an uploader wrapper in the format `__uploader://fileid`.

## Data Structures

### ContentInfo

Holds information about content to be processed:

```go
type ContentInfo struct {
    Source      ContentSource  // http, uploader, base64, local
    FileType    FileType       // image, audio, pdf, word, excel, etc.
    ContentType string         // MIME type
    URL         string         // Original URL or file ID
    Data        []byte         // File data

    // For uploader wrapper
    UploaderName string
    FileID       string
}
```

### ProcessedContent

Result of content processing:

```go
type ProcessedContent struct {
    Text        string              // Extracted text
    ContentPart *context.ContentPart // For model input
    Metadata    map[string]interface{}
    Error       error
}
```

## Usage Example

```go
import (
    "github.com/yaoapp/yao/agent/content"
    "github.com/yaoapp/yao/agent/context"
)

// Process messages before sending to LLM
processedMessages, err := content.Vision(
    ctx,
    capabilities,  // Model capabilities
    messages,      // Original messages
    uses,         // Tool specifications (vision, audio, etc.)
)
```

## Performance Optimization

### File Processing Cache

**Problem**: Same file (uploader wrapper) might appear in multiple messages or be referenced multiple times.

**Solution**: Three-level caching strategy:

1. **In-memory cache** (`processedFiles` map):
   - Caches processed text for the duration of the Vision() call
   - Key: file ID from uploader wrapper
   - Value: extracted text content

2. **Attachment preview** (attachment.GetText with preview):
   - Tries to get preview (first 2000 chars) from attachment manager
   - If file was previously processed and saved, preview is available immediately
   - Much faster than full file processing

3. **Full processing** (only if needed):
   - Falls back to complete file processing if no cache/preview available
   - Result is cached in memory and optionally saved to attachment manager

### Cache Flow

```go
// For uploader://file_id
1. Check processedFiles[file_id]
   └── Found? → Return cached text ⚡ (fastest)

2. Not in cache → Call attachment.GetText(file_id, false) // preview only
   └── Has preview? → Cache and return ⚡ (fast)

3. No preview → Process file fully 🔄 (slower)
   └── Cache result in processedFiles
   └── Optional: Save to attachment using SaveText for future use
```

### Benefits

- **Avoid duplicate processing**: Same file processed only once per Vision() call
- **Fast preview access**: Leverage pre-processed content from attachment manager
- **Reduced latency**: Especially important for large documents (PDFs, Word, Excel)
- **Resource efficient**: Less CPU/memory usage for repeated file references

## Implementation Status

### ✅ Completed

- [x] Package structure
- [x] Type definitions
- [x] Interface definitions
- [x] Skeleton functions with TODO comments
- [x] File processing cache infrastructure
- [x] Cache helper functions (tryGetCachedText, cacheProcessedText)

### 🚧 To Implement

- [ ] tryGetCachedText implementation (attachment.GetText integration)
- [ ] cacheProcessedText implementation (attachment.SaveText integration)
- [ ] HTTP fetching logic
- [ ] Uploader wrapper parsing and fetching
- [ ] Image processing (base64, vision API)
- [ ] Audio processing (transcription)
- [ ] PDF text extraction
- [ ] Word document parsing
- [ ] Excel spreadsheet parsing
- [ ] Text/CSV formatting
- [ ] Content part processing logic
- [ ] Model capability detection
- [ ] Agent/MCP tool invocation

## Configuration

Content processing behavior is controlled by:

1. **Model Capabilities** (`openai.Capabilities`)

   - Determines if model can handle images/audio directly
   - Specifies vision format (OpenAI vs Claude)

2. **Uses** (`context.Uses`)
   ```go
   type Uses struct {
       Vision string // "agent" or "mcp:server_id"
       Audio  string // "agent" or "mcp:server_id"
       Search string
       Fetch  string
   }
   ```

## Error Handling

- Errors during processing are logged but don't stop the entire pipeline
- Original content is kept if processing fails
- Graceful degradation: if advanced processing fails, fall back to simpler methods
