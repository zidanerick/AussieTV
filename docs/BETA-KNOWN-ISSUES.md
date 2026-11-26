# Docker beta known issues and release gate

AussieTV is an early beta, not a production multi-user streaming service.

| Area | Limitation |
| --- | --- |
| Streams | DRM, regional access, source changes and unsupported HLS can block playback |
| MPEG-TS/Fast Tune | Experimental; some channels may lack audio or fail to start |
| Jellyfin startup | Client buffering and transcoding can dominate tune latency |
| Artwork | TVmaze coverage incomplete; XMLTV Primary import varies; repair opt-in |
| Public broadcasters | Manifest readiness only; not end-to-end playback or EPG verification |
| HDHomeRun | Bridge broadcast discovery unreliable; disabled by default |
| Remote / captions | Browser and hardware behaviour varies |
| Security | Private network/HTTPS required; further rate-limit, CSP and dependency audit work remains |
| Unraid | Static template tests, not automated real GUI deployment |

Stable release requires real client cold/warm tuning, audio/video sync,
long-running and multi-client tests, safe updates/rollback on amd64/arm64,
real Unraid and proxy checks, and security review. Docker CI tests packaging
and local listeners, not live broadcast rights or client playback. Do not
share secrets, signed URLs or unredacted logs.
