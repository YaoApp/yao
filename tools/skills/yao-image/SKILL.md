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
tai tool image_read --image_path /path/to/image.png --prompt "Describe this image"
```

### URL:
```bash
tai tool image_read --image_path https://example.com/photo.jpg --prompt "What is shown?"
```

### With a specific vision provider:
```bash
tai tool image_read --image_path /path/to/image.png --prompt "Describe" --provider llm.my-openai:gpt-4o
```

| Parameter  | Type    | Required | Description                                                     |
| ---------- | ------- | -------- | --------------------------------------------------------------- |
| image_path | string  | yes      | Image file path or URL                                          |
| prompt     | string  | no       | Analysis instruction (default: describe in detail)              |
| max_size   | integer | no       | Max dimension in pixels for longest edge (default: 1080)        |
| provider   | string  | no       | Vision provider connector ID. If omitted, uses default vision model |

Images are automatically resized (preserving aspect ratio) before sending to the vision model.
Supported formats: PNG, JPEG, GIF, WebP.

## image_generate

Generate a new image from a text prompt (text-to-image). For editing an existing image, use `image_edit` instead.

### Basic usage (always specify output):
```bash
tai tool image_generate --prompt "A serene mountain landscape at sunset" --output landscape.png
```

### With specific provider, model and size:
```bash
tai tool image_generate --prompt "A futuristic city skyline" --provider llm.my-openai --model gpt-image-1 --dimensions 1792x1024 --output output/city.png
```

### Transparent background (for icons, stickers, product shots):
```bash
tai tool image_generate --prompt "A cute fox mascot" --background transparent --output_format png --output fox.png
```

### WebP output with quality control:
```bash
tai tool image_generate --prompt "A landscape painting" --output_format webp --quality high --output painting.webp
```

| Parameter | Type   | Required | Description                                                       |
| --------- | ------ | -------- | ----------------------------------------------------------------- |
| prompt     | string | yes      | Text description of the image to generate                         |
| output     | string | yes      | Output file path for the generated image                          |
| provider   | string | no       | Provider connector ID (use `image_providers` to list). Auto-selects if omitted |
| dimensions | string | no       | Image dimensions (default: 1024x1024). Common: 1024x1024, 1024x1792, 1792x1024 |
| model      | string | no       | Model name to use. Overrides the provider's default model         |
| background | string | no       | `transparent`, `opaque`, or `auto`. Use `transparent` for PNG/WebP with no background |
| output_format | string | no    | `png`, `jpeg`, or `webp`. Default: png                            |
| output_compression | integer | no | Compression level 0-100 for jpeg/webp. Higher = better quality. Default: 100 |
| quality    | string | no       | `low`, `medium`, `high`, or `auto`. Higher takes longer. Default: auto |
| extra      | JSON   | no       | Provider-specific parameters as a JSON object (e.g. `--extra '{"moderation":"low"}'`) |

If `output` is omitted, the image is saved to a default path in the working directory.

## image_edit

Edit or transform an existing image based on a text prompt (image-to-image). Use for style transfer, background replacement, adding/removing elements, or any modification that requires a reference image.

### Basic usage:
```bash
tai tool image_edit --image_path /path/to/photo.png --prompt "Change the background to a beach scene" --output edited.png
```

### With URL image:
```bash
tai tool image_edit --image_path https://example.com/photo.jpg --prompt "Make it look like a watercolor painting" --output watercolor.png
```

### With specific provider and model:
```bash
tai tool image_edit --image_path /path/to/original.png --prompt "Remove the person in the foreground" --provider llm.my-openai --model gpt-image-1 --dimensions 1024x1024 --output result.png
```

### With mask (edit only the masked region):
```bash
tai tool image_edit --image_path /path/to/photo.png --mask /path/to/mask.png --prompt "Replace with a garden" --output edited.png
```

### Transparent background edit:
```bash
tai tool image_edit --image_path /path/to/product.png --prompt "Remove background" --background transparent --output_format png --output cutout.png
```

| Parameter  | Type   | Required | Description                                                       |
| ---------- | ------ | -------- | ----------------------------------------------------------------- |
| image_path | string | yes      | Reference image file path or URL                                  |
| prompt     | string | yes      | Text description of the desired edit or transformation            |
| output     | string | yes      | Output file path for the edited image                             |
| provider   | string | no       | Provider connector ID (use `image_providers` with `capability=image_editing`). Auto-selects if omitted |
| dimensions | string | no       | Output dimensions (default: 1024x1024). Common: 1024x1024, 1024x1792, 1792x1024 |
| model      | string | no       | Model name to use. Overrides the provider's default model         |
| mask       | string | no       | Mask image path/URL. Transparent areas in the mask define the editable region |
| background | string | no       | `transparent`, `opaque`, or `auto`. Use `transparent` for PNG/WebP with no background |
| output_format | string | no    | `png`, `jpeg`, or `webp`. Default: png                            |
| output_compression | integer | no | Compression level 0-100 for jpeg/webp. Higher = better quality. Default: 100 |
| quality    | string | no       | `low`, `medium`, `high`, or `auto`. Higher takes longer. Default: auto |
| extra      | JSON   | no       | Provider-specific parameters as a JSON object (e.g. `--extra '{"input_fidelity":"high"}'`) |

If `output` is omitted, the image is saved to a default path in the working directory.

## image_providers

List available image providers filtered by capability.

### List image generation providers (default):
```bash
tai tool image_providers
```

### List image editing providers:
```bash
tai tool image_providers --capability image_editing
```

### List vision (image reading) providers:
```bash
tai tool image_providers --capability vision
```

| Parameter  | Type   | Required | Description                                                 |
| ---------- | ------ | -------- | ----------------------------------------------------------- |
| capability | string | no       | `image_generation` (default), `image_editing`, or `vision`  |

Returns a list of providers with their available models and connector IDs that can be passed to `image_generate`, `image_edit`, or `image_read`.

## Constraints

Use only the parameters listed above for each tool. The supported first-class parameters are: `prompt`, `output`, `provider`, `dimensions`, `model`, `background`, `output_format`, `output_compression`, `quality`, `mask` (edit only), and `extra`.

Do **not** pass `n`, `style`, or `response_format` — they are unsupported and will be ignored or cause errors.

For provider-specific parameters not covered above (e.g. `moderation`, `input_fidelity`), pass them through the `extra` JSON object.
