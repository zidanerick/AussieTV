# Jellyfin artwork and MPEG-TS

Current configuration and troubleshooting are in [Jellyfin](JELLYFIN.md).
The Docker image includes FFmpeg; no native installation or updater is
needed. Set AUSSIETV_JELLYFIN_STREAM_MODE to redirect (default), relay,
mpegts or fast-tune and recreate the container. Refresh the tuner when
changing HLS/TS URL extensions. The guarded relay preserves standard HLS
encryption and rejects unsupported protection. Do not bypass access controls.

Programme artwork uses exact-title TVmaze matching, a bounded persistent
cache under /data and asynchronous lookups. Missing/ambiguous matches stay
blank. XMLTV Primary import varies; optional direct repair requires a
separate Jellyfin API key. See [Jellyfin artwork repair](JELLYFIN.md).

Use authenticated diagnostics and [benchmark protocol](JELLYFIN-BENCHMARK.md)
to distinguish provider failures, FFmpeg startup and client buffering.
Historical v2.0.x troubleshooting remains in Git history; synthetic results
are not real-world playback evidence.
