# Ultra-minimal - Your Go code uses Docker API, not CLI
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build static binary
RUN CGO_ENABLED=0 \
    go build \
    -ldflags='-w -s' \
    -trimpath \
    -o app \
    ./cmd/worker

RUN ls -lh /build/app

# Final stage: Pure distroless - NO DOCKER CLI
FROM gcr.io/distroless/static-debian12:nonroot

LABEL \
    org.opencontainers.image.title="Namor" \
    org.opencontainers.image.description="Lightweight. Realtime. Orchestration." \
    org.opencontainers.image.source="https://github.com/rndmcodeguy20/namor"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/app /worker/app

USER nonroot

WORKDIR /worker

ENTRYPOINT ["./app"]