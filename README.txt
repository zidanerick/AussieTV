AussieTV v1.5 - Live Australian Television for Windows
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
3. An Edge/Chrome app-style window opens when available. Otherwise the
   normal default browser opens. If no window appears, visit:
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
- Original AussieTV play-screen logo and favicon throughout the application.
- Browser favicon, install icon and local site manifest.
- Multi-size AussieTV.ico included in the Windows ZIP for shortcut icons.
- Icon generator source is kept with the repository and runs during CI.

NEW IN v1.5
- AussieTV.exe embeds its Windows Explorer/shortcut icon directly.
- Microsoft Edge or Google Chrome opens a separate app-style window when
  installed; otherwise the ordinary default browser is used.
- Set AUSSIE_TV_BROWSER_MODE=default before launching for normal browser
  tabs instead. The console/log and URL fallback remain available.
- The app window remains browser-owned, not an independent native WebView.
