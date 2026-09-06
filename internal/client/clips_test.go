package client

import (
	"testing"
)

func TestResolveClipURL(t *testing.T) {
	baseURL := "http://192.0.2.10:8091"

	tests := []struct {
		name          string
		clipName      string
		detectionDate string
		expected      string
	}{
		{
			name:          "standard filename with timestamp",
			clipName:      "sialia_sialis_96p_20260829T192351Z.wav",
			detectionDate: "2026-08-29",
			expected:      "http://192.0.2.10:8091/2026/08/sialia_sialis_96p_20260829T192351Z.wav",
		},
		{
			name:          "fallback to date when filename has no standard timestamp",
			clipName:      "bluebird_custom.wav",
			detectionDate: "2026-09-04",
			expected:      "http://192.0.2.10:8091/2026/09/bluebird_custom.wav",
		},
		{
			name:          "sanitizes path traversal",
			clipName:      "../../../etc/passwd",
			detectionDate: "2026-08-29",
			expected:      "http://192.0.2.10:8091/2026/08/passwd",
		},
		{
			name:          "empty clip name",
			clipName:      "",
			detectionDate: "2026-08-29",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveClipURL(baseURL, tt.clipName, tt.detectionDate)
			if got != tt.expected {
				t.Errorf("ResolveClipURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}
