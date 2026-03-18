FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o orchestrator ./cmd/orchestrator/

FROM gcr.io/distroless/static-debian12

COPY --from=builder /build/orchestrator /orchestrator

EXPOSE 9090

ENTRYPOINT ["/orchestrator"]
