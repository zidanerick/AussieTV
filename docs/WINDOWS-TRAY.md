# Windows launcher and Settings: v0.1.0-beta.1

AussieTV.exe is a native Windows GUI-subsystem process and uses the tray icon
near the clock. Its TV and Settings commands open URLs in your **Windows
system default browser**. Closing the browser does not end AussieTV. Use
right-click tray > **Exit AussieTV** to stop the loopback server and optional
Jellyfin listener.

If a tray icon cannot be registered, AussieTV should open the browser anyway.
Launch AussieTV-Diagnostics.exe for a visible console and startup log; the
user-profile log is at `%LOCALAPPDATA%\AussieTV\startup.log`.

## Distribution

- Portable ZIP: extract and launch, no installer or admin required.
- Setup EXE: per-user Inno Setup wizard, default installation location
  `%LOCALAPPDATA%\Programs\AussieTV`, Start Menu shortcuts and uninstaller.
- The setup wizard optionally downloads and verifies FFmpeg, with a selectable
  target directory. Start Menu > AussieTV > **Update FFmpeg** repeats that
  operation and retains the chosen FFmpeg path.
- The installer is unsigned: Windows SmartScreen may show a warning.
- Exit the existing tray process before installing/updating or uninstalling.
- Local-only playback requires no firewall exception. Choosing Jellyfin LAN
  access in Settings requests explicit administrator approval for a
  Private/LocalSubnet firewall rule. Never port-forward ports 45781/45782.

Native Windows testing and user feedback are still required; passing a
cross-compile is not a substitute. [Known issues](BETA-KNOWN-ISSUES.md).
