# Development instructions for future AI-assisted sessions

1. Read `README.md`, `CHANGELOG.md`, `docs/CONTINUITY.md`, open GitHub issues and the current `main` tree before editing.
2. Treat GitHub source as canonical. Do not reconstruct the code from conversation summaries when the repository is accessible.
3. Preserve `versions/v1.0`, `versions/v1.1`, `versions/v1.2` and `versions/v1.3` snapshot branches and their original commits.
4. Keep Windows 10/11 x64 compatibility; avoid dependencies or installers without a demonstrated need.
5. Maintain the current working launcher and browser fallback. Do not hide all errors by default.
6. Run `go test ./...` and build a Windows x64 executable for meaningful code changes.
7. Clearly separate Linux automated tests, cross-compilation, native Windows launch tests and live Australian streaming tests.
8. Never promise that third-party streams will work indefinitely; retain legal official-player fallbacks.
9. Check URL validation, local binding, stream proxy and SSRF regression behaviour when changing network code.
10. Update `CHANGELOG.md` and `docs/CONTINUITY.md` with each significant version, test evidence and open issues.
11. Use small commits and pull requests where practical. Do not rewrite version-history branches or force-push main.

12. Preserve v1.3 favourite/guide interactions, trusted stream-root policy, signed HLS subresources, redirect and DNS checks; exercise security regression tests before changing streaming code.
13. Do not confuse headless Chromium with real Windows/browser playback validation. Track unverified streams in issue #1.

14. For branding edits, keep SVG and generated PNG/ICO visuals consistent. Run `go run ./tools/iconbuilder` before build/test and preserve working broadcaster logos.

14. Run `go run ./tools/iconbuilder` before building; it generates `rsrc_windows_amd64.syso` with the embedded EXE icon. Keep generator tests and inspect the icon resource on cross-built executables.
15. The preferred branded app window is Edge/Chrome `--app`, not a native WebView. Preserve the default browser and console fallbacks, and avoid claiming a fixed browser taskbar identity.

14. Keep v1.5.1's bundled channel-logo fallbacks and their allowlisted SVG routes; retain the feed-provided logo as preferred artwork. Test the broken-remote-image path when changing logo rendering.

15. For v1.6+ captions, never claim a live stream carries captions merely because the broadcaster captions its terrestrial broadcasts or official player. UI should only list actual tracks exposed in the current source and remain off by default. Preserve and test Hls.subtitleTrack/HTMLMediaElement.textTracks selection, and distinguish simulated browser tracks from real broadcasts.
16. AussieTV Text 100/200/300/400 is a simulated locally-generated information overlay. Do not describe it as receiving Austext or legacy broadcast teletext packets. Preserve the original HTML/CSS interface and local fallback launch behavior.

17. v1.7 remote UI lives in `static/remote.js`. Preserve D-pad/OK/Back and help modal behavior, spatial focus, native select support, quick tile/guide focus on redraw and numeric entry. Run Node remote tests and browser mock integration for UI changes.
18. A separate Docker server mode must change localhost-only lifecycle and authentication/Host/Origin assumptions before exposure; avoid trivial 0.0.0.0 binding. Jellyfin supports M3U/XMLTV without a plugin. See docs/INTEGRATIONS.md.
