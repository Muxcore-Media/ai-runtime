# Changelog

## [Unreleased]

### Security
- OpenAI-compatible completion requests go through netguard Integration (private LAN and loopback allowed for a local model server; link-local, cloud metadata, and non-HTTP schemes refused) (NFR-SEC-009).

## [0.1.2] - 2026-10-05

### Changed
- Built on core v0.6.14 / sdk/go/module v0.6.4: unregisters on shutdown and re-registers after core restarts (ADR-0022).

## [0.1.1] - 2026-10-05


### Changed
- Reported version comes from muxcore.json (ADR-0021); built on core v0.6.12 / sdk/go/module v0.6.3 (mesh enrollment, ADR-0017).

## [0.1.0] - 2026-10-05

### Changed
- CI runs on GitHub-hosted runners from the umbrella template; retired-origin workflows removed.
- Dependencies resolve from published GitHub tags (no filesystem `replace`); requires core v0.6.0.

### Added

- Initial scaffold from muxcore-module-starter cookiecutter
