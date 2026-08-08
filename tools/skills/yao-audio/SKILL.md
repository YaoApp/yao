---
name: yao-audio
description: Audio expert. ALWAYS invoke this skill when the user asks to transcribe, recognize, or convert speech/audio to text.
---

# Audio Tools

Use these tools to transcribe audio files to text using speech-to-text models.

## audio_transcribe

Transcribe an audio file to text.

```bash
tai tool audio_transcribe --audio_path /path/to/meeting.m4a
```

```bash
tai tool audio_transcribe --audio_path /path/to/recording.wav --language en --provider llm.my-openai:whisper-1
```

| Parameter  | Type   | Required | Description                                                       |
| ---------- | ------ | -------- | ----------------------------------------------------------------- |
| audio_path | string | yes      | Audio file path. Supported: mp3, m4a, wav, webm, mp4, mpeg, mpga  |
| language   | string | no       | ISO 639-1 language code (e.g. `en`, `zh`, `ja`). Auto-detected if omitted |
| provider   | string | no       | STT provider connector ID. If omitted, uses the default STT provider |

## audio_providers

List available speech-to-text providers and models.

### List STT providers (default):
```bash
tai tool audio_providers
```

| Parameter  | Type   | Required | Description                          |
| ---------- | ------ | -------- | ------------------------------------ |
| capability | string | no       | Filter by capability (default: `audio`) |

Returns a list of providers with their available models and connector IDs that can be passed to `audio_transcribe`.

## Constraints

Only use the parameters listed above for each tool. Do not pass unsupported parameters — they will be ignored or cause errors.
