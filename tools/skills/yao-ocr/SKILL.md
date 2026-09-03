---
name: yao-ocr
description: OCR text recognition expert. ALWAYS invoke this skill when you need to extract text from images or PDFs — including invoices, receipts, ID cards, bank cards, business licenses, tables, handwritten documents, or any visual text content.
---

# OCR Tools

Two tools for optical character recognition, supporting both VLM-OCR (vision language models) and traditional OCR APIs (Baidu, Google, Azure, PaddleOCR).

## ocr_recognize

Extract text from images or PDF files using OCR.

### Basic usage (plain text output):
```bash
tai tool ocr_recognize --source /path/to/image.png
```

### With URL:
```bash
tai tool ocr_recognize --source https://example.com/document.jpg
```

### Table extraction as Markdown:
```bash
tai tool ocr_recognize --source /path/to/table.png --type table --output_format markdown
```

### Invoice structured extraction:
```bash
tai tool ocr_recognize --source /path/to/invoice.pdf --type invoice --output_format json
```

### With specific provider:
```bash
tai tool ocr_recognize --source /path/to/doc.png --provider baidu
```

### VLM-OCR with custom prompt:
```bash
tai tool ocr_recognize --source /path/to/doc.png --provider llm:qwen-ocr --prompt "只提取表格中的金额列"
```

### PDF page range:
```bash
tai tool ocr_recognize --source /path/to/report.pdf --pages "1-5" --output_format markdown
```

| Parameter     | Type   | Required | Description                                                                                  |
| ------------- | ------ | -------- | -------------------------------------------------------------------------------------------- |
| source        | string | yes      | Image or PDF file path/URL to recognize                                                      |
| provider      | string | no       | LLM connector ID (`llm:xxx`) or OCR settings key (`baidu`/`paddleocr`/`google`/`azure`). Auto-selects if omitted |
| type          | string | no       | Recognition type (default: `general`). See type table below                                  |
| output_format | string | no       | `text` (default), `json` (with coordinates/fields), or `markdown` (structured)               |
| mode          | string | no       | `accurate` (default, best quality) or `standard` (faster)                                    |
| language      | string | no       | Language hint (ISO 639-1, e.g. `en`, `zh`, `ja`). Auto-detected if omitted                   |
| prompt        | string | no       | Custom instruction for VLM-OCR only, appended to system prompt. Ignored by traditional OCR   |
| pages         | string | no       | PDF page range, e.g. `1-5` or `1,3,7`. All pages if omitted                                 |
| extra         | JSON   | no       | Provider-specific parameters as a JSON object                                                |

### Recognition types

| Type             | Description                | Best output_format |
| ---------------- | -------------------------- | ------------------ |
| `general`        | General text (default)     | text               |
| `table`          | Table extraction           | markdown           |
| `handwriting`    | Handwritten text           | text               |
| `document`       | Document layout parsing    | markdown           |
| `invoice`        | Invoice (VAT)              | json               |
| `receipt`        | Receipt / ticket           | json               |
| `id_card`        | ID card                    | json               |
| `bank_card`      | Bank card                  | json               |
| `license`        | Business license           | json               |
| `vehicle_license` | Vehicle license           | json               |
| `passport`       | Passport                   | json               |
| `license_plate`  | License plate              | json               |

If the chosen provider does not support the requested type, it automatically degrades to `general` and annotates the response metadata with `degraded_from`. VLM-OCR supports all types via prompt adaptation.

## ocr_providers

List available OCR providers and their supported recognition types.

```bash
tai tool ocr_providers
```

Returns a list of providers including VLM-OCR models (from LLM connectors with `ocr` capability) and traditional API providers (from OCR settings). Each entry includes `id`, `name`, `type` (`vlm` or `traditional`), and `supported_types`.

## PDF support

| Provider   | PDF | Notes                                    |
| ---------- | --- | ---------------------------------------- |
| Baidu      | yes | pdf_file parameter                       |
| Azure      | yes | Document Intelligence native support     |
| PaddleOCR  | yes | pdf + fileType=0                         |
| Google     | no  | Sync API does not support PDF            |
| VLM (llm:) | no  | Vision models accept images only         |

For providers that do not support PDF, use Baidu, Azure, or PaddleOCR instead.

## Multi-page PDF response

Multi-page PDFs are automatically split page-by-page. Instead of printing all text, the tool returns a JSON summary with file paths for each page result:

```json
{
  "source": "report.pdf",
  "total_pages": 10,
  "pages": 3,
  "results": [
    {"page": 1, "file": ".tool-tmp/ocr-a1b2c3d4/page-1.txt", "preview": "Invoice No: INV-001..."},
    {"page": 2, "file": ".tool-tmp/ocr-a1b2c3d4/page-2.txt", "preview": "Invoice No: INV-002..."},
    {"page": 3, "file": ".tool-tmp/ocr-a1b2c3d4/page-3.txt", "preview": "Invoice No: INV-003..."}
  ]
}
```

To read full content of a specific page, use `cat`:
```bash
cat .tool-tmp/ocr-a1b2c3d4/page-2.txt
```

Single-page PDFs and images return inline text as usual (no file indirection).

## Guidelines

- Use `output_format=text` (default) when you just need the text content — simplest for LLM processing
- Use `output_format=json` for structured types (invoice, id_card, etc.) to get key-value fields
- Use `output_format=markdown` for documents and tables to preserve layout
- The `prompt` parameter only works with VLM-OCR providers; traditional OCR ignores it
- For structured document types (invoice, receipt, id_card, etc.), prefer `json` output to get `fields` with key-value pairs
- Use `ocr_providers` first to check which providers are available and what types they support
- Multi-page PDFs return a JSON summary with temporary file paths; use `cat <file>` to read specific pages
- Google Vision and VLM providers do not support PDF input directly; use Baidu, Azure, or PaddleOCR for PDF files
