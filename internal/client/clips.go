package client

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Matches filenames like: species_name_96p_20260829T192351Z.wav
	clipTimestampRegex = regexp.MustCompile(`_(\d{4})(\d{2})\d{2}T\d{6}Z\.wav$`)
)

// ResolveClipURL builds the accessible LAN URL for a clip.
// Clips are served on Caddy (:8091) organized by year and month:
// http://192.0.2.10:8091/{year}/{month}/{clipName}
func ResolveClipURL(baseURL, clipName, detectionDate string) string {
	if clipName == "" {
		return ""
	}

	// Sanitize against path traversal or malicious inputs
	cleanName := filepath.Base(clipName)
	cleanName = strings.TrimSpace(cleanName)
	if strings.Contains(cleanName, "..") || cleanName == "." || cleanName == "/" {
		return ""
	}

	baseURL = strings.TrimRight(baseURL, "/")

	// Attempt to extract year and month from the clip filename
	matches := clipTimestampRegex.FindStringSubmatch(cleanName)
	if len(matches) == 3 {
		year := matches[1]
		month := matches[2]
		return fmt.Sprintf("%s/%s/%s/%s", baseURL, year, month, url.PathEscape(cleanName))
	}

	// Fallback to detectionDate (YYYY-MM-DD)
	if len(detectionDate) >= 7 && detectionDate[4] == '-' {
		parts := strings.Split(detectionDate, "-")
		if len(parts) >= 2 {
			year := parts[0]
			month := parts[1]
			return fmt.Sprintf("%s/%s/%s/%s", baseURL, year, month, url.PathEscape(cleanName))
		}
	}

	// Final fallback: root of clips server
	return fmt.Sprintf("%s/%s", baseURL, url.PathEscape(cleanName))
}
