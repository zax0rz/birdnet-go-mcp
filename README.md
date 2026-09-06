# birdnet-go-mcp

<p align="center">
  <br>
  <strong>A high-performance Model Context Protocol (MCP) server & CLI for <a href="https://github.com/tphakala/birdnet-go">BirdNET-Go</a> bioacoustic observatories.</strong>
  <br>
  <em>Connect Claude, Antigravity, OpenClaw, and local LLMs to your backyard bird monitoring station.</em>
</p>

<p align="center">
  <a href="https://github.com/zax0rz/birdnet-go-mcp/releases"><img src="https://img.shields.io/github/v/release/zax0rz/birdnet-go-mcp?style=flat-square&color=3b82f6" alt="Latest Release"></a>
  <a href="https://www.npmjs.com/package/birdnet-go-mcp"><img src="https://img.shields.io/npm/v/birdnet-go-mcp?style=flat-square&color=cb3837&logo=npm" alt="npm package"></a>
  <a href="https://glama.ai/mcp/servers/zax0rz/birdnet-go-mcp"><img src="https://glama.ai/mcp/servers/zax0rz/birdnet-go-mcp/badges/score.svg" alt="Glama MCP Server"></a>
  <a href="https://github.com/zax0rz/birdnet-go-mcp/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-emerald?style=flat-square" alt="MIT License"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-Standard-purple?style=flat-square" alt="Model Context Protocol"></a>
  <a href="https://github.com/tphakala/birdnet-go"><img src="https://img.shields.io/badge/BirdNET--Go-v2-orange?style=flat-square" alt="BirdNET-Go"></a>
</p>

---

## Highlights

- **Native BirdNET-Go Support**: Interfaces directly with BirdNET-Go's v2 REST API over LAN or localhost. No raw database locking or unmaintained Python dependencies.
- **Instant `npx` Run**: Launch immediately with `npx -y birdnet-go-mcp` — zero Go toolchain required.
- **Zero-Dependency Static Binary**: Single Go binary (`CGO_ENABLED=0`) compiled for Linux, macOS, and Windows.
- **Dual-Mode (CLI + MCP)**: Human-friendly CLI for quick terminal health checks (`birdnet-mcp status`), plus full stdio & SSE MCP server for AI agents.
- **Context-Protected (8KB Envelope)**: Hard-capped output preventing multi-hundred detection queries from overflowing model context windows.
- **Read-Only & Parallel-Safe**: Omits all mutating/destructive endpoints. Annotates all tools with `readOnlyHint` and `idempotentHint` for fast parallel agent calls.
- **Audio & Clip Access**: Resolves LAN Caddy/Nginx `.wav` clip URLs, with built-in base64 audio streaming for multimodal models.

---

## Quick Start

### 1. Instant Run via NPX (Recommended for Claude Desktop & Node users)

No Go installation needed. Downloads the native binary for your platform automatically:

```bash
# Check station health in your terminal
npx -y birdnet-go-mcp status

# Or set target host
BIRDNET_BASE_URL="http://192.168.1.130:8080" npx -y birdnet-go-mcp recent
```

### 2. Install via Go

```bash
go install github.com/zax0rz/birdnet-go-mcp/cmd/birdnet-mcp@latest
```

### 3. Pre-Compiled Binaries

