FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o relay ./cmd/relay
RUN go build -o agent ./cmd/agent  
RUN go build -o client ./cmd/client

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/relay .
COPY --from=builder /app/agent .
COPY --from=builder /app/client .

EXPOSE 443

CMD ["./relay"]
