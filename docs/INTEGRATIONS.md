# AussieTV future hosting / Jellyfin integration plan

This document is a plan, not a claim that the current Windows executable is a secure self-hosted server or Jellyfin plugin.

## 1. Separate desktop and server lifecycles

Currently the Go app binds localhost, launches the default browser and shuts itself down after desktop-client inactivity. It has a quit endpoint and loopback Host/Origin checks. Never solve Docker support by changing the bind address to `0.0.0.0` without addressing these security and lifecycle assumptions.

**Future server mode**: a separate `serve` mode should bind a user-selected network interface, disable browser auto-launch, heartbeat auto-quit and remote `/api/quit`, and provide a stable health endpoint. Add authentication and CSRF/origin controls if accessible beyond a trusted local network, rate-limit outbound proxy and artwork requests, and cap caches/storage. Retain SSRF/DNS/rebinding restrictions around the HLS and logo proxies. Supply a multi-stage Dockerfile, non-root runtime, read-only filesystem where feasible and a persistent configuration volume. Document Docker/Unraid `ports`, hostnames, reverse proxy and HTTPS.

Browser clients should continue using relative `/api` and `/hls` URLs so a same-origin reverse proxy works. Do not expose unauthenticated stream proxies to the internet. Browser-local favourites and caption preferences are presently per-browser; decide whether a server-hosted account/preferences API is needed later.

## 2. Jellyfin integration: use built-in Live TV first

Official Jellyfin Live TV supports **M3U tuner playlists** and **XMLTV guide feeds**, so a new Jellyfin plugin is *not required for initial channel playback or guide data*. This is the fastest interoperable path.

Suggested future AussieTV endpoints:

- `GET /export/channels.m3u`: curated regional channels with stable `tvg-id`, channel number, displayed name and logo URLs. Keep stream URLs useful to Jellyfin, not ephemeral signed browser-only `/hls` URLs.
- `GET /export/guide.xml`: XMLTV matching those stable IDs with regional EPG entries; test UTC offsets and XML escaping.
- `GET /api/v1/channels`: stable JSON for web, integrations and future clients. Do not expose secrets/tokens in querystrings or logs.

Jellyfin Admin > Live TV > M3U tuner points to the playlist; add an XMLTV provider and map channels. A DVB-T/TVHeadend backend could supply the same feeds. Underlying sources may use geo-restrictions, auth or DRM and are not guaranteed to work in Jellyfin.

## 3. Only then consider a Jellyfin plugin

A server plugin, if still useful, would add AussieTV-specific configuration, channel management or metadata and would be written against the supported Jellyfin plugin API/runtime. Reproducing the entire AussieTV TV remote interface inside Jellyfin would be a **client/web UI task**, not automatically achieved by a server-side plugin. Keep the channel model and EPG as testable, client-agnostic services. Confirm Jellyfin server version and plugin compatibility before selecting a template/SDK.

References: https://jellyfin.org/docs/general/server/live-tv/setup-guide/ and https://jellyfin.org/docs/general/server/plugins/

## Integration sequence

1. Ship/test remote navigation and modern-browser compatibility.
2. Factor the feed/channel/EPG model into versioned APIs with stable IDs.
3. Add an authenticated server mode and Docker image, verifying localhost and LAN security separately.
4. Implement M3U/XMLTV exports, import into Jellyfin's standard Live TV, validate real streams and guide matching.
5. Scope a plugin only for gaps that Jellyfin Live TV does not already cover, with its own project/tests.

## Initial Jellyfin feeds (v1.8)

See [Jellyfin setup](JELLYFIN.md) for opt-in M3U/XMLTV endpoints, stream redirects, separate authenticated LAN listener, and feed-only Docker/Unraid recipe. This is **not** full Docker-hosted browser UI or HDHomeRun emulation.