Download the latest static binary for your architecture from [GitHub Releases](https://github.com/zax0rz/birdnet-go-mcp/releases):
- `darwin-arm64` (Apple Silicon M1/M2/M3/M4)
- `darwin-amd64` (Intel Mac)
- `linux-amd64` (x86_64 servers, Proxmox LXC, Docker)
- `linux-arm64` (Raspberry Pi 4 / 5)
- `linux-armv7` (Raspberry Pi 2 / 3 / Zero 2 W)
- `windows-amd64`

---

## Client & Harness Setup

### Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

#### Option A: Using `npx` (Easiest)
```json
{
  "mcpServers": {
    "birdnet": {
      "command": "npx",
      "args": ["-y", "birdnet-go-mcp", "serve"],
      "env": {
        "BIRDNET_BASE_URL": "http://192.168.1.130:8080",
        "CLIPS_BASE_URL": "http://192.168.1.130:8091"
      }
    }
  }
}
```

#### Option B: Using Native Binary
```json
{
  "mcpServers": {
    "birdnet": {
      "command": "/usr/local/bin/birdnet-mcp",
      "args": ["serve"],
      "env": {
        "BIRDNET_BASE_URL": "http://192.168.1.130:8080",
        "CLIPS_BASE_URL": "http://192.168.1.130:8091"
      }
    }
  }
}
```

---

### Claude Code

Add directly via CLI:
```bash
claude mcp add birdnet -- npx -y birdnet-go-mcp serve
```

---

### Google Antigravity

In your Antigravity MCP configuration (`~/.gemini/antigravity/mcp/` or project settings):

```json
{
  "mcpServers": {
    "birdnet": {
      "command": "birdnet-mcp",
      "args": ["serve"],
      "env": {
        "BIRDNET_BASE_URL": "http://192.168.1.130:8080",
        "CLIPS_BASE_URL": "http://192.168.1.130:8091"
      }
    }
  }
}
```

---

### OpenClaw

In `~/.openclaw/openclaw.json`:

```json
{
  "mcp": {
    "servers": {
      "birdnet": {
        "command": "/Users/zach/.openclaw/mcp-servers/birdnet-go-mcp/bin/birdnet-mcp",
        "args": [],
        "env": {
          "BIRDNET_BASE_URL": "http://192.168.1.130:8080",
          "CLIPS_BASE_URL": "http://192.168.1.130:8091"
        },
        "toolFilter": {
          "include": ["*"]
        }
      }
    }
  }
}
```

In your agent's `tools.allow` list:
```json
"tools": {
  "allow": [
    "birdnet__get_recent_detections",
    "birdnet__search_detections",
    "birdnet__get_detection_detail",
    "birdnet__get_today_summary",
    "birdnet__get_new_arrivals",
    "birdnet__get_station_health",
    "birdnet__get_audio_clip",
    "birdnet__get_audio_clip_base64"
  ]
}
```

---

### Remote / Headless Agents (SSE HTTP Mode)

If your agent runs on another machine or in the cloud without access to local stdio:

```bash
# Start background SSE server on port 8092
birdnet-mcp serve --sse --port 8092
```

Point your agent to:
`http://<your-server-ip>:8092/sse`

---

## Interactive CLI Commands

`birdnet-mcp` is a full CLI tool for humans as well as an MCP server for agents.

### Check Station & RTSP Mic Health
```bash
$ birdnet-mcp status

🔍 Connecting to BirdNET-Go at http://192.168.1.130:8080...

=== AUDIO STREAMS ===
NAME          TYPE   HEALTH      STATE     THROUGHPUT   LAST RECEIVED
birdz0rz-pi   rtsp   🟢 HEALTHY   running   74.8 KB/s    2026-09-04T20:38:53-04:00

=== DETECTOR HOST ===
Host:         birdz0rz (Debian Linux, x86_64)
CPU:          AMD Ryzen 5 PRO 2400G (2 cores)
Uptime:       15d 3h 6m (Host) | 6d 4h 47m (BirdNET-Go)
Kernel:       7.0.14-12-pve
Environment:  LXC
```

### View Recent Sightings
```bash
$ birdnet-mcp recent --limit 5 --min-conf 0.80

ID     TIME                  SPECIES   COMMON NAME        CONF   NEW?   CLIP URL
#129   2026-09-04 19:59:35   easblu    Eastern Bluebird   84%    -      http://192.168.1.130:8091/2026/09/sialia_sialis_84p_20260904T195937Z.wav
#128   2026-09-04 19:58:17   easblu    Eastern Bluebird   95%    -      http://192.168.1.130:8091/2026/09/sialia_sialis_95p_20260904T195819Z.wav
#127   2026-09-04 19:05:23   blujay    Blue Jay           84%    -      http://192.168.1.130:8091/2026/09/cyanocitta_cristata_84p_20260904T190525Z.wav
#126   2026-09-04 18:46:16   carwre    Carolina Wren      96%    -      http://192.168.1.130:8091/2026/09/thryothorus_ludovicianus_96p_20260904T184618Z.wav
#124   2026-09-04 18:35:34   houfin    House Finch        90%    -      http://192.168.1.130:8091/2026/09/haemorhous_mexicanus_90p_20260904T183536Z.wav
```

### Download Audio Recording
```bash
$ birdnet-mcp clip sialia_sialis_95p_20260904T195819Z.wav --download --out bluebird.wav
✅ Saved 1440044 bytes to bluebird.wav
```

---

## MCP Tool Reference

| Tool | Parameters | Description |
| :--- | :--- | :--- |
| `get_recent_detections` | `limit` (int, 1-25)<br>`min_confidence` (0.0-1.0)<br>`species_code` (string) | Fetches the most recent bird acoustic detections with confidence scores, timestamps, and audio clip URLs. |
| `search_detections` | `date` (YYYY-MM-DD)<br>`species` (name/code)<br>`min_confidence` (0.0-1.0)<br>`limit` (int) | Historical search through detection records stored in SQLite. |
| `get_detection_detail` | `id` (int, required) | Inspects a single detection: weather conditions at detection time, confidence breakdown, and audio clip info. |
| `get_today_summary` | *(none)* | Aggregate summary of today's observatory run: total count, species diversity, and top visitors. |
| `get_new_arrivals` | *(none)* | Identifies species heard for the first time ever or new this season/year (vital for tracking migration). |
| `get_station_health` | *(none)* | Real-time diagnostic telemetry: RTSP mic stream state, ingest bit rate, dropped packets, and host uptime. |
| `get_audio_clip` | `clip_name` (string)<br>`detection_date` (string) | Resolves the accessible LAN URL on Caddy/Nginx (:8091) for Discord/web embedding. |
| `get_audio_clip_base64` | `clip_name` (string)<br>`detection_date` (string) | Downloads and base64-encodes the raw `.wav` audio clip for multimodal models with direct audio input capabilities.<br>⚠️ *Token budget warning: A 15-second WAV clip is ~1.4–1.9MB base64 (~350,000–500,000 tokens into Gemini/Claude multimodal models). Use selectively for verification; never put this in automated high-frequency briefing crons!* |

### MCP Resources

- `birdnet://station/health` — Live JSON snapshot of the Pi mic stream and BirdNET-Go engine.
- `birdnet://detections/recent` — The 10 most recent detections.
- `birdnet://species/summary` — Aggregated species occurrence summary.

### MCP Prompts

- `daily_backyard_brief` — Morning dispatch workflow template for resident bird agents.
- `investigate_detection` — Deep-dive template for evaluating anomalous sightings against eBird.

---

## Configuration Reference

| Environment Variable | CLI Flag | Default | Description |
| :--- | :--- | :--- | :--- |
| `BIRDNET_BASE_URL` | `--birdnet-url` | `http://localhost:8080` | BirdNET-Go v2 REST API base URL |
| `CLIPS_BASE_URL` | `--clips-url` | `http://localhost:8091` | Base URL for audio clips file server |
| `BIRDNET_USERNAME` | `--user` | *(empty)* | Optional HTTP Basic Auth username |
| `BIRDNET_PASSWORD` | `--pass` | *(empty)* | Optional HTTP Basic Auth password |
| `BIRDNET_AUTH_TOKEN`| `--token` | *(empty)* | Optional Bearer authorization token |
| `REQUEST_TIMEOUT_SECONDS` | *(none)* | `5` | HTTP client timeout in seconds |

---

## Field-Tested in Production: The Origin Story & Hardening

`birdnet-go-mcp` wasn't built in a theoretical vacuum — it was forged and verified against a live backyard bioacoustic observatory:

- **Mic Station**: Raspberry Pi Zero 2 W mounted outdoors with a weather-sealed electret microphone streaming 48kHz mono audio via RTSP (`mediamtx`) at ~75 KB/s over Wi-Fi.
- **Detector Host**: [BirdNET-Go](https://github.com/tphakala/birdnet-go) running inside a Debian 12 Proxmox LXC container (`amd64`, AMD Ryzen 5 PRO), analyzing audio chunks with the Cornell Lab of Ornithology neural network.
- **Clip Web Server**: Caddy reverse proxy serving `/var/lib/birdnet-go/clips/` over HTTP on port 8091.
- **Agent Mesh**: [OpenClaw](https://github.com/openclaw/openclaw) on an Apple Silicon Mac Mini running autonomous resident agents:
  - **Blenda** (Infra agent, GLM-5.3): Monitors server health, manages gateway restarts, and oversees tool permissions.
  - **Leopold** (Resident Naturalist, Gemini 3.8 Flash): Composes daily backyard wildlife briefings, flags unusual species (Eastern Bluebirds, Carolina Wrens, Pileated Woodpeckers), and inspects audio spectrograms.

### Real-World Lessons & "Incident Zero"

1. **The Caddy Directory Permission Trap (Incident Zero)**:
   BirdNET-Go creates monthly clip directories (`/clips/YYYY/MM/`) with `750` permissions (`drwxr-x---`). External web servers (Caddy, Nginx) running as their own system user will hit **HTTP 403 Forbidden** when serving audio clips to agents or Discord webhooks.
   *Fix*: Set permissions to `755` on existing month folders and add your web server user to the `birdnet` group:
   ```bash
   sudo chmod -R 755 /var/lib/birdnet-go/clips
   sudo usermod -aG birdnet caddy
   ```

2. **Apple Silicon AMFI & Cross-Compilation**:
   When cross-compiling Go binaries from Linux for macOS (`GOOS=darwin GOARCH=arm64`) with stripped debug symbols (`-ldflags="-s -w"`), macOS Apple Mobile File Integrity (AMFI) will immediately terminate the process with `SIGKILL` (exit code 137). All Darwin ARM64 releases are properly ad-hoc codesigned (`codesign -s - --force`).

3. **Context Window Token Budget Guard**:
   Feeding raw base64 WAV recordings into multimodal LLMs is magical for verifying difficult bird calls, but a single 15-second WAV consumes **~1.9MB (approx. 500,000 tokens)**. That can consume 50% of a 1M-token context window in one tool invocation.
   `birdnet-go-mcp` strictly caps all structured JSON tool responses at an **8KB envelope cap**, while keeping `get_audio_clip_base64` explicitly exempt so models can call it intentionally without risk of accidental context blowup in daily briefing routines.

---

## Contributing & Development

```bash
# Clone
git clone https://github.com/zax0rz/birdnet-go-mcp.git
cd birdnet-go-mcp

# Run unit tests
make test

# Build for local OS
make build

# Cross-compile for Darwin ARM64 (Apple Silicon)
make build-mac
```

---

## Credits & License

- Built with [`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go).
- Designed for [`tphakala/birdnet-go`](https://github.com/tphakala/birdnet-go).
- Bird identification neural network developed by the **Cornell Lab of Ornithology** and **Chemnitz University of Technology**.
- Licensed under the [MIT License](LICENSE).

