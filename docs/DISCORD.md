# Discord Activity experimental preview

This target is a Discord embedded Activity, not a chat bot or a public HLS
relay. It uses AussieTV's regional programme guide and official broadcaster
links. It does not yet implement in-Discord video, watch-party sync,
authentication, or native Discord distribution. No desktop management
endpoints, feed keys, upstream stream URLs or request headers are served.

## Build

Requires Go 1.22+, Node 22+ and npm. From repository root:

    npm --prefix discord install
    npm --prefix discord test
    npm --prefix discord run build
    mkdir -p dist/discord-preview
    CGO_ENABLED=0 go build -trimpath -o dist/discord-preview/aussietv .
    cp -R discord/dist dist/discord-preview/discord-web

Or extract the Discord CI artifact (Linux x64). It includes the server and
built browser assets. Keep discord-web beside the executable.

## Configure and launch

1. Create an application at https://discord.com/developers/applications
   and enable Activities for it. Copy its public Application ID.
2. Start the isolated server:

       cd dist/discord-preview
       AUSSIETV_DISCORD_APP_ID=YOUR_APPLICATION_ID ./aussietv --discord-activity

   The default address is 127.0.0.1:45783. Open it in a local browser to
   preview. The optional AUSSIETV_DISCORD_BIND overrides the listen address.
3. Put only this Activity listener behind an HTTPS reverse proxy or tunnel
   on a dedicated hostname. Do not forward the normal desktop or Jellyfin
   listener. The guide endpoint is publicly readable; use rate limits and
   monitor traffic. Verify that your guide data usage is permitted.
4. In the Discord Developer Portal, set Activities > URL Mappings, prefix
   / to your HTTPS Activity hostname (without the scheme, per portal UI).
   Enable the Activity entry point and add test accounts as appropriate.
5. Test launching from Discord. The Embedded App SDK connects without
   requesting user identity or OAuth scopes; external broadcaster links
   use openExternalLink. No bot token or client secret is required by this
   read-only implementation.

See https://docs.discord.com/developers/activities/building-an-activity and
https://docs.discord.com/developers/developer-tools/embedded-app-sdk.

Live playback and watch-party functionality require separate broadcaster
permission/availability, Discord proxy and media compatibility validation,
and a security review. Do not proxy signed HLS URLs or bypass DRM.
