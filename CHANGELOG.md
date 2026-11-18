# Changelog

## v0.1.0-beta.1, 18 November 2026

First publicly labelled beta. This version **supersedes the development-only
version strings 1.0 through 2.0.9**, without implying that the application has
previously been released as a stable v1 or v2.

- Consolidates the working Windows desktop/tray, TV guide and regional playback code.
- Adds a real Windows setup wizard alongside a self-contained portable edition.
- Setup includes an optional, checksum-verified FFmpeg downloader with a custom
  target directory; Start Menu offers future FFmpeg updates.
- Portable package includes a verified FFmpeg binary and update shortcut.
- Installed app can find FFmpeg in a custom directory recorded by the setup wizard.
- Retains authenticated Jellyfin M3U/XMLTV and experimental MPEG-TS, HDHomeRun
  discovery, Teletext and artwork prototypes. **Those integrations remain beta.**
- Adds release gating, troubleshooting notes and a known-issues list.

### Known beta limitations

- The FFmpeg MPEG-TS relay is not validated for all channels, and SBS has had
  picture-without-audio reports.
- Jellyfin HDHomeRun autodiscovery has not reliably worked across LAN/Docker.
- Jellyfin Web may need several seconds to begin displaying a stream.
- XMLTV programme artwork may not import as Jellyfin Primary images even if
  the XMLTV icon URL is valid.
- Native Windows installer, tray, and UAC interaction need hands-on testing.
- Current provider data, network rules and DRM can prevent particular streams.

Older development milestones are preserved in [Development history](docs/DEVELOPMENT-HISTORY.md)
and the repository's historical branches. They are **not** semver production releases.
