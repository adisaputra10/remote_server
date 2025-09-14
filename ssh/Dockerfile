# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk update && apk add --no-cache git ca-certificates tzdata

# Create appuser
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o relay ./cmd/relay
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o agent ./cmd/agent
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o client ./cmd/client

# Final stage - multi-stage build for smaller image
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy binaries from builder
COPY --from=builder /build/relay /relay
COPY --from=builder /build/agent /agent
COPY --from=builder /build/client /client

# Use non-root user
USER appuser

# Default to relay
ENTRYPOINT ["/relay"]
CMD ["serve"]

# Expose ports
EXPOSE 8080