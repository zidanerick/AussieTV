# Development continuity

Canonical beta: v0.1.0-beta.1. Docker is the active distribution. The main
registry tag remains experimental-v0.1.0-beta.1 for existing installations,
with docker-preview for the trusted development branch and immutable SHA
tags. Previous native packages are historical and unchanged.

Go embeds a browser UI; authenticated web 45781 and keyed Jellyfin 45782 are
independently configured. Headless lifecycle, no desktop startup or hidden
feed-only web listener. Docker bundles FFmpeg and persists caches under /data.
Preserve source IDs, numbers and existing secrets. See [Docker](DOCKER.md),
[Jellyfin](JELLYFIN.md), [lineup](LINEUP.md), [providers](PROVIDERS.md).

CI publishes exact tested multi-architecture container images after source
and container gates. Automated tests are not proof of broadcaster playback.
Priorities: live audio/video and Fast Tune stability, remaining selected
provider integrations, artwork coverage, proxy security and client hardware
tests. Legacy helper code is dormant; remove only after checking test
dependencies. See [developer instructions](../AGENTS.md).
