> **Version note (v0.1.0-beta.1):** This document may describe pre-beta v1.x/v2.x prototypes. Current Windows beta status and support boundaries are in [README](../README.md) and [Known issues](BETA-KNOWN-ISSUES.md).

# AussieTV v2.0.5 preview: XMLTV programme art and copy-only transport stream

## XMLTV programme art

AussieTV exports `<programme><icon src="https://static.tvmaze.com/..."/></programme>` only for fresh, confidently matched TVmaze poster URLs. It uses the existing in-memory artwork cache shared with the UI (`main.go`). An initial guide request always returns immediately with whatever is already cached, including channel logos and unchanged `tvg-id`/XMLTV channel IDs. Subsequent guide requests include newly resolved images after the background workers complete.

Two asynchronous workers with one shared **700 ms** request ticker bound outgoing TVmaze searches. The pending queue holds at most 128 unique titles; excess misses can be queued on a subsequent guide request. Negative matches are cached for 30 minutes and valid images for 12 hours. Search results must contain **exactly one** case/punctuation-normalized title match. Broadcast news, weather and similar ambiguous titles are excluded. Movie art outside TVmaze's shows database is not invented. API results must have an HTTPS poster on `static.tvmaze.com`; never use arbitrary artwork sources.

Jellyfin can import these URLs from XMLTV, but there is an [open Jellyfin 12 import issue #18209](https://github.com/jellyfin/jellyfin/issues/18209) for missing Primary images despite valid imported `<icon>` values. No user Jellyfin server is available for direct verification in this build.

## Experimental MPEG-TS relay

**Default: `redirect`**, the original working M3U HLS redirect. `relay` continues to offer the existing same-origin HLS rewrite. `mpegts` is a separate, strictly opt-in experiment:

- Windows Settings → Jellyfin channel stream delivery → **Experimental MPEG-TS copy relay**; the bundled Windows GitHub Actions build includes `ffmpeg/ffmpeg.exe`; source-only builds require `SETUP-FFmpeg.cmd` or an FFmpeg already on PATH.
- Server/Docker environment: `AUSSIETV_JELLYFIN_STREAM_MODE=mpegts`; the default remains unset/`redirect`.
- M3U entries change to `/jellyfin/stream/<channel>.ts?key=...` only when this mode is selected. Auth, regional lineup, logos and guide IDs stay the same. Jellyfin may require Refresh Tuner (and keeping **Allow stream sharing enabled**) when changing modes.
- Each GET starts a separate FFmpeg process, reading HLS through a **loopback-only authenticated** AussieTV relay at `127.0.0.1:45782` with the existing HMAC-validated and SSRF-guarded playlist/segment endpoints. It uses `-c:v copy -c:a copy -f mpegts` and does not transcode. The client request context cancels FFmpeg when Jellyfin disconnects; the feed's HTTP write deadline is lifted only for this continuous stream.
- MP4/fMP4 and MPEG-TS HLS support depends on FFmpeg, browser/network conditions and source codec. DRM and provider restrictions are **not bypassed**. Running one FFmpeg per stream costs additional process/network resources; do not enable without measured benefit.
- MPEG-TS is **not** an HDHomeRun RF emulator and does not fix UDP discovery. The HDHomeRun virtual tuner remains an independent experimental feature.

**Security note:** The key is passed to a loopback HLS URL as an FFmpeg command-line input, so users with local process-inspection privileges may be able to see it. Treat that machine as trusted and use a dedicated feed key. The app never includes the key or broadcaster token in diagnostics.

## Tests and release gate

`go test -race ./... -count=1`, `node --test tests/remote_keys.test.cjs tests/settings_page.test.cjs`, and Windows x64 cross-compilation. Synthetic FFmpeg/FFprobe integration tests exercise H.264 + AAC copy, auth, segment rewriting and disconnection. Those tests do **not** count as real Jellyfin Web benchmarking. Use [Jellyfin benchmark protocol](JELLYFIN-BENCHMARK.md) and keep the deployed tuner unchanged until accepted.

### FFmpeg executable lookup (v2.0.6)

Experimental MPEG-TS first attempts an explicit absolute `AUSSIETV_FFMPEG_PATH` (when provided), otherwise `ffmpeg/ffmpeg.exe` next to AussieTV.exe, then `ffmpeg.exe` next to the application, then system `PATH`. The Teletext video encoder uses the same resolver. A source-only ZIP is not a bundled Windows executable distribution; use the GitHub Actions artifact or `SETUP-FFmpeg.cmd`. The Windows packaging workflow verifies the publisher's SHA-256 digest before including the binary, and retains FFmpeg's GPLv3 notice.

### v2.0.7 MPEG-TS startup diagnostics

The Windows portable archive contains `ffmpeg/ffmpeg.exe`. The Settings page now reports **FFmpeg: Ready (bundled)** when the selected stream mode is `mpegts` and the executable starts successfully. It reports a separate message if the binary is missing or blocked. A copy of the executable outside its `ffmpeg` subdirectory is not required.

For a failed `mpegts` tune, the authenticated `/jellyfin/diagnostics?key=...` endpoint now includes `ffmpegPresent`, `ffmpegRunnable`, `ffmpegSource` and a bounded `mpegtsStartupTimeoutSeconds`. Entries under `events` distinguish `mpegts-executable-start-failed`, `mpegts-source-forbidden`, `mpegts-invalid-source-media`, `mpegts-feed-authentication`, `mpegts-loopback-connection` and `mpegts-startup-timeout`. Raw FFmpeg stderr is never logged or returned, since it can contain channel access tokens and the private feed key. Before v2.0.7, errors were silently discarded and a failed FFmpeg process could leave the request waiting indefinitely for its first MPEG-TS packet. The timeout applies **only before the first transport packet**, not to ongoing streams.

**Test discipline:** a successful `ffmpeg -version`, synthetic H.264/AAC remux, and first-packet test establish executable and pipeline health but do not prove compatibility with every Australian broadcaster or Jellyfin Web client. If the new diagnostics show an upstream HTTP failure, keep `redirect` mode until the protected HLS path is resolved. Do not publish the diagnostics URL itself, since it includes your feed key.

## v2.0.8: SBS World Movies MPEG-TS troubleshooting

The v2.0.7 log for `mjh-sbs-4syd` showed HLS manifests returning HTTP 200 in 37–72 ms, followed by `mpegts-no-transport-packets` HTTP 502 in roughly 320–350 ms. This is an early FFmpeg input failure, not the 18-second startup timeout. No claim is made that the root cause has been verified without raw FFmpeg logs and a live client test.

The regional source index supplies a custom User-Agent for SBS World Movies. Previous Jellyfin media requests ignored this source field and always used Chrome; v2.0.8 now retains and safely forwards only indexed User-Agent and public Referer headers. It also emits extension-bearing, signed local HLS links, accommodating FFmpeg HLS media-extension validation without disabling security checks.

In authenticated `/jellyfin/diagnostics`, compare counters before/after one experimental channel attempt:

- `relayManifestRequests`: playlist fetches reaching AussieTV.
- `relayMediaRequests`: successful upstream media responses, including segments.
- `relayMediaFailures`: upstream media errors or interrupted transfers.
- `relayMediaBytes`: total bytes actually transferred to the local FFmpeg client.

If manifest requests increase but media requests stay at zero, FFmpeg isn't successfully fetching segments. If media failures increase, investigate upstream delivery/header/restrictions. If media bytes increase but FFmpeg outputs no TS, investigate codec/segment parsing or MPEG-TS muxing. Never share the diagnostics request URL or private feed key; share only the sanitized JSON body. These counts accumulate since process start.

**Validation:** Offline H.264/AAC fragmented-MP4 master+variant playlist remux test passes using FFmpeg 7.1.5, with signed local HLS links. An extensionless control also passed on this Linux FFmpeg build, so the media-extension change is protective, not a proven explanation. No SBS World Movies live test, native Windows bundled FFmpeg test, Jellyfin Web playback, or click-to-picture benchmark has been performed. Do not change the deployed tuner until verified; default `redirect` remains unchanged.
