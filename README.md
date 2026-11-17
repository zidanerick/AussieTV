# AussieTV

A lightweight, portable Windows 10/11 application for watching Australian free-to-air TV streams with a television-style electronic programme guide (EPG).

**Current source version: v1.7.1.** Windows x64, Go 1.22+, browser-based UI served locally by the executable. There is no installation or Python runtime.

## Run

Download a **Windows build artifact** from the latest successful [GitHub Actions build](https://github.com/zidanerick/AussieTV/actions/workflows/windows-build.yml) or build from source. Extract the ZIP and run `RUN-AussieTV.cmd` while keeping the console open.

If your browser does not open, visit `http://127.0.0.1:45781`. If the port is occupied, use the alternate address printed by the application. Logs are written to `%LOCALAPPDATA%\AussieTV\startup.log`.

## Features

- Full TV remote navigation using D-pad, OK, Back, channel up/down, numeric tuning and a remote-help overlay; use keyboard or remote bridge that emits browser key events.
- Player-controlled Play/Pause and Mute for browsers with inconsistent native media controls, plus high-contrast focus and resilient guide focus.
- Real upstream logo images for all curated channels through a guarded local image endpoint; artwork contrast adapts to light/dark marks.

- Australian channels from ABC, SBS, Seven, Nine and Network 10, with selected secondary channels
- Television grid guide, channel selection, programme descriptions and shortcuts
- Direct HLS playback where available, plus links to official network live players
- Default region Melbourne, with other major Australian capital-city regions available
- Original AussieTV icon, wordmark, favicon and browser install icon
- Embedded multi-resolution icon in the Windows EXE; dedicated Edge/Chrome app-style browser window when installed, with system-default browser fallback
- Network/channel logos from the guide provider
- Optional TV programme posters from TVmaze, with attribution and strict title matching
- Saved favourites with a dedicated filter, keyboard/remote-like EPG navigation, and guide focus preservation
- Visible playback timeout and single-recovery limit for HLS media errors
- Proxy hardening: stream URL signatures, DNS/IP checks, redirects and localhost Host/Origin validation
- Single localhost-bound server with embedded web UI, no external installation

## Build from source

```bash
go run ./tools/iconbuilder
go test ./...
go build -o AussieTV.exe .
```

On Linux, cross-compile for Windows x64 with:

```bash
go run ./tools/iconbuilder
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o AussieTV.exe .
```

Package `AussieTV.exe`, `RUN-AussieTV.cmd`, `README.txt` and `assets/AussieTV.ico` together. The ICO is also embedded in the executable through the generated `rsrc_windows_amd64.syso` resource. Icon generation uses Go standard libraries. Edge/Chrome app windows remain browser-owned; set `AUSSIE_TV_BROWSER_MODE=default` to open normal browser tabs. GitHub Actions automates the build and packaging on `main`.

## Version history

| Version | Historical snapshot | Highlights |
| --- | --- | --- |
| v1.0 | [versions/v1.0](https://github.com/zidanerick/AussieTV/tree/versions/v1.0) | First programme guide and channel player |
| v1.1 | [versions/v1.1](https://github.com/zidanerick/AussieTV/tree/versions/v1.1) | Visible launcher, startup logs, browser-opening fixes |
| v1.2 | [versions/v1.2](https://github.com/zidanerick/AussieTV/tree/versions/v1.2) | Real channel logos, programme posters, updated guide |
| v1.3 | [versions/v1.3](https://github.com/zidanerick/AussieTV/tree/versions/v1.3) | Favourites, remote-style guide navigation, playback timeout and stream proxy safeguards |
| v1.4 | [versions/v1.4](https://github.com/zidanerick/AussieTV/tree/versions/v1.4) | Original app logo, browser favicon, multi-size Windows shortcut icon and brand asset tooling |
| v1.5.1 | `main` | Embedded Windows EXE icon and app-style window launch with reliable fallback |

Full details: [CHANGELOG.md](CHANGELOG.md).

## Important limitations

AussieTV does not operate or own television streams. The channel index and guide come from [i.mjh.nz](https://i.mjh.nz/au/), which can change or be unavailable. Browser-based HLS playback also depends on an externally loaded HLS.js library. Some broadcasters restrict streams using authentication, DRM or region rules: use the supplied official broadcaster links in those cases. TVmaze art is optional and won't match all programmes. Melbourne listings may differ from Gippsland local feeds.

**Validation:** The v1.5.1 Go resource generator, tests and Windows cross-build passed on Linux; direct PE resource inspection confirmed seven icon sizes. Native Windows app-mode/taskbar behaviour remains unverified. Earlier validation:  Go tests, race detector, Windows cross-compilation, localhost HTTP checks, and headless Chromium UI interaction checks (with mocked channel data) passed in Linux. Actual free-to-air stream playback, Windows launcher behaviour for v1.3 and the Windows UI have not been independently verified on a Windows PC. Saved favourites use browser localStorage for this application's local address.

## Working on this project in a future chat

Begin with [docs/CONTINUITY.md](docs/CONTINUITY.md) and [AGENTS.md](AGENTS.md). Read the latest source and open issues from this repository instead of relying on earlier chat context. Create small, reviewable commits, test before publishing and update the continuity notes after each significant change.

## Attribution

The AussieTV logo is original artwork. Network marks and TV programme art belong to their respective owners. TVmaze API artwork is attributed in the application. Project license: see [LICENSE](LICENSE).

### Channel-logo reliability (v1.5.1)

ABC NEWS, SBS, SBS2, SBS World Movies, SBS Food and NITV use their feed-provided logos where available. If a remote image is blocked or unavailable, AussieTV uses a bundled, readable SVG identifier tile. These custom tiles are not copies of official artwork. No playback or EPG logic changed.

## Captions and AussieTV Text (v1.6)

When a live stream actually supplies captions, the player shows a **CC track selector**. This supports HLS WebVTT subtitle renditions and browser-exposed in-band caption tracks (such as CEA-608/708 where HLS.js and the browser decode them). Use **C** to toggle a detected track; if no tracks exist, the selector stays unavailable. The preferred subtitle language is saved locally. **No captions are generated or retrieved from third-party subtitle scraping.** The broadcaster's official player may have captions even when AussieTV's third-party stream does not.

Use **TEXT** or **T** to open the Teletext-inspired local information overlay: page **100** index, **200** current programme and next, **300** channel directory, and **400** help. This displays existing EPG data; it is not a decoder for original broadcast Teletext/Austext (discontinued in 2009).

UI interaction tests were performed in headless Chromium with a synthetic native caption track. Live Australian caption tracks and a native Windows browser window remain unverified. The HLS proxy and channel ingestion were not modified in this version.

## Remote controls and future integrations (v1.7)

Read [TV remote and browser compatibility](docs/REMOTE.md) and [Docker/Jellyfin integration roadmap](docs/INTEGRATIONS.md). Only Chromium interaction with mock guide data was tested; actual smart-TV remote key support and other browser playback modes are not yet verified. Current app remains localhost-only and does not expose a Jellyfin plugin or secure Docker-hosted network API.

### v1.7.1 remote navigation repair

Left/right now stays inside the same visual row and control group, and guide edge navigation no longer jumps to unrelated player controls. Region and caption dropdowns require **OK to edit**, **Up/Down to choose**, **OK to apply**, and **Back/Escape to cancel**, without changing the region merely from focus navigation. See [docs/REMOTE.md](docs/REMOTE.md). The portable Windows launcher and playback code are otherwise unchanged.
