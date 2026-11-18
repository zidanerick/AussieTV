# AussieTV branding

The AussieTV icon and wordmark are original project artwork. The icon uses a navy television silhouette, aqua play glyph and gold accent.

- Canonical browser icon: `static/app-icon.svg`
- Wide transparent wordmark: `static/aussietv-logo.svg`
- Generated outputs: `static/favicon-32.png`, `static/favicon-192.png` and `assets/AussieTV.ico`.
- Executable icon resource: `rsrc_windows_amd64.syso` (generated, not committed).

Run `go run ./tools/iconbuilder` before a distributable build. It also embeds the
ICO as a Windows AMD64 COFF resource using only Go standard libraries. Generated files are excluded from source control and included in CI releases. Browsers capable of SVG favicons can use `/app-icon.svg` even without generating the PNGs.

Unlike broadcaster channel logos and TVmaze show images, these brand assets are original AussieTV artwork. Do not incorporate broadcaster logos into the AussieTV app icon.
