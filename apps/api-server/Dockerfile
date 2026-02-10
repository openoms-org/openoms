# ── Build stage ──────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Copy workspace and dependency files first (layer caching)
COPY go.work go.work.sum ./
COPY apps/api-server/go.mod apps/api-server/go.sum ./apps/api-server/
COPY packages/allegro-go-sdk/go.mod ./packages/allegro-go-sdk/
COPY packages/inpost-go-sdk/go.mod ./packages/inpost-go-sdk/
COPY packages/dhl-go-sdk/go.mod ./packages/dhl-go-sdk/
COPY packages/order-engine/go.mod ./packages/order-engine/

RUN cd apps/api-server && go mod download

# Copy source code
COPY apps/api-server/ ./apps/api-server/
COPY packages/ ./packages/

RUN cd apps/api-server && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

# ── Runtime stage ────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

COPY --from=builder /server /server
COPY apps/api-server/migrations/ /migrations/

EXPOSE 8080
ENTRYPOINT ["/server"]
