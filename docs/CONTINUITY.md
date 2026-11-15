# AussieTV project continuity / future-chat handoff

**Last documented:** 15 November 2026  
**Current code:** v1.2 on `main`  
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

## Exact historical versions

- v1.0: commit `681b9aa580f0ac5f2ff961f2804c654866631391` ; branch `versions/v1.0`
- v1.1: commit `c78cc5a2a3674bedd0277dadbd9771b4fe5ad8e3` ; branch `versions/v1.1`
- v1.2: commit `72bc93cb9d55aa4b2a7db9bc603df52c9cb3e1c9` ; branch `versions/v1.2`
- Later docs/automation commits on main are not themselves new user-facing product versions.

## Build / test / launch

```bash
go test ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o AussieTV.exe .
```

Bundle `AussieTV.exe`, `RUN-AussieTV.cmd`, `README.txt`. GitHub Actions workflow `.github/workflows/windows-build.yml` builds and uploads this ZIP automatically. Local address defaults to http://127.0.0.1:45781.

## Verification state

- v1.2 unit tests passed in Linux on 15 Nov 2026.
- Windows x64 executable cross-compilation passed, with PE console subsystem.
- Source and local HTTP endpoints were checked in earlier work.
- User previously confirmed that the v1.1 Windows app launched successfully.
- **Not proven**: universal broadcaster playback, native Windows GUI automation, sustained stream reliability, artwork matching for every programme, or Australian geolocation behaviour. Do not claim those tests passed.

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
- Improve EPG navigation with keyboard/remote controls, artwork caching and now/next programmes.
- Investigate official/legal streaming integrations and proper regional selection.
- Add release tags and release attachments after a real Windows smoke test.

## How to continue in a new chat

Say: "Continue development of https://github.com/zidanerick/AussieTV. Read README.md, AGENTS.md, docs/CONTINUITY.md and current GitHub issues. Inspect the actual main branch before changing code. Work from the current version, preserve historical branches, run tests, and update the handoff document."

If a change is made, update this document with: exact commit SHA, affected files, tests executed, Windows device evidence (or an explicit gap), regressions, and the next task.
