# AussieTV

**Early public beta (`v0.1.0-beta.1`), Docker-only deployment.**

AussieTV is a self-hosted Australian live television server, browser TV guide
and optional Jellyfin M3U/XMLTV source. One container includes the Go server,
browser interface, FFmpeg, fonts and certificate bundle. No separate desktop
app, native installer or host FFmpeg installation is required.

## Install

Supported images: Linux amd64 and arm64, including Unraid, Docker Engine,
Docker Compose and Linux containers on Docker Desktop. Windows and macOS
users access the same browser UI; native executables are no longer distributed.

| Image | Purpose |
| --- | --- |
| `ghcr.io/zidanerick/aussietv:experimental-v0.1.0-beta.1` | Main branch beta (existing compatible tag) |
| `ghcr.io/zidanerick/aussietv:docker-preview` | Development branch, published after tests |
| `ghcr.io/zidanerick/aussietv:sha-<commit>` | Exact tested multi-architecture build |

A tag only receives the Docker-first changes when its branch passes CI and
publishes. [Container registry](https://github.com/zidanerick/AussieTV/pkgs/container/aussietv)
and [CI builds](https://github.com/zidanerick/AussieTV/actions/workflows/experimental-builds.yml).

For browser TV plus Jellyfin, clone this repository, copy `.env.example` to
`.env`, restrict its permissions, and set separate random secrets (at least
24 characters, e.g. generate twice using `openssl rand -hex 24`).

```sh
cp .env.example .env
chmod 600 .env
# Edit .env before starting
docker compose -f compose.web.yaml -f compose.jellyfin.yaml up -d --wait
```

Open http://127.0.0.1:45781 on the Docker host using the configured username
and password. Both published ports default to loopback. Set private network
bindings and a reachable Jellyfin base URL as needed; use HTTPS for browser
access beyond the host. For feed-only deployments use `docker compose up -d`;
for web-only use `docker compose -f compose.web.yaml up -d`.
[Full Docker and Unraid instructions](docs/DOCKER.md).

## Features and configuration

- Regional guide, favourites, remote-friendly browser controls, channel logos
  and TVmaze programme artwork.
- Authenticated Jellyfin feeds on 45782, with direct HLS redirect default
  and optional guarded relay, experimental MPEG-TS and Fast Tune.
- Selected expanded lineup (`AUSSIETV_EXTRA_AU_CHANNELS=1`), public
  broadcaster manifest readiness checks and optional approved local M3U/XMLTV.
- Teletext pages and optional in-vision channel 995; not an analogue encoder.

See [Jellyfin](docs/JELLYFIN.md), [lineup](docs/LINEUP.md),
[providers](docs/PROVIDERS.md), [Teletext](docs/TELETEXT.md) and
[remote controls](docs/REMOTE.md). Configuration is through container
environment variables and files in the persistent /data volume.
Closing the browser does not stop the service. Update by pulling and recreating
the container, preserving /data and secrets.

## Development

```sh
docker build -t aussietv:local .
# Set AUSSIETV_IMAGE=aussietv:local in .env
docker compose -f compose.web.yaml -f compose.jellyfin.yaml up -d --wait
```

CI runs Go, browser and template tests, builds both image architectures,
checks authenticated listener modes, Compose, FFmpeg and shutdown, and publishes
the exact tested images. Source-level Go/Node tests remain available for
developers; native packaging is retired. See [developer instructions](AGENTS.md).

## Security and limitations

Keep both listeners private, use distinct feed and browser credentials, and
use a trusted HTTPS reverse proxy for LAN browser access. Do not port-forward
AussieTV. Never share signed upstream URLs or feed keys. Source availability,
DRM, artwork coverage and live Jellyfin playback vary by provider. Docker
health confirms listener availability, not successful television playback.
See [known issues](docs/BETA-KNOWN-ISSUES.md).

Channel index: [i.mjh.nz](https://i.mjh.nz/au/); artwork:
[TVmaze](https://www.tvmaze.com/api). Unaffiliated with the services.
AussieTV code is [MIT](LICENSE); third-party media and FFmpeg have their own
terms. Existing Windows releases remain available as unsupported historical
artifacts, not the current installation path. [Development history](docs/DEVELOPMENT-HISTORY.md).
