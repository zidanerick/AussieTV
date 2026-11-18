# AussieTV

Portable Australian free-to-air television web interface with TV-style guide. The desktop build runs on Windows 10/11 x64 via a lightweight Go localhost server and a modern web browser.

**Version: v1.7.1** (remote navigation, browser-facing controls, all-channel logo proxy and contrast improvements).

## Windows

Extract the Windows ZIP and run `RUN-AussieTV.cmd`. If the app window fails to appear, open the local address printed in the console. Console/log output is intentionally visible. Video streams may require broadcaster authentication, be geo-restricted or use DRM unavailable through a generic HLS player; the official players remain linked.

## Features

- v1.8: D-pad navigation can exit every guide/control-group edge; no Escape-only dead ends; retains explicit OK-to-edit region/subtitle selectors.
- v1.8: Opt-in authenticated Jellyfin M3U/XMLTV feed service on a separate LAN listener, with Docker/Unraid example (see [Jellyfin guide](docs/JELLYFIN.md)).

- ABC, SBS, Seven, Nine and 10 selected main and secondary channels.
- Electronic programme guide, now/next information, favourites, regional selection.
- Real provider logos where available, programme artwork, on-screen Text information panel, optional caption tracks.
- Remote and keyboard operation across the interface using D-pad, OK, Back, channel buttons and numeric keys, plus high-contrast focus. Left/right stays in the same visual row. Region and caption selectors require OK to edit, up/down to choose and OK to apply, with Back to cancel.
- HLS.js-based playback in browsers that support MSE and native HLS fallback in compatible browsers.
- Original brand icon embedded in Windows executable.

## Developer setup

```sh
go run ./tools/iconbuilder
go test ./... -count=1
go test -race ./... -count=1
node --test tests/remote_keys.test.cjs
# optional: Python Playwright and Chromium for mocked browser interactions
python tests/remote_browser_smoke.py
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o AussieTV.exe .
```

The Windows build is normally produced by GitHub Actions. `RUN-AussieTV.cmd`, `README.txt` and `AussieTV.ico` accompany the `.exe`.

See [remote controls](docs/REMOTE.md), [Docker/Jellyfin roadmap](docs/INTEGRATIONS.md), [project continuity](docs/CONTINUITY.md), [changelog](CHANGELOG.md) and [future-chat instructions](AGENTS.md).

**Current limitations:** Native Windows remote input, Firefox/Safari remote behaviour and real Australian stream playback were not all verified in this environment. The current application is localhost-only; it is not yet a multi-user Docker server or Jellyfin plugin. Australian channel data comes from a third party and may change. Copyright in broadcaster logos and television artwork remains with the respective owners.

## v1.8 limitations

The network listener is **opt-in and feed-only**; it does not expose the TV app UI over LAN or implement HDHomeRun emulation. Jellyfin is expected to follow upstream HLS redirects. Stream playback from Jellyfin has not been tested with live Australian broadcasters.
