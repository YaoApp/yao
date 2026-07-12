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

Generate a new image from a text prompt (text-to-image). For editing an existing image, use `image_edit` instead.

### Basic usage (always specify output):
```bash
tai tool image_generate '{"prompt": "A serene mountain landscape at sunset", "output": "landscape.png"}'
```

### With specific provider, model and size:
```bash
tai tool image_generate '{"prompt": "A futuristic city skyline", "provider": "llm.my-openai", "model": "gpt-image-1", "size": "1792x1024", "output": "output/city.png"}'
```

| Parameter | Type   | Required | Description                                                       |
| --------- | ------ | -------- | ----------------------------------------------------------------- |
| prompt    | string | yes      | Text description of the image to generate                         |
| output    | string | yes      | File path to save the generated image (parent dirs created automatically) |
| provider  | string | no       | Provider connector ID (use `image_providers` to list). Auto-selects if omitted |
| size      | string | no       | Image dimensions (default: 1024x1024). Common: 1024x1024, 1024x1792, 1792x1024 |
| model     | string | no       | Model name to use. Overrides the provider's default model         |

**Important**: Always pass `output`. The tool saves the image directly and returns only the file path and size. Without `output`, the raw base64 data is returned which may exceed output limits.

Use relative paths (e.g. `"output": "fox.png"`) — they resolve relative to the current working directory (`$WORKDIR`). No need to prepend `$WORKDIR` manually.

## image_edit

Edit or transform an existing image based on a text prompt (image-to-image). Use for style transfer, background replacement, adding/removing elements, or any modification that requires a reference image.

### Basic usage:
```bash
tai tool image_edit '{"image_path": "/path/to/photo.png", "prompt": "Change the background to a beach scene", "output": "edited.png"}'
```

### With URL image:
```bash
tai tool image_edit '{"image_path": "https://example.com/photo.jpg", "prompt": "Make it look like a watercolor painting", "output": "watercolor.png"}'
```

### With specific provider and model:
```bash
tai tool image_edit '{"image_path": "workspace://ws-id/uploads/original.png", "prompt": "Remove the person in the foreground", "provider": "llm.my-openai", "model": "gpt-image-1", "size": "1024x1024", "output": "result.png"}'
```

| Parameter  | Type   | Required | Description                                                       |
| ---------- | ------ | -------- | ----------------------------------------------------------------- |
| image_path | string | yes      | Reference image: file path, URL, workspace://, or attach:// URI   |
| prompt    | string | yes      | Text description of the desired edit or transformation            |
| output    | string | yes      | File path to save the edited image (parent dirs created automatically) |
| provider  | string | no       | Provider connector ID (use `image_providers` with `capability=image_editing`). Auto-selects if omitted |
| size      | string | no       | Output dimensions (default: 1024x1024). Common: 1024x1024, 1024x1792, 1792x1024 |
| model     | string | no       | Model name to use. Overrides the provider's default model         |

**Important**: Always pass `output`. Same rules as `image_generate`.

Local image files are automatically read and converted to data URIs before sending to the server.

## image_providers

List available image providers filtered by capability.

### List image generation providers (default):
```bash
tai tool image_providers '{}'
```

### List image editing providers:
```bash
tai tool image_providers '{"capability": "image_editing"}'
```

### List vision (image reading) providers:
```bash
tai tool image_providers '{"capability": "vision"}'
```

| Parameter  | Type   | Required | Description                                                 |
| ---------- | ------ | -------- | ----------------------------------------------------------- |
| capability | string | no       | `image_generation` (default), `image_editing`, or `vision`  |

Returns a list of providers with their available models and connector IDs that can be passed to `image_generate`, `image_edit`, or `image_read`.

## Constraints

Only use the parameters listed above for each tool. Do not pass unsupported parameters (such as `quality`, `style`, `n`, `response_format`, etc.) — they will be ignored or cause errors.
