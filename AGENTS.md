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
