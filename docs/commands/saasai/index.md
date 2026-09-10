# SaaS AI

`grn saas-ai` uses the global [Speech Text API](https://docs.api.greennode.ai/service-docs/saas-ai.html) at `https://ai-speech-text.api.vngcloud.vn/speech-api` with the shared bearer-authenticated machine/user profile. No region, project, or portal-user ID is required.

| Command | Required flags | Optional flags |
| --- | --- | --- |
| `speech-to-text transcribe` | `--audio-file`, `--encoding-type wav\|mp3` | `--dry-run` |
| `text-to-speech synthesize` | `--input`, `--output-file` | `--speed 0.8..1.2`, `--speaker-id 0..3`, `--encode-type 0\|1`, `--dry-run`, `--force` |

Both requests are billable POSTs and are never retried. Transcription streams multipart fields `encoding_type` and `audio_file`; synthesis sends JSON and writes the returned audio to the explicit file. Encoding values are 0 for WAV and 1 for MP3.

Dry-runs validate flags and local file metadata without reading audio bytes, loading credentials, calling the API, prompting, or writing output. Existing output files require confirmation or `--force`; unforced non-interactive overwrite fails. Symlinks and non-regular output files are refused, and newly appearing files are not overwritten without authorization.

Responses require HTTP 200 and the published content type: JSON for transcription, `audio/mpeg` for synthesis. Invalid responses leave output untouched.

```bash
grn saas-ai speech-to-text transcribe --audio-file sample.wav --encoding-type wav --dry-run
grn saas-ai text-to-speech synthesize --input 'Sample narration.' --encode-type 1 --output-file sample.mp3 --dry-run
```

Public fixtures and local HTTP tests cover multipart fields, binary output, retry refusal, and file safety. No billable provider call or speech-quality validation was performed.
