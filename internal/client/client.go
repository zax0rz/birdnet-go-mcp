package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// BirdNETClient provides typed read-only access to BirdNET-Go v2 API
type BirdNETClient struct {
	baseURL      string
	clipsBaseURL string
	username     string
	password     string
	authToken    string
	httpClient   *http.Client
}

// NewClient constructs a new BirdNETClient, respecting env vars
func NewClient(baseURL, clipsBaseURL string, timeout time.Duration) *BirdNETClient {
	if baseURL == "" {
		baseURL = os.Getenv("BIRDNET_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}
	}
	if clipsBaseURL == "" {
		clipsBaseURL = os.Getenv("CLIPS_BASE_URL")
		if clipsBaseURL == "" {
			clipsBaseURL = "http://localhost:8091"
		}
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &BirdNETClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clipsBaseURL: strings.TrimRight(clipsBaseURL, "/"),
		username:     os.Getenv("BIRDNET_USERNAME"),
		password:     os.Getenv("BIRDNET_PASSWORD"),
		authToken:    os.Getenv("BIRDNET_AUTH_TOKEN"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetAuth sets HTTP Basic Auth credentials
func (c *BirdNETClient) SetAuth(username, password string) {
	c.username = username
	c.password = password
}

// SetToken sets Bearer authorization token
func (c *BirdNETClient) SetToken(token string) {
	c.authToken = token
}

// GetBaseURL returns the configured BirdNET-Go API base URL
func (c *BirdNETClient) GetBaseURL() string {
	return c.baseURL
}

// GetClipsBaseURL returns the configured base URL for audio clips
func (c *BirdNETClient) GetClipsBaseURL() string {
	return c.clipsBaseURL
}

func (c *BirdNETClient) applyAuth(req *http.Request) {
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	} else if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *BirdNETClient) get(ctx context.Context, endpoint string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "birdnet-go-mcp/1.0")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed for %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error from %s (status %d): %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

// DownloadClip downloads audio data for a clip up to maxBytes (default 5MB)
func (c *BirdNETClient) DownloadClip(ctx context.Context, clipName, detectionDate string, maxBytes int64) ([]byte, error) {
	clipURL := ResolveClipURL(c.clipsBaseURL, clipName, detectionDate)
	if clipURL == "" {
		return nil, fmt.Errorf("invalid or unsafe clip name: %s", clipName)
	}

	if maxBytes <= 0 {
		maxBytes = 5 * 1024 * 1024 // 5MB limit
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clipURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create clip download request: %w", err)
	}
	req.Header.Set("User-Agent", "birdnet-go-mcp/1.0")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch clip from %s: %w", clipURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clip server returned status %d for %s", resp.StatusCode, clipURL)
	}

	limitedReader := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("error reading clip stream: %w", err)
	}

	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("audio clip exceeds size limit of %d bytes", maxBytes)
	}

	return data, nil
}

// GetRecentDetections fetches recent detections from /api/v2/detections/recent or /api/v2/detections
func (c *BirdNETClient) GetRecentDetections(ctx context.Context, limit int, minConfidence float64, speciesCode string) ([]Detection, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	body, err := c.get(ctx, "/api/v2/detections/recent")
	var detections []Detection
	if err == nil {
		_ = json.Unmarshal(body, &detections)
	}

	if len(detections) == 0 {
		params := url.Values{}
		params.Set("limit", strconv.Itoa(limit*2))
		if speciesCode != "" {
			params.Set("species_code", speciesCode)
		}
		endpoint := fmt.Sprintf("/api/v2/detections?%s", params.Encode())
		data, err := c.get(ctx, endpoint)
		if err != nil {
			return nil, err
		}

		var pageResp DetectionsResponse
		if err := json.Unmarshal(data, &pageResp); err != nil {
			return nil, fmt.Errorf("failed to parse detections json: %w", err)
		}
		detections = pageResp.Data
	}

	filtered := make([]Detection, 0, limit)
	for i := range detections {
		d := detections[i]
		if d.Confidence < minConfidence {
			continue
		}
		if speciesCode != "" && !strings.EqualFold(d.SpeciesCode, speciesCode) {
			continue
		}
		d.ClipURL = ResolveClipURL(c.clipsBaseURL, d.ClipName, d.Date)
		filtered = append(filtered, d)
		if len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

// SearchDetections queries past detections with filters
func (c *BirdNETClient) SearchDetections(ctx context.Context, date, species string, minConfidence float64, limit int) ([]Detection, error) {
	if limit <= 0 || limit > 50 {
		limit = 15
	}

	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit*2))
	if date != "" {
		params.Set("date", date)
	}
	if species != "" {
		if len(species) <= 6 && !strings.Contains(species, " ") {
			params.Set("species_code", strings.ToLower(species))
		} else {
			params.Set("common_name", species)
		}
	}

	endpoint := fmt.Sprintf("/api/v2/detections?%s", params.Encode())
	data, err := c.get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var pageResp DetectionsResponse
	if err := json.Unmarshal(data, &pageResp); err != nil {
		return nil, fmt.Errorf("failed to parse detections search json: %w", err)
	}

	results := make([]Detection, 0, limit)
	for i := range pageResp.Data {
		d := pageResp.Data[i]
		if minConfidence > 0 && d.Confidence < minConfidence {
			continue
		}
		d.ClipURL = ResolveClipURL(c.clipsBaseURL, d.ClipName, d.Date)
		results = append(results, d)
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// GetDetectionDetail retrieves full details for a single detection by ID
func (c *BirdNETClient) GetDetectionDetail(ctx context.Context, id int) (*Detection, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid detection id: %d", id)
	}

	endpoint := fmt.Sprintf("/api/v2/detections/%d", id)
	data, err := c.get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var d Detection
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to parse detection detail json: %w", err)
	}

	d.ClipURL = ResolveClipURL(c.clipsBaseURL, d.ClipName, d.Date)
	return &d, nil
}

// GetSpeciesSummary fetches aggregated species counts and activity
func (c *BirdNETClient) GetSpeciesSummary(ctx context.Context) ([]SpeciesSummaryItem, error) {
	data, err := c.get(ctx, "/api/v2/analytics/species/summary")
	if err != nil {
		return nil, err
	}

	var items []SpeciesSummaryItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse species summary json: %w", err)
	}

	return items, nil
}

// GetNewArrivals fetches species detected for the first time or in the current season
func (c *BirdNETClient) GetNewArrivals(ctx context.Context) ([]NewArrivalItem, error) {
	data, err := c.get(ctx, "/api/v2/analytics/species/detections/new")
	if err != nil {
		return nil, err
	}

	var items []NewArrivalItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse new arrivals json: %w", err)
	}

	return items, nil
}

// GetStreamHealth returns RTSP input stream metrics and connection status
func (c *BirdNETClient) GetStreamHealth(ctx context.Context) ([]StreamHealthItem, error) {
	data, err := c.get(ctx, "/api/v2/streams/health")
	if err != nil {
		return nil, err
	}

	var items []StreamHealthItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse stream health json: %w", err)
	}

	return items, nil
}

// GetSystemInfo returns host and runtime stats from CT 122
func (c *BirdNETClient) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	data, err := c.get(ctx, "/api/v2/system/info")
	if err != nil {
		return nil, err
	}

	var info SystemInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse system info json: %w", err)
	}

	return &info, nil
}
