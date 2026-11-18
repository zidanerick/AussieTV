# Agent instructions

1. GitHub `zidanerick/AussieTV` is canonical. Inspect actual code/branch head and open issues before editing.
2. Preserve historical `versions/` branches. Never force-push `main`.
3. Keep portable Windows 10/11 x64 build and visible launcher diagnostics.
4. Run Go tests and Windows cross-compilation; run Node remote tests for interface changes. Optional Chromium tests use mocked data and are not video playback validation.
5. Retain local-only backend security, HLS and logo proxy URL restrictions, signed manifest resources and DNS/redirect rules.
6. Respect built-in TV remote input; use explicit OK to start editing region/CC selects, never let arrows mutate a non-editing select, and keep focus visible and recoverable when the guide redraws.
7. Do not imply every browser or remote type works until tested. Remote special key delivery depends on the TV OS.
8. Use `.github/workflows/windows-build.yml` to build Windows zip and include `AussieTV.ico`.
9. For Docker, redesign desktop-specific lifecycle and add network auth before exposing ports. For Jellyfin, test M3U/XMLTV before developing a custom server plugin.
10. Update docs/CONTINUITY.md and CHANGELOG.md after each substantial change, with exact tests and unverified gates.


14. v1.8 remote navigation has explicit section graph exit routes: test all guide and player boundaries. Do not restore Escape-only dead ends or arrow-driven dropdown editing.
15. Jellyfin integration uses opt-in, tokenized M3U and XMLTV on a **separate** restricted listener. Never expose the desktop quit or image/HLS proxy to LAN without a new reviewed authentication/security design.
