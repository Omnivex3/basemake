# syntax=docker/dockerfile:1
# Multi-stage build — produces a ~15MB Alpine image with basemake

# ── Stage 1: Build ──────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /basemake .

# ── Stage 2: Runtime ────────────────────────────────────────────
FROM alpine:3.24

RUN apk add --no-cache ca-certificates

COPY --from=builder /basemake /usr/local/bin/basemake

RUN adduser -D basemake
USER basemake

ENTRYPOINT ["basemake"]
CMD ["--help"]
