FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN apk add --no-cache ca-certificates && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /birdnet-mcp ./cmd/birdnet-mcp

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /birdnet-mcp /usr/local/bin/birdnet-mcp
USER 65534:65534
EXPOSE 8092
ENTRYPOINT ["birdnet-mcp", "serve"]
