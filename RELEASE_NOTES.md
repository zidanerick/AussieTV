# AussieTV v0.1.0-beta.1

**First early beta for Windows 10/11 x64. Not a stable release.**

Two downloads are attached to this one beta release:

- **AussieTV-Portable-v0.1.0-beta.1-win-x64.zip:** extract and launch, including FFmpeg and licensing information.
- **AussieTV-Setup-v0.1.0-beta.1-win-x64.exe:** Windows installation/uninstallation wizard with an optional checksum-verified FFmpeg download and a selectable FFmpeg install directory. Launch and FFmpeg update shortcuts are in the Start Menu.

AussieTV opens in your default browser and runs in the Windows system tray.
It offers Australian EPG/live streams, a TV remote-style interface, TVmaze
artwork, a Teletext prototype, and an authenticated Jellyfin M3U/XMLTV feed.

**Known problems:** Jellyfin auto HDHomeRun detection is experimental, and
experimental MPEG-TS playback can lack audio on SBS. Real-world Jellyfin stream
startup can take several seconds. The Windows installer is unsigned and may
trigger SmartScreen. See `docs/BETA-KNOWN-ISSUES.md` before reporting a bug.

The previous 1.x / 2.0.x preview numbers were pre-beta development labels,
not stable production releases. Future platforms (Linux, macOS, interactive
Docker UI) are deliberately deferred.
