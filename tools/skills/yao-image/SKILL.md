---
name: yao-image
description: Image expert. ALWAYS invoke this skill when you need to read, analyze, describe, or generate images. Use for screenshots, photos, charts, diagrams, AI-generated images, or any visual content.
---

# Image Tools

Use these tools when you encounter images you cannot read natively, or when you need to generate new images.

## image_read

Send an image to a vision-capable model and get a text description.

### Local file (most common):
```bash
tai tool image_read '{"image_path": "/path/to/image.png", "prompt": "Describe this image"}'
```

### URL:
```bash
tai tool image_read '{"image_path": "https://example.com/photo.jpg", "prompt": "What is shown?"}'
```

### Cross-workspace file:
```bash
tai tool image_read '{"image_path": "workspace://ws-id/path/to/image.png", "prompt": "Analyze"}'
```

### Attachment file:
```bash
tai tool image_read '{"image_path": "attach://__yao.attachment/file-id-123", "prompt": "Describe"}'
```

### With a specific vision provider:
```bash
tai tool image_read '{"image_path": "/path/to/image.png", "prompt": "Describe", "provider": "llm.my-openai:gpt-4o"}'
```

| Parameter  | Type    | Required | Description                                                     |
| ---------- | ------- | -------- | --------------------------------------------------------------- |
| image_path | string  | yes      | File path, URL, workspace://, attach://, or yao:// URI          |
| prompt     | string  | no       | Analysis instruction (default: describe in detail)              |
| max_size   | integer | no       | Max dimension in pixels for longest edge (default: 1080)        |
| provider   | string  | no       | Vision provider connector ID. If omitted, uses default vision model |

Images are automatically resized (preserving aspect ratio) before sending to the vision model.
Supported formats: PNG, JPEG, GIF, WebP.

## image_generate

Generate an image from a text prompt and save it to a file.

### Basic usage (always specify output):
```bash
tai tool image_generate '{"prompt": "A serene mountain landscape at sunset", "output": "landscape.png"}'
```

### With specific provider and size:
```bash
tai tool image_generate '{"prompt": "A futuristic city skyline", "provider": "llm.my-openai:dall-e-3", "size": "1792x1024", "output": "output/city.png"}'
```

| Parameter | Type   | Required | Description                                                       |
| --------- | ------ | -------- | ----------------------------------------------------------------- |
| prompt    | string | yes      | Text description of the image to generate                         |
| output    | string | yes      | File path to save the generated image (parent dirs created automatically) |
| provider  | string | no       | Provider connector ID (use `image_providers` to list). Auto-selects if omitted |
| size      | string | no       | Image dimensions (default: 1024x1024). Common: 1024x1024, 1024x1792, 1792x1024 |

**Important**: Always pass `output`. The tool saves the image directly and returns only the file path and size. Without `output`, the raw base64 data is returned which may exceed output limits.

Use relative paths (e.g. `"output": "fox.png"`) — they resolve relative to the current working directory (`$WORKDIR`). No need to prepend `$WORKDIR` manually.

## image_providers

List available image providers filtered by capability.

### List image generation providers (default):
```bash
tai tool image_providers '{}'
```

### List vision (image reading) providers:
```bash
tai tool image_providers '{"capability": "vision"}'
```

| Parameter  | Type   | Required | Description                                                 |
| ---------- | ------ | -------- | ----------------------------------------------------------- |
| capability | string | no       | `image_generation` (default) or `vision`                    |

Returns a list of providers with their available models and connector IDs that can be passed to `image_generate` or `image_read`.
