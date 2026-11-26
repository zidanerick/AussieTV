# AussieTV branding

The AussieTV icon and wordmark are original project artwork. The icon uses a navy television silhouette, aqua play glyph and gold accent.

- Canonical browser icon: `static/app-icon.svg`
- Wide transparent wordmark: `static/aussietv-logo.svg`
- Generated outputs: `static/favicon-32.png`, `static/favicon-192.png` and `assets/AussieTV.ico`.
- Executable icon resource: `rsrc_windows_amd64.syso` (generated, not committed).

The Docker build runs go run ./tools/iconbuilder to generate browser icons. Legacy ICO/resource generation remains an internal helper; no native packages are distributed. Generated files are excluded from source control. Browsers capable of SVG favicons can use `/app-icon.svg` even without generating the PNGs.

Unlike broadcaster channel logos and TVmaze show images, these brand assets are original AussieTV artwork. Do not incorporate broadcaster logos into the AussieTV app icon.
