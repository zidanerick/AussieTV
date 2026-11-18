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
