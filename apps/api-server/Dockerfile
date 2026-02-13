# ── Build stage ──────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /src

# Copy workspace-level dependency files first (layer caching).
# go.work tells Go this is a multi-module workspace.
COPY go.work ./
COPY go.work.su[m] ./

# Copy go.mod (and go.sum where present) for every module in the workspace
# so that `go mod download` can fetch all dependencies in a cached layer.
COPY apps/api-server/go.mod apps/api-server/go.sum ./apps/api-server/
COPY packages/allegro-go-sdk/go.mod       ./packages/allegro-go-sdk/
COPY packages/amazon-sp-sdk/go.mod        ./packages/amazon-sp-sdk/
COPY packages/dhl-go-sdk/go.mod           ./packages/dhl-go-sdk/
COPY packages/dpd-go-sdk/go.mod           ./packages/dpd-go-sdk/
COPY packages/erli-go-sdk/go.mod          ./packages/erli-go-sdk/
COPY packages/fakturownia-go-sdk/go.mod   ./packages/fakturownia-go-sdk/
COPY packages/gls-go-sdk/go.mod           ./packages/gls-go-sdk/
COPY packages/inpost-go-sdk/go.mod        ./packages/inpost-go-sdk/
COPY packages/iof-parser/go.mod           ./packages/iof-parser/
COPY packages/mirakl-go-sdk/go.mod        ./packages/mirakl-go-sdk/
COPY packages/order-engine/go.mod         ./packages/order-engine/
COPY packages/ups-go-sdk/go.mod           ./packages/ups-go-sdk/
COPY packages/poczta-polska-go-sdk/go.mod ./packages/poczta-polska-go-sdk/
COPY packages/orlen-paczka-go-sdk/go.mod  ./packages/orlen-paczka-go-sdk/
COPY packages/fedex-go-sdk/go.mod         ./packages/fedex-go-sdk/
COPY packages/smsapi-go-sdk/go.mod        ./packages/smsapi-go-sdk/
COPY packages/woocommerce-go-sdk/go.mod   ./packages/woocommerce-go-sdk/
COPY packages/ebay-go-sdk/go.mod          ./packages/ebay-go-sdk/
COPY packages/kaufland-go-sdk/go.mod      ./packages/kaufland-go-sdk/
COPY packages/ksef-go-sdk/go.mod          ./packages/ksef-go-sdk/
COPY packages/olx-go-sdk/go.mod           ./packages/olx-go-sdk/

RUN cd apps/api-server && go mod download

# Copy full source for all workspace modules
COPY apps/api-server/ ./apps/api-server/
COPY packages/         ./packages/

# Build a statically-linked binary with stripped debug info
RUN cd apps/api-server && \
    CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-s -w" \
      -o /server \
      ./cmd/server

# Build a tiny healthcheck binary for distroless (no curl/wget)
RUN printf 'package main\nimport("net/http";"os")\nfunc main(){r,e:=http.Get("http://localhost:8080/health");if e!=nil||r.StatusCode!=200{os.Exit(1)}}' > /tmp/healthcheck.go && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /healthcheck /tmp/healthcheck.go

# ── Runtime stage ────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

# Import CA certs and timezone data from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary and healthcheck
COPY --from=builder /server /server
COPY --from=builder /healthcheck /healthcheck

# Copy migrations (used by the separate migrate service)
COPY apps/api-server/migrations/ /migrations/

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=5 \
  CMD ["/healthcheck"]

ENTRYPOINT ["/server"]
