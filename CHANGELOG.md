# Changelog

## v1.8.0 (18 November 2026)
- Replace dead-end D-pad section bounds with deterministic cross-section exit routes; preserve guide row navigation and dropdown edit safeguards.
- Add opt-in, token-authenticated Jellyfin M3U and XMLTV feeds on a separate, restricted LAN listener; stream URLs redirect to current upstream sources.
- Provide a feed-only Dockerfile, Compose example and Unraid/Jellyfin setup instructions.
- Add tests for remote movement and feed authentication/channel mapping/XML output; CI still verifies Windows builds.
- Live Jellyfin playback and native TV-remote validation remain unverified.


## v1.7.1 (17 November 2026)
- Keep horizontal D-pad movement within the same visual row and control group; stop crossing from the programme guide into unrelated player controls at its boundaries.
- Require explicit OK-to-edit mode for region and caption selectors. Up/down changes values only while editing, OK commits and dispatches change, Back/Escape restores the previous value without applying it.
- Keep non-editing selects navigable, fix Back recovery from a select, and retain mouse/touch use.
- Add unit and Chromium mocked-guide regression tests at 1280px and 768px; Go race tests and Windows cross-build pass. Actual remote hardware and browser playback remain unverified.


## v1.7.0 (16 November 2026)
- Add a dedicated remote/keyboard navigation layer for the whole web interface with D-pad/arrow spatial focus, OK, Back, channel step, numeric tuning, Text, captions and help shortcuts.
- Add remote help overlay, strong visible focus indicators and keyboard-accessible player Play/Pause and Mute controls.
- Preserve guide focus on redraw and quick-channel focus when rebuilding quick buttons.
- Improve logo artwork handling across every curated channel with a safe same-origin image endpoint, aspect/background contrast analysis and original logo fallback chain. Keep the original broadcaster marks preferred; the channel-specific generated placeholders are last resort.
- Add Node tests for key aliases and spatial navigation, and Playwright's mocked Chromium UI smoke test.
- Add architecture notes for later Docker self-hosting and Jellyfin M3U/XMLTV integration, without exposing the current localhost proxy over the LAN.
- Pass local Go tests/race detector, Node tests, Chromium mock-UI interactions and Windows x64 compilation. Live streams, native Windows remotes and Firefox/Safari remain unverified.

## v1.6 (16 November 2026)
- Added stream caption selector and local Text information panel using the existing EPG.

## v1.5.1 (16 November 2026)
- Added local fallback channel artwork for ABC News and SBS services.

## v1.5 and earlier
- See GitHub preserved source branches `versions/v1.0` through `versions/v1.5` for historical versions, including embedded app branding, favourites and launch fixes.
