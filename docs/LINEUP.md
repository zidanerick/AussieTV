# Selected channel lineup

Source: completed user channel selection workbook (84 Keep, 445 Undecided).
Only explicit Keep rows are included in the built-in lineup. This supersedes
the previous 110-entry Round 2 shortlist. All 22 existing channels remain
selected, plus optional AussieTV Text (995). Previously unselected indexed
expansion entries are removed, including from offline guide fallback.
Selected single-show and seasonal streams are intentional exceptions to the
earlier general-channel preference. No audio-only entries are selected.

Enable using AUSSIETV_EXTRA_AU_CHANNELS=1 and refresh Jellyfin guide data.
Set 0 to retain only the original lineup. Existing source channel IDs,
numbers for retained channels, feed URLs and Fast Tune remain unchanged.
Custom administrator-configured providers are separate and unaffected.

## Existing selection

Workbook IDs: CH-001, CH-002, CH-003, CH-004, CH-005, CH-006, CH-007, CH-008, CH-009, CH-010, CH-011, CH-012, CH-013, CH-014, CH-015, CH-016, CH-017, CH-026, CH-213, CH-235, CH-236, CH-269. Teletext CH-020 remains optional.

## 37 indexed expansion candidates

Source: https://i.mjh.nz/au/Melbourne/list.html and tv.json (regional).
Exact name/network matching, guarded public HTTPS HLS relay, no new
device-specific headers. Presence in the index does not verify playback,
DRM status or redistribution permission. Missing entries are omitted.
10 Sport 2 and Hockey One were not present in the checked 27 November list;
they remain exact-match candidates if the source returns them. Some
channels lack schedule data or are short/event-only feeds. No made-up EPG.
New numbers are assigned only where no previously assigned number exists.

| Workbook ID | Number | Channel | Network | Category |
| --- | --- | --- | --- | --- |
| CH-018 | 95 | 9Rush | Nine | General / FTA |
| CH-021 | 44 | Channel 31 | Community | Community / Public |
| CH-028 | 1209 | 7NEWS Spotlight | Seven | News / Business |
| CH-030 | 1210 | ABC News in 90 Seconds | ABC | News / Business |
| CH-033 | 1202 | Al Jazeera | International index | News / Business |
| CH-035 | 1200 | BBC News | International index | News / Business |
| CH-036 | 1207 | Bloomberg TV | International index | News / Business |
| CH-039 | 1203 | DW English | International index | News / Business |
| CH-040 | 1206 | euronews | SBS | News / Business |
| CH-041 | 1204 | France 24 | SBS | News / Business |
| CH-043 | 1201 | Sky News Now | Sky | News / Business |
| CH-064 | 1101 | Air Crash Investigations | Seven | Science / Tech / Nature |
| CH-066 | 1100 | Mythbusters | Seven | Science / Tech / Nature |
| CH-068 | 1410 | Taronga 24/7 | Nine | Science / Tech / Nature |
| CH-127 | 1600 | A Country Practice | Seven | Drama / Series |
| CH-128 | 1601 | All Saints | Seven | Drama / Series |
| CH-129 | 1602 | Blue Heelers | Seven | Drama / Series |
| CH-131 | 1603 | Home and Away Classics | Seven | Drama / Series |
| CH-150 | 1502 | Adult Animation | Ten | Comedy |
| CH-151 | 1500 | BBC Comedy | Nine | Comedy |
| CH-176 | 1503 | 9Crime | Nine | Crime / Mystery |
| CH-177 | 1504 | 9Mystery&Paranormal | Nine | Crime / Mystery |
| CH-190 | 1604 | Rush | Ten | Crime / Mystery |
| CH-214 | 1507 | All-Star Cinema | Seven | Movies |
| CH-215 | 1700 | Christmas Movies | Ten | Movies |
| CH-217 | 1508 | Moviesphere | Ten | Movies |
| CH-241 | 1710 | NickToons 90s | Ten | Kids / Family |
| CH-272 | 1720 | BBC Antiques Roadshow | Nine | Food / Lifestyle |
| CH-273 | 1509 | BBC Food | Nine | Food / Lifestyle |
| CH-274 | 1510 | BBC Home & Garden | Nine | Food / Lifestyle |
| CH-275 | 1721 | Better Homes And Gardens | Seven | Food / Lifestyle |
| CH-276 | 1722 | Bondi Vet | Seven | Food / Lifestyle |
| CH-336 | 1730 | Big Brother Live Stream | Ten | Reality / Entertainment |
| CH-398 | 1301 | 10 Sport 2 | Ten | Sport |
| CH-399 | 1300 | 7AFL | Seven | Sport |
| CH-402 | 1306 | Hockey One | Seven | Sport |
| CH-408 | 1304 | Redbull TV | International index | Sport |

