# Dockerfile for GoTeleport Server
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o goteleport-server-db goteleport-server-db.go

# Production image
FROM alpine:3.18

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/goteleport-server-db .

# Copy config file
COPY server-config-db.json .

# Expose ports
EXPOSE 8080

# Run the server
CMD ["./goteleport-server-db", "server-config-db.json"]
