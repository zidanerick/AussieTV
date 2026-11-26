# Browser remote-control and compatibility

The existing TV interface can now be operated without a mouse when a remote emits keyboard events through HDMI-CEC, a Bluetooth/USB remote, a PC remote bridge or an embedded TV webview. This is a web application, not a native Samsung/LG/Android TV app. The TV OS may reserve special keys instead of delivering them to webpages.

## Remote controls

| Key / button | Action |
| --- | --- |
| Arrow keys / D-pad | Left/right move along the same row; up/down move between rows and programmes |
| OK, Select, Enter | Activate the highlighted control; start/commit an explicit dropdown edit |
| Back, Escape, Backspace | Cancel dropdown edits, close help/Text/fullscreen or return to guide |
| CH+/CH-, Page Up/Page Down | Previous/next channel |
| Numbers | Enter broadcast channel number (e.g. 2 then 4 for ABC News 24); commits after about 1 second |
| G, Home | Jump to guide |
| M | Focus network browse filters |
| H | Remote help screen |
| T | Text information screen |
| C | Toggle captions when a track exists |
| F | Fullscreen, where the browser permits it |
| Space or media Play/Pause | Toggle stream playback; Space activates a focused button |

**In-app controls**: the player also exposes Play/Pause and Mute buttons. All buttons, links, favourites, source regions, CC selector and Text pages remain accessible via D-pad/OK. On a focused region or CC dropdown, directional keys navigate the screen until you press **OK**. While editing, Up/Down changes the selected item; **OK** commits, while **Back/Escape** restores the previous selection without applying it. Outside edit mode, Back returns to the guide. The help overlay traps D-pad navigation until closed.

Visual focus uses a high-contrast yellow outline, and an active dropdown edit has a green outline. The guide is a strict row-based navigation grid: at a row edge, left/right stops rather than jumping to the player. Scrolling is performed when focus moves, including inside the guide. Programme guide navigation retains time proximity when moving between channels.

## Browser support and verification

- **Automated evidence**: Chromium headless with simulated guide and channels, `tests/remote_browser_smoke.py`; plus Node built-in test runner, `node --test tests/remote_keys.test.cjs`; Go tests and Docker image smoke checks.
- **Expected modern browsers**: Microsoft Edge and Google Chrome (Hls.js), Firefox (Hls.js), and Safari (native HLS where available). The UI uses standard DOM keyboard events. Actual playback depends on the browser's codecs, stream and DRM rules. Only headless Chromium navigation was exercised in this release, not all browsers.
- **Smart-TV browsers**: standard arrows and OK may be exposed, but Back, channel, numeric and media keys may be reserved by the device. Samsung's own documentation warns that special TV remote keys are unavailable in its Web browser. For guaranteed key coverage, a native TV app or compatible keyboard/remote bridge would be needed.
- Native `video` controls and fullscreen are browser-specific; the player includes its own controls, and fullscreen failure is reported without breaking navigation.

## Tests

```bash
go test -race ./...
node --test tests/remote_keys.test.cjs
# Optional: Python Playwright + local Chromium installed
python tests/remote_browser_smoke.py
```

The Python test renders HTML in memory with mocked guide JSON, avoiding network access and live streams. It is an interaction test, not a broadcaster video test. Set `AUSSIETV_CHROMIUM=/path/to/chromium` if Chromium is elsewhere.

## Remaining work

- Real hardware passes with HDMI-CEC/USB remote and TV web engines, including native key codes.
- Firefox, Safari/WebKit and Edge manual compatibility matrix and HLS/DRM playback validation.
- Make the programme guide more efficient with large channel counts; track focus restoration when channel data is refreshed concurrently.
- Use the authenticated Docker web origin and private HTTPS proxy; do not publish the service to the Internet.

## Focus exit routes

When a group or guide edge has no next item in the requested direction, the remote moves to a named neighboring section instead of remaining stuck. The section graph covers the sidebar, top controls, player controls, on-air actions, quick channels, guide controls and EPG. Modal Text/help retain Back-to-close. Region/caption dropdowns remain explicit OK-to-edit with Back-to-cancel. Headless Chromium with mock guide and focusable elements validates these transitions; real TV browser hardware is not yet tested.
