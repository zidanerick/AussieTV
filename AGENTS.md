# AussieTV developer handoff

**Current branch:** `main` targets **`v0.1.0-beta.1`**, the first early
Windows-only beta. Previous `v1.*`/`v2.0.*` strings were pre-beta build labels.
Do not revive them as release versions. Keep code and documentation versioned
from `RELEASE_VERSION` and the `main.go` constant until these can be unified.

## Scope

- Supported beta target: Windows 10/11 x64, portable and Inno Setup installer.
- Normal TV binds localhost `127.0.0.1:45781` (or a free loopback port).
- Optional Jellyfin authenticated feed is TCP 45782, LAN gated via Settings;
  HDHomeRun discovery is optional and experimental (UDP 65001 / TCP 80).
- Direct Jellyfin HLS redirect is the default. FFmpeg MPEG-TS, Teletext video,
  and HDHomeRun auto-discovery are experimental and can fail on real sources.
- macOS, Linux and full Docker-served browser UI are **deferred**.

## Build / packaging

- `go test ./... -count=1 && go vet ./...`
- `go test -race ./...`
- `node --test tests/remote_keys.test.cjs tests/settings_page.test.cjs`
- `go run ./tools/iconbuilder` then cross-compile Windows GUI and diagnostic.
- `.github/workflows/beta-release.yml` builds two assets and only creates a
  prerelease after all Windows tests, FFmpeg checksum validation, and Inno
  Setup compilation succeed. Do not overwrite an existing beta tag.
- `installer/AussieTV.iss` includes optional FFmpeg download with a user-
  selected folder and Start Menu update action. Script:
  `tools/ffmpeg/fetch-windows.ps1` validates the SHA-256 and executable.

## Review policy

Preserve Jellyfin feed authentication, SSRF/redirect rules, loopback host and
origin checks, firewall Private-only rules, and FFmpeg cancellation. Never log
signed broadcaster URLs or feed tokens. Keep new transport changes opt-in.

Use `docs/BETA-KNOWN-ISSUES.md` for unresolved concerns, and
`docs/DEVELOPMENT-HISTORY.md` for old snapshots. The README should describe
only active beta functionality. **Do not delete historical branches or releases
without an explicit retention decision.**
