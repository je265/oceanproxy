# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/oceanproxy cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata 3proxy nginx curl bash jq

# Create app user
RUN addgroup -g 1001 -S oceanproxy && \
    adduser -u 1001 -S oceanproxy -G oceanproxy

# Create directories
RUN mkdir -p /var/log/oceanproxy /etc/3proxy /app/scripts && \
    chown -R oceanproxy:oceanproxy /var/log/oceanproxy /etc/3proxy

WORKDIR /app

# Copy binary and configs
COPY --from=builder /app/bin/oceanproxy .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/scripts ./scripts

# Make scripts executable
RUN chmod +x scripts/proxy/*.sh scripts/system/*.sh

# Switch to app user
USER oceanproxy

EXPOSE 8080

CMD ["./oceanproxy"]