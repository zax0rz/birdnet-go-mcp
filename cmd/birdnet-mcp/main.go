package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/zax0rz/birdnet-go-mcp/internal/client"
	"github.com/zax0rz/birdnet-go-mcp/internal/server"

	mcpServer "github.com/mark3labs/mcp-go/server"
)

var (
	version   = "1.0.0"
	commit    = "none"
	date      = "unknown"
	builtBy   = "source"
)

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func printBanner() {
	fmt.Fprintf(os.Stderr, "🐦 birdnet-go-mcp v%s (MCP Server & CLI for BirdNET-Go)\n", version)
}

func printHelp() {
	printBanner()
	fmt.Fprintf(os.Stderr, `
USAGE:
  birdnet-mcp [command] [flags]

MCP SERVER COMMANDS:
  serve                 Run as an MCP server (auto-detected when stdin is piped)
    --stdio             Serve over stdio JSON-RPC 2.0 (default)
    --sse               Serve over Server-Sent Events (SSE) HTTP transport
    --port <port>       Port for SSE server (default: 8092)

CLI INSPECTION COMMANDS:
  status, health        Inspect outdoor station stream health & detector host
  recent                Display recent detections in a terminal table
    --limit <n>         Number of detections to show (default: 10)
    --min-conf <val>    Minimum confidence threshold 0.0-1.0 (default: 0.70)
    --species <code>    Filter by 6-letter species code (e.g. 'easblu')
  summary               Display species activity leaderboard
  arrivals              Display newly detected species (migration arrivals)
  clip <clip_name>      Resolve and optionally download an audio recording
    --download          Save the WAV audio file to disk
    --out <path>        Output file path (default: current directory)
  version               Display version and build information

GLOBAL FLAGS:
  --birdnet-url <url>   BirdNET-Go REST API URL (env: BIRDNET_BASE_URL, default: http://localhost:8080)
  --clips-url <url>     Audio clips server URL (env: CLIPS_BASE_URL, default: http://localhost:8091)
  --user <user>         HTTP Basic Auth username (env: BIRDNET_USERNAME)
  --pass <pass>         HTTP Basic Auth password (env: BIRDNET_PASSWORD)
  --token <token>       Bearer authorization token (env: BIRDNET_AUTH_TOKEN)

EXAMPLES:
  # Run as MCP server in Claude Desktop / OpenClaw (stdio mode)
  birdnet-mcp serve --stdio

  # Start SSE server for remote agents
  birdnet-mcp serve --sse --port 8092

  # Quick terminal check of the outdoor mic
  birdnet-mcp status

  # View the 5 most recent high-confidence detections
  birdnet-mcp recent --limit 5 --min-conf 0.85
`)
}

func main() {
	// If stdin is piped/redirected and no arguments were passed, automatically run MCP stdio server
	if len(os.Args) == 1 && !isTerminal(os.Stdin) {
		runStdioServer("", "", 5*time.Second)
		return
	}

	if len(os.Args) < 2 {
		printHelp()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "serve":
		handleServe(args)
	case "status", "health":
		handleStatus(args)
	case "recent":
		handleRecent(args)
	case "summary":
		handleSummary(args)
	case "arrivals":
		handleArrivals(args)
	case "clip":
		handleClip(args)
	case "version", "-v", "--version":
		fmt.Printf("birdnet-go-mcp %s (commit: %s, date: %s, builtBy: %s)\n", version, commit, date, builtBy)
	case "help", "-h", "--help":
		printHelp()
	default:
		// If first arg starts with a dash, check if user ran flags directly without subcommand
		if strings.HasPrefix(cmd, "-") {
			// If stdin is not terminal, assume stdio server with flags
			if !isTerminal(os.Stdin) {
				handleServe(os.Args[1:])
				return
			}
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'birdnet-mcp help' for usage.\n", cmd)
		os.Exit(1)
	}
}

func getEnvOrDefault(envKey, defVal string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return defVal
}

func buildClientFromFlags(fs *flag.FlagSet, birdnetURL, clipsURL, user, pass, token *string) *client.BirdNETClient {
	bURL := *birdnetURL
	if bURL == "" {
		bURL = getEnvOrDefault("BIRDNET_BASE_URL", "http://localhost:8080")
	}

	cURL := *clipsURL
	if cURL == "" {
		cURL = getEnvOrDefault("CLIPS_BASE_URL", "http://localhost:8091")
	}

	c := client.NewClient(bURL, cURL, 6*time.Second)

	u := *user
	if u == "" {
		u = os.Getenv("BIRDNET_USERNAME")
	}
	p := *pass
	if p == "" {
		p = os.Getenv("BIRDNET_PASSWORD")
	}
	if u != "" || p != "" {
		c.SetAuth(u, p)
	}

	t := *token
	if t == "" {
		t = os.Getenv("BIRDNET_AUTH_TOKEN")
	}
	if t != "" {
		c.SetToken(t)
	}

	return c
}

func handleServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	stdioMode := fs.Bool("stdio", true, "Run in stdio mode (JSON-RPC on stdin/stdout)")
	sseMode := fs.Bool("sse", false, "Run in SSE HTTP server mode")
	port := fs.Int("port", 8092, "Port for SSE server")
	birdnetURL := fs.String("birdnet-url", "", "BirdNET-Go REST API URL")
	clipsURL := fs.String("clips-url", "", "Audio clips server URL")
	user := fs.String("user", "", "Basic Auth username")
	pass := fs.String("pass", "", "Basic Auth password")
	token := fs.String("token", "", "Bearer token")
	_ = fs.Parse(args)

	// Logging must strictly go to stderr
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	apiClient := buildClientFromFlags(fs, birdnetURL, clipsURL, user, pass, token)
	mcpSrv := server.NewServer(apiClient)

	if *sseMode {
		addr := fmt.Sprintf("0.0.0.0:%d", *port)
		log.Printf("[birdnet-mcp] Starting SSE server on %s", addr)
		log.Printf("[birdnet-mcp] Endpoint: http://%s/sse", addr)
		sse := mcpServer.NewSSEServer(mcpSrv)
		if err := sse.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[birdnet-mcp] SSE server failed: %v", err)
		}
		return
	}

	if *stdioMode || true {
		log.Printf("[birdnet-mcp] Initializing MCP server v%s (stdio)", version)
		log.Printf("[birdnet-mcp] Target API: %s", apiClient.GetBaseURL())
		if err := mcpServer.ServeStdio(mcpSrv); err != nil {
			log.Fatalf("[birdnet-mcp] Server exited: %v", err)
		}
	}
}

