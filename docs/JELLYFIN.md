> **Version note (v0.1.0-beta.1):** This document may describe pre-beta v1.x/v2.x prototypes. Current Windows beta status and support boundaries are in [README](../README.md) and [Known issues](BETA-KNOWN-ISSUES.md).

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

## v1.9 experimental HDHomeRun detection

AussieTV can optionally advertise a **virtual HDHomeRun-compatible IPTV tuner** to software on the same private LAN. This is an emulator for discovery and lineup integration, not an actual RF/DVB-T tuner. It does not provide SiliconDust firmware, tuner control via TCP 65001, MPEG-TS conversion, hardware transcoding, or dedicated channel buffers. Detection or HDHomeRun-mode playback in Jellyfin cannot be guaranteed. The keyed M3U + XMLTV tuner is still the preferred, supported integration.

On Windows, right-click AussieTV in the notification area, open **Settings**, tick **Enable Jellyfin**, tick **Allow Jellyfin on other computers**, select the correct LAN IP and region, then optionally tick **Advertise as a virtual HDHomeRun tuner**. Click **Save and apply**. For Docker/server mode, set `AUSSIETV_HDHR_DISCOVERY=1` along with `AUSSIETV_JELLYFIN_BIND=0.0.0.0:45782` and the usual feed credentials, and ensure the host can receive UDP broadcast on port 65001. Prefer Docker host networking for this experiment; regular bridged containers typically do not receive LAN broadcasts.

The optional LAN-only endpoints on TCP 45782 are `/discover.json`, `/lineup.json`, `/lineup_status.json`, and `/auto/v<channel-number>`. These four interfaces are deliberately **not protected by the M3U key** when discovery is enabled because clients cannot supply it during tuner discovery. They are limited to private source IPs and must be additionally protected by the host firewall. `GET /jellyfin/channels.m3u?key=...` and `GET /jellyfin/guide.xml?key=...` remain authenticated.

