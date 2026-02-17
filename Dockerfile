# Stage 1: Build
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build with CGO for sqlite3
COPY . .
RUN CGO_ENABLED=1 go build -o dc-management-tool ./cmd/server

# Stage 2: Runtime
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends sqlite3 curl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/dc-management-tool .
COPY --from=builder /build/migrations/ ./migrations/
COPY --from=builder /build/templates/ ./templates/
COPY --from=builder /build/static/ ./static/
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

RUN mkdir -p data static/uploads

EXPOSE 8080

ENTRYPOINT ["./docker-entrypoint.sh"]
