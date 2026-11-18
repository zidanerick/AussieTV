# AussieTV

**Status: early public beta, `v0.1.0-beta.1` (Windows x64).** This is the **first beta**, not version 2.0. Previous `v1.*` and `v2.0.*` names were internal development milestones and prototypes, not production releases. See [Development history](docs/DEVELOPMENT-HISTORY.md).

AussieTV is a self-hosted desktop application for Australian live TV with an electronic programme guide, remote-control navigation, Teletext-style pages, and optional Jellyfin M3U/XMLTV feeds. It is not affiliated with broadcasters, TVmaze, Jellyfin, or the EPG provider.

## Windows downloads

The same beta version has **two different distribution files**, published as assets of a single GitHub prerelease:

| Edition | File | What you get |
| --- | --- | --- |
| **Portable** | `AussieTV-Portable-v0.1.0-beta.1-win-x64.zip` | Extract and launch `AussieTV.exe`; includes FFmpeg and no installer |
| **Windows Setup** | `AussieTV-Setup-v0.1.0-beta.1-win-x64.exe` | Standard setup wizard, install folder, Start Menu shortcuts, uninstaller, and optional verified FFmpeg download/update to a chosen folder |

Find both on [GitHub Releases](https://github.com/zidanerick/AussieTV/releases). The setup EXE is currently **unsigned**, so Windows SmartScreen may request confirmation. Do not download builds from untrusted mirrors.

## Start and exit

1. Install or extract the app and launch `AussieTV.exe`.
2. The TV opens in your **Windows default browser** and AussieTV remains in the notification area by the clock.
3. Closing the browser tab does **not** stop AussieTV. Right-click the tray icon to reopen the TV, open **Settings**, view logs or **Exit AussieTV**.
4. If the window doesn't open, use `http://127.0.0.1:45781` or launch `AussieTV-Diagnostics.exe`. The local TV interface is bound to loopback only.

The setup wizard defaults to a **per-user installation** (`%LOCALAPPDATA%\Programs\AussieTV`); no administrator permission is necessary until you explicitly enable LAN firewall rules from Settings. The installer offers an FFmpeg folder selection and download/update option. You can update FFmpeg again from the Start Menu via **Update FFmpeg**, without reinstalling AussieTV. Portable users can run `UPDATE-FFmpeg.cmd`.

## What works and what remains experimental

- Remote-friendly TV guide, regional channel lists, favourites, broadcaster logos, programme artwork and optional broadcaster-supplied captions.
- Teletext-inspired web pages, TTI export and an optional video information channel. This **is not** a real analogue PAL/VBI Teletext transmitter.
- Optional **Jellyfin M3U tuner** with XMLTV guide, programme artwork when accurately identified by TVmaze, and authenticated LAN feeds. Direct redirect is the **default** stream transport.
- Experimental HDHomeRun autodiscovery and **FFmpeg MPEG-TS stream copy** are **not considered stable**. Playback may be slow or fail, and some SBS channels have had missing-audio reports. Leave the MPEG-TS mode disabled unless testing.
- Real Windows tray, firewall elevation, remote-device input, multiple Jellyfin clients, and programme-artwork import have not passed comprehensive hardware tests. File issues with the beta version, channel name, player diagnostic summary and reproduction steps. **Never include feed API keys or signed HLS URLs in logs or screenshots.**

### Jellyfin

Enable the feed through the tray **Settings**. For Jellyfin on a different machine, enable LAN access and allow the narrowly scoped Windows Private firewall exception when requested. Use the URLs shown in Settings under Jellyfin Dashboard > Live TV > Add M3U Tuner and Add XMLTV. *Allow stream sharing* should remain enabled if disabling it causes playback errors. There is **no** need to forward router ports to the Internet. Refer to [Jellyfin setup and known limitations](docs/JELLYFIN.md).

### FFmpeg and legal notices

The portable package includes a separately licensed, checksum-verified FFmpeg binary from the distribution source linked by [FFmpeg.org](https://ffmpeg.org/download.html). The setup wizard downloads FFmpeg only when selected and verifies the provider-published SHA-256. Source/provenance and GPL notices accompany the download; FFmpeg is **not** an original AussieTV component. FFmpeg is only needed for experimental MPEG-TS or Teletext video output, not ordinary browser TV playback.

## Build and test

Go 1.22+ and Node 22+:

```sh
go test ./... -count=1
go test -race ./...
go vet ./...
node --test tests/remote_keys.test.cjs tests/settings_page.test.cjs
go run ./tools/iconbuilder
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='-s -w -H=windowsgui' -o AussieTV.exe .
```

A GitHub Actions Windows release workflow additionally compiles a portable ZIP and a real Inno Setup installer, including an optional FFmpeg setup page. Work on macOS, Linux and a full Docker-hosted interactive UI is **deferred until Windows beta issues are resolved**. The existing feed-only Docker proof of concept is experimental, not a supported release.

## Data providers and privacy

Channel/EPG index: [i.mjh.nz](https://i.mjh.nz/au/); series metadata/artwork: [TVmaze](https://www.tvmaze.com/api). Public broadcaster streams may change, require authentication or DRM, or be unavailable in some locations. AussieTV does not supply or bypass protected streams. Feed credentials are stored locally. Do not expose Jellyfin's feed service or experimental HDHomeRun endpoints to the public Internet.

Documentation: [release notes](RELEASE_NOTES.md), [Teletext](docs/TELETEXT.md), [remote controls](docs/REMOTE.md), [known issues](docs/BETA-KNOWN-ISSUES.md), [developer instructions](AGENTS.md).

License: [MIT](LICENSE), excluding third-party content and FFmpeg, which carry their own terms.
