package server

import (
	"context"
	"fmt"

	"github.com/zax0rz/birdnet-go-mcp/internal/client"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterResources registers standard birdnet:// resources
func RegisterResources(s *server.MCPServer, api *client.BirdNETClient) {
	// 1. birdnet://station/health
	healthRes := mcp.NewResource(
		"birdnet://station/health",
		"Station Health & Telemetry",
		mcp.WithResourceDescription("Real-time telemetry for the outdoor Pi Zero mic stream and BirdNET-Go detector"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(healthRes, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		streams, err := api.GetStreamHealth(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch stream health: %w", err)
		}
		sysInfo, _ := api.GetSystemInfo(ctx)
		healthReport := map[string]any{
			"streams": streams,
			"system":  sysInfo,
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "birdnet://station/health",
				MIMEType: "application/json",
				Text:     formatEnvelope(healthReport),
			},
		}, nil
	})

	// 2. birdnet://detections/recent
	recentRes := mcp.NewResource(
		"birdnet://detections/recent",
		"Recent Detections",
		mcp.WithResourceDescription("The 10 most recent bird detections from the backyard observatory"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(recentRes, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		detections, err := api.GetRecentDetections(ctx, 10, 0.70, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch recent detections: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "birdnet://detections/recent",
				MIMEType: "application/json",
				Text:     formatEnvelope(detections),
			},
		}, nil
	})

	// 3. birdnet://species/summary
	summaryRes := mcp.NewResource(
		"birdnet://species/summary",
		"Species Activity Summary",
		mcp.WithResourceDescription("Aggregated species counts and first/last heard timestamps"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(summaryRes, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		summary, err := api.GetSpeciesSummary(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch species summary: %w", err)
		}
		if len(summary) > 20 {
			summary = summary[:20]
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "birdnet://species/summary",
				MIMEType: "application/json",
				Text:     formatEnvelope(summary),
			},
		}, nil
	})
}
