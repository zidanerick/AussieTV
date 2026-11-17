# AussieTV v1.7.1 remote-control and browser compatibility

The existing TV interface can now be operated without a mouse when a remote emits keyboard events through HDMI-CEC, a Bluetooth/USB remote, a PC remote bridge or an embedded TV webview. This is a web application, not a native Samsung/LG/Android TV app. The TV OS may reserve special keys instead of delivering them to webpages.

## Remote controls

| Key / button | Action |
| --- | --- |
| Arrow keys / D-pad | Left/right stays in a visual row; up/down moves between rows and guide channels |
| OK, Select, Enter | Activate an item, or start/commit editing a region/subtitle selector |
| Back, Escape, Backspace | Cancel selector editing, close overlays/fullscreen or return to guide |
| CH+/CH-, Page Up/Page Down | Previous/next channel |
| Numbers | Enter broadcast channel number (e.g. 2 then 4 for ABC News 24); commits after about 1 second |
| G, Home | Jump to guide |
| M | Focus network browse filters |
| H | Remote help screen |
| T | Text information screen |
| C | Toggle captions when a track exists |
| F | Fullscreen, where the browser permits it |
| Space or media Play/Pause | Toggle stream playback; Space activates a focused button |

**In-app controls**: the player also exposes Play/Pause and Mute buttons. All buttons, links, favourites, source regions, CC selector and Text pages remain accessible via D-pad/OK. Region and CC `<select>` controls do **not** change when arrows move focus. Press **OK** to enter edit mode (green focus ring), **Up/Down** to preview a different option, **OK** again to commit and **Back/Escape** to cancel. Normal mouse/touch selection still works. The help overlay traps D-pad navigation until closed.

Visual focus uses a high-contrast yellow outline. Scrolling is performed when focus moves, including inside the guide. Programme guide navigation retains time proximity when moving between channels.

## Browser support and verification

- **Automated evidence**: Chromium headless with simulated guide and channels, `tests/remote_browser_smoke.py`; plus Node built-in test runner, `node --test tests/remote_keys.test.cjs`; Go tests on Linux and Windows x64 cross-build.
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
- Use an authenticated network host when Docker mode is implemented. Existing desktop-only endpoints, heartbeat and localhost security rules are *not* ready to expose to a LAN.

**v1.7.1:** Horizontal navigation is deliberately limited to visually aligned controls, and guide boundary arrows do not jump to the player. Regression tests use mocked programmes and two Chromium viewport widths. Other browsers and physical TV remotes still need direct validation.
