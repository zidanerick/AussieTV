AUSSIE TV 0.1.0 BETA 1 / WINDOWS X64
===================================
This is an early beta with known bugs, not a stable production release.

PORTABLE: Extract the entire ZIP, then launch AussieTV.exe. FFmpeg is bundled.
INSTALLER: Run the Setup EXE and choose an installation folder. The wizard
can optionally download or update FFmpeg into a folder you select. The
installer verifies the published SHA-256 checksum. Internet access required.

The TV opens in your Windows DEFAULT browser. AussieTV stays running in the
notification area after you close the browser. Right-click its icon by the
clock to open Settings, view logs, or EXIT.

Diagnostics: launch AussieTV-Diagnostics.exe; local TV is normally at
http://127.0.0.1:45781. A console error is not a reason to open firewall
ports for the TV UI, which must remain localhost-only.

FFMPEG: Ordinary live TV works without it. For experimental Jellyfin MPEG-TS
or Teletext video mode, use Start Menu > AussieTV > Update FFmpeg, or run
UPDATE-FFmpeg.cmd in portable mode. Download location can be customised in
the installer. Updates verify upstream checksums before replacing FFmpeg.

JELLYFIN: Configure authenticated M3U/XMLTV through Settings. Direct redirect
is the default stream mode. Experimental MPEG-TS is known to have audio and
compatibility bugs, including SBS reports. Enabling LAN access prompts for
Private-network firewall rules; NO router port forwarding is needed.

WARNINGS: HDHomeRun discovery is experimental, and not all broadcaster streams,
EPG posters, captions, or Teletext pages are available. Do not publish the
feed key or expose the feed publicly. Windows SmartScreen may warn because this
beta setup executable is not signed.

Project: https://github.com/zidanerick/AussieTV
Issues: https://github.com/zidanerick/AussieTV/issues
License: LICENSE (MIT for AussieTV code; FFmpeg and media separately licensed).
