> **Version note (v0.1.0-beta.1):** This document may describe pre-beta v1.x/v2.x prototypes. Current Windows beta status and support boundaries are in [README](../README.md) and [Known issues](BETA-KNOWN-ISSUES.md).

# AussieTV project continuity

**Latest implementation:** v1.7.0, 16 November 2026. GitHub: https://github.com/zidanerick/AussieTV

AussieTV is a Go Windows console app serving an embedded HTML/CSS/JS TV guide on a localhost web port, normally `127.0.0.1:45781`, with a browser app window and fallback address. It selects Australian live streams from third-party TV metadata and exposes a bounded HLS proxy. The user confirmed a previous Windows version worked, but actual broadcaster playback tests are not systematic.

## v1.7 changes

- `static/remote.js` handles TV remote input throughout the web UI, including spatial D-pad navigation, OK, Back, browser legacy key codes, numeric tuning, channel step and focus management.
- `static/index.html` adds high-contrast focus, remote help, Play/Pause/Mute and a separately loaded remote module. Original player/guide functionality remains.
- `main.go` allowlists `/remote.js` as an embedded asset, bumps version and preserves localhost service protections.
- `logo.go` and `logo_test.go` from the preceding unpublished all-channel logo fix add real guide-logo proxy with SSL and DNS safeguards, short-lived image caching and contrast-aware rendering. The service remains restricted to curated channel IDs.
- `tests/remote_keys.test.cjs` (Node only) and `tests/remote_browser_smoke.py` (optional Playwright/Chromium mock UI) cover core remote inputs.
- `docs/REMOTE.md` and `docs/INTEGRATIONS.md` document compatibility and the Docker/Jellyfin plan.

## Evidence

Go tests and race detector passed on Linux; Node remote-key tests and in-memory Chromium guide/navigation tests passed; Windows amd64 cross-compile passed. Browser test uses fake schedules and does *not* validate live video, native Windows mouse/remote or Safari/Firefox/TV-browser behaviour.

## Next tasks

1. Confirm UI navigation on the user's TV remote, determine model/transport (HDMI-CEC, Android TV, IR-USB, Bluetooth) if key events differ.
2. Build a browser matrix for Edge, Firefox and Safari where hardware is available; validate HLS.js native playback, captions, actual streams and fullscreen limitations.
3. Extract shared region/channel/EPG data API and stable IDs.
4. Implement a separate authenticated network/server mode and Docker package; do *not* simply bind existing desktop API on `0.0.0.0` (quit, heartbeat and host/origin security must be redesigned).
5. Expose M3U/XMLTV data for Jellyfin built-in Live TV; consider a plugin only for remaining features.

## Resume in another chat

Read `README.md`, `AGENTS.md`, `docs/REMOTE.md`, `docs/INTEGRATIONS.md`, GitHub open issues and the current branch head. Keep earlier snapshot branches unchanged. Do not claim successful remote input tests on hardware without an actual device test.

## v1.7.1 remote navigation repair (17 November 2026)

- Mainline bug: v1.7 guide boundary key presses fell through to full-page nearest-element matching, allowing diagonals to player/now-playing controls. Region and CC `<select>` elements always received native arrows, changing selections unintentionally and effectively trapping D-pad focus.
- Fix: in `static/remote.js` restrict horizontal direction to same-row candidates and relevant control groups; guide arrow navigation never falls through at its boundaries. Region/caption selects only edit after OK; Up/Down previews; OK fires change on commit; Back/Escape cancels. Edited select uses a distinctive focus outline.
- Tests: Go unit and race tests; Node key/spatial unit tests; mocked Chromium interactions at 1280x800 and 768x800, including no accidental region changes, commit/cancel, guide edge and player control group; Windows x64 cross-build. Physical TV remote, Safari, Firefox, native Windows player and streams are not validated.
- For Docker: preserve these remote input semantics and keep auth/server-mode changes separate from the desktop fix.


## v1.8.0 (18 November 2026): remote exits and Jellyfin feed

- The main Windows app retains localhost-only UI and desktop launcher.
- Remote navigation adds explicit section transitions when a spatial move hits a boundary. Guide arrow keys still move through logical channel/programme cells first. New browser smoke tests exercise escaping guide and control-group edges.
- The separate opt-in Jellyfin feed listener requires a long token and explicit bind/base URL. Only M3U/XMLTV, redirect and health endpoints are exposed. Server mode disables the desktop quit route and browser heartbeat.
- New files: `jellyfin.go`, `jellyfin_test.go`, `docs/JELLYFIN.md`, `Dockerfile`, `compose.yaml`, `.dockerignore`.
- Tests: `go test -race ./...`, `node --test tests/remote_keys.test.cjs`, `python tests/remote_browser_smoke.py`, local Windows cross-compilation, and local server health checks.
- Not yet verified: native remote hardware, Jellyfin live-stream playback, actual Docker build in a Docker runtime, multi-client authenticated browser UI.

