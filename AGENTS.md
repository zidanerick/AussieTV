# AussieTV developer instructions

Current beta marker: v0.1.0-beta.1. Docker is the sole supported distribution
and server runtime. Do not rename the version without updating RELEASE_VERSION
and the Go constant. Do not publish standalone Windows/Linux releases.

- Preserve the Go server, embedded browser UI and authenticated web/feed
  listeners (45781/45782). No desktop startup, settings, tray, firewall or
  unauthenticated fallback listener. Feed-only mode has no web listener.
- Linux amd64/arm64 container, UID/GID 10001, /data persistent, FFmpeg included.
  Use bridge networking, dropped capabilities, read-only root and explicit
  secrets. Existing feed URLs, channel IDs, volume and tag compatibility matter.
- Keep redirect default; relay/MPEG-TS/Fast Tune remain opt-in. Preserve
  cancellation, resource bounds, HLS key validation and public-IP SSRF rules.
- No device/geographic restriction bypass. Do not log credentials or signed
  media URLs. Do not manufacture missing EPG or artwork.
- CI: .github/workflows/experimental-builds.yml (historical filename) runs
  Go tests, focused race tests, go vet, Node tests, Unraid template contract,
  both image builds, authenticated web/feed-only and combined Compose tests,
  FFmpeg and SIGTERM smoke tests. Publish only exact tested images on trusted
  pushes. The preview branch never changes the main beta tag.
- Development: docker build -t aussietv:local .; see docs/DOCKER.md.
  Source tests: go run ./tools/iconbuilder; go test ./... -count=1; go vet ./...;
  node --test tests/*.test.cjs; python3 -m unittest discover -s tests -p 'test_unraid_template.py'.
- Legacy native helper code/tests may remain for compatibility and gradual
  cleanup; do not reactivate native packaging. Previous releases and Git
  history are preserved. Do not delete historical release assets or tags.
- Distinguish synthetic tests, image smoke checks and live client evidence.
  Keep README, docs/DOCKER.md, docs/JELLYFIN.md and known issues accurate.
