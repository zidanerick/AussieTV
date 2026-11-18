# Jellyfin integration: AussieTV v0.1.0-beta.1

**The dependable integration is manually configured M3U/XMLTV with direct HLS
redirect.** The HDHomeRun emulator and FFmpeg MPEG-TS are experiments, not
part of the stable beta acceptance criteria.

## Setup

Use the AussieTV notification-area Settings command to enable **Jellyfin
feed**. A strong per-install access key is generated and stored in your
Windows user profile. If Jellyfin is on another PC or server, enable LAN access
and allow the optional Windows Private/LocalSubnet firewall rule for TCP 45782.
Never port-forward the feed on your router or expose the API key publicly.

In Jellyfin Dashboard > Live TV, select **M3U Tuner**, then use the M3U URL
shown by AussieTV Settings:

```text
http://YOUR-AUSSIETV-PC:45782/jellyfin/channels.m3u?key=YOUR_KEY
```

Add an **XMLTV guide provider** using:

```text
http://YOUR-AUSSIETV-PC:45782/jellyfin/guide.xml?key=YOUR_KEY
```

Use the same region in both feeds. Channel IDs and guide IDs match. Cached
programme posters are exported where TVmaze has an unambiguous title match;
Jellyfin clients do not always import programme icons as Primary artwork.
The bearer key is embedded in both URLs and must be kept private.

## Stream transport modes

| Mode | Use | State |
| --- | --- | --- |
| `redirect` | Jellyfin opens the broadcaster HLS URL | **Default**; reported working, sometimes 6-11 s startup |
| `relay` | Authenticated HLS playlist and segments through AussieTV | Experimental; network dependent |
| `mpegts` | FFmpeg copies HLS video and audio to continuous MPEG-TS | Experimental; SBS audio may be missing or fail startup |

Ordinary TV playback does not require FFmpeg. The setup wizard optionally
fetches FFmpeg and checks its published SHA-256. From the Start Menu choose
**Update FFmpeg** to get an updated binary later. Portable builds bundle FFmpeg.
Keep Jellyfin's **Allow stream sharing** enabled if switching it off causes a
playback error. A fast AussieTV redirect does not mean Jellyfin Web can
immediately display its first picture; client probing, buffering, and HLS
remuxing happen downstream.

## Discovery limitations

AussieTV may optionally advertise a **virtual HDHomeRun** tuner via UDP
65001, plus TCP 80 on the LAN IP if available. Some Jellyfin versions and
Docker network layouts cannot detect or use it. This does not prove a feed
failure. For reliability, retain the M3U/XMLTV tuner and disable discovery.
If testing discovery, keep firewalls restricted to Private/LocalSubnet and
prefer Docker host networking on the Jellyfin side so broadcast traffic
reaches the server. HDHomeRun-mode HTTP can expose an unauthenticated tuner
lineup on the LAN, so do not expose these ports beyond trusted clients.

## Diagnostics and evidence

AussieTV Settings shows whether ports actually bound, HDHomeRun probe/reply
counts, FFmpeg health and stream timing categories. Never share the raw
`?key=...` URLs, signed HLS URLs, or unredacted server logs. Media requests
that finish with HTTP 499 may simply be FFmpeg cancelling unneeded segments;
502 indicates a relay failure. Tests based on synthetic sources do **not**
prove live SBS audio or Jellyfin stability.

Known problems: [BETA-KNOWN-ISSUES.md](BETA-KNOWN-ISSUES.md). See
[JELLYFIN-BENCHMARK.md](JELLYFIN-BENCHMARK.md) for a repeatable Web benchmark.
