# AussieTV project continuity / future-chat handoff

**Last documented:** 18 November 2026  
**Current code:** v1.8.0 on `main` (PR #13 merged; `6e3a900ccd9dcf5f1fdb82c7e9479dd742334029`)  
**Repository:** https://github.com/zidanerick/AussieTV

## User goal

Create a simple, TV-like Windows interface for Australia's main free-to-air broadcasters (ABC, SBS, Seven, Nine, 10), with a real programme guide, channel logos and relevant artwork. The user is in Victoria; Melbourne channel selection is currently the default. A possible later goal is running similar frontend functionality on Raspberry Pi 3B.

## Implemented

- Portable Windows x64 Go application serving embedded HTML/CSS/JS on localhost.
- Reads regional channel listing and EPG data from https://i.mjh.nz/au/ (third-party service).
- HLS video via browser/HLS.js where playable; broadcaster official live-TV pages as fallbacks.
- Main broadcaster filtering, selected secondary channels and full-screen playback.
- v1.1: visible launcher `RUN-AussieTV.cmd`, browser start, log at `%LOCALAPPDATA%\AussieTV\startup.log` and fallback local URL.
- v1.2: guide-provided channel logos, brand-coloured fallbacks, optional TVmaze show artwork for exact title matches.
- v1.3: persistent browser-local favourite channels, Favourites filter, and visible star for saved channels.
- v1.3: EPG directional focus navigation, correct Enter activation on actual buttons, focus retained when guide refreshes, and single redraw per minute.
- v1.3: startup timeout for HLS playback and one maximum HLS media recovery.
- v1.3: signed HLS segment/variant URL rewriting, strict channel-root validation, outbound DNS/public IP and redirect guards, local Host/Origin checks.

- v1.4: original AussieTV logo and wordmark, favicons and multi-size ICO; reproducible Go icon generator and brand asset HTTP tests. The existing guide/player behaviour is unchanged.
- Original branding source: `static/app-icon.svg`, `static/aussietv-logo.svg`, `tools/iconbuilder/main.go` and `assets/BRANDING.md`.
- Run `go run ./tools/iconbuilder` before building to produce generated PNG and ICO files. The GitHub Actions build runs this automatically.
- Windows ZIP contains `assets/AussieTV.ico` for shortcut customization. The original v1.4 icon was supplied separately; v1.5 now also embeds it in the executable.

- v1.5: embeds seven icon sizes in the PE executable via `tools/iconbuilder/winresource.go`, generated `rsrc_windows_amd64.syso`. Prefers Edge/Chrome `--app` window, retains v1.1 shell/default browser fallback and supports `AUSSIE_TV_BROWSER_MODE=default`.

## Exact historical versions

- v1.0: commit `681b9aa580f0ac5f2ff961f2804c654866631391` ; branch `versions/v1.0`
- v1.1: commit `c78cc5a2a3674bedd0277dadbd9771b4fe5ad8e3` ; branch `versions/v1.1`
- v1.2: commit `72bc93cb9d55aa4b2a7db9bc603df52c9cb3e1c9` ; branch `versions/v1.2`
- v1.3: branch `versions/v1.3` stores the latest source checkpoint; see GitHub commit history for the exact version commit.
- Later docs/automation commits on main are not themselves new user-facing product versions.

## Build / test / launch

```bash
go run ./tools/iconbuilder
go test ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o AussieTV.exe .
```

Bundle `AussieTV.exe`, `RUN-AussieTV.cmd`, `README.txt` and `assets/AussieTV.ico`. GitHub Actions workflow `.github/workflows/windows-build.yml` builds and uploads this ZIP automatically. Local address defaults to http://127.0.0.1:45781.

## Verification state

- v1.2 unit tests passed in Linux on 15 Nov 2026.
- v1.3 Go unit tests and race detector passed in Linux on 15 Nov 2026, including signed URL, forged HLS and private-DNS tests.
- v1.3 headless Chromium interaction checks passed using mocked channel data: 5 networks, favourite persistence serialization/filtering, guide arrow navigation, Enter activation, focus preservation. A screenshot was inspected in the development environment.
- v1.3 local server responded HTTP 200; invalid Host requests returned HTTP 403.
- v1.3 Windows x64 cross-compilation passed in Linux.
- Windows x64 executable cross-compilation passed, with PE console subsystem.
- Source and local HTTP endpoints were checked in earlier work.
- User previously confirmed that the v1.1 Windows app launched successfully.
- **Not proven**: live Australian stream playback, Windows native startup for v1.3, sustained stream reliability, artwork matching for every programme, or Australian geolocation behaviour. Browser interaction tests used mock EPG data and did not test video. Do not claim those tests passed.

- v1.5 local verification: Go tests passed, Windows amd64 executable cross-built, and Windows PE .rsrc structure inspected with seven icon sizes. The native Windows app-mode/window/taskbar behaviour is not independently tested on Windows; the browser still owns that window.

## Technical risks / boundaries

1. Third-party stream index is not guaranteed to stay online and does not imply rights to redistribute channels.
2. Some streams need DRM, tokens, login or location checks and cannot simply be played in a generic HLS player. Use official broadcaster links as fallback. Do not add bypasses.
3. Artwork from TVmaze can misidentify names; keep conservative title matching and visible attribution.
4. Local streaming proxy makes outbound requests. Preserve SSRF restrictions and loopback-only bind; add regression tests for redirects/host validation if modifying it.
5. Latest code is a proof-of-concept, not a digitally signed production distribution.
6. Melbourne guide may not match regional Gippsland schedules.

## Next useful improvements

- Verify a native Windows build from the GitHub Actions artifact and check actual stream-by-stream playback.
- Log which channels fail and why (DRM, expired URL, CORS, region, missing EPG), without recording secrets.
- Improve EPG further for TV remotes, including test coverage and regional channel guides. Keyboard navigation and favourites first landed in v1.3.
- Evaluate artwork caching and now/next improvements without unnecessary API traffic.
- Investigate official/legal streaming integrations and proper regional selection.
- Add release tags and release attachments after a real Windows smoke test.

## How to continue in a new chat

Say: "Continue development of https://github.com/zidanerick/AussieTV. Read README.md, AGENTS.md, docs/CONTINUITY.md and current GitHub issues. Inspect the actual main branch before changing code. Work from the current version, preserve historical branches, run tests, and update the handoff document."

If a change is made, update this document with: exact commit SHA, affected files, tests executed, Windows device evidence (or an explicit gap), regressions, and the next task. Last v1.3 files: `main.go`, `main_test.go`, `static/index.html`, `README.txt`, `README.md`, `CHANGELOG.md`, `AGENTS.md` and this continuity file.

## v1.4 validation and next step
- Go icon generation, Go tests, Windows cross-compilation and local asset HTTP smoke test passed on Linux on 16 Nov 2026.
- PNG and ICO output include transparent imagery; the ICO was structurally inspected with Pillow for all seven icon resolutions.
- GitHub Actions Windows build and native user-facing UI are separate validation gates. Live channel playback was not changed by this branding update.
- Further improvement: embed the Windows icon into PE resources if wanted, and track channel-by-channel playback evidence.

## v1.5 handoff

Before each Windows build run `go run ./tools/iconbuilder`, which produces `rsrc_windows_amd64.syso` (ignored by Git). Then run `go test ./...` and compile for Windows amd64. Keep the existing console launcher and default browser fallback. Windows EXE icon is verified by PE structure inspection, not native Windows Explorer UI. Do not claim a separate native window, as Edge/Chrome still provides the app window.

## v1.5.1 channel logo repair (16 November 2026)

- Six locally bundled fallback channel identifiers at `static/channel-logo-*.svg`, served by the existing allowlisted asset handler.
- Mapping and browser image-error recovery in `static/index.html`: ABC NEWS, SBS, SBS2, SBS World Movies, SBS Food and NITV. The original feed URL remains the first choice.
- Tests: local Go unit tests and race detector passed; Windows x64 cross-compiled; SVG syntax and JS syntax validated. Headless browser navigation blocked by the test environment; no native Windows verification yet. No change to HLS proxy or playback.
- Next: confirm logo fallback for each named channel on Windows, particularly when upstream artwork URLs fail.

## v1.6: Captions and AussieTV Text (16 November 2026)

- New caption track selector and C shortcut in `static/index.html`; HLS subtitle renditions selected through Hls.subtitleTrack and in-band CEA caption tracks through HTMLMediaElement.textTracks.
- Caption selector is disabled when the current IPTV source lacks accessible tracks. Preference for language saved in localStorage. No subtitles are generated; playback on broadcaster's official site remains fallback.
- New Teletext-inspired information overlay uses the EPG: 100 index, 200 now/next, 300 channels, 400 help. It is NOT decoding old Austext or DVB teletext packets. T opens/closes, Escape closes.
- Existing stream proxy and safety checks unchanged; only front-end plus version and docs changed.
- Local: Go tests and race detector pass; JS syntax check passes; browser UI smoke using mocked EPG and injected native caption track passes. Windows x64 cross-compilation passed. No native Windows or real broadcast caption testing yet.
- Next: test ABC News, ABC TV, SBS and commercial network caption selector on actual Windows streams; capture a per-channel caption capability table. Determine if DVB-T support is needed for any original broadcast-only captions.

## v1.7 remote-control integration (16 November 2026)

- Added `static/remote.js` (whole-interface D-pad/OK/Back, numeric tune, CH+/-, guide, filters and overlays), high-contrast focus and explicit player play/mute buttons.
- The browser UI remains a client of same-origin relative API URLs, facilitating future host/service separation. Current process still uses localhost-only binding, auto quit/heartbeat and desktop browser launch, **not safe to expose as-is to the LAN**.
- Integrated prior all-channel logo proxy work via `logo.go` and `logo_test.go`, including DNS restrictions and bounded image cache. No HLS proxy logic was changed.
- Automated checks: `go test -race ./...`, `go vet ./...`, `node --test tests/remote_keys.test.cjs`, `python tests/remote_browser_smoke.py` with mocked TV schedules, and Windows x64 Go build. All passed locally; Firefox/Safari, real streams and TV hardware not verified.
- Roadmap in `docs/REMOTE.md` and `docs/INTEGRATIONS.md`. For Jellyfin, validate standard M3U+XMLTV Live TV integration before building a new plugin.

## v1.7.1 remote-control navigation correction (17 November 2026)

- **Reason:** v1.7 used full-page nearest-element navigation when a guide move hit its boundary; focus could jump diagonally into the player or other shows. Region and caption `<select>` elements used native arrows even during ordinary D-pad navigation, inadvertently changing settings.
- **Implementation:** `static/remote.js` restricts horizontal moves to the same visible row and local control group. A guide boundary does not fall through to global navigation. Region and CC selects have explicit OK-to-edit, up/down preview, OK-to-commit, Back/Escape cancel. `static/index.html` adds edit-state highlighting and UI help.
- **Tests:** Go unit and race, Node remote unit, mocked Chromium at 1280x800 and 768x800, and Windows x64 cross-build passed locally. No real device or broadcaster playback test performed.
- **Future:** Docker/Jellyfin server-mode work remains separate and must retain these remote-navigation semantics.


## v1.8.0 (18 November 2026): deterministic remote exits and Jellyfin

- Retain desktop loopback UI and launcher. D-pad navigation now has explicit cross-section routes between sidebar, player controls, now-playing actions, quick channels, guide actions, and the programme guide. Each row still uses program-aware movement. Browser tests verify escape routes, dropdown edit/cancel, playback controls and overlays without Escape-only traps.
- A new **opt-in** Jellyfin feed listener serves only token-protected `/jellyfin/channels.m3u`, `/jellyfin/guide.xml`, `/jellyfin/stream/{id}` (redirect) and `/jellyfin/health`. No desktop APIs, HLS proxy, or unauthenticated UI are exposed on its port. `AUSSIETV_MODE=server` prevents desktop shutdown and heartbeat exit.
- M3U tvg-id and XMLTV channel IDs match. Channels without valid public upstream links are excluded. Guide timestamps are UTC. Stream URLs redirect to upstream HLS sources: playback still depends on external stream restrictions.
- New files: `jellyfin.go`, `jellyfin_test.go`, `Dockerfile`, `compose.yaml`, `docs/JELLYFIN.md`, `.github/workflows/docker-feed.yml`.
- Evidence: local `go test -race ./...`, `node --test tests/remote_keys.test.cjs`, `python tests/remote_browser_smoke.py` (mock feed), Windows x64 Go build, and local feed isolation/authentication HTTP checks passed. Real Android/TV remote and Jellyfin playback, and Docker on Unraid, remain unverified.