## 24 additional selected rows not exposed as streams

No unverified provider URLs or access-control bypass. One row is an
alternate provider for BBC Home & Garden, already in the indexed list.
All others need an authorised compatible source and EPG verification.

| Workbook ID | Channel | Candidate source | Status |
| --- | --- | --- | --- |
| CH-019 | 7Bravo | Seven / MJH | DRM restriction; not enabled |
| CH-044 | ABC News Live (US) | ABC US | Awaiting supported source verification |
| CH-045 | CBS News 24/7 | CBS | Awaiting supported source verification |
| CH-046 | NBC News NOW | NBC | Awaiting supported source verification |
| CH-047 | Sky News UK | Sky | Awaiting supported source verification |
| CH-053 | CBC News | Plex | Awaiting supported source verification |
| CH-058 | NEWSMAX | Plex | Awaiting supported source verification |
| CH-062 | USA TODAY | Plex | Awaiting supported source verification |
| CH-070 | NASA+ | NASA | Live event/on-demand service; continuous linear HLS not established |
| CH-083 | PBS Science | Samsung TV Plus / Plex | Awaiting supported source verification |
| CH-084 | Popular Science | Samsung TV Plus / Plex | Awaiting supported source verification |
| CH-085 | Real Wild | Samsung TV Plus / Plex | Awaiting supported source verification |
| CH-088 | WeatherSpy | Samsung TV Plus | Awaiting supported source verification |
| CH-141 | Law & Order | Samsung TV Plus | Awaiting supported source verification |
| CH-145 | Stargate | Samsung TV Plus | Awaiting supported source verification |
| CH-169 | Mr. Bean | Plex | Awaiting supported source verification |
| CH-255 | Pokémon | Samsung TV Plus | Awaiting supported source verification |
| CH-285 | BBC Home and Garden | Samsung TV Plus | Alternate source for selected BBC Home & Garden (CH-274); not a second channel |
| CH-286 | BBC Travel | Samsung TV Plus | Awaiting supported source verification |
| CH-304 | The Bob Ross Channel | Samsung TV Plus | Awaiting supported source verification |
| CH-367 | Deal or No Deal | Samsung TV Plus | Awaiting supported source verification |
| CH-424 | NFL Channel | Plex | Awaiting supported source verification |
| CH-435 | TNA Wrestling Channel | Plex | Awaiting supported source verification |
| CH-443 | NHK World-Japan | NHK | Awaiting supported source verification |

## Playback troubleshooting

Expanded streams keep all validated HLS variants in Fast Tune. Inline HLS
session metadata is supported; URI-based session metadata, content steering,
variable substitution and protected HLS remain unsupported. Authenticated
Jellyfin diagnostics retain a separate bounded recentFailures history with
upstreamStatus and reason where known. A provider HTTP 401/403 is not the
same as an FFmpeg remux failure; no access restrictions are bypassed.

### HLS encryption diagnostics

Ordinary AES-128 HLS with the default or explicitly declared identity key
format is supported without altering the encryption declaration. Non-identity
key formats, unsupported encryption methods and malformed key declarations
remain blocked in both browser and Jellyfin playback. The diagnostics report
`protected-hls-keyformat`, `unsupported-hls-encryption-method`,
`unsupported-hls-key-version` or `invalid-hls-key-tag` rather than confusing
all key declarations with DRM. No DRM license handling or circumvention is
implemented. A live 7AFL report showed a playlist rejection but did not
identify its exact method or key format; live compatibility is unverified.
