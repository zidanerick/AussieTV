# AussieTV v1.8: Jellyfin IPTV tuner and programme guide

Jellyfin natively supports **M3U tuners + XMLTV guide providers**. That is the integration used here. It does not claim to emulate an HDHomeRun tuner, perform SSDP/UPnP discovery, or transcode video to MPEG-TS. No Jellyfin plugin needs to be installed.

## Endpoints

The optional feed listener exposes only these authenticated GET endpoints:

- `/jellyfin/channels.m3u?key=YOUR_KEY` — current regional TV lineup (streamable channels only).
- `/jellyfin/guide.xml?key=YOUR_KEY` — XMLTV schedule with channel IDs matching the playlist.
- `/jellyfin/stream/ID?key=YOUR_KEY` — temporary HTTP 307 redirect to the channel's current external stream URL.
- `/jellyfin/health?key=YOUR_KEY` — `ok` (authentication test).

The stream redirect is deliberate for the first integration milestone. Jellyfin/FFmpeg must be capable of playing the upstream HLS stream on its own. Some broadcasters use authentication, custom headers, token expiry, regional restrictions or DRM. Such streams may fail even though the playlist imports correctly. AussieTV cannot bypass restrictions. The external channel provider can also change without warning.

### Windows: opt-in local network feed

For example, on a PC with LAN IP `192.168.1.50`, set these environment variables **before** starting `RUN-AussieTV.cmd`:

```powershell
$env:AUSSIETV_JELLYFIN_BIND='0.0.0.0:45782'
$env:AUSSIETV_JELLYFIN_BASE_URL='http://192.168.1.50:45782'
$env:AUSSIETV_JELLYFIN_KEY='REPLACE-WITH-A-LONG-UNIQUE-RANDOM-TOKEN'
$env:AUSSIETV_JELLYFIN_REGION='Melbourne'
.\AussieTV.exe
```

The desktop app continues listening on `127.0.0.1:45781` and retains the normal browser launcher. The second port, `45782`, exposes *only* the protected Jellyfin feeds, not the UI, heartbeat, shutdown API or unrestricted HLS/image proxy. Allow the port only from Jellyfin's LAN address in the host firewall. The token appears in URLs. Never publish the port to the public internet or share the playlist publicly. For reverse-proxy access, use TLS and additional access controls.

### Docker / Unraid

This is a **feed-only** container for Jellyfin; it does not yet serve the full AussieTV UI on the LAN. To run via Docker Compose, place these lines in `.env` next to `compose.yaml`:

```dotenv
AUSSIETV_JELLYFIN_BASE_URL=http://192.168.1.50:45782
AUSSIETV_JELLYFIN_KEY=REPLACE-WITH-A-LONG-UNIQUE-RANDOM-TOKEN
AUSSIETV_JELLYFIN_REGION=Melbourne
```

Replace the example LAN IP with the **Docker host's reachable LAN IP**, and the key with a random secret of at least 24 characters. Generate a secret, for example, with `openssl rand -hex 24`. Run `docker compose up -d --build`. Port 45782 must be reachable from your Jellyfin server, and must not be exposed to the internet. The container runs as an unprivileged user and persists guide cache to `/data`. If running on Unraid, create a Docker template with the same environment variables and port mapping.

### Configure Jellyfin

1. In Jellyfin Dashboard → **Live TV** → **Tuner Devices** → `+` → choose **M3U Tuner**.
2. Paste `http://192.168.1.50:45782/jellyfin/channels.m3u?key=YOUR_KEY` and save.
3. In **TV Guide Data Providers** → `+` → choose **XMLTV**.
4. Paste `http://192.168.1.50:45782/jellyfin/guide.xml?key=YOUR_KEY` and save.
5. Refresh guide data and check channel mapping. The generated `tvg-id` and XMLTV `<channel id>` values match exactly.
6. Test one ABC, SBS, Seven, Nine and 10 channel at a time. A populated guide does **not** prove the videos are playable.

The playlist redirects to public upstream HLS URLs rather than acting as an HDHomeRun transport-stream tuner. Jellyfin's transcoding, recording support, and source restrictions are determined by Jellyfin and each original stream. The M3U/guide endpoints are intentionally bound to a **separate, opt-in authenticated port**.

### Validation status

- Go unit tests cover authentication, guide/channel ID matching, XML escaping, UTC timestamps, inaccessible channels, redirects, and network endpoint isolation.
- Windows cross-build and Linux build work. A synthetic localhost feed health request was exercised.
- Docker image builds and live Jellyfin playback have **not** been tested in this development environment. Host-level port firewall and local network access must be configured by the operator.
- Future enhancement: optionally proxy authenticated compatible HLS segments where permitted and justified, implement a proper multi-user LAN TV UI, evaluate HDHomeRun discovery only if native M3U/XMLTV proves insufficient.
