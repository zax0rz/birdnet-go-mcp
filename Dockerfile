FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /birdnet-mcp ./cmd/birdnet-mcp

FROM alpine:latest
COPY --from=builder /birdnet-mcp /usr/local/bin/birdnet-mcp
ENTRYPOINT ["birdnet-mcp", "serve"]
