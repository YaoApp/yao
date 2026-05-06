---
name: yao-vision
description: Image understanding expert. ALWAYS invoke this skill when you need to read, analyze, or describe an image that you cannot process directly. Use for screenshots, photos, charts, diagrams, or any visual content.
---

# Vision Tools

Use when you encounter images you cannot read natively (e.g., as a text-only model).

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

### Yao data file:
```bash
tai tool image_read '{"image_path": "yao://uploads/photo.jpg", "prompt": "Describe"}'
```

| Parameter  | Type    | Required | Description                                                     |
| ---------- | ------- | -------- | --------------------------------------------------------------- |
| image_path | string  | yes      | File path, URL, workspace://, attach://, or yao:// URI          |
| prompt     | string  | no       | Analysis instruction (default: describe in detail)              |
| max_size   | integer | no       | Max dimension in pixels for longest edge (default: 1080)        |

Images are automatically resized (preserving aspect ratio) before sending to the vision model.
Supported formats: PNG, JPEG, GIF, WebP.
