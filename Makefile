.PHONY: all build build-mac test clean deploy-mac

BINARY_NAME=birdnet-mcp
GO_BIN?=/usr/local/go/bin/go
ifeq (,$(wildcard $(GO_BIN)))
	GO_BIN=go
endif

all: test build build-mac

build:
	CGO_ENABLED=0 $(GO_BIN) build -ldflags="-s -w" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/birdnet-mcp

build-mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO_BIN) build -ldflags="-s -w" -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/birdnet-mcp

test:
	$(GO_BIN) test -v ./...

clean:
	rm -rf dist/

deploy-mac: build-mac
	ssh mac-mini 'mkdir -p ~/.openclaw/mcp-servers/birdnet-go-mcp/bin'
	scp dist/$(BINARY_NAME)-darwin-arm64 zach@mac-mini:~/.openclaw/mcp-servers/birdnet-go-mcp/bin/$(BINARY_NAME)
	ssh mac-mini 'chmod +x ~/.openclaw/mcp-servers/birdnet-go-mcp/bin/$(BINARY_NAME) && codesign -s - --force ~/.openclaw/mcp-servers/birdnet-go-mcp/bin/$(BINARY_NAME)'
