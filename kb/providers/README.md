# Knowledge Base Providers

This directory contains all the providers for the Knowledge Base (KB) system. Providers are modular components that handle different aspects of document processing, including chunking, embedding, extraction, fetching, and conversion.

## Table of Contents

- [Overview](#overview)
- [Provider Types](#provider-types)
  - [Chunking Providers](#chunking-providers)
  - [Embedding Providers](#embedding-providers)
  - [Extraction Providers](#extraction-providers)
  - [Fetcher Providers](#fetcher-providers)
  - [Converter Providers](#converter-providers)
- [Configuration Format](#configuration-format)
- [Examples](#examples)

## Overview

The provider system is designed to be modular and extensible. Each provider type handles a specific aspect of document processing:

- **Chunking**: Splits documents into manageable pieces
- **Embedding**: Converts text into vector representations
- **Extraction**: Extracts entities and relationships for knowledge graphs
- **Fetching**: Retrieves documents from various sources
- **Conversion**: Transforms different file formats into processable text

All providers implement a common interface with `Make()`, `Options()`, and `Schema()` methods.

## Provider Types

### Chunking Providers

#### Structured Chunking (`__yao.structured`)

Splits documents based on structural elements like headings, paragraphs, and sections.

**Configuration Fields:**

| Field             | Type            | Default | Description                                  | Requirements |
| ----------------- | --------------- | ------- | -------------------------------------------- | ------------ |
| `size`            | `int`/`float64` | `300`   | Maximum chunk size in characters             | > 0          |
| `overlap`         | `int`/`float64` | `20`    | Character overlap between chunks             | ≥ 0          |
| `max_depth`       | `int`/`float64` | `3`     | Maximum nesting depth for structure analysis | ≥ 1          |
| `size_multiplier` | `int`/`float64` | `3`     | Multiplier for dynamic sizing                | ≥ 1          |
| `max_concurrent`  | `int`/`float64` | `10`    | Maximum concurrent processing threads        | ≥ 1          |

**Example Configuration:**

```json
{
  "properties": {
    "size": 500,
    "overlap": 50,
    "max_depth": 5,
    "size_multiplier": 2,
    "max_concurrent": 15
  }
}
```

#### Semantic Chunking (`__yao.semantic`)

Uses AI models to create semantically coherent chunks based on content meaning.

**Configuration Fields:**

| Field                     | Type            | Default    | Description                             | Requirements |
| ------------------------- | --------------- | ---------- | --------------------------------------- | ------------ |
| `size`                    | `int`/`float64` | `300`      | Base chunk size in characters           | > 0          |
| `overlap`                 | `int`/`float64` | `50`       | Character overlap between chunks        | ≥ 0          |
| `max_depth`               | `int`/`float64` | `3`        | Maximum nesting depth                   | ≥ 1          |
| `size_multiplier`         | `int`/`float64` | `3`        | Size multiplier for analysis            | ≥ 1          |
| `max_concurrent`          | `int`/`float64` | `10`       | Maximum concurrent processing threads   | ≥ 1          |
| `connector`               | `string`        | `""`       | AI connector name for semantic analysis | Must exist   |
| `toolcall`                | `bool`          | `false`    | Enable AI tool calling                  | -            |
| `context_size`            | `int`/`float64` | `size * 6` | Context window size for AI analysis     | > 0          |
| `options`                 | `string`        | `""`       | Additional AI model options             | -            |
| `prompt`                  | `string`        | `""`       | Custom prompt for semantic analysis     | -            |
| `max_retry`               | `int`/`float64` | `3`        | Maximum retry attempts for AI calls     | ≥ 0          |
| `semantic_max_concurrent` | `int`/`float64` | `10`       | Max concurrent semantic operations      | ≥ 1          |

**Example Configuration:**

```json
{
  "properties": {
    "size": 400,
    "overlap": 80,
    "connector": "openai.gpt-4o-mini",
    "toolcall": true,
    "context_size": 2400,
    "max_retry": 5,
    "semantic_max_concurrent": 8
  }
}
```

### Embedding Providers

#### OpenAI Embedding (`__yao.openai`)

Uses OpenAI's embedding models to convert text into vector representations.

**Configuration Fields:**

| Field        | Type            | Default | Description                     | Requirements        |
| ------------ | --------------- | ------- | ------------------------------- | ------------------- |
| `connector`  | `string`        | `""`    | OpenAI connector name           | Must exist          |
| `dimensions` | `int`/`float64` | `1536`  | Embedding vector dimensions     | > 0, model-specific |
| `concurrent` | `int`/`float64` | `10`    | Maximum concurrent API requests | ≥ 1                 |
| `model`      | `string`        | `""`    | Specific model name (optional)  | Valid OpenAI model  |

**Example Configuration:**

```json
{
  "properties": {
    "connector": "openai.text-embedding-3-small",
    "dimensions": 1536,
    "concurrent": 20,
    "model": "text-embedding-3-small"
  }
}
```

#### Fastembed Embedding (`__yao.fastembed`)

Uses local FastEmbed models for embedding generation without API calls.

**Configuration Fields:**

| Field        | Type            | Default | Description                      | Requirements        |
| ------------ | --------------- | ------- | -------------------------------- | ------------------- |
| `connector`  | `string`        | `""`    | Fastembed service connector      | Must exist          |
| `dimensions` | `int`/`float64` | `384`   | Embedding vector dimensions      | > 0, model-specific |
| `concurrent` | `int`/`float64` | `5`     | Maximum concurrent requests      | ≥ 1                 |
| `model`      | `string`        | `""`    | FastEmbed model name             | Valid model name    |
| `host`       | `string`        | `""`    | FastEmbed service host           | Valid URL/IP        |
| `key`        | `string`        | `""`    | Authentication key (if required) | -                   |

**Example Configuration:**

```json
{
  "properties": {
    "connector": "fastembed.sentence-transformers",
    "dimensions": 384,
    "concurrent": 8,
    "model": "BAAI/bge-small-en-v1.5",
    "host": "localhost:8080"
  }
}
```

### Extraction Providers

#### OpenAI Extraction (`__yao.openai`)

Extracts entities and relationships from documents using OpenAI models for knowledge graph construction.

**Configuration Fields:**

| Field            | Type            | Default | Description                                   | Requirements           |
| ---------------- | --------------- | ------- | --------------------------------------------- | ---------------------- |
| `connector`      | `string`        | `""`    | OpenAI connector name                         | Must exist             |
| `toolcall`       | `bool`          | `true`  | Enable tool calling for structured extraction | -                      |
| `temperature`    | `float64`/`int` | `0.1`   | Model temperature for generation              | 0.0-2.0                |
| `max_tokens`     | `int`/`float64` | `4000`  | Maximum tokens per request                    | > 0                    |
| `concurrent`     | `int`/`float64` | `5`     | Maximum concurrent requests                   | ≥ 1                    |
| `model`          | `string`        | `""`    | Specific model name (optional)                | Valid OpenAI model     |
| `prompt`         | `string`        | `""`    | Custom extraction prompt                      | -                      |
| `retry_attempts` | `int`/`float64` | `3`     | Number of retry attempts                      | ≥ 0                    |
| `retry_delay`    | `float64`/`int` | `1.0`   | Delay between retries (seconds)               | ≥ 0                    |
| `tools`          | `[]interface{}` | `nil`   | Custom extraction tools                       | Valid tool definitions |

**Example Configuration:**

```json
{
  "properties": {
    "connector": "openai.gpt-4o-mini",
    "toolcall": true,
    "temperature": 0.2,
    "max_tokens": 8000,
    "concurrent": 10,
    "retry_attempts": 5,
    "retry_delay": 2.0
  }
}
```

### Fetcher Providers

#### HTTP Fetcher (`__yao.http`)

Downloads files from HTTP/HTTPS URLs with configurable headers and timeout.

**Configuration Fields:**

| Field        | Type                     | Default                  | Description                | Requirements       |
| ------------ | ------------------------ | ------------------------ | -------------------------- | ------------------ |
| `headers`    | `map[string]interface{}` | `{}`                     | Custom HTTP headers        | String values only |
| `user_agent` | `string`                 | `"GraphRAG-Fetcher/1.0"` | Custom User-Agent header   | -                  |
| `timeout`    | `int`/`float64`          | `300`                    | Request timeout in seconds | > 0                |

**Example Configuration:**

```json
{
  "properties": {
    "headers": {
      "Authorization": "Bearer token123",
      "Accept": "application/json",
      "Custom-Header": "custom-value"
    },
    "user_agent": "MyApp/2.0",
    "timeout": 60
  }
}
```

#### MCP Fetcher (`__yao.mcp`)

Retrieves files using Model Context Protocol (MCP) tools for intelligent fetching.

**Configuration Fields:**

| Field                  | Type                     | Default   | Description                                 | Requirements       |
| ---------------------- | ------------------------ | --------- | ------------------------------------------- | ------------------ |
| `id`                   | `string`                 | `""`      | MCP client identifier                       | Must exist         |
| `tool`                 | `string`                 | `"fetch"` | MCP tool name to call                       | Valid tool name    |
| `arguments_mapping`    | `map[string]interface{}` | `nil`     | Template mapping for tool arguments         | String values only |
| `result_mapping`       | `map[string]interface{}` | `nil`     | Template mapping for parsing results        | String values only |
| `output_mapping`       | `map[string]interface{}` | `nil`     | Alias for result_mapping (compatibility)    | String values only |
| `notification_mapping` | `map[string]interface{}` | `nil`     | Template mapping for progress notifications | String values only |

**Example Configuration:**

```json
{
  "properties": {
    "id": "fetcher",
    "tool": "fetch_document",
    "arguments_mapping": {
      "url": "{{.url}}",
      "format": "text"
    },
    "result_mapping": {
      "content": "{{.result.content}}",
      "mime_type": "{{.result.mime_type}}"
    },
    "notification_mapping": {
      "progress": "{{.notification.progress}}",
      "status": "{{.notification.status}}"
    }
  }
}
```

### Converter Providers

#### UTF8 Converter (`__yao.utf8`)

Converts plain text and UTF-8 encoded files to processable text format.

**Configuration Fields:**

| Field        | Type     | Default   | Description                       | Requirements        |
| ------------ | -------- | --------- | --------------------------------- | ------------------- |
| `encoding`   | `string` | `"utf-8"` | Text encoding to assume           | Valid encoding name |
| `remove_bom` | `bool`   | `true`    | Remove Byte Order Mark if present | -                   |

**Example Configuration:**

```json
{
  "properties": {
    "encoding": "utf-8",
    "remove_bom": true
  }
}
```

#### Vision Converter (`__yao.vision`)

Processes images and visual documents using AI vision models.

**Configuration Fields:**

| Field        | Type            | Default  | Description                    | Requirements          |
| ------------ | --------------- | -------- | ------------------------------ | --------------------- |
| `connector`  | `string`        | `""`     | Vision AI connector name       | Must exist            |
| `quality`    | `string`        | `"auto"` | Image processing quality       | "low", "high", "auto" |
| `detail`     | `string`        | `"auto"` | Level of detail in analysis    | "low", "high", "auto" |
| `max_tokens` | `int`/`float64` | `4000`   | Maximum tokens for description | > 0                   |
| `prompt`     | `string`        | `""`     | Custom vision analysis prompt  | -                     |

**Example Configuration:**

```json
{
  "properties": {
    "connector": "openai.gpt-4-vision",
    "quality": "high",
    "detail": "high",
    "max_tokens": 8000,
    "prompt": "Describe this image in detail"
  }
}
```

#### Whisper Converter (`__yao.whisper`)

Converts audio files to text using speech recognition models.

**Configuration Fields:**

| Field             | Type            | Default  | Description                    | Requirements                   |
| ----------------- | --------------- | -------- | ------------------------------ | ------------------------------ |
| `connector`       | `string`        | `""`     | Audio processing connector     | Must exist                     |
| `language`        | `string`        | `"auto"` | Audio language for recognition | ISO language code or "auto"    |
| `temperature`     | `float64`/`int` | `0.0`    | Model temperature              | 0.0-1.0                        |
| `response_format` | `string`        | `"text"` | Output format                  | "text", "json", "verbose_json" |

**Example Configuration:**

```json
{
  "properties": {
    "connector": "openai.whisper-1",
    "language": "en",
    "temperature": 0.2,
    "response_format": "text"
  }
}
```

#### MCP Converter (`__yao.mcp`)

Uses MCP tools for custom document conversion workflows.

**Configuration Fields:**

| Field               | Type                     | Default     | Description                  | Requirements       |
| ------------------- | ------------------------ | ----------- | ---------------------------- | ------------------ |
| `id`                | `string`                 | `""`        | MCP client identifier        | Must exist         |
| `tool`              | `string`                 | `"convert"` | MCP tool name for conversion | Valid tool name    |
| `arguments_mapping` | `map[string]interface{}` | `nil`       | Template for tool arguments  | String values only |
| `result_mapping`    | `map[string]interface{}` | `nil`       | Template for result parsing  | String values only |

**Example Configuration:**

```json
{
  "properties": {
    "id": "converter",
    "tool": "convert_document",
    "arguments_mapping": {
      "file_path": "{{.path}}",
      "format": "text"
    },
    "result_mapping": {
      "content": "{{.result.text}}",
      "metadata": "{{.result.meta}}"
    }
  }
}
```

#### OCR Converter (`__yao.ocr`)

Optical Character Recognition for extracting text from images and scanned documents.

**Configuration Fields:**

| Field        | Type                     | Default  | Description                    | Requirements                     |
| ------------ | ------------------------ | -------- | ------------------------------ | -------------------------------- |
| `vision`     | `map[string]interface{}` | Required | Vision converter configuration | Must contain valid vision config |
| `language`   | `string`                 | `"auto"` | OCR language hint              | ISO language code or "auto"      |
| `dpi`        | `int`/`float64`          | `300`    | Image DPI for processing       | > 0                              |
| `preprocess` | `bool`                   | `true`   | Enable image preprocessing     | -                                |

**Example Configuration:**

```json
{
  "properties": {
    "vision": {
      "converter": "__yao.vision",
      "properties": {
        "connector": "openai.gpt-4-vision",
        "quality": "high"
      }
    },
    "language": "en",
    "dpi": 300,
    "preprocess": true
  }
}
```

#### Video Converter (`__yao.video`)

Extracts content from video files using frame analysis and audio transcription.

**Configuration Fields:**

| Field            | Type                     | Default  | Description                         | Requirements                     |
| ---------------- | ------------------------ | -------- | ----------------------------------- | -------------------------------- |
| `vision`         | `map[string]interface{}` | Required | Vision converter for frame analysis | Must contain valid vision config |
| `audio`          | `map[string]interface{}` | Required | Audio converter for transcription   | Must contain valid audio config  |
| `frame_interval` | `int`/`float64`          | `30`     | Seconds between frame captures      | > 0                              |
| `max_frames`     | `int`/`float64`          | `10`     | Maximum frames to analyze           | > 0                              |

**Example Configuration:**

```json
{
  "properties": {
    "vision": {
      "converter": "__yao.vision",
      "properties": {
        "connector": "openai.gpt-4-vision"
      }
    },
    "audio": {
      "converter": "__yao.whisper",
      "properties": {
        "connector": "openai.whisper-1"
      }
    },
    "frame_interval": 60,
    "max_frames": 20
  }
}
```

#### Office Converter (`__yao.office`)

Processes Microsoft Office documents (Word, Excel, PowerPoint) and PDFs.

**Configuration Fields:**

| Field                 | Type                     | Default  | Description                         | Requirements                     |
| --------------------- | ------------------------ | -------- | ----------------------------------- | -------------------------------- |
| `vision`              | `map[string]interface{}` | Required | Vision converter for image content  | Must contain valid vision config |
| `video`               | `map[string]interface{}` | Optional | Video converter for embedded videos | Must contain valid video config  |
| `audio`               | `map[string]interface{}` | Optional | Audio converter for embedded audio  | Must contain valid audio config  |
| `extract_images`      | `bool`                   | `true`   | Extract and process embedded images | -                                |
| `extract_tables`      | `bool`                   | `true`   | Extract and format table data       | -                                |
| `preserve_formatting` | `bool`                   | `false`  | Preserve original formatting        | -                                |

**Example Configuration:**

```json
{
  "properties": {
    "vision": {
      "converter": "__yao.vision",
      "properties": {
        "connector": "openai.gpt-4-vision"
      }
    },
    "video": {
      "converter": "__yao.video",
      "properties": {
        "vision": {
          "converter": "__yao.vision",
          "properties": {
            "connector": "openai.gpt-4-vision"
          }
        },
        "audio": {
          "converter": "__yao.whisper",
          "properties": {
            "connector": "openai.whisper-1"
          }
        }
      }
    },
    "extract_images": true,
    "extract_tables": true,
    "preserve_formatting": false
  }
}
```

## Configuration Format

All providers use a consistent configuration format:

```json
{
  "id": "provider_id",
  "properties": {
    "field_name": "field_value"
  }
}
```

### Data Type Handling

The configuration system automatically handles type conversion:

- **Numeric fields**: Accept both `int` and `float64`, converted as needed
- **String fields**: Must be strings, other types are ignored
- **Boolean fields**: Must be boolean values
- **Map fields**: Accept `map[string]interface{}`, non-string values filtered out
- **Array fields**: Accept `[]interface{}`, with element type validation

### Default Values

All providers provide sensible default values for optional fields. Required fields (like `connector` names) must be explicitly configured.

## Examples

### Complete KB Configuration

```json
{
  "chunking": {
    "id": "__yao.semantic",
    "properties": {
      "size": 400,
      "overlap": 80,
      "connector": "openai.gpt-4o-mini",
      "toolcall": true
    }
  },
  "embedding": {
    "id": "__yao.openai",
    "properties": {
      "connector": "openai.text-embedding-3-small",
      "dimensions": 1536,
      "concurrent": 15
    }
  },
  "extraction": {
    "id": "__yao.openai",
    "properties": {
      "connector": "openai.gpt-4o-mini",
      "toolcall": true,
      "temperature": 0.1
    }
  },
  "fetcher": {
    "id": "__yao.http",
    "properties": {
      "timeout": 60,
      "headers": {
        "User-Agent": "KB-System/1.0"
      }
    }
  },
  "converters": [
    {
      "id": "__yao.utf8",
      "properties": {
        "encoding": "utf-8"
      }
    },
    {
      "id": "__yao.vision",
      "properties": {
        "connector": "openai.gpt-4-vision",
        "quality": "high"
      }
    }
  ]
}
```

### Error Handling

All providers implement robust error handling:

- **Invalid configurations**: Ignored with defaults applied
- **Missing dependencies**: Clear error messages
- **Type mismatches**: Automatic type conversion or field skipping
- **Network failures**: Retry mechanisms where applicable

For detailed implementation examples and test cases, see the corresponding `*_test.go` files in each provider directory.
