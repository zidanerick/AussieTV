# Experimental provider manager

The Australian regional lineup remains unchanged. Extra providers are disabled
by default; enable only sources you have permission to play in third-party
clients. No FAST service is integrated or claimed working yet.

## Docker / Unraid

Under the existing appdata mapping, create providers/providers.json and copy
the example in config/providers.example.json. Add local .m3u and optional
XMLTV files beside the manifest. Set AUSSIETV_PROVIDERS_FILE to
/data/providers/providers.json and restart. The same file can be used on
Linux or Windows via an absolute path. Set enabled=true for approved sources.

Example M3U (reserved placeholder domain, not a real stream):

    #EXTM3U
    #EXTINF:-1 tvg-id="example.tv" tvg-logo="https://example.org/logo.png",Example TV
    https://example.org/live/master.m3u8

Each source needs a unique stable tvg-id. The XMLTV programme channel
attribute must match it; use timestamps with explicit UTC offsets. The
provider manager assigns stable namespaced IDs and numbers beginning at
numberStart (1000-9899), preserving all Australian IDs. Omit xmltv if no
authorised schedule is available. Missing guides stay empty, not fabricated.
Import changes require a restart and Jellyfin guide refresh.

Limits: eight provider entries, 200 additional channels total, 1 MiB M3U
and 8 MiB XMLTV per provider, 20,000 programmes per guide. Invalid enabled
providers disable the extra lineup without changing the Australian source.
Only local regular files beside the manifest; no arbitrary URL list uploads.
Public HTTPS HLS URLs without query parameters, nonstandard ports, embedded
credentials, or player directives. No local network endpoints, DRM or
device/geographic restrictions bypass. Additional channels use the guarded
relay even in redirect mode; MPEG-TS/Fast Tune remains optional and unchanged.
Image URLs go through the existing bounded logo proxy. External XMLTV images
are ignored until a safe import policy is implemented.

Candidate sources to verify separately: 9Rush, C31 Melbourne, C44 Adelaide,
NHK World-Japan, DW English, France 24, CNA, and authorised FAST sources.
Official online playback is not permission for arbitrary third-party
redistribution; no unverified streams are shipped. Check rights, HLS
compatibility, audio, artwork and guide mapping before inclusion.

## Hardening completed in this stage

Docker uses Go 1.27.4 and Alpine 3.24.4. Media DNS/address checks now block
carrier NAT and additional special-use ranges, and Jellyfin media responses
reject active HTML/script/SVG content. Full vulnerability scanning, CSP for
the app UI, dependency pinning, and authentication rate limits remain open.
