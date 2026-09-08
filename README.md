# AI Runtime

Shared inference gateway for MuxCore AI feature modules. Households work offline with the built-in **heuristic** provider. An OpenAI-compatible chat endpoint is optional (`AI_RUNTIME_API_KEY` / `OPENAI_API_KEY`).

Feature modules (`ai-subtitles`, `ai-recommend`, `ai-tickets`, `ai-filter`, `ai-librarian`) call this capability (`ai.runtime`). Existing non-AI modules must not embed model clients.

## Ports

| Service | Default |
|---------|---------|
| gRPC / mesh | `127.0.0.1:9760` |
| Health / HTTP | `127.0.0.1:9761` |

## HTTP

| Method | Path | Notes |
|--------|------|-------|
| GET | `/healthz` | liveness |
| GET | `/v1/providers` | heuristic always; openai when keyed |
| POST | `/v1/complete` | chat / structured JSON |
| POST | `/v1/embed` | 32-d local hasher |
| POST | `/v1/transcribe` | timed cues from `transcript_hint` |
| POST | `/v1/classify` | label confidences |
| POST | `/v1/translate` | tiny offline glossary |

## Env

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_RUNTIME_PROVIDER` | heuristic | Preferred provider name |
| `AI_RUNTIME_API_KEY` / `OPENAI_API_KEY` | unset | Enables openai provider |
| `AI_RUNTIME_BASE_URL` | `https://api.openai.com` | OpenAI-compatible base |
| `AI_RUNTIME_MODEL` | `gpt-4o-mini` | Chat model |
| `AI_RUNTIME_GRPC_ADDR` | `127.0.0.1:9760` | gRPC bind |
| `MUXCORE_HTTP_ADDR` | `127.0.0.1:9761` | HTTP bind |

## Privacy

Transcripts and prompts stay on the host unless an operator sets an API key. Heuristic never transmits data.

## Build

```bash
cd ai-runtime
go test ./...
```
