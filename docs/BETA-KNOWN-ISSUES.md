# Beta known issues and release gate

This is an **early Windows beta**, not a production release or a claim of
full Jellyfin compatibility. The following limitations are deliberately open:

| Area | Known behaviour | Action |
| --- | --- | --- |
| Jellyfin MPEG-TS | Some SBS HLS renditions produce no usable audio even when video is present | Keep Direct redirect as the default; collect anonymised ffmpeg/TS metadata, verify audio sync before graduation |
| Jellyfin startup | ~6-11 s reported for some channels in Jellyfin Web, despite fast AussieTV redirects | Compare click-to-first-picture in the same client; no performance claim yet |
| HDHomeRun | LAN auto-detection unreliable, especially across Docker bridge networks | Keep manual authenticated M3U/XMLTV supported; diagnose broadcast reachability separately |
| XMLTV images | XMLTV programme icons may not become Jellyfin Primary artwork | Verify exported URLs first; track Jellyfin import separately |
| Captions and Teletext | Network-dependent captions; no real analogue VBI and no live supertext | Not a hardware Teletext product |
| Windows desktop | Tray and default browser work in automated tests but require more native hardware checks | Verify on Windows 10 and 11 |
| Windows setup | Inno Setup EXE is unsigned | SmartScreen warning possible; verify SHA-256 from GitHub Release |
| Provider streams | Authentication, DRM, region restrictions and EPG provider changes may break channels | Do not bypass DRM; use official broadcaster playback when necessary |

**Before a stable release:** test installer install/update/uninstall; FFmpeg
install and update from a nondefault folder; native tray exit; Private-network
firewall rules and disabled state; EPG artwork, subtitles, TV remote navigation;
Jellyfin client audio/video stability; and packaging on real Windows PCs.

Do not put signed stream URLs, feed keys, private IPs or personal logs in public
issues. Future Linux/macOS/full Docker UI work stays out of the Windows beta gate.
