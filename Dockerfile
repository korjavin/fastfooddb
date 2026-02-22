FROM golang:1.24-alpine AS builder
WORKDIR /app

# Cache dependencies separately
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata su-exec
WORKDIR /app

COPY --from=builder /app/server .
COPY entrypoint.sh /entrypoint.sh

# Create non-root user and switch to it
RUN chmod +x /entrypoint.sh && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
CMD ["./server"]
