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

Use the same region in both feeds. Channel IDs and guide IDs match. Programme posters use the same exact-title TVmaze matching as the web app. Server mode warms upcoming programme metadata in the background and persists a bounded cache in /data/AussieTV/programme-artwork.json (Docker). XMLTV exports legacy icon and typed poster, plus a genuine landscape backdrop when available. Curated channel logos use the same bounded image endpoint as the web player, with authenticated feed URLs. Jellyfin clients do not always import programme icons as Primary artwork.
The bearer key is embedded in both URLs and must be kept private.

## Stream transport modes

| Mode | Use | State |
| --- | --- | --- |
| `redirect` | Jellyfin opens the broadcaster HLS URL | **Default**; reported working, sometimes 6-11 s startup |
| `relay` | Authenticated HLS playlist and segments through AussieTV | Experimental; network dependent |
| `mpegts` | FFmpeg copies HLS video and audio to continuous MPEG-TS | Experimental; SBS audio may be missing or fail startup |
| `fast-tune` | Shared on-demand copy-only MPEG-TS with bounded rolling media | Experimental; warm joins only, real broadcaster/Jellyfin performance unverified |

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


## Experimental fast tune

Set AUSSIETV_JELLYFIN_STREAM_MODE=fast-tune and refresh the M3U tuner (TS
URLs rather than HLS). Keep redirect as the reliable fallback. Fast tune
shares the existing validated FFmpeg relay between readers of the same
channel. Each channel retains approximately 6 MiB of TS packets, allows up
to eight readers and stays active for 20 seconds after the last disconnect.
At most four channel sessions run concurrently. Slow clients that fall
behind the ring are disconnected rather than accumulating memory. Joins
wait for a PAT, video random-access signalling and audio; this is not a
guarantee of decodability on all clients. No background prewarming, no
re-encoding, and no relaxed upstream URL/authentication rules.

For simple master playlists that explicitly advertise H.264/AAC, fast tune selects one variant (highest advertised bandwidth up to 6 Mbit/s, or lowest compatible above that), retaining original audio rendition declarations. Unknown or malformed masters remain unchanged. This may reduce duplicate variant probes but can limit quality; it is deliberately restricted to fast-tune mode. Cold starts still pay FFmpeg input preparation costs. Compare cold and
repeat tune times separately. Authenticated diagnostics include
fastTuneSessionsStarted, fastTuneSessionReuses, fastTuneLagDrops,
fast-tune-first-packet and fast-tune-cold/shared response timings.
These do not measure Jellyfin's own click-to-first-picture latency.

## Artwork troubleshooting

After updating, keep the same M3U/XMLTV URLs and run Jellyfin Refresh Guide Data after the artwork queue has populated. Check the authenticated diagnostics for programmeArtworkCached, programmeArtworkMisses and programmeArtworkPending. The guide request never blocks on TVmaze; a cold installation may need a later refresh. The cache persists if /data is mounted. Ambiguous titles, bulletins and programmes with no matching artwork deliberately remain blank. TVmaze landscape images are series backdrops, not episode stills. Jellyfin 12 imports typed backdrop art, but upstream [issue #18209](https://github.com/jellyfin/jellyfin/issues/18209) tracks missing Primary artwork from XMLTV icons. This change cannot guarantee a particular Jellyfin client displays images. Do not clear the entire Jellyfin database to refresh artwork.


## Optional direct artwork repair (Docker server mode)

Jellyfin issue #18209 documents successful direct Base64 image uploads for
Live TV programs whose XMLTV image import failed. The opt-in repair service
runs every two minutes, checks up to 40 missing posters per cycle, and only
acts on the next 24 hours of listings. It requires exactly one configured
M3U tuner and XMLTV provider, with URLs matching this AussieTV instance
(including the feed key). It then matches unique channel name/number and
exact programme title, UTC start and UTC end. Existing images are skipped;
no library, recording, channel or server configuration is modified.

In Jellyfin Dashboard > API Keys, create a dedicated administrator API key.
Set the following variables on the AussieTV container:

    AUSSIETV_JELLYFIN_ARTWORK_SYNC=1
    AUSSIETV_JELLYFIN_SERVER_URL=http://YOUR-JELLYFIN-SERVER:8096
    AUSSIETV_JELLYFIN_API_KEY=YOUR-JELLYFIN-API-KEY

Prefer HTTPS. The key has broad server permissions; restrict the container
to trusted hosts and revoke the key if no longer used. The optional
AUSSIETV_JELLYFIN_API_KEY_FILE can replace the environment key with a mounted
secret file. The service never follows API redirects or forwards the API key
to artwork hosts. Do not enable on a multi-tuner server; it fails closed.
After refreshing Jellyfin guide data, let the artwork cache populate.
The keyed diagnostics endpoint reports jellyfinArtworkSync status, matched,
existing, uploaded, verified (HEAD 200) and failed counts for the last pass.
These are not proof of display in every client. A title without confidently
matched TVmaze artwork remains blank. Disable with SYNC=0; this does not
delete any images already repaired.
