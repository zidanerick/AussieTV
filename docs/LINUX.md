# Linux experimental preview

The supported release remains Windows beta 1. These are separate experimental
CI artifacts, not additions to the existing immutable beta release.

Extract the archive for your architecture, enter its directory, then run:

    ./aussietv

The default browser opens the full TV UI at loopback port 45781 (or a free
loopback port, printed in the terminal). Linux uses the existing browser
lifecycle, not a system tray; closing the browser may stop the process after
the heartbeat timeout. Ctrl+C stops a foreground instance.

Requirements: a modern browser, CA certificates and xdg-open (usually
xdg-utils). FFmpeg is optional for experimental MPEG-TS and Teletext video.
Install it from your distribution rather than using Windows binaries.

CachyOS / Arch: sudo pacman -S --needed xdg-utils ffmpeg ca-certificates
Debian / Ubuntu: sudo apt install xdg-utils ffmpeg ca-certificates

Build from source (Go 1.22 or newer):

    bash tools/build-linux.sh amd64
    bash tools/build-linux.sh arm64

Artifacts and checksums appear in dist/linux. Settings and cache follow
the existing Go user config/cache locations. Do not expose the desktop UI
to the network. Optional Jellyfin LAN feed requires explicit settings and
manual Linux firewall configuration; Windows firewall automation does not
apply. ARM64 is cross-compiled, not device-tested. Real playback and
Wayland/X11 browser/remote controls still need hardware smoke testing.
