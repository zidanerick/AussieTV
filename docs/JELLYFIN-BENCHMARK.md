# Jellyfin Web timing protocol (Docker)

**Status: no measurements of the user's live Jellyfin Web client have been made.** The 24-row CSV template is intentionally blank; do not mistake a Go/FFmpeg synthetic test for the requested real-world comparison.

## Preflight

1. Keep the currently deployed tuner, authentication key, and guide provider unchanged. Back up the private container environment and persistent /data before testing. Run the preview separately or in a planned test slot, not at the same time on the production ports.
2. Ensure `ffmpeg` is installed and on the PATH of the **AussieTV process**. The optional `mpegts` mode uses a copy-only H.264/AAC-to-MPEG-TS remux, **not** a video transcoder. FFmpeg is bundled in the Docker image.
3. Set AUSSIETV_JELLYFIN_STREAM_MODE=redirect, recreate the container, and start and Jellyfin **Allow stream sharing enabled**. Confirm both Nickelodeon and 7two play.
4. Use the same Jellyfin server and **the same Jellyfin Web browser/profile and playback quality** for both modes. Leave Jellyfin transcoding, hardware acceleration, and client playback options unchanged.
5. Use the two identified channels (Nickelodeon, 7two) in the same region and channel IDs for both modes. Jellyfin must refresh the M3U tuner when the URL extension changes from `.m3u8` to `.ts`. Do not change the deployed tuner without a backup.

## Trials

For each `(channel, mode)`: perform **three cold starts** and **three repeat starts**. Cold: stop playback, wait for the Jellyfin playback session to close, and start anew. Repeat: reselect the same channel after one brief stop, without restarting Jellyfin. Record the true elapsed milliseconds from clicking the channel in Jellyfin Web to the **first visible moving video frame**. Do not time the first HTTP response, spinner disappearance, audio start, or the appearance of a still programme poster.

Watch each run for **two minutes**, noting stalls, missing audio and lip-sync drift. Check the Jellyfin FFmpeg/transcode log for input protocol, stream copy/transcode, probes, errors and timestamps. Do not copy signed broadcaster URLs or AussieTV feed keys into shared logs.

Use `benchmarks/jellyfin-web-results-template.csv` (24 blank rows). Calculate the median and 90th percentile separately for cold and repeat runs. A useful gain should be **repeatable for both channels**, not an outlier in one trial, and should not introduce audio sync or stability regressions. If the median gain is trivial, the extra FFmpeg process and relay traffic are not justified. Keep the direct mode as default.

## Verification of programme art

1. Request `/jellyfin/guide.xml?key=YOUR_KEY` using the existing authenticated feed URL. Note the number of `<programme>` and programme `<icon>` tags. Initially, missing titles are queued for lookup and export finishes without waiting.
2. Allow time for background TVmaze resolution, request the XML again, and find a programme with a confidently unique title such as **The Simpsons** (only if it is listed in the EPG). Verify `<programme channel="...">...<icon src="https://static.tvmaze.com/..."/></programme>` and open its exact `src` URL from the Jellyfin server/network. **Bluey** is ambiguous in TVmaze (there are distinct 2018 and 1976 series), so it is intentionally excluded by title-only matching.
3. In Jellyfin Dashboard → Live TV, choose **Refresh Guide**. Open the programme via Jellyfin Web. If artwork is absent, query Jellyfin's program/item JSON for `ImageTags.Primary`, and `/Items/{id}/Images/Primary` if available to distinguish import from display failures. Jellyfin issue [#18209](https://github.com/jellyfin/jellyfin/issues/18209) reports that Jellyfin 12 may not persist XMLTV `<programme><icon>` into Primary images even when the XML and URL are correct. **This must not be labelled as a failed AussieTV XML export.**

## Evidence gathered offline for the earlier pipeline

- Synthetic FFmpeg H.264/AAC transport-stream remux (no re-encode) passed in Go integration tests, including validating output codecs with `ffprobe`.
- FFmpeg was terminated by cancelling the HTTP client request; an active process counter returned to zero.
- Audio/video start PTS difference for the synthetic fixture was approximately **0.021 seconds**, with first transport packets after approximately **63–72 ms** in the local test environment. These are **not Jellyfin click-to-first-picture or real broadcaster measurements**.
- Cross-browser Jellyfin Web, Nickelodeon/7two stream stability, real lip sync and the requested cold/repeat benchmark: **not tested; no access to the user's Jellyfin client**.

If actual Jellyfin Web timings remain 6–11 seconds, use Jellyfin's playback/FFmpeg logs to compare probe, startup, and remux phase duration against AussieTV `mpegts-first-packet` events. FFmpeg re-muxing on AussieTV cannot remove an additional buffering period inside Jellyfin Web.
