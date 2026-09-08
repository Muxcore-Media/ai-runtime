# Compatibility

| Module Version | Core Version | Status |
|----------------|-------------|--------|
| v0.1.0         | v0.5.8+     | Current |

## Capabilities

- `ai.runtime`
- `settings`

## Contracts

- `contracts-ai` events + `infer` types

Existing non-AI modules must not import this module to run models. Feature AI modules call `ai.runtime`.
