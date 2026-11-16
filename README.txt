AussieTV v1.7 - Live Australian Television for Windows
======================================================

WHAT'S NEW IN V1.2
- Real channel logos from the Australian EPG/channel index appear in the
  network filter, five quick-tune buttons, TV guide, and Now Playing panel.
- Optional show posters appear for confidently matched TV series, with
  attribution to TVmaze. The idle player background uses that artwork too.
- Channel/brand-coloured fallback tiles are shown if logos or images fail.
- Existing channel tuning, guide, playback controls and v1.1 launcher remain.

RUNNING ON WINDOWS 10/11
1. Extract the complete ZIP to a folder.
2. Double-click RUN-AussieTV.cmd (recommended), or AussieTV.exe.
3. An Edge/Chrome app-style window opens when available; otherwise
   Windows uses your default browser. If no window opens, visit:
   http://127.0.0.1:45781
4. Keep the command window running while watching TV.
5. Select a channel in the guide or use the five large quick-tune buttons.

The app runs a local HTTP server bound to 127.0.0.1 only. If port 45781 is
occupied, the app selects a free port and prints the actual URL.
The startup log is at %LOCALAPPDATA%\AussieTV\startup.log.

ARTWORK AND LOGOS
- Channel logos come directly from the live channel index. They require a
  working internet connection and may be hosted by the broadcasters or by
  the channel index provider. Images may change or become unavailable.
- Program images are optional. AussieTV queries TVmaze for the currently
  highlighted show's title, strips common season/episode suffixes, checks
  for an exact title match, and uses the resulting poster if available.
  Many shows, news bulletins and films will NOT have matched artwork.
- Programme images and network marks belong to their respective owners.
  TVmaze API metadata is provided under CC BY-SA; credit is in the app.
- TVmaze lookups only run for the selected programme and are cached for
  the current session. No TVmaze account or API key is required.

LIMITATIONS
- Australian free-to-air streams are provided by third parties and can
  change or be blocked by network/region/provider requirements.
- Some protected streams require the broadcaster's official player.
- Video playback depends on the HLS.js library, which is loaded online.
- v1.3 tested by Go unit tests and local HTTP server check. The executable
  was cross-compiled for Windows x64, not exercised directly on Windows.
- Your television and guide listings are supplied by https://i.mjh.nz/au/
  under the operator's availability and usage limitations.

SOURCES & ATTRIBUTION
- IPTV and channel logos: https://i.mjh.nz/au/
- Programme art and metadata: https://www.tvmaze.com/api
- Broadcaster live sites are linked from each selected channel.

KEYS
Up / Down: Switch channels
Enter: Play selected channel
F: Full-screen player

No installer, admin rights, subscription, or bundled Python runtime required.


NEW IN v1.3
- Favourite channels are saved in your browser (select a channel then click Favourite).
- Use the Favourites sidebar filter; it follows you across city regions by network/name.
- Navigate guide cells with arrow keys and press Enter to tune; interactive buttons keep native keyboard behaviour.
- Playback now reports a timeout instead of connecting indefinitely.
- Stream proxy restricted to channel stream and signed resources from channel HLS manifests.

The TV guide is community supplied, and actual Windows broadcaster streaming is not verified.

NEW IN v1.4
- Original AussieTV play-screen icon and SVG logo, used throughout the interface.
- Browser favicon, 192px install icon, and site manifest; works offline after launch.
- AussieTV.ico included in the Windows ZIP for shortcuts, independent of
  broadcaster channel logos.
- Logo artwork and icon-generation source kept in the GitHub repository.


NEW IN v1.5
- Windows Explorer and shortcuts now show AussieTV's icon directly from
  AussieTV.exe, without needing a separate custom icon file.
- Prefer a browser app-style window on Microsoft Edge or Google Chrome,
  with an AussieTV page title and favicon, instead of a regular browser tab.
- If you prefer your default browser's ordinary tabs, set
  AUSSIE_TV_BROWSER_MODE=default before running RUN-AussieTV.cmd.
- The v1.1 default-browser fallback and console diagnostics remain intact.
- A browser app window is still hosted by the browser; its taskbar identity
  may be controlled by Edge/Chrome and Windows. It is not a native WebView2
  executable.

NEW IN v1.5.1
- Local artwork fallbacks for ABC NEWS, SBS, SBS2, SBS World Movies,
  SBS Food and NITV when the remote channel logos fail or are missing.
- Playback code and channel stream URLs are unchanged.

NEW IN V1.6
- HLS subtitle and closed-caption selector (only tracks available in the live stream).
- Press C to toggle available captions. Preferences are saved locally.
- Press T or click TEXT for a Teletext-style local program information display.
  This uses the existing EPG and is NOT original broadcast Teletext.
- No artificial subtitles are generated. If none are present, CC is unavailable.

AUSSIETV v1.7: REMOTE / CROSS-BROWSER INPUT
- Arrow/D-pad moves focus across sidebar, quick channels, guide, player controls, captions, Text pages and settings.
- OK/Enter activates the highlighted item; Back/Escape closes overlays or returns to the guide.
- CH+/CH- or PageUp/PageDown change channels. Numeric remote keys enter a channel number.
- H opens the remote help screen, M opens network filters, G returns to the guide.
- Hardware special keys must be delivered to the browser by the TV OS or remote adapter.
- HLS playback is provided through Hls.js or native HLS where the browser supports it.
