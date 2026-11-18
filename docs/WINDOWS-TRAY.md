> **Version note (v0.1.0-beta.1):** This document may describe pre-beta v1.x/v2.x prototypes. Current Windows beta status and support boundaries are in [README](../README.md) and [Known issues](BETA-KNOWN-ISSUES.md).

# AussieTV Windows notification-area application (v2.0.1 preview)

AussieTV is a portable Windows GUI-subsystem executable. Launch `AussieTV.exe` directly after extracting the ZIP. It opens the existing TV interface in your Windows default browser (Firefox, Chrome, Edge or whichever you have selected) and shows an AussieTV icon in the notification area beside the clock.

**Closing the browser window does not stop AussieTV.** To reopen it, double-click the icon. Right-click the icon for:

- Open AussieTV
- Settings
- Jellyfin feed status
- Open startup log
- Exit AussieTV

**Exit AussieTV** is the supported shutdown path. It stops the local HTTP service and Jellyfin feed/discovery listeners. Closing the Windows app via the notification area also leaves no console window open. The Settings window is a local web GUI served on `127.0.0.1:45781/settings`, not a native WinUI window. It stores options and a random Jellyfin access key under `%LOCALAPPDATA%\AussieTV\settings.json`. No password/token is written to the startup log. The local API requires an origin check and a per-process anti-CSRF token for changes.

**Jellyfin setup:** Use Settings to enable a feed and, if needed, a LAN listener. A local Jellyfin installation can use loopback. For remote Jellyfin, check the app's suggested LAN address; choose the IP Windows can actually reach from Jellyfin. Saving a LAN-enabled Jellyfin feed now requests Windows administrator approval to configure the application-owned Private-profile firewall rules (TCP 45782 for the LAN feed and optional UDP 65001 for HDHomeRun). The rules are restricted to the local subnet. Turning these features off removes the app-owned rules after approval. Existing LAN installations may use Settings → Repair firewall access. The local-only feed requires no firewall changes. No public-network rules or router forwarding are added. See `docs/JELLYFIN.md` for the required ports and the optional, experimental HDHomeRun discovery.

The old `RUN-AussieTV.cmd` remains a compatibility shortcut but no longer needs to remain open. For new releases, Windows CI must use `-ldflags="-s -w -H=windowsgui"` and generate icon resources before building. Native tray icon/menu/window behaviour cannot be completely tested in Linux and needs a Windows smoke test.

## v2.0.1 Settings and browser repair

The v2.0 preview Settings page shipped with an extra `}` in its JavaScript, so
its interactive controls never initialised. A JS syntax and Settings DOM test is
now part of Windows CI. The tray's Open/Settings commands both call the Windows
`ShellExecuteW` URL handler, respecting the user's default browser. An app
window is **not** forced into Edge/Chrome.

When LAN permissions change, Settings asks for elevation through a short-lived
copy of AussieTV itself. Its helper supports only the fixed, application-owned
TCP 45782 / UDP 65001 rules, with `profile=private` and
`remoteip=localsubnet`. If approval is refused, the requested app settings
still save but Settings displays a firewall warning and offers **Repair
firewall access**. The Windows firewall prompt and resulting rules need a native
Windows 10/11 test; Linux cross-compilation cannot grant or verify Windows rules.
