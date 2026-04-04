# =============================================================
# Stage 1: Build UI static assets
# =============================================================
FROM node:22-alpine AS ui-builder

WORKDIR /build/ui

# Install dependencies first (layer caching).
COPY ui/package.json ui/package-lock.json ./
RUN npm ci --ignore-scripts

# Copy UI source and build.
COPY ui/ ./
RUN npm run build

# =============================================================
# Stage 2: Build Go binary
# =============================================================
FROM golang:1.26-alpine AS go-builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go module files first (layer caching).
COPY go.mod go.sum ./
RUN go mod download

# Copy source code.
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

# Build static binary (CGO_ENABLED=0 with modernc.org/sqlite pure Go).
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o portal ./cmd

# =============================================================
# Stage 3: Runtime (minimal Alpine)
# =============================================================
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy binary from builder.
COPY --from=go-builder /build/portal .

# Copy built UI assets.
COPY --from=ui-builder /build/ui/build/ ui/build/

# Create data directory for SQLite volume mount.
RUN mkdir -p /data

# Create non-root user (UID 1001 per WitFoo Way).
RUN addgroup -g 1001 -S portal && \
    adduser -u 1001 -S portal -G portal && \
    chown -R portal:portal /app /data

USER portal

ENV DD_DB_PATH=/data/portal.db
ENV DD_UI_PATH=/app/ui/build

EXPOSE 8080 8443

# Health check (wget -O /dev/null per WitFoo Way, not --spider).
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/health || exit 1

ENTRYPOINT ["./portal"]
