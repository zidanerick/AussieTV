# Changelog

## v1.8.0 (18 November 2026)
- Use a deterministic D-pad navigation graph to exit guide/control-group boundaries, preserving logical EPG rows and explicit select editing.
- Add opt-in Jellyfin M3U and XMLTV feeds with matching stable channel IDs, via a separate authenticated read-only LAN port and current-source stream redirects.
- Add a feed-only, non-root Dockerfile/Compose example, Unraid setup guidance, server-only mode, and separate Docker CI smoke test.
- Tests: Go race/unit tests, Node navigation tests, mocked Chromium remote browser test, local feed authentication and isolation checks, Windows cross-build. Real device playback is not verified.


## v1.7.1 (17 November 2026)
- Keep D-pad horizontal navigation within a single visible row and control group; programme-guide row boundaries no longer fall through to page-wide spatial navigation.
- Require explicit OK-to-edit for Region/CC selectors; Up/Down changes a draft choice, OK commits, Back/Escape cancels without dispatching change.
- Fix select Back recovery and retain mouse/touch controls.
- Add Node unit and Chromium mocked UI regression checks at desktop and narrower layouts; Go race checks and Windows x64 build passed locally. Physical remotes, Firefox/Safari and live streams remain unverified.


## v1.7.0 (16 November 2026)
- Add remote.js whole-page focus navigation: D-pad/arrow, OK, Back, channel step and numeric entry, with modal focus recovery and a help screen.
- Add explicit Play/Pause and Mute controls, CSS focus treatment, quick-channel focus preservation and multi-browser-friendly fullscreen fallback.
- Incorporate verified original channel logos for all curated channels via the image proxy, safe DNS/IP checks and contrast-aware logo backgrounds; retain safe fallback tiles.
- Add Node keyboard mapping/spatial tests, a mocked headless Chromium browser UI test and future Docker/Jellyfin architectural notes.
- Local Go test/race detector, Node tests, Chromium mocked UI, Go vet and Windows x64 cross-compilation passed. Live TV playback and real TV remote hardware not yet verified.


## v1.6 (16 November 2026)
- Add CC/subtitle track selector for streams that expose HLS WebVTT renditions or browser-native caption text tracks.
- Provide C shortcut, caption-language preference and explicit No captions state; avoid pretending tracks exist.
- Add Teletext-style AussieTV Text panel with 100 index, 200 now/next, 300 channels and 400 help pages using the existing EPG, not defunct Austext broadcast data.
- Preserve playback and network proxy behaviour. Add no external caption source.
- Local Go unit tests/race checks, JS syntax checks, Windows x64 cross-compilation, and headless Chromium UI checks passed with mocked programme data and captions. Live stream captions and native Windows window still need real-device verification.


## v1.5.1 (16 November 2026)
- Add bundled SVG fallback tiles for ABC NEWS, SBS, SBS2, SBS World Movies, SBS Food and NITV.
- When a feed-provided logo is missing or fails to load, use a local channel-specific identifier tile rather than a generic network block.
- Add allowlisted routes for SVGs and server tests; playback unchanged.
- Verified local Go tests, race detector, Windows x64 cross-compilation, SVG parsing and JS syntax. Native Windows display not yet verified.


## v1.5 (16 November 2026)
- Embed a multi-size native Windows executable icon (16 to 256 px) with a reproducible Go-generated PE resource, without external tools or runtime dependencies.
- Prefer app-style Edge/Chrome windows and retain the proven default-browser fallback. Add an environment switch for normal browser tabs.
- Clean up the browser window title and guide version badge; include the ICO at ZIP root.
- Add COFF resource structure and browser-path tests. Local Go tests, Windows cross-build and PE icon tree inspection passed; actual native Windows app window and taskbar behaviour remain unverified.


Historical source versions are preserved as separate commits and branches. These commits were imported from the original ChatGPT-produced source archives on 15 November 2026. Their history reflects version order, not the original local file creation times.

## v1.4 (16 November 2026)
- Introduce an original AussieTV brandmark and wordmark, separate from broadcasters' copyrighted channel marks.
- Replace the placeholder sidebar glyph and idle-player symbol with the new project icon.
- Serve SVG favicon, PNG favicons, multi-size ICO and a site manifest through allow-listed local asset routes.
- Add a dependency-free Go icon builder and package `AussieTV.ico` for Windows shortcuts.
- Add route tests, generate icons in GitHub Actions and document the branding source.
- This version changes branding only; real-stream and native Windows validation remain separate.

## v1.3 (15 November 2026)
- Save favourite channels locally in the browser, with Favourites filtering and visual markers in the EPG.
- Add EPG directional navigation, preserve keyboard focus on guide redraw, and restore native Enter handling for active controls.
- Replace repeated minute-long guide redraws with one redraw at the minute boundary.
- Add a 35-second live playback startup timeout and limit media-error recovery attempts.
- Harden localhost stream proxy: require indexed channel roots or backend-signed HLS subresources, block redirects/DNS targets to local and private addresses, and reject mismatched Host/Origin.
- Expand Go security tests, check Chrome/Chromium guide interactions with mocked programme data, and retain the working v1.1 launcher.
- The Windows binary cross-compiles, but native Windows and real broadcaster playback remain unverified.

Source snapshot: [versions/v1.3](https://github.com/zidanerick/AussieTV/tree/versions/v1.3).

## v1.2 (15 November 2026)
- Display broadcaster/channel logos from the channel feed in navigation and EPG.
- Add quick-tune network buttons and channel branding.
- Look up optional programme posters on TVmaze using strict programme-title matching.
- Show poster artwork as an idle background with fallback branded tiles.
- Preserve the v1.1 launcher and diagnostics.
- Unit tests expanded for artwork matching and image URL validation.

Source snapshot: [versions/v1.2](https://github.com/zidanerick/AussieTV/tree/versions/v1.2).

## v1.1 (15 November 2026)
- Fix silent startup failure by retaining a visible console.
- Add `RUN-AussieTV.cmd`, log file, and fallback browser address.
- Make opening the default browser more resilient.
- Keep local server loopback-only and print alternate port if occupied.

Source snapshot: [versions/v1.1](https://github.com/zidanerick/AussieTV/tree/versions/v1.1).

## v1.0 (15 November 2026)
- Initial Windows TV interface, guide, stream playback and main Australian broadcaster/channel filter.
- Default Melbourne guide plus configurable major-city region.
- Go localhost HTTP server and embedded HTML/JavaScript frontend.
- Initial Go tests.

Source snapshot: [versions/v1.0](https://github.com/zidanerick/AussieTV/tree/versions/v1.0).

## Future versions

For every change: update this changelog; bump the app version in `main.go`, launcher and README; maintain `docs/CONTINUITY.md`; use pull requests for review. GitHub Actions produces Windows build artifacts for commits to main.