In Jellyfin, try Dashboard → Live TV → Tuner Devices → **Detect My Devices**. If the virtual tuner does not appear, add a **HDHomeRun** tuner manually at `http://YOUR-PC-LAN-IP:45782` (Jellyfin's port handling varies), or choose the more reliable **M3U tuner** and use the URLs from AussieTV Settings. Add the separate XMLTV guide even if Jellyfin detects the virtual tuner. HDHomeRun clients may require MPEG-TS rather than the HLS playlist redirects provided today. If channels import but fail to play, revert to M3U/XMLTV.

Firewall: allow inbound TCP **45782** only from your Jellyfin server, and UDP **65001** from your private LAN for discovery. The TV player is localhost-only at 127.0.0.1:45781 and needs no inbound rule. **Do not forward these ports on your router.** Other hosts must be on the same broadcast segment unless you have explicitly configured broadcast relaying. The UDP protocol test and HTTP requests have passed locally; real Jellyfin autoconfiguration, DVR and streaming remain unverified.

## Teletext information channel (v2.0 preview)

Enable `AUSSIETV_TELETEXT_VIDEO=1` on the AussieTV Jellyfin feed server **and install FFmpeg**. The authenticated M3U at `/jellyfin/channels.m3u?key=...` then includes virtual channel **995 AussieTV Text**, with XMLTV guide ID `aussietv.teletext`. This streams PAL-resolution MPEG-TS in-vision information pages; it is not an HDHomeRun RF tuner or embedded DVB Teletext PES stream. Without FFmpeg or the environment flag, the channel is intentionally omitted. For text/data exports, use the authenticated `/teletext/`, `/teletext/page/100`, `/teletext/tti/100` and `/teletext/export.zip` endpoints with the same `?key=` query. Keep tokens private. See [Teletext documentation](TELETEXT.md).

## Windows desktop firewall automation (v2.0.1 preview)

In the Windows tray application's **Settings**, enable Jellyfin feed and **Allow
Jellyfin on other computers (LAN)**, then save. AussieTV asks for administrator
approval once when permissions change. The generated inbound rules are named
`AussieTV Jellyfin Feed Private LAN` (TCP 45782) and, only with HDHomeRun
discovery enabled, `AussieTV HDHomeRun Discovery Private LAN` (UDP 65001).
Both use the **Private** profile and `LocalSubnet` remote scope. Existing rules
from other programs are not modified. These rules do not apply to Docker/Unraid:
configure the host firewall there as appropriate. If Windows is configured with
a **Public** network profile, the Private rules do not apply; change the trusted
home network profile through Windows Settings if appropriate, rather than opening
AussieTV to every network. Never forward the ports on your router.

If Windows elevation is cancelled or the firewall command fails, AussieTV saves
the feed settings and shows a warning. Use **Repair firewall access** to retry.
It is safe to use Jellyfin on the same PC without any inbound rule.

## v2.0.2 preview: Slow tune and discovery troubleshooting

### Slow channel start (sometimes 30+ seconds)

Jellyfin may send an HTTP `HEAD` request before it opens an M3U channel, and remote HLS providers can stall on `HEAD`. The new local `/jellyfin/stream/<id>.m3u8` path returns a quick HLS MIME-type reply for `HEAD`, then uses the original HTTP redirect for playback `GET`. Older `/jellyfin/stream/<id>` links still work. The guide cache also now serves stale channel records while refreshing asynchronously. This removes two AussieTV-related sources of delay, **not** Jellyfin transcoding time or broadcaster/region/DRM restrictions.

After installing v2.0.2, fully **Exit AussieTV** using the tray menu and restart it, then refresh your Live TV tuner/channel list in Jellyfin. Some Jellyfin installations may need the M3U tuner removed and re-added before they use updated stream URLs. Keep the existing XMLTV guide URL and key. If streams continue taking 30+ seconds, inspect Jellyfin's Live TV log for `HEAD request`, `shared stream`, or `ffmpeg` lines. Keep **Allow stream sharing** enabled if disabling it causes the stream to fail (as observed in this installation). Avoid changing transcoding settings blindly.

### HDHomeRun automatic discovery

When enabled in Windows Settings, AussieTV now attempts to listen on your **configured private IPv4 address at TCP 80** for `/discover.json` and `/lineup.json`, as well as UDP 65001, because some Jellyfin discovery paths query HTTP port 80 regardless of the port in the advertised URL. The keyed feeds remain on TCP 45782. No HTTP port 80 listener runs when discovery is disabled. Only HDHomeRun-compatible, read-only endpoints are available on TCP 80; desktop app control endpoints are not exposed.

Windows Firewall rules are Private profile and LocalSubnet scope only: **TCP 80, TCP 45782, UDP 65001**. If discovery was already enabled before this upgrade, use Settings → **Repair firewall access** to add TCP 80. If another application uses that same LAN IP and port 80, Settings reports discovery unavailable while leaving M3U working. This is intentionally nonfatal. Test `http://YOUR-PC-IP/discover.json` only from your local network; it should return a virtual tuner JSON object when enabled.

Jellyfin in Docker ordinarily needs **host networking** to receive the UDP broadcast. HDHomeRun detection also works only within the same broadcast subnet without explicit relaying; VPNs, VLAN boundaries, Wi-Fi client isolation and host firewalls may prevent it. Actual real-device discovery has **not** been verified for this emulator. Its HLS redirects are not guaranteed to work as a native HDHomeRun MPEG-TS source; manual M3U/XMLTV remains the recommended tuner.

## v2.0.3: HLS relay experiment and auto-detection diagnosis

Keep Jellyfin's **Allow stream sharing** enabled if your streams fail with it disabled. In Windows Settings, select **Jellyfin channel stream delivery → AussieTV HLS relay** and save to test whether eliminating broadcaster redirects helps. The M3U and XMLTV URLs do not change. To revert, select **Direct broadcaster redirect**. In Docker/server mode set `AUSSIETV_JELLYFIN_STREAM_MODE=relay` (the default is `redirect`).

The relay fetches public HLS resources through AussieTV and gives Jellyfin rewritten, authenticated manifest and media links. It cannot decrypt DRM or grant access to region-locked streams. It uses additional bandwidth and may be slower if the provider depends on cookies or referer headers. `GET /jellyfin/diagnostics?key=YOUR_KEY` returns the selected stream mode, recent response timings and discovery counters, but no broadcaster URLs or token values. Keep the diagnostics key private.

To distinguish a firewall/network discovery problem from a protocol problem: enable HDHomeRun discovery, confirm Settings reports the UDP and port 80 listeners as active, then click **Detect My Devices** in Jellyfin followed by **Refresh discovery counters** in AussieTV Settings. If the UDP probe count stays zero, Jellyfin's broadcasts never reached AussieTV. Docker-hosted Jellyfin typically needs host networking and must share the same broadcast subnet; Windows Firewall must allow UDP 65001 (and TCP 80) on the **Private** profile, LocalSubnet scope. If probes and replies increase but Jellyfin still doesn't list a tuner, protocol compatibility remains incomplete. Keep the manually configured M3U/XMLTV tuner. HDHomeRun-mode playback may need MPEG-TS output rather than the current HLS redirect.

Do not expose ports 80, 45782 or UDP 65001 to the public internet. Discovery on different VLANs does not work automatically.

## v2.0.4 discovery and startup diagnostics

The authenticated `GET /jellyfin/diagnostics?key=<your-private-key>` report now includes:

- `appVersion`: confirms which executable actually owns the feed port.
- `streamMode`: the effective `redirect` or `relay` mode. An empty saved setting means `redirect`, not an unknown mode.
- `events`: successful HEAD (200), direct stream redirects (307), and relay manifest/segment activity (where used). A quick redirect means AussieTV answered quickly; it **does not** measure Jellyfin's upstream fetch, probe or buffer time.
- `hdhomerunEnabled`: configured preference only, not proof of active discovery.
- `hdhomerunUdpListening` and `hdhomerunHttpListening`: whether UDP 65001 and the optional LAN TCP 80 listener were successfully opened.
- `hdhomerunUdpProbesReceived`, `hdhomerunRepliesSent`, `hdhomerunHttpRequests`: requests seen since startup. These should increase when Jellyfin can reach the appropriate listener.

Interpreting combinations:

- `udpListening=false`: check the Settings status for a port conflict. UDP requests cannot be counted by this process.
- `udpListening=true`, probes remain zero during **Detect My Devices**: Jellyfin's broadcasts have not reached AussieTV, often due to Docker bridge networking, a routed subnet, or an inbound firewall rule. In Docker, test host networking; do not open the service to the Internet.
- probes increase but no replies: check whether the received discovery frames have the correct HDHomeRun format; the app only responds to valid requests.
- replies increase but Jellyfin does not list the tuner: investigate HDHomeRun compatibility and TCP 80 reachability. This virtual tuner is experimental and advertises HLS redirects, not physical tuner MPEG-TS.
- `events=[]` after a successful **new** tune on v2.0.4: Jellyfin may be using cached broadcaster URLs or another source, rather than requesting AussieTV's current M3U entry.
- a direct `redirect` event completes in milliseconds while playback takes 8-9 seconds: the additional time is outside AussieTV's HTTP redirect, typically the upstream player/Jellyfin buffering path. Check Jellyfin's FFmpeg logs before changing source protocols.

Keep **Allow stream sharing** enabled if disabling it produces errors on your existing Jellyfin tuner. Normal broadcast HLS startup often requires waiting for enough media data; do not claim 8–9 seconds can be eliminated by a firewall or redirect change.

## v2.0.5 isolated programme artwork and MPEG-TS trial
See [artwork/transport notes](JELLYFIN-ARTWORK-TS.md) and [Jellyfin Web measurement protocol](JELLYFIN-BENCHMARK.md). The original redirect is still the default; additional XMLTV programme artwork is opportunistic and asynchronous. Jellyfin may not import XMLTV artwork as Primary images in version 12 (jellyfin/jellyfin#18209). **Do not alter the deployed tuner until a supervised comparison succeeds.**

## v2.0.9 experimental MPEG-TS audio detection

The optional MPEG-TS tuner now **requires audio**. The old FFmpeg `-map 0:a:0?` option silently omitted an audio stream when an HLS feed's audio rendition was not detected in the small probe window. This is now `-map 0:a:0` with a 3-second/2 MiB maximum probe budget, followed by an MPEG-TS PAT/PMT check that ensures the output actually contains a video PID and an audio PID carrying payload. This mode will show a 502 instead of a silent picture when no audio is available. The default Direct mode is unchanged.

In `/jellyfin/diagnostics`, a successful experimental tune is recorded as `mpegts-audio-video-ready`. Failures are categorised as `mpegts-audio-track-not-found`, `mpegts-audio-track-missing`, `mpegts-audio-packets-missing`, or an existing FFmpeg/HLS failure code. These classifications do not reveal URLs or access tokens.

A successful `mpegts-audio-video-ready` result verifies transport packets, **not audible sound in Jellyfin**. If Jellyfin still shows a silent picture with this result, compare the client's audio selection and Jellyfin's remux log; this points to a downstream audio-selection/decoding issue rather than a video-only AussieTV transport stream. Keep Allow stream sharing enabled. Do not replace a working Direct tuner until real playback is verified.
