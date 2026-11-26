# Round 2 channel shortlist

Approved selection workbook: AussieTV_Channel_Selection_Round_2.xlsx (110
entries, all cells Undecided; user approved the shortlist as a whole).
No existing channels are removed. No radio, audio-only, ambient or
single-show feeds from the old experimental expansion are included.

## Enable

Set AUSSIETV_EXTRA_AU_CHANNELS=1 and restart, then refresh Jellyfin Live TV
guide data. Existing channel IDs, numbers and feed URLs are unchanged.
The source is https://i.mjh.nz/au/Melbourne/list.html and its regional JSON
counterpart. Availability varies by region. Matching uses exact indexed name
and network and a valid public HTTPS HLS URL. No playback or rights testing
has been completed for all candidates. No inferred or fabricated schedules.
Source entries without listings can have an empty guide. Sports event
channels may not be continuously available. Set the variable to 0 to revert.

## 22 existing channels + optional Teletext

Retain the existing curated lineup and optional AussieTV Text (995).

## 38 indexed candidates

| Number | Channel | Source network | Category |
| --- | --- | --- | --- |
| 95 | 9Rush | Nine | General & community |
| 44 | Channel 31 | Community | General & community |
| 1000 | GOOD. | International index | General & community |
| 1001 | YouTV | Ten | General & community |
| 1200 | BBC News | International index | News & business |
| 1201 | Sky News Now | Sky | News & business |
| 1202 | Al Jazeera | International index | News & business |
| 1203 | DW English | International index | News & business |
| 1204 | France 24 | SBS | News & business |
| 1205 | Channel News Asia (CNA) | International index | News & business |
| 1206 | euronews | SBS | News & business |
| 1207 | Bloomberg TV | International index | News & business |
| 1208 | ausbiz | Seven | News & business |
| 1300 | 7AFL | Seven | Sports |
| 1301 | 10 Sport 2 | Ten | Sports |
| 1302 | FIFA+ | SBS | Sports |
| 1303 | Cricket Gold | Seven | Sports |
| 1304 | Redbull TV | International index | Sports |
| 1305 | Racing.com | Seven | Sports |
| 1306 | Hockey One | Seven | Sports |
| 1307 | WNBL 2026/27 | Nine | Sports |
| 1400 | Wildest | Ten | Factual & documentaries |
| 1401 | H2 | Nine | Factual & documentaries |
| 1402 | History & Warfare | Ten | Factual & documentaries |
| 1403 | True Stories | Ten | Factual & documentaries |
| 1404 | Travel Australia | Seven | Factual & documentaries |
| 1500 | BBC Comedy | Nine | Multi-show entertainment |
| 1501 | Sitcom Central | Ten | Multi-show entertainment |
| 1502 | Adult Animation | Ten | Multi-show entertainment |
| 1503 | 9Crime | Nine | Multi-show entertainment |
| 1504 | 9Mystery&Paranormal | Nine | Multi-show entertainment |
| 1505 | Crime Time | Ten | Multi-show entertainment |
| 1506 | True Crime Files | Seven | Multi-show entertainment |
| 1507 | All-Star Cinema | Seven | Multi-show entertainment |
| 1508 | Moviesphere | Ten | Multi-show entertainment |
| 1509 | BBC Food | Nine | Multi-show entertainment |
| 1510 | BBC Home & Garden | Nine | Multi-show entertainment |
| 1511 | NickToons | Ten | Multi-show entertainment |

## 49 pending or restricted candidates

The remainder of the approved shortlist is tracked here, not inserted as
dead tuner channels. Provider and regional availability, public compatible
HLS, third-party playback permission, artwork and EPG need confirmation.
Samsung TV Plus and Plex catalogues are not generic M3U export APIs.
Do not use device emulation, protected endpoints or access-control bypass.

| Channel | Candidate provider | Status |
| --- | --- | --- |
| 7Bravo | Seven | DRM / official player only |
| Channel 44 Adelaide | CTV+ | official live player; direct feed not verified |
| NHK World-Japan | NHK | official HLS and guide verification required |
| ARTE | ARTE | Australian availability and supported HLS required |
| KBS World | KBS | supported third-party feed required |
| Samsung Television Network | Samsung TV Plus | Awaiting supported source and guide verification |
| Sony One FAVES | Samsung TV Plus | Awaiting supported source and guide verification |
| Sky News UK | Sky | Awaiting supported source and guide verification |
| ABC News Live (US) | ABC US | Awaiting supported source and guide verification |
| CBS News 24/7 | CBS | Awaiting supported source and guide verification |
| NBC News NOW | NBC | Awaiting supported source and guide verification |
| CBC News | Plex | Awaiting supported source and guide verification |
| TRT World | TRT | Awaiting supported source and guide verification |
| beIN SPORTS XTRA | Samsung TV Plus | Awaiting supported source and guide verification |
| NFL Channel | Plex | Awaiting supported source and guide verification |
| PGA Tour | Samsung TV Plus | Awaiting supported source and guide verification |
| Tennis TV | Plex | Awaiting supported source and guide verification |
| Tennis+ | Samsung TV Plus | Awaiting supported source and guide verification |
| Women's Sports Network | Plex | Awaiting supported source and guide verification |
| FIFA+ Women | Samsung TV Plus | Awaiting supported source and guide verification |
| World Surf League 24/7 | Plex | Awaiting supported source and guide verification |
| SportsGrid | Plex | Awaiting supported source and guide verification |
| Documentary+ | Plex | Awaiting supported source and guide verification |
| Adventure Earth | Plex | Awaiting supported source and guide verification |
| NatureTime | Samsung TV Plus | Awaiting supported source and guide verification |
| Real Wild | Samsung TV Plus / Plex | Awaiting supported source and guide verification |
| PBS Science | Samsung TV Plus / Plex | Awaiting supported source and guide verification |
| Popular Science | Samsung TV Plus / Plex | Awaiting supported source and guide verification |
| Modern Marvels | Samsung TV Plus / Plex | Awaiting supported source and guide verification |
| History Hit | Samsung TV Plus / Plex | Awaiting supported source and guide verification |
| Wonder | Samsung TV Plus | Awaiting supported source and guide verification |
| CNA Originals | Samsung TV Plus | Awaiting supported source and guide verification |
| Bloomberg Originals | Samsung TV Plus | Awaiting supported source and guide verification |
| SPEEDVISION | Plex | Awaiting supported source and guide verification |
| BBC Travel | Samsung TV Plus | Awaiting supported source and guide verification |
| Real Crime | Samsung TV Plus | Awaiting supported source and guide verification |
| Grit TV | Samsung TV Plus | Awaiting supported source and guide verification |
| Homeful Channel | Samsung TV Plus | Awaiting supported source and guide verification |
| Gusto TV | Samsung TV Plus | Awaiting supported source and guide verification |
| PBS Kids | Samsung TV Plus | Awaiting supported source and guide verification |
| Kartoon Channel | Samsung TV Plus | Awaiting supported source and guide verification |
| Vevo Pop | Samsung TV Plus | Awaiting supported source and guide verification |
| Vevo Rock | Samsung TV Plus | Awaiting supported source and guide verification |
| Vevo 80s | Samsung TV Plus | Awaiting supported source and guide verification |
| Vevo 90's | Samsung TV Plus | Awaiting supported source and guide verification |
| Vevo 2K | Samsung TV Plus | Awaiting supported source and guide verification |
| Vevo Country | Samsung TV Plus | Awaiting supported source and guide verification |
| That's ROCK | Samsung TV Plus | Awaiting supported source and guide verification |
| VEEPS Concert Pass | Plex | Awaiting supported source and guide verification |