func runStdioServer(birdnetURL, clipsURL string, timeout time.Duration) {
	log.SetOutput(os.Stderr)
	apiClient := client.NewClient(birdnetURL, clipsURL, timeout)
	mcpSrv := server.NewServer(apiClient)
	if err := mcpServer.ServeStdio(mcpSrv); err != nil {
		log.Fatalf("[birdnet-mcp] Server exited: %v", err)
	}
}

func handleStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	birdnetURL := fs.String("birdnet-url", "", "BirdNET-Go REST API URL")
	clipsURL := fs.String("clips-url", "", "Audio clips server URL")
	user := fs.String("user", "", "Basic Auth username")
	pass := fs.String("pass", "", "Basic Auth password")
	token := fs.String("token", "", "Bearer token")
	_ = fs.Parse(args)

	apiClient := buildClientFromFlags(fs, birdnetURL, clipsURL, user, pass, token)
	ctx := context.Background()

	fmt.Printf("🔍 Connecting to BirdNET-Go at %s...\n\n", apiClient.GetBaseURL())

	streams, err := apiClient.GetStreamHealth(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to query stream health: %v\n", err)
		os.Exit(1)
	}

	sysInfo, err := apiClient.GetSystemInfo(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Note: Could not fetch system info: %v\n", err)
	}

	fmt.Println("=== AUDIO STREAMS ===")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tHEALTH\tSTATE\tTHROUGHPUT\tLAST RECEIVED")
	for _, s := range streams {
		healthStr := "🟢 HEALTHY"
		if !s.IsHealthy {
			healthStr = "🔴 UNHEALTHY"
		}
		thru := fmt.Sprintf("%.1f KB/s", s.BytesPerSecond/1024.0)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", s.Name, s.Type, healthStr, s.ProcessState, thru, s.LastDataReceived)
	}
	w.Flush()

	if sysInfo != nil {
		fmt.Println("\n=== DETECTOR HOST ===")
		fmt.Printf("Host:         %s (%s, %s)\n", sysInfo.Hostname, sysInfo.OSDisplay, sysInfo.Architecture)
		fmt.Printf("CPU:          %s (%d cores)\n", sysInfo.CPUModel, sysInfo.NumCPU)
		fmt.Printf("Uptime:       %s (Host) | %s (BirdNET-Go)\n", formatDuration(sysInfo.UptimeSeconds), formatDuration(sysInfo.AppUptimeSeconds))
		fmt.Printf("Kernel:       %s\n", sysInfo.KernelVersion)
		fmt.Printf("Environment:  %s\n", sysInfo.Environment)
	}
}

