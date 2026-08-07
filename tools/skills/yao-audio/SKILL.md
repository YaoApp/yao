---
name: yao-audio
description: Audio expert. ALWAYS invoke this skill when the user asks to transcribe, recognize, or convert speech/audio to text.
---

# Audio Tools

Use these tools to transcribe audio files to text using speech-to-text models.

## audio_transcribe

Transcribe an audio file to text.

### Local file:
```bash
tai tool audio_transcribe '{"audio_path": "/workspace/meeting.m4a"}'
```

### URL:
```bash
tai tool audio_transcribe '{"audio_path": "https://example.com/podcast.mp3", "language": "en"}'
```

### With a specific STT provider:
```bash
tai tool audio_transcribe '{"audio_path": "/path/to/recording.wav", "provider": "llm.my-openai:whisper-1"}'
```

| Parameter  | Type   | Required | Description                                                       |
| ---------- | ------ | -------- | ----------------------------------------------------------------- |
| audio_path | string | yes      | File path or URL to the audio file                                |
| language   | string | no       | ISO 639-1 language code (e.g. `en`, `zh`, `ja`). Auto-detected if omitted |
| provider   | string | no       | STT provider connector ID. If omitted, uses the default STT provider |

Supported formats: mp3, mp4, mpeg, mpga, m4a, wav, webm (Whisper-compatible formats).

Files larger than 25MB are automatically split into chunks and transcribed sequentially. The user does not need to handle splitting manually.

## audio_providers

List available speech-to-text providers and models.

### List STT providers (default):
```bash
tai tool audio_providers '{}'
```

| Parameter  | Type   | Required | Description                          |
| ---------- | ------ | -------- | ------------------------------------ |
| capability | string | no       | Filter by capability (default: `audio`) |

Returns a list of providers with their available models and connector IDs that can be passed to `audio_transcribe`.

## Constraints

Only use the parameters listed above for each tool. Do not pass unsupported parameters — they will be ignored or cause errors.
