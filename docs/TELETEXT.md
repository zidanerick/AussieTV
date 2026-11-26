# AussieTV Text / Teletext generator

This project produces 24 x 40 7-bit ASCII pages with Teletext colour control metadata, `.tti` exports for [VBIT2](https://github.com/peterkvt80/vbit2), and a browser viewer. The output is a prototype, **not** a direct RF/PAL encoder. The authenticated Docker browser serves the viewer at your configured origin plus /teletext/.

## Configuration

Place teletext.json in the persistent /data directory and set AUSSIETV_TELETEXT_CONFIG=/data/teletext.json. The repository's canonical default file is `config/teletext.json`. The file defines magazine ranges, page descriptions, manual lines, weather observation feeds and refresh periods. No Go code changes are needed to repurpose pages. Restart after editing. Invalid Teletext configuration leaves the main TV player running but disables the new Teletext endpoint.

Ranges include 100 index; 101-103 notices; 104-119 news; 120-139 weather; 140-159 racing/sport; 160-179 interest; 180-199 local TV guide; 200 diagnostics; 201-299 reserved; 300-399 SBS; 400-499 Seven; 800/801 future captions; 900-999 administrative. Magazine 9 is **never** exported or exposed. The index is rendered every 30 seconds. Data pages have a last-updated stamp at bottom right and HTTP ETags. Pages with no verified source display an unavailable status rather than fabricated content.

Pages have **24 visible rows of 40 characters** each. The first row is a heading, second row blank, and last row includes a return-to-index cue and update timestamp. Browser colours are white/yellow/cyan. TTI rows use the corresponding 7-bit Teletext ESC control sequences and `OL` rows 1-24. The VBIT2 exporter includes a `PN,mppss`, `PS,8000` (transmit) and `CT` header; 900-999 and unimplemented caption pages are excluded from broadcasts. TTI source is text with CRLF line endings.

## Browser and API

- `http://127.0.0.1:45781/teletext/`: browser page viewer, numeric page selection, keyboard/remote arrows and Escape-to-TV.
- `/teletext/page/100`: 24x40 JSON page, cache ETag.
- `/teletext/tti/100`: VBIT2-compatible `.tti` file.
- `/teletext/export.zip`: ZIP of currently listed public pages only.
- The separate Jellyfin listener on TCP 45782 exposes these same routes when a valid Jellyfin feed key is supplied in `?key=`. Keep both listeners private and protect credentials.

### Export for a Teletext decoder

Download /teletext/export.zip from the authenticated browser endpoint, or the
keyed Jellyfin endpoint. Extract its TTI files to a local directory.

Install VBIT2 separately (GPL). Example VBIT2 input/output command (adjust to your installation):

```bash
vbit2 --dir ./tti --format t42 > stream.t42
# For DVB-TXT packetised output, if supported by your installed VBIT2:
vbit2 --dir ./tti --format ts > teletext.ts
```

A VBIT2-capable decoder/inserter can consume the resulting data. A PAL 625-line TV still requires a compatible analogue signal inserter or a Raspberry Pi composite teletext encoder (e.g. raspi-teletext). HDMI or MPEG video alone does not embed analogue Teletext in the VBI. Native end-to-end VBIT2/TV decoder testing has **not** been performed on hardware.

## Jellyfin in-vision channel

With FFmpeg on the server's `PATH` and environment variable `AUSSIETV_TELETEXT_VIDEO=1`, AussieTV adds *AussieTV Text* (virtual channel **995**) to its authenticated M3U and XMLTV feeds. This is a repeating **720x576, 25fps PAL-resolution MPEG-TS video** with optional silent audio generated live from the current pages, not a DVB-TXT auxiliary stream. It rotates pages every 12 seconds and updates the text. Jellyfin's built-in M3U tuner can try it like any other channel. Actual decoding with a specific Jellyfin client is not yet verified.

The Docker image includes FFmpeg and DejaVu fonts; set AUSSIETV_TELETEXT_VIDEO=1 in the container environment. Without FFmpeg, the text channel is omitted instead of presenting a broken channel. This live encoder is CPU-intensive on low-powered hardware, including Raspberry Pi 3B.

## Data sources and gaps

- **Weather:** BOM JSON observations for Melbourne (Olympic Park), Sydney, Brisbane Airport and Darwin; other cities are named placeholders awaiting verified products. Example Melbourne input: `https://www.bom.gov.au/fwo/IDV60901/IDV60901.95936.json`. Current temperature, humidity, wind and rain fields are parsed; the optional five-day forecast and sunrise/sunset aren't currently available and are labelled as such.
- **EPG:** loaded from AussieTV's curated channel guide via `loadChannels`, including upcoming programmes. The standalone `go run ./cmd/teletext` utility has no TV backend and reports EPG unavailable, while the integrated Docker server does.
- **News, racing, lottery, captions:** content isn't synthesized. Populate supported static `pages[].lines` in the config for manually curated information, or add a licensed/permissioned provider before offering live data. Pages 800/801 are placeholders only, never claimed to be live synchronized captions.
- **BOM licence:** anonymous BOM data is for personal/noncommercial use under the Bureau's conditions. Do not redistribute public Teletext weather or share the service beyond a personal setup without confirming licensing. Official warnings should always be checked at BOM.

**Verification:** Go tests, race detector, 24x40 output checks, TTI packet syntax validation, HTTP auth/ETags and a locally generated MPEG-TS stream passed. Real PAL/VBI hardware output, VBIT2 decoding, live BOM network access and Jellyfin playback are not yet proven. Do not equate the generated video channel with embedded Teletext service data.
