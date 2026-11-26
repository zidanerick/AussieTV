# Docker experimental build

The Windows beta release remains unchanged. Docker supports Linux amd64 and
arm64. FFmpeg, CA certificates and fonts are included. The process runs as
UID/GID 10001; use a named volume for /data or set bind-mount ownership
accordingly. No privileged mode or host networking is required.

After the experimental workflow passes on main, it publishes
`ghcr.io/zidanerick/aussietv:experimental-v0.1.0-beta.1` and a commit SHA tag.
The repository owner may need to make the GHCR package public. PR builds
instead provide architecture-specific docker save archives. Extract the
Actions artifact ZIP, verify SHA256SUMS, then `docker load -i <archive.tar.gz>`;
the local loaded tag is `aussietv:ci`.

## Browser TV (opt-in)

    export AUSSIETV_WEB_PASSWORD="$(openssl rand -hex 24)"
    docker compose -f compose.web.yaml up -d

Open http://127.0.0.1:45781 and log in as viewer using the generated password.
Store the password in a private .env file for persistent deployments. For
local testing of the loaded CI image, set AUSSIETV_IMAGE=aussietv:ci.
Alternatively build locally with `docker build -t aussietv:local .` and
set AUSSIETV_IMAGE=aussietv:local.

Unraid: use the same image, bridge networking, /data mapped to an appdata
directory owned by 10001, and container TCP 45781. Set AUSSIETV_MODE=server,
AUSSIETV_WEB_BIND=0.0.0.0:45781, AUSSIETV_WEB_USER, a random password of at
least 24 characters in AUSSIETV_WEB_PASSWORD, AUSSIETV_WEB_ORIGIN to the
exact browser origin (e.g. https://tv.example.test), and set
AUSSIETV_JELLYFIN_BIND to an empty string unless configuring the feed.

HTTP Basic authentication is not encryption. Use an HTTPS reverse proxy
with a trusted certificate for LAN access, preserving Host and using the
exact HTTPS origin. Bind the published port to loopback when using a local
proxy; never expose the service directly to the Internet. If you explicitly
publish on a LAN address, restrict direct access to trusted clients/proxy
with a firewall. No multi-user account management or browser settings/quit
endpoints are exposed. Real playback, subtitles and remote input still
require testing. This is not a stable multi-user server.

## Jellyfin-only (original mode)

Use the original compose.yaml with the required BASE_URL and KEY variables:

    export AUSSIETV_JELLYFIN_BASE_URL=http://YOUR_SERVER_IP:45782
    export AUSSIETV_JELLYFIN_KEY="$(openssl rand -hex 24)"
    docker compose up -d

Use the existing authenticated M3U/XMLTV endpoints and keep TCP 45782
private. See [Jellyfin](JELLYFIN.md). The web UI remains loopback-only
inside the container unless explicitly enabled. For combined deployment,
publish 45782 separately and configure the Jellyfin environment alongside
the web environment. HDHomeRun discovery is not enabled by default.
