# syntax=docker/dockerfile:1

ARG UI_DIR=cmd/omnihance-a3-agent/omnihance-a3-agent-ui

FROM --platform=$BUILDPLATFORM node:22-alpine AS ui-builder
ARG UI_DIR
RUN npm install --global pnpm@11
WORKDIR /src/ui
COPY ${UI_DIR}/package.json ${UI_DIR}/pnpm-lock.yaml ${UI_DIR}/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY ${UI_DIR}/ ./
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS go-builder
ARG UI_DIR
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /src/ui/dist ./${UI_DIR}/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /out/omnihance-a3-agent ./cmd/omnihance-a3-agent

FROM alpine:3.22

LABEL org.opencontainers.image.source="https://github.com/omnihance/omnihance-a3-agent" \
      org.opencontainers.image.description="Web interface to control A3 Online MMO game server" \
      org.opencontainers.image.licenses="GPL-3.0"

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 omnihance \
    && adduser -D -u 1000 -G omnihance omnihance \
    && mkdir -p /data \
    && chown omnihance:omnihance /data

COPY --from=go-builder /out/omnihance-a3-agent /usr/local/bin/omnihance-a3-agent

ENV RUNNING_IN_DOCKER=true \
    PORT=8080 \
    LOG_DIR=/data/logs \
    DATABASE_URL="file:/data/omnihance-a3-agent.db?cache=shared&mode=rwc" \
    REVISIONS_DIRECTORY=/data/.revisions \
    BACKUPS_DIRECTORY=/data/.backups \
    DIRECTORY_DOWNLOADS_DIRECTORY=/data/.directory-download

USER omnihance
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --spider --quiet "http://127.0.0.1:${PORT}/health" || exit 1

ENTRYPOINT ["/usr/local/bin/omnihance-a3-agent"]
