AUSSIE TV FOR WINDOWS - VERSION 1.1
===================================

FIXED FROM VERSION 1.0
- Startup is no longer invisible: a console window shows the local web address and failures.
- Browser launch uses your Windows default web browser, with Edge/Chrome fallbacks.
- Startup logs are saved to %LOCALAPPDATA%\AussieTV\startup.log.
- Uses http://127.0.0.1:45781 unless another program already occupies that port.

HOW TO RUN
1. Extract the whole ZIP to a folder, NOT just the .exe from inside ZIP.
2. Double-click RUN-AussieTV.cmd (recommended) or AussieTV.exe.
3. Wait for the browser to open. If nothing opens, type this in Edge or Chrome:
     http://127.0.0.1:45781
4. Keep the black console window open while watching.
5. If startup fails, read the message in the console and check the log above.

This app creates a TV-style guide for ABC, SBS, Seven, Nine and Ten channels.
It uses community-maintained schedules and streams from https://i.mjh.nz/au/.
Streams may be unavailable, expire, require login, or be restricted. Use the
app's Official Player links for protected broadcaster streams.

No Python, Go, or administrator privileges are needed to run this executable.
Microsoft Edge or Google Chrome and an internet connection are recommended.
Executable is unsigned and not verified by Windows SmartScreen.

This release is cross-compiled from Go on Linux. Its internal HTTP interface,
shutdown and tests have been verified, but launching it on Windows and playing
Australian streams have not been tested end-to-end.
