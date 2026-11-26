# Integration roadmap after the first Windows beta

**Windows-only, `v0.1.0-beta.1`.** The next release milestone is reliability,
not platform expansion. Current optional integrations:

- Jellyfin M3U/XMLTV feed with keyed access on a separate listener. Default
  redirect transport remains the stable path to test.
- Experimental FFmpeg HLS-to-MPEG-TS copy relay, not yet validated for all
  Australian broadcaster audio renditions.
- Experimental HDHomeRun discovery, which depends on a shared broadcast domain
  and nonstandard tuner interoperability.
- Teletext-inspired local information pages and TTI export; not an analogue
  RF/PAL VBI transmitter.

## Deferred until core bugs are addressed

- Docker-hosted full browser TV interface and multi-user authentication.
- macOS desktop packaging and platform-native Linux/macOS tray integrations.
- Dedicated Jellyfin plugin, DVR optimisation, stable remote hardware control.

Keep authenticated feeds behind a private-network firewall. Do not turn the
Windows desktop's loopback-only HTTP service into a public network server.
Start with [Jellyfin setup](JELLYFIN.md) and [known issues](BETA-KNOWN-ISSUES.md).

Experimental CI previews: [Linux portable](LINUX.md) and [Discord Activity](DISCORD.md). Neither changes the Windows release gate.
