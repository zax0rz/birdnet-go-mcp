package server

import (
	"github.com/zax0rz/birdnet-go-mcp/internal/client"

	"github.com/mark3labs/mcp-go/server"
)

// NewServer builds and registers tools, resources, and prompts for birdnet-go-mcp
func NewServer(api *client.BirdNETClient) *server.MCPServer {
	s := server.NewMCPServer(
		"birdnet-go",
		"1.0.0",
		server.WithDescription("BirdNET-Go Model Context Protocol server for Leopold and the birdz0rz observatory"),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	RegisterTools(s, api)
	RegisterResources(s, api)
	RegisterPrompts(s, api)

	return s
}
