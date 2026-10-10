# =========================================================================
# Multi-Stage Dockerfile: Fintech User Test Sandbox
# Ringan (~18 MB), Cepat, dan Siap untuk Local Docker Compose & Cloudflare Tunnel
# =========================================================================

# Tahap 1: Builder
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Salin go.mod & go.sum
COPY go.mod go.sum* ./
RUN go mod download || true

# Salin seluruh kode sumber
COPY . .

# Kompilasi binary Go statis untuk target Linux
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/sandbox ./cmd/sandbox

# Tahap 2: Minimal Runtime Runner
FROM alpine:3.20

WORKDIR /app

# Tambahkan sertifikat SSL & curl untuk healthcheck
RUN apk --no-cache add ca-certificates curl tzdata

# Salin binary hasil kompilasi
COPY --from=builder /app/sandbox /app/sandbox

# Default port
EXPOSE 8080
ENV PORT=8080

# Health check probe
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

# Jalankan binary
ENTRYPOINT ["/app/sandbox"]

