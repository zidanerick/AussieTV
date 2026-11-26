# AussieTV Docker beta

The project now distributes only Linux amd64/arm64 Docker images. The beta
version remains v0.1.0-beta.1; existing native release assets are historical
and unchanged. Docker/Unraid installs keep the same image tags, /data mount,
environment configuration and Jellyfin endpoints.

- Headless authenticated web, feed-only and combined modes.
- Bundled FFmpeg, Fast Tune, guarded HLS and identity AES-128 support.
- Selected regional lineup, readiness-gated international candidates,
  optional approved providers and Jellyfin artwork repair.
- Automatic tested image publication; no new native installers or archives.

**Known problems:** upstream streams, EPG, artwork, live audio/video and
HDHomeRun discovery may still fail. Docker readiness is not playback
verification. See [known issues](docs/BETA-KNOWN-ISSUES.md).