## v1.9.0 (18 November 2026)

- Source-of-truth goal: preserve the Windows desktop build and support a notification area process with no console, separate Settings GUI, persistent Jellyfin feed settings and optional LAN HDHomeRun discovery.
- Files: `settings.go`, `settings_test.go`, `tray_windows.go`, `tray_other.go`, `static/settings.html`, `hdhr.go`, `hdhr_test.go`, updated `jellyfin.go`, `main.go`, TV quit button and Windows CI.
- HTTP `discover.json`, `lineup.json`, `lineup_status.json`, `auto/vNUMBER` enabled only when explicitly opted into LAN discovery, private-IP gated, no M3U key on these HDHR-specific endpoints. M3U/XMLTV still require secret. UDP broadcast 65001 responds to valid CRC-checked HDHomeRun discovery requests.
- Tests: Go/race, Windows GUI-subsystem cross-build, Node remote tests, synthetic Linux server HTTP 200/401 and live UDP discovery reply. **No actual native Windows tray click test and no real Jellyfin tuner import/playback test.**
- Security: no browser quit in tray mode; settings API same-origin and anti-CSRF; no router port forwarding. Keep LAN firewall restriction (TCP 45782 and optionally UDP 65001).
- Known limitation: native HDHomeRun-compatible clients may require HTTP port 80 or MPEG-TS stream delivery; auto-discovery and HDHR playback are experimental. M3U/XMLTV recommended.

## v2.0.0-preview (18 November 2026): Teletext and startup

The v1.9 silent launch was a Go HTTP mux route-pattern panic: `GET /` conflicts with methodless `/api/settings` in Go 1.22+. Fixed by method-specific Settings handlers. Rebuilt console diagnostic EXE accompanies the GUI app; Jellyfin port or tray failures no longer prevent TV startup. **New tests include an actual Linux startup with port 45782 held by another process.**

`teletext/` is an independent 40x24 page model with TTI exporter, tests and optional FFmpeg MPEG-TS in-vision channel. Default editable config in `config/teletext.json`; Windows picks up `teletext.json` next to its executable. Pages are served on the local UI and (authenticated) Jellyfin feed. See `docs/TELETEXT.md`. GUI full viewer is at `/teletext/`, export `/teletext/export.zip`. `AUSSIETV_TELETEXT_VIDEO=1` with FFmpeg makes the Jellyfin M3U/XMLTV virtual channel 995 available.

IMPORTANT: There is no actual PAL/VBI modulator here; use VBIT2 and hardware to transmit Teletext for an analogue TV. Native VBIT2 decoding, physical TV hardware, live BOM/weather sources and Jellyfin stream playback still require validation. Do not claim captions 800/801 are real or synced. The browser and video views do not convey a DVB Teletext auxiliary stream.

## v2.0.1 preview (18 November 2026): Settings, browser, and firewall

- Based on `AussieTV-Source-v2.0-Teletext-PREVIEW.zip`, not GitHub `main`
  (GitHub was last at v1.8; the v1.9+ sources have not yet been merged).
- Reproduced Settings failure: extraneous closing brace following `save()`
  caused a global syntax error, preventing `load()` from ever running.
- Corrected Settings JS and added Node syntax/DOM/save regression test in CI.
- Both tray Open and Settings launch through Windows ShellExecuteW's `open`
  handler (system default browser). Removed forced Edge/Chrome logic.
- On desktop Settings save, changes to LAN/HDHomeRun permissions prompt for
  administrator approval in a short-lived helper process. The helper only
  manages two app-owned inbound Windows Firewall rules, Private profile and
  LocalSubnet remote scope, TCP 45782 and optional UDP 65001. Local-only
  usage makes no firewall changes. A cancelled UAC prompt produces a visible
  warning without re-enabling a disabled feed; Repair allows retrying.
- Test evidence: local Go tests/race detector, Node remote/settings tests,
  Windows GUI and diagnostic cross-compilation, HTTP/Settings verification.
  Native UAC and default-browser behaviour require an actual Windows PC test.
- Continue Teletext and Jellyfin work from this preview source, not old main.

## v2.0.2 Preview, 18 November 2026: Jellyfin tuning latency and HDHR discovery

