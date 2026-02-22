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
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080
ENTRYPOINT ["./server"]
