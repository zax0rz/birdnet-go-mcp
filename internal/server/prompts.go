package server

import (
	"context"
	"fmt"

	"github.com/zax0rz/birdnet-go-mcp/internal/client"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterPrompts registers prompt workflow templates for Leopold
func RegisterPrompts(s *server.MCPServer, api *client.BirdNETClient) {
	// 1. daily_backyard_brief
	dailyBrief := mcp.NewPrompt("daily_backyard_brief",
		mcp.WithPromptDescription("Workflow template to generate Leopold's morning backyard observatory dispatch for Discord #birdz0rz"),
	)
	s.AddPrompt(dailyBrief, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		promptText := `You are Leopold VanDerlyn, resident ornithologist of #birdz0rz.
Please compile your morning backyard dispatch based on the observatory data:
1. Call birdnet__get_today_summary to review overnight and dawn activity.
2. Call birdnet__get_new_arrivals to check for newly arriving migrants or seasonal firsts.
3. Call birdnet__get_station_health to confirm the station (birdz0rz-pi) is running smoothly.
4. If an exciting species was heard, call birdnet__get_recent_detections with that species to get its clipName, and birdnet__get_audio_clip to get the LAN clip URL.
5. Compose your dispatch in your signature slow-pour Piedmont honey voice. Keep prose under 1600 characters, no markdown tables, and emphasize real natural history over corporate summaries.`

		return mcp.NewGetPromptResult(
			"Morning backyard observatory dispatch template",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(promptText)),
			},
		), nil
	})

	// 2. investigate_detection
	investigatePrompt := mcp.NewPrompt("investigate_detection",
		mcp.WithPromptDescription("Workflow template for inspecting an unusual or borderline bird detection"),
		mcp.WithArgument("detection_id",
			mcp.ArgumentDescription("ID of the detection to investigate"),
			mcp.RequiredArgument(),
		),
	)
	s.AddPrompt(investigatePrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		detectionID := req.Params.Arguments["detection_id"]
		promptText := fmt.Sprintf(`Please investigate detection ID %s:
1. Call birdnet__get_detection_detail with id=%s to inspect confidence, time of day, and audio clip info.
2. Cross-reference the species with regional Greenville SC observations using bird-brain__ebird_recent or bird-brain__ebird_notable.
3. Call birdnet__get_audio_clip to obtain the recording URL.
4. Give your ornithological assessment: is this a solid identification, an acoustic mimic, or a rare visitor worth flagging for Zach?`, detectionID, detectionID)

		return mcp.NewGetPromptResult(
			"Acoustic detection investigation template",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(promptText)),
			},
		), nil
	})
}
