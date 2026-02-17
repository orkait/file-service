# ── Build stage ──────────────────────────────────────────────
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o file-service .

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/file-service .
COPY .env .env

# Memory tuning for 1GB RAM environment
# GOMEMLIMIT: hard memory limit for Go runtime (~800MB, leaving 200MB for OS)
# GOGC: trigger GC more aggressively (default 100, lower = more frequent GC)
ENV GOMEMLIMIT=800MiB
ENV GOGC=50

# Profiling — set to "true" to enable /debug/pprof and /metrics/* endpoints
ENV ENABLE_PROFILING=true

# Disable rate limiter during stress tests
ENV DISABLE_RATE_LIMITER=true

EXPOSE 8080

CMD ["./file-service"]
