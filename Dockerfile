# Multi-stage build for smaller image
FROM golang:1.22-alpine AS builder

# Install dependencies in alphabetical order
RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries with CGO disabled for smaller static binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o relay ./cmd/relay && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent ./cmd/agent && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o client ./cmd/client

# Final stage with specific alpine version
FROM alpine:3.18

# Install ca-certificates and create user in one layer
RUN apk --no-cache add ca-certificates && \
    addgroup -g 1001 -S tunnel && \
    adduser -u 1001 -S tunnel -G tunnel

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/relay .
COPY --from=builder /app/agent .
COPY --from=builder /app/client .

# Create directories for certificates and logs
RUN mkdir -p /app/certs /app/logs && \
    chown -R tunnel:tunnel /app

# Switch to non-root user
USER tunnel

# Expose HTTPS port
EXPOSE 443

# Default command runs relay server
CMD ["./relay", "-addr", ":443"]
