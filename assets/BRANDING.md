# AussieTV branding

The project icon and wordmark are original AussieTV artwork. The icon combines a navy television silhouette, aqua play symbol and gold accent.

- Canonical SVG icon: `static/app-icon.svg`
- Wide transparent wordmark: `static/aussietv-logo.svg`
- Generated PNG outputs: `static/favicon-32.png`, `static/favicon-192.png`
- Generated ICO outputs: `static/favicon.ico`, `assets/AussieTV.ico`

Run `go run ./tools/iconbuilder` before a distributable build. The GitHub Actions workflow generates these files before tests and compilation. The files are intentionally ignored by Git.

The Windows ICO is distributed alongside the EXE and can be assigned to a shortcut manually. Embedding the icon in PE resources is a separate enhancement.

Do not incorporate network broadcaster logos into the AussieTV application icon.