- Code based on the unmerged v2.0.1 Teletext/Settings preview, NOT repository main (which remains v1.8).
- In `jellyfin.go`, return immediate HTTP 200 HLS MIME response for HEAD and suffix the M3U HLS links `.m3u8`; GET still uses the original upstream redirect.
- In `main.go`, retain expired cached channel data immediately, refresh in the background with a 1-minute attempt throttle. Does not avoid initial cold fetch.
- In `hdhr.go`, correct virtual DeviceID checksum and validate a private IPv4 host for optional discovery HTTP port 80.
- In `settings.go`, add a second LAN-only listener on TCP 80 for HDHomeRun discovery when enabled; fail softly if unavailable, maintaining authenticated M3U on TCP 45782. Settings status exposes the warning.
- In `firewall_windows.go`, add Private/LocalSubnet TCP80 rule on discovery and remove on disable. Existing installations must use Repair firewall access.
- Tests: `go test ./...`, `go test -race ./...`, Node remote/settings tests and Windows GUI/console crossbuild passed locally. Synthetic tests do not establish real Jellyfin playback or physical network discovery.
- Next: user-run Windows test with actual Jellyfin and its UDP broadcast path, confirm no 30+ second tuning when cache refresh occurs, monitor Live TV HEAD and ffmpeg logging.

## v2.0.3 Preview, 18 November 2026: Jellyfin relay and discovery troubleshooting

- Developed from the unmerged v2.0.2 Windows/Teletext preview. GitHub main remains on v1.8 until this preview is separately reviewed and merged.
- Added `jellyfin_relay.go` and tests; direct 307 redirects remain the default. Optional HLS relay mode is selectable in Settings or via `AUSSIETV_JELLYFIN_STREAM_MODE=relay` for server mode. Existing M3U URLs still work.
- New authenticated `/jellyfin/diagnostics` endpoint reports timings and HDHomeRun UDP probe/reply/HTTP counts without exposing stream URLs or keys. Settings includes a refresh button for these counters.
- Tests verify playlist rewriting, segment passthrough, denied private/forged URLs, auth, actual localhost UDP discovery response, Go race checks, JS Settings/remote tests, Windows GUI/diagnostic cross-build. Not a claim of real Jellyfin detection or live broadcaster timing improvements.
- User confirmed manual M3U works, autodetection does not, and disabling Jellyfin **Allow stream sharing** breaks playback. Recommend keeping stream sharing enabled and using the M3U tuner while debugging UDP reachability.
- Next: compare Jellyfin startup latency in redirect versus relay, and inspect whether UDP probes are observed when Jellyfin searches. If no UDP received check Docker host networking/subnet/firewall; if probes received but no detection, investigate remaining HDHomeRun protocol requirements.

## v2.0.4 (18 November 2026): Jellyfin diagnostics and discovery partial bind

- Based on v2.0.3 preview, not GitHub main (older v1.8).
- Direct stream mode previously emitted `events:[]` with `streamMode:""`; now records 200 HEAD and 307 redirect events and returns effective mode. This does not measure the broadcaster/Jellyfin time after redirect.
- HDHomeRun UDP listener can now operate when TCP 80 is unavailable; authenticated `/jellyfin/diagnostics` reports both bind states separately. Listening does not imply Jellyfin receives UDP or accepts the tuner.
- Keep sharing enabled in Jellyfin for the known working tuner. Measure actual 8-9 s behaviour before further buffering changes.

## v2.0.5 isolated artwork/MPEG-TS experiment (18 November 2026)
Source is based on v2.0.4-preview, not the repository's old main (v1.8). No production changes. See `docs/JELLYFIN-ARTWORK-TS.md` and `docs/JELLYFIN-BENCHMARK.md`. XMLTV programme art uses existing TVmaze cache with bounded background workers, and MPEG-TS runs copy-only FFmpeg via authenticated localhost HLS. Default redirect mode unchanged. Offline integration covers H.264/AAC, auth, cancellation and PTS; live Jellyfin Web timings and programme Primary-image imports are **not yet verified**. Do not merge the preview into main or ask users to alter their working tuner before device validation.

## v2.0.9 preview (18 Nov 2026)

Branch release remains standalone; GitHub `main` remains older. Experimentally mapped audio was optional in v2.0.8, allowing FFmpeg to return a video-only MPEG-TS and a misleading 200. `jellyfin_mpegts.go` now requires first audio stream, uses bounded larger probes, and verifies PAT/PMT and elementary stream packet presence with `ts_audio_probe.go`. Go tests verify no-audio upstream returns 502 and A/V fixtures remain valid. No direct Jellyfin playback confirmation is claimed.
