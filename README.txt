AUSSIE TV FOR WINDOWS  |  Version 1.0
=======================================

HOW TO RUN
1. Extract the ZIP to any folder on your Windows 10/11 64-bit PC.
2. Double-click AussieTV.exe.
3. Microsoft Edge (or Chrome) opens in a dedicated, app-style TV window.
4. Select a channel, then choose "Watch live".

No Python, Go, Java, admin permissions, installation or tuner required.
Internet access and Microsoft Edge or Google Chrome are required.
If Windows SmartScreen warns about an unfamiliar unsigned executable,
check that you downloaded it from the expected location before choosing
whether to run it. The build is not code-signed.

CHANNELS
ABC: ABC TV, News, Kids, Entertains, Family
SBS: SBS, SBS2, World Movies, Food, NITV
Seven: Seven, 7two, 7mate, 7flix
Nine: Channel 9, 9Gem, 9Go!, 9Life
10: 10, 10 Drama, 10 Comedy, Nickelodeon

The app offers capital-city regions, starting with Melbourne/Victoria.
Latrobe Valley broadcast schedules may differ from Melbourne streams.
Channels not supplied by the community-maintained index will be absent.

GUIDE AND PLAYER
- TV guide with current and upcoming shows when supplied by the feed.
- Select any channel or guide entry to tune to live TV.
- Arrow Up / Down: previous / next channel
- Enter: play channel
- F: fullscreen video
- Earlier / Later / Now: browse the four-hour EPG view
- "Official player": open the broadcaster's official live TV website
- "M3U": save the selected playlist for use in VLC or another player
- "Exit": stops the local server and closes the app window

PLAYBACK NOTES
Channel metadata, schedules and stream addresses come from the public,
third-party Australian IPTV index at https://i.mjh.nz/au/
The index is independently maintained, not supplied or endorsed by broadcasters.
This program plays available HLS streams through a localhost proxy. It does
not bypass DRM, account logins or geoblocking; streams may fail or change.
For channels that require a broadcaster's own authenticated or protected
player, use the clearly labelled "Official player" button.

Your browser loads hls.js from the jsDelivr CDN, so internet access to that
CDN is needed as well as access to the TV feeds. Video goes through a local
server on 127.0.0.1. Only this computer can connect to that server.
No account credentials are stored by AussieTV. Official websites may ask
you to log in using your normal browser session.

TROUBLESHOOTING
- Black screen, 403 or unavailable: try the "Official player" button.
- Guide blank: check internet, click Refresh; schedules depend on the feed.
- No window: confirm Edge or Chrome is installed and launch AussieTV.exe again.
- M3U opens instead of playing: open it in VLC on your PC.
- Windows Firewall prompt: the player listens on the loopback interface only.
- Some channels' schedules or streams may be for Melbourne, not local Gippsland.

LIMITATIONS
Live stream playback has not been end-to-end tested against all the broadcasters
on a Windows machine; some streams may be blocked or incompatible.
The app is a proof-of-concept, not a formally supported or signed installer.

SOURCES / THIRD-PARTY SERVICES
Community EPG / URLs: https://i.mjh.nz/au/
HLS playback: hls.js (Apache-2.0) https://github.com/video-dev/hls.js
Official live-TV links: ABC iview, SBS On Demand, 7plus, 9Now, 10.

Source code is available in the separate AussieTV-Source.zip archive.
