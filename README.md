# AussieTV

A lightweight, portable Windows 10/11 application for watching Australian free-to-air TV streams with a television-style electronic programme guide (EPG).

**Current source version: v1.3.** Windows x64, Go 1.22+, browser-based UI served locally by the executable. There is no installation or Python runtime.

## Run

Download a **Windows build artifact** from the latest successful [GitHub Actions build](https://github.com/zidanerick/AussieTV/actions/workflows/windows-build.yml) or build from source. Extract the ZIP and run `RUN-AussieTV.cmd` while keeping the console open.

If your browser does not open, visit `http://127.0.0.1:45781`. If the port is occupied, use the alternate address printed by the application. Logs are written to `%LOCALAPPDATA%\AussieTV\startup.log`.

## Features

- Australian channels from ABC, SBS, Seven, Nine and Network 10, with selected secondary channels
- Television grid guide, channel selection, programme descriptions and shortcuts
- Direct HLS playback where available, plus links to official network live players
- Default region Melbourne, with other major Australian capital-city regions available
- Network/channel logos from the guide provider
- Optional TV programme posters from TVmaze, with attribution and strict title matching
- Saved favourites with a dedicated filter, keyboard/remote-like EPG navigation, and guide focus preservation
- Visible playback timeout and single-recovery limit for HLS media errors
- Proxy hardening: stream URL signatures, DNS/IP checks, redirects and localhost Host/Origin validation
- Single localhost-bound server with embedded web UI, no external installation

## Build from source

```bash
go test ./...
go build -o AussieTV.exe .
```

On Linux, cross-compile for Windows x64 with:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o AussieTV.exe .
```

Package `AussieTV.exe`, `RUN-AussieTV.cmd`, and `README.txt` together. GitHub Actions automates the build and packaging on `main`.

## Version history

| Version | Historical snapshot | Highlights |
| --- | --- | --- |
| v1.0 | [versions/v1.0](https://github.com/zidanerick/AussieTV/tree/versions/v1.0) | First programme guide and channel player |
| v1.1 | [versions/v1.1](https://github.com/zidanerick/AussieTV/tree/versions/v1.1) | Visible launcher, startup logs, browser-opening fixes |
| v1.2 | [versions/v1.2](https://github.com/zidanerick/AussieTV/tree/versions/v1.2) | Real channel logos, programme posters, updated guide |
| v1.3 | [versions/v1.3](https://github.com/zidanerick/AussieTV/tree/versions/v1.3) | Favourites, remote-style guide navigation, playback timeout and stream proxy safeguards |

Full details: [CHANGELOG.md](CHANGELOG.md).

## Important limitations

AussieTV does not operate or own television streams. The channel index and guide come from [i.mjh.nz](https://i.mjh.nz/au/), which can change or be unavailable. Browser-based HLS playback also depends on an externally loaded HLS.js library. Some broadcasters restrict streams using authentication, DRM or region rules: use the supplied official broadcaster links in those cases. TVmaze art is optional and won't match all programmes. Melbourne listings may differ from Gippsland local feeds.

**Validation:** Go tests, race detector, Windows cross-compilation, localhost HTTP checks, and headless Chromium UI interaction checks (with mocked channel data) passed in Linux. Actual free-to-air stream playback, Windows launcher behaviour for v1.3 and the Windows UI have not been independently verified on a Windows PC. Saved favourites use browser localStorage for this application's local address.

## Working on this project in a future chat

Begin with [docs/CONTINUITY.md](docs/CONTINUITY.md) and [AGENTS.md](AGENTS.md). Read the latest source and open issues from this repository instead of relying on earlier chat context. Create small, reviewable commits, test before publishing and update the continuity notes after each significant change.

## Attribution

Network marks and TV programme art belong to their respective owners. TVmaze API artwork is attributed in the application. Project license: see [LICENSE](LICENSE).
