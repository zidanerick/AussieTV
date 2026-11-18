# Read-only, opt-in Jellyfin feed container. The interactive TV UI remains
# localhost-only until a separately authenticated LAN interface is developed.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/aussietv .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg font-dejavu && \
    addgroup -S -g 10001 aussietv && \
    adduser -S -D -u 10001 -G aussietv aussietv && \
    mkdir -p /data && chown -R aussietv:aussietv /data
COPY --from=build /out/aussietv /usr/local/bin/aussietv
USER aussietv
ENV XDG_CACHE_HOME=/data \
    AUSSIETV_MODE=server \
    AUSSIETV_JELLYFIN_BIND=0.0.0.0:45782 \
    AUSSIETV_JELLYFIN_REGION=Melbourne
WORKDIR /data
EXPOSE 45782
ENTRYPOINT ["/usr/local/bin/aussietv"]
