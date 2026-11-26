# Docker and Unraid installation

Docker is the supported deployment for AussieTV. Linux amd64/arm64 images
include FFmpeg, fonts and certificates. Use Docker Engine with Compose v2,
Unraid or Docker Desktop in Linux-container mode. The container runs as
UID/GID 10001. No privileged mode, host networking or host FFmpeg is required.

## Image selection

- Main beta: `ghcr.io/zidanerick/aussietv:experimental-v0.1.0-beta.1`
  (historical tag retained for upgrades).
- Development: `ghcr.io/zidanerick/aussietv:docker-preview`.
- Pin an exact tested revision: `ghcr.io/zidanerick/aussietv:sha-<commit>`.

CI publishes only after tests pass. A successful branch push is not proof
that its image has been published. Use [Actions](https://github.com/zidanerick/AussieTV/actions)
and [GHCR](https://github.com/zidanerick/AussieTV/pkgs/container/aussietv).
If GHCR requires authentication, use an authorised package read token or
load the tested architecture artifact with `docker load -i archive.tar.gz`
after verifying SHA256SUMS. The loaded tag is aussietv:ci.

## Docker Compose

Copy .env.example to .env, chmod 600 and edit it. Generate two distinct
random secrets with openssl rand -hex 24. Empty secrets fail validation.

| Mode | Command |
| --- | --- |
| Browser and Jellyfin | `docker compose -f compose.web.yaml -f compose.jellyfin.yaml up -d --wait` |
| Jellyfin only | `docker compose up -d --wait` |
| Browser only | `docker compose -f compose.web.yaml up -d --wait` |

The three files are intentionally retained for existing installations.
Use the same file arguments for pull, up, logs and down. A single container
serves both listeners in combined mode; do not start the modes as separate
instances on conflicting ports. Browser UI defaults to http://127.0.0.1:45781
on the host. Both published ports default to host loopback. If Jellyfin is on
another host, set AUSSIETV_FEED_HOST_BIND to a reachable private host IP
and AUSSIETV_JELLYFIN_BASE_URL to its reachable address (not Jellyfin's own
web UI). For browser LAN viewing use an HTTPS reverse proxy with a trusted
certificate, preserve Host, set AUSSIETV_WEB_ORIGIN to the exact browser
origin, and restrict direct access to the backend.

## Unraid

The [manual template](../unraid/AussieTV.xml) is not listed in Community
Applications. In Docker > Add Container, use the fields below or import
the template using your usual Unraid template workflow.

| Setting | Value |
| --- | --- |
| Repository | Main beta tag above, or docker-preview for development |
| Network | Bridge |
| Privileged | No |
| Appdata | /mnt/user/appdata/aussietv mapped to /data |
| Browser port | 45781/TCP (if enabled) |
| Jellyfin feed port | 45782/TCP (if enabled) |
| Extra parameters | --read-only --cap-drop ALL --security-opt no-new-privileges --tmpfs /tmp:rw,noexec,nosuid,size=16m |

For a **new** appdata directory only:
`mkdir -p /mnt/user/appdata/aussietv && chown 10001:10001 /mnt/user/appdata/aussietv && chmod 700 /mnt/user/appdata/aussietv`.
Do not recursively change unrelated appdata ownership. Existing working
mounts need no change. Set WebUI to the configured browser origin in Advanced
View. The template leaves it blank for safety. Do not expose these ports to
the Internet. A manual template has static CI tests but no automated Unraid GUI test.

## Environment reference

| Variable | Purpose / example |
| --- | --- |
| AUSSIETV_MODE | server (image default; required) |
| AUSSIETV_WEB_BIND | 0.0.0.0:45781 or empty to disable |
| AUSSIETV_WEB_ORIGIN | Exact http(s) origin, no trailing slash |
| AUSSIETV_WEB_USER | viewer or chosen user |
| AUSSIETV_WEB_PASSWORD | Unique secret, minimum 24 characters |
| AUSSIETV_JELLYFIN_BIND | 0.0.0.0:45782 or empty to disable |
| AUSSIETV_JELLYFIN_BASE_URL | Reachable AussieTV URL, e.g. http://192.168.1.10:45782 |
| AUSSIETV_JELLYFIN_KEY | Separate secret, minimum 24 characters |
| AUSSIETV_JELLYFIN_REGION | Melbourne or supported region |
| AUSSIETV_JELLYFIN_STREAM_MODE | redirect (default), relay, mpegts, fast-tune |
| AUSSIETV_EXTRA_AU_CHANNELS | 1 for selected expansion, otherwise 0 |
| AUSSIETV_PROVIDERS_FILE | Optional /data/providers/providers.json |
| AUSSIETV_TELETEXT_CONFIG | Optional /data/teletext.json; embedded default otherwise |
| AUSSIETV_TELETEXT_VIDEO | 1 for channel 995, otherwise 0 |
| AUSSIETV_JELLYFIN_ARTWORK_SYNC | 1 for opt-in direct artwork repair |
| AUSSIETV_JELLYFIN_SERVER_URL | Jellyfin server URL for artwork repair |
| AUSSIETV_JELLYFIN_API_KEY | Dedicated Jellyfin administrator API key (not feed key) |

Compose-only variables: AUSSIETV_IMAGE, AUSSIETV_HOST_BIND,
AUSSIETV_FEED_HOST_BIND. Environment values are deployment configuration;
the browser UI has no admin settings or quit endpoint. At least one
authenticated listener is required. Feed-only does not start a web listener.
Browser Basic authentication and feed query keys are not encryption.
Protect .env, backups, Docker access and reverse proxy logs.

## Updates and rollback

Use Unraid Check for Updates / Update, or the same Compose file arguments
used to start the service with pull then up -d --wait. Preserve /data,
environment variables, existing feed URLs and keys. Back up appdata
and private deployment configuration before updates. Do not use down -v
unless deliberately deleting the named volume. Roll back by pinning the
previous tested SHA image and recreating the container.

Inspect docker ps, docker logs AussieTV or docker compose logs. Health checks
test authenticated listeners locally, not external stream availability.
Use /jellyfin/diagnostics?key=YOUR_KEY for redacted playback status (never
share the URL/key). A browser tab closing does not stop the container.
SIGTERM triggers shutdown. Docker restarts use restart: unless-stopped.

## Local development

```sh
docker build -t aussietv:local .
# Set AUSSIETV_IMAGE=aussietv:local in .env
docker compose -f compose.web.yaml -f compose.jellyfin.yaml up -d --wait
```

No native binary package is produced. Source-level tests remain available
to contributors; [developer instructions](../AGENTS.md).
