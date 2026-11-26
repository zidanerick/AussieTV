# Multi-architecture experimental server. Feed-only by default; web UI opt-in.
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY . .
RUN go run ./tools/iconbuilder && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags="-s -w" -o /out/aussietv .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg font-dejavu && \
    addgroup -S -g 10001 aussietv && \
    adduser -S -D -u 10001 -G aussietv aussietv && \
    mkdir -p /data && chown -R aussietv:aussietv /data
COPY --from=build /out/aussietv /usr/local/bin/aussietv
COPY LICENSE /usr/share/doc/aussietv/LICENSE
LABEL org.opencontainers.image.source="https://github.com/zidanerick/AussieTV"
USER 10001:10001
ENV XDG_CACHE_HOME=/data \
    AUSSIETV_MODE=server \
    AUSSIETV_JELLYFIN_BIND=0.0.0.0:45782 \
    AUSSIETV_JELLYFIN_REGION=Melbourne
WORKDIR /data
EXPOSE 45781 45782
ENTRYPOINT ["/usr/local/bin/aussietv"]
