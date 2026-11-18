# Development continuity: first beta

**Canonical version:** `v0.1.0-beta.1` (18 November 2026). This is the first
Windows early beta; prior `v1.x` and `v2.0.x` names described development
experiments, not stable version milestones. See [pre-beta development history](DEVELOPMENT-HISTORY.md)
for the former historical notes.

## Source and release

- Primary source: `main` after beta PR #16 merges.
- The beta build workflow runs Go, Node and compiler tests on PRs; on the merged
  `main` commit it creates a GitHub **prerelease** with two distinct assets.
- Windows **portable ZIP** includes `AussieTV.exe`, diagnostic EXE, built-in
  icons, FFmpeg, licences, and configuration. No installation.
- Windows **Setup EXE** uses an Inno wizard: application install path,
  Start Menu/uninstall, optional checksum-verified FFmpeg download and custom
  FFmpeg directory, plus separate manual FFmpeg updater.
- Future Linux/macOS and fully interactive Docker modes are deferred.

## Architecture and security

- Go app embeds the browser UI, serves it on loopback (`127.0.0.1:45781`).
- Settings/tray never change that loopback bind to LAN; feed service separately
  binds TCP 45782 when explicitly enabled.
- Jellyfin M3U/XMLTV paths require a generated access key. Signed HLS rewrite
  endpoints validate external locations and DNS/IPs to avoid LAN SSRF.
- Optional HDHomeRun discovery (UDP 65001 and TCP 80) remains experimental,
  restricted to the private LAN and disabled by default.
- FFmpeg MPEG-TS relay remains opt-in and can fail audio on SBS. Direct
  redirect is default. Stop on client disconnect; do not log tokens.
- Local settings and secrets live in the Windows user profile. The optional
  FFmpeg installer writes a path hint by the app for custom install folders.

## Acceptance evidence

Linux development tests: `go test ./... -count=1`, `go test -race ./...`,
`go vet ./...`, Node navigation/settings/release tests, and Windows x64 cross-
compile passed. GitHub Actions Windows installer compilation is a separate
release gate, not equivalent to native Windows installation or live Jellyfin
playback on a device.

## Active priorities before stable

1. Resolve SBS audio in experimental MPEG-TS. Keep the default M3U/direct
   transport unchanged.
2. Test HDHomeRun discovery on the same broadcast domain as Jellyfin, including
   host-networking Docker scenarios.
3. Measure Jellyfin Web startup latency and XMLTV artwork imports on real clients.
4. Exercise Windows installation, FFmpeg update path choices, uninstallation,
   system tray, firewall prompts, remote-control focus and default browser.
5. Reconsider Linux/macOS/Docker UI only after Windows beta reliability improves.

See [Known issues](BETA-KNOWN-ISSUES.md) and [developer instructions](../AGENTS.md).
