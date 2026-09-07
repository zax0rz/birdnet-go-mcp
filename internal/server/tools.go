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

// RegisterTools registers all 8 read-only, parallel-safe tools on the MCP server
func RegisterTools(s *server.MCPServer, api *client.BirdNETClient) {
	// 1. get_recent_detections
	recentTool := mcp.NewTool("get_recent_detections",
		mcp.WithDescription("Retrieve the most recent bird acoustic detections from the BirdNET-Go observatory. Returns species names, confidence scores, timestamps, and audio clip URLs. Use this as the primary tool for 'what birds were heard recently?' questions; use search_detections for historical lookups or get_today_summary for aggregates. All output is capped at 8KB to protect the agent context window."),
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
		mcp.WithDescription("Search historical detection records by date, species name/code, and minimum confidence. Use this to answer questions about a specific day, date range, or species across history. For the latest activity use get_recent_detections instead. Dates must be YYYY-MM-DD."),
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
		mcp.WithDescription("Get comprehensive details for a single detection by its numeric ID (obtained from get_recent_detections or search_detections): weather conditions at detection time, confidence score, and audio clip URL. Use after an interesting detection ID has been identified."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithInteger("id",
			mcp.Required(),
			mcp.Description("Numeric detection ID, as returned by get_recent_detections or search_detections"),
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
		mcp.WithDescription("Get an aggregate summary of today's observatory activity: per-species detection counts plus first and last heard times. Use for daily briefings or 'what happened today' questions."),
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
		mcp.WithDescription("List species detected for the first time ever, or newly arrived this season/year. Use this to spot migration events, rare visitors, or new residents."),
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
		mcp.WithDescription("Get real-time health telemetry for the monitoring station: audio stream state and ingest throughput, packet loss, and host uptime/CPU. Use this to diagnose a silent station or missing detections before other tools."),
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
		mcp.WithDescription("Resolve a detection's audio clip (.wav) into a full HTTP URL on the clips file server, for sharing or embedding in messages and web pages. The URL is only reachable from the same network as the clips server. Use get_audio_clip_base64 instead when the audio data must be returned inline to the model."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("clip_name",
			mcp.Required(),
			mcp.Description("Clip file name exactly as returned in a detection record, e.g. 'sialia_sialis_96p_20260829T192351Z.wav'"),
		),
		mcp.WithString("detection_date",
			mcp.Description("Detection date in YYYY-MM-DD format; used as a fallback to locate the clip's folder when the filename embeds no parseable date"),
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
		mcp.WithDescription("Download a detection's audio clip (.wav) and return it base64-encoded, for multimodal models that accept audio input directly. WARNING: a 15-second clip is roughly 350,000-500,000 tokens. Call sparingly for single-clip verification only — never inside high-frequency or scheduled workflows. Use get_audio_clip for a lightweight shareable URL instead."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("clip_name",
			mcp.Required(),
			mcp.Description("Clip file name exactly as returned in a detection record, e.g. 'sialia_sialis_96p_20260829T192351Z.wav'"),
		),
		mcp.WithString("detection_date",
			mcp.Description("Detection date in YYYY-MM-DD format; used as a fallback to locate the clip's folder when the filename embeds no parseable date"),
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

