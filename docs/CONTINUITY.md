# AussieTV project continuity / future-chat handoff

**Last documented:** 15 November 2026  
**Current code:** v1.3 on `main`  
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

## Exact historical versions

- v1.0: commit `681b9aa580f0ac5f2ff961f2804c654866631391` ; branch `versions/v1.0`
- v1.1: commit `c78cc5a2a3674bedd0277dadbd9771b4fe5ad8e3` ; branch `versions/v1.1`
- v1.2: commit `72bc93cb9d55aa4b2a7db9bc603df52c9cb3e1c9` ; branch `versions/v1.2`
- v1.3: branch `versions/v1.3` stores the latest source checkpoint; see GitHub commit history for the exact version commit.
- Later docs/automation commits on main are not themselves new user-facing product versions.

## Build / test / launch

```bash
go test ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o AussieTV.exe .
```

Bundle `AussieTV.exe`, `RUN-AussieTV.cmd`, `README.txt`. GitHub Actions workflow `.github/workflows/windows-build.yml` builds and uploads this ZIP automatically. Local address defaults to http://127.0.0.1:45781.

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
