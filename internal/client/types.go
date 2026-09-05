package client

// SourceInfo describes the audio input source (e.g. Pi Zero RTSP mic)
type SourceInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
}

// DetectionWeather captures environmental conditions at the moment of detection
type DetectionWeather struct {
	WeatherIcon      string  `json:"weatherIcon,omitempty"`
	WeatherMain      string  `json:"weatherMain,omitempty"`
	Description      string  `json:"description,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	WindSpeed        float64 `json:"windSpeed,omitempty"`
	Humidity         int     `json:"humidity,omitempty"`
	Units            string  `json:"units,omitempty"`
	MoonPhaseName    string  `json:"moonPhaseName,omitempty"`
	MoonIllumination float64 `json:"moonIllumination,omitempty"`
}

// Detection represents a single bird acoustic detection
type Detection struct {
	ID                 int               `json:"id"`
	Date               string            `json:"date"`
	Time               string            `json:"time"`
	Timestamp          string            `json:"timestamp"`
	Source             SourceInfo        `json:"source"`
	BeginTime          string            `json:"beginTime"`
	EndTime            string            `json:"endTime"`
	SpeciesCode        string            `json:"speciesCode"`
	ScientificName     string            `json:"scientificName"`
	CommonName         string            `json:"commonName"`
	Confidence         float64           `json:"confidence"`
	ClipName           string            `json:"clipName"`
	ModelType          string            `json:"modelType"`
	Verified           string            `json:"verified"`
	Locked             bool              `json:"locked"`
	Weather            *DetectionWeather `json:"weather,omitempty"`
	TimeOfDay          string            `json:"timeOfDay,omitempty"`
	IsNewSpecies       bool              `json:"isNewSpecies"`
	DaysSinceFirstSeen int               `json:"daysSinceFirstSeen"`
	IsNewThisYear      bool              `json:"isNewThisYear"`
	IsNewThisSeason    bool              `json:"isNewThisSeason"`
	DaysThisYear       int               `json:"daysThisYear"`
	DaysThisSeason     int               `json:"daysThisSeason"`
	CurrentSeason      string            `json:"currentSeason"`
	ClipURL            string            `json:"clipUrl,omitempty"`
}

// DetectionsResponse is the paginated response from GET /api/v2/detections
type DetectionsResponse struct {
	Data        []Detection `json:"data"`
	Total       int         `json:"total"`
	Limit       int         `json:"limit"`
	Offset      int         `json:"offset"`
	CurrentPage int         `json:"current_page"`
	TotalPages  int         `json:"total_pages"`
}

// SpeciesSummaryItem represents an aggregate summary for a species
type SpeciesSummaryItem struct {
	ScientificName string  `json:"scientific_name"`
	CommonName     string  `json:"common_name"`
	SpeciesCode    string  `json:"species_code"`
	Count          int     `json:"count"`
	FirstHeard     string  `json:"first_heard"`
	LastHeard      string  `json:"last_heard"`
	AvgConfidence  float64 `json:"avg_confidence"`
	MaxConfidence  float64 `json:"max_confidence"`
	ThumbnailURL   string  `json:"thumbnail_url,omitempty"`
}

// NewArrivalItem represents a species newly detected in the tracking window
type NewArrivalItem struct {
	ScientificName string `json:"scientific_name"`
	CommonName     string `json:"common_name"`
	FirstHeardDate string `json:"first_heard_date"`
	ThumbnailURL   string `json:"thumbnail_url,omitempty"`
	CountInPeriod  int    `json:"count_in_period"`
}

// StreamHealthItem represents telemetry for an RTSP audio stream
type StreamHealthItem struct {
	Name                 string  `json:"name"`
	Type                 string  `json:"type"`
	URL                  string  `json:"url"`
	IsHealthy            bool    `json:"is_healthy"`
	ProcessState         string  `json:"process_state"`
	LastDataReceived     string  `json:"last_data_received"`
	TimeSinceDataSeconds float64 `json:"time_since_data_seconds"`
	RestartCount         int     `json:"restart_count"`
	TotalBytesReceived   int64   `json:"total_bytes_received"`
	BytesPerSecond       float64 `json:"bytes_per_second"`
	IsReceivingData      bool    `json:"is_receiving_data"`
}

// SystemInfo represents host and runtime info from CT 122
type SystemInfo struct {
	Hostname         string `json:"hostname"`
	PlatformVersion  string `json:"platform_version"`
	KernelVersion    string `json:"kernel_version"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	BootTime         string `json:"boot_time"`
	AppStartTime     string `json:"app_start_time"`
	AppUptimeSeconds int64  `json:"app_uptime_seconds"`
	NumCPU           int    `json:"num_cpu"`
	TimeZone         string `json:"time_zone"`
	OSDisplay        string `json:"os_display"`
	Architecture     string `json:"architecture"`
	CPUModel         string `json:"cpu_model"`
	Environment      string `json:"environment"`
}