func handleRecent(args []string) {
	fs := flag.NewFlagSet("recent", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Max detections to display")
	minConf := fs.Float64("min-conf", 0.70, "Minimum confidence (0.0 - 1.0)")
	species := fs.String("species", "", "Filter by species code")
	birdnetURL := fs.String("birdnet-url", "", "BirdNET-Go REST API URL")
	clipsURL := fs.String("clips-url", "", "Audio clips server URL")
	user := fs.String("user", "", "Basic Auth username")
	pass := fs.String("pass", "", "Basic Auth password")
	token := fs.String("token", "", "Bearer token")
	_ = fs.Parse(args)

	apiClient := buildClientFromFlags(fs, birdnetURL, clipsURL, user, pass, token)
	ctx := context.Background()

	detections, err := apiClient.GetRecentDetections(ctx, *limit, *minConf, *species)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to query recent detections: %v\n", err)
		os.Exit(1)
	}

	if len(detections) == 0 {
		fmt.Println("No recent detections matched criteria.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTIME\tSPECIES\tCOMMON NAME\tCONF\tNEW?\tCLIP URL")
	for _, d := range detections {
		newFlag := "-"
		if d.IsNewSpecies {
			newFlag = "★ NEW"
		} else if d.IsNewThisSeason {
			newFlag = "SEASON"
		}
		confStr := fmt.Sprintf("%.0f%%", d.Confidence*100)
		fmt.Fprintf(w, "#%d\t%s %s\t%s\t%s\t%s\t%s\t%s\n",
			d.ID, d.Date, d.Time, d.SpeciesCode, d.CommonName, confStr, newFlag, d.ClipURL)
	}
	w.Flush()
}

func handleSummary(args []string) {
	fs := flag.NewFlagSet("summary", flag.ExitOnError)
	birdnetURL := fs.String("birdnet-url", "", "BirdNET-Go REST API URL")
	clipsURL := fs.String("clips-url", "", "Audio clips server URL")
	user := fs.String("user", "", "Basic Auth username")
	pass := fs.String("pass", "", "Basic Auth password")
	token := fs.String("token", "", "Bearer token")
	_ = fs.Parse(args)

	apiClient := buildClientFromFlags(fs, birdnetURL, clipsURL, user, pass, token)
	ctx := context.Background()

	summary, err := apiClient.GetSpeciesSummary(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to query summary: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "RANK\tCOMMON NAME\tSCIENTIFIC NAME\tCOUNT\tAVG CONF\tMAX CONF\tLAST HEARD")
	for i, s := range summary {
		fmt.Fprintf(w, "#%d\t%s\t%s\t%d\t%.0f%%\t%.0f%%\t%s\n",
			i+1, s.CommonName, s.ScientificName, s.Count, s.AvgConfidence*100, s.MaxConfidence*100, s.LastHeard)
	}
	w.Flush()
}

func handleArrivals(args []string) {
	fs := flag.NewFlagSet("arrivals", flag.ExitOnError)
	birdnetURL := fs.String("birdnet-url", "", "BirdNET-Go REST API URL")
	clipsURL := fs.String("clips-url", "", "Audio clips server URL")
	user := fs.String("user", "", "Basic Auth username")
	pass := fs.String("pass", "", "Basic Auth password")
	token := fs.String("token", "", "Bearer token")
	_ = fs.Parse(args)

	apiClient := buildClientFromFlags(fs, birdnetURL, clipsURL, user, pass, token)
	ctx := context.Background()

	arrivals, err := apiClient.GetNewArrivals(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to query new arrivals: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "SPECIES\tSCIENTIFIC NAME\tFIRST RECORDED")
	for _, a := range arrivals {
		fmt.Fprintf(w, "%s\t%s\t%s\n", a.CommonName, a.ScientificName, a.FirstHeardDate)
	}
	w.Flush()
}

func handleClip(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: birdnet-mcp clip <clip_name> [--download] [--out <file>]\n")
		os.Exit(1)
	}

	clipName := args[0]
	fs := flag.NewFlagSet("clip", flag.ExitOnError)
	download := fs.Bool("download", false, "Download the audio recording to disk")
	outPath := fs.String("out", "", "Output filename")
	clipsURL := fs.String("clips-url", "", "Audio clips server URL")
	user := fs.String("user", "", "Basic Auth username")
	pass := fs.String("pass", "", "Basic Auth password")
	token := fs.String("token", "", "Bearer token")
	_ = fs.Parse(args[1:])

	apiClient := buildClientFromFlags(fs, new(string), clipsURL, user, pass, token)
	resolvedURL := client.ResolveClipURL(apiClient.GetClipsBaseURL(), clipName, "")

	fmt.Printf("Clip Name: %s\n", clipName)
	fmt.Printf("LAN URL:   %s\n", resolvedURL)

	if *download {
		targetFile := *outPath
		if targetFile == "" {
			targetFile = clipName
		}
		fmt.Printf("Downloading to %s...\n", targetFile)
		data, err := apiClient.DownloadClip(context.Background(), clipName, "", 10*1024*1024)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Download failed: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(targetFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Saving file failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Saved %d bytes to %s\n", len(data), targetFile)
	}
}

func formatDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}
