package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zax0rz/birdnet-go-mcp/internal/client"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// MaxEnvelopeChars caps tool output to ~8KB to protect agent context from flooding
	MaxEnvelopeChars = 8000
)

// formatEnvelope marshals data to indented JSON and enforces the 8KB envelope cap
func formatEnvelope(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to format output: %s"}`, err.Error())
	}

	if len(data) > MaxEnvelopeChars {
		capped := string(data[:MaxEnvelopeChars])
		// find last newline for clean break
		lastNL := strings.LastIndex(capped, "\n")
		if lastNL > MaxEnvelopeChars/2 {
			capped = capped[:lastNL]
		}
		return fmt.Sprintf("%s\n\n... [Response capped at 8KB envelope to protect context window]", capped)
	}

	return string(data)
}

// RegisterTools registers all 7 read-only, parallel-safe tools on the MCP server
func RegisterTools(s *server.MCPServer, api *client.BirdNETClient) {
	// 1. get_recent_detections
	recentTool := mcp.NewTool("get_recent_detections",
		mcp.WithDescription("Get recent bird detections from the backyard observatory. Returns species names, confidence scores, timestamps, and audio clip URLs."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithInteger("limit",
			mcp.Description("Number of detections to return (default: 10, max: 25)"),
			mcp.DefaultNumber(10),
			mcp.Min(1),
			mcp.Max(25),
		),
		mcp.WithNumber("min_confidence",
			mcp.Description("Minimum confidence score between 0.0 and 1.0 (default: 0.70)"),
			mcp.DefaultNumber(0.70),
			mcp.Min(0.0),
			mcp.Max(1.0),
		),
		mcp.WithString("species_code",
			mcp.Description("Optional 6-letter species code filter (e.g. 'easblu', 'blujay', 'rebwoo')"),
		),
	)

	s.AddTool(recentTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := req.GetInt("limit", 10)
		if limit > 25 {
			limit = 25
		}
		minConf := req.GetFloat("min_confidence", 0.70)
		speciesCode := req.GetString("species_code", "")

		detections, err := api.GetRecentDetections(ctx, limit, minConf, speciesCode)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to query recent detections: %v", err)), nil
		}

		return mcp.NewToolResultText(formatEnvelope(detections)), nil
	})

	// 2. search_detections
	searchTool := mcp.NewTool("search_detections",
		mcp.WithDescription("Search historical detections in SQLite by date (YYYY-MM-DD), species name/code, and minimum confidence."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("date",
			mcp.Description("Filter by date in YYYY-MM-DD format (e.g. '2026-08-29')"),
		),
		mcp.WithString("species",
			mcp.Description("Species common name or 6-letter code (e.g. 'Eastern Bluebird' or 'easblu')"),
		),
		mcp.WithNumber("min_confidence",
			mcp.Description("Minimum confidence score between 0.0 and 1.0 (default: 0.70)"),
			mcp.DefaultNumber(0.70),
			mcp.Min(0.0),
			mcp.Max(1.0),
		),
		mcp.WithInteger("limit",
			mcp.Description("Max results to return (default: 10, max: 25)"),
			mcp.DefaultNumber(10),
			mcp.Min(1),
			mcp.Max(25),
		),
	)

	s.AddTool(searchTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		date := req.GetString("date", "")
		species := req.GetString("species", "")
		minConf := req.GetFloat("min_confidence", 0.70)
		limit := req.GetInt("limit", 10)
		if limit > 25 {
			limit = 25
		}

		results, err := api.SearchDetections(ctx, date, species, minConf, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
		}

		return mcp.NewToolResultText(formatEnvelope(results)), nil
	})

	// 3. get_detection_detail
	detailTool := mcp.NewTool("get_detection_detail",
		mcp.WithDescription("Get comprehensive details for a specific detection ID, including weather at detection time, confidence, and audio clip URL."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithInteger("id",
			mcp.Required(),
			mcp.Description("Detection ID number"),
		),
	)

	s.AddTool(detailTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		detail, err := api.GetDetectionDetail(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get detection %d: %v", id, err)), nil
		}

		return mcp.NewToolResultText(formatEnvelope(detail)), nil
	})

	// 4. get_today_summary
	summaryTool := mcp.NewTool("get_today_summary",
		mcp.WithDescription("Get an aggregate summary of species activity recorded by the observatory (species count, total calls, first and last heard times)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(summaryTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		summary, err := api.GetSpeciesSummary(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch species summary: %v", err)), nil
		}

		// Cap summary list if large
		if len(summary) > 20 {
			summary = summary[:20]
		}

		return mcp.NewToolResultText(formatEnvelope(summary)), nil
	})

	// 5. get_new_arrivals
	arrivalsTool := mcp.NewTool("get_new_arrivals",
		mcp.WithDescription("Get species newly detected by the station (first time ever or new this season/year). Useful for monitoring migration."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(arrivalsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arrivals, err := api.GetNewArrivals(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch new arrivals: %v", err)), nil
		}

		if len(arrivals) > 20 {
			arrivals = arrivals[:20]
		}

		return mcp.NewToolResultText(formatEnvelope(arrivals)), nil
	})

	// 6. get_station_health
	healthTool := mcp.NewTool("get_station_health",
		mcp.WithDescription("Check real-time health of the outdoor station (Pi Zero 2 W mic RTSP stream, ffmpeg ingest, bytes/second, and CT 122 host uptime)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(healthTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streams, err := api.GetStreamHealth(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream health: %v", err)), nil
		}

		sysInfo, _ := api.GetSystemInfo(ctx)

		healthReport := map[string]any{
			"streams": streams,
			"system":  sysInfo,
		}

		return mcp.NewToolResultText(formatEnvelope(healthReport)), nil
	})

	// 7. get_audio_clip
	clipTool := mcp.NewTool("get_audio_clip",
		mcp.WithDescription("Resolve the full LAN URL for an audio clip on Caddy (:8091) so Leopold can attach it to Discord #birdz0rz messages."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("clip_name",
			mcp.Required(),
			mcp.Description("Clip file name from detection record (e.g. 'sialia_sialis_96p_20260829T192351Z.wav')"),
		),
		mcp.WithString("detection_date",
			mcp.Description("Optional detection date in YYYY-MM-DD format as fallback for folder path"),
		),
	)

	s.AddTool(clipTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clipName, err := req.RequireString("clip_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		detectionDate := req.GetString("detection_date", "")

		clipURL := client.ResolveClipURL(api.GetClipsBaseURL(), clipName, detectionDate)
		if clipURL == "" {
			return mcp.NewToolResultError("Invalid or unsafe clip name provided"), nil
		}

		resp := map[string]string{
			"clip_name": clipName,
			"url":       clipURL,
			"format":    "audio/x-wav",
			"notes":     "LAN-accessible URL. Download via curl/http and attach to Discord message.",
		}

		return mcp.NewToolResultText(formatEnvelope(resp)), nil
	})

	// 8. get_audio_clip_base64
	clipDataTool := mcp.NewTool("get_audio_clip_base64",
		mcp.WithDescription("Download and return base64-encoded audio (WAV) for a detection clip. Useful for multimodal models with audio input capabilities or off-LAN agents."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("clip_name",
			mcp.Required(),
			mcp.Description("Clip file name from detection record (e.g. 'sialia_sialis_96p_20260829T192351Z.wav')"),
		),
		mcp.WithString("detection_date",
			mcp.Description("Optional detection date in YYYY-MM-DD format as fallback for folder path"),
		),
	)

	s.AddTool(clipDataTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clipName, err := req.RequireString("clip_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		detectionDate := req.GetString("detection_date", "")

		// Download up to 2MB for base64 safety
		data, err := api.DownloadClip(ctx, clipName, detectionDate, 2*1024*1024)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch clip audio: %v", err)), nil
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		clipURL := client.ResolveClipURL(api.GetClipsBaseURL(), clipName, detectionDate)

		resp := map[string]any{
			"clip_name":  clipName,
			"url":        clipURL,
			"mime_type":  "audio/x-wav",
			"size_bytes": len(data),
			"base64":     encoded,
		}

		// Don't envelope truncate the base64 audio data
		b, err := json.Marshal(resp)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal audio payload: %v", err)), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	})
}

