package extension

import (
	"testing"
)

func TestGetEffectiveTier(t *testing.T) {
	tests := []struct {
		name     string
		ext      *Extension
		expected string
	}{
		{
			name:     "nil extension returns free",
			ext:      nil,
			expected: "free",
		},
		{
			name: "official text_simplifier returns free",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "text_simplifier",
					Tier: "pro", // Manifest claim ignored for official extension
				},
			},
			expected: "free",
		},
		{
			name: "official audio_overview returns pro",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "audio_overview",
					Tier: "free", // Manifest claim ignored for official extension
				},
			},
			expected: "pro",
		},
		{
			name: "official youtube returns free",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "youtube",
					Tier: "pro", // Manifest claim ignored for official extension
				},
			},
			expected: "free",
		},
		{
			name: "official deep_pdf returns pro",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "deep_pdf",
					Tier: "free", // Even if local manifest says free, compiled tier takes precedence
				},
			},
			expected: "pro",
		},
		{
			name: "third party extension with pro manifest tier returns pro",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "custom_search_pro",
					Tier: "pro",
				},
			},
			expected: "pro",
		},
		{
			name: "third party extension with Pro case-insensitive manifest tier returns pro",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "custom_search_pro_caps",
					Tier: "PRO",
				},
			},
			expected: "pro",
		},
		{
			name: "third party extension with free manifest tier returns free",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "custom_search_free",
					Tier: "free",
				},
			},
			expected: "free",
		},
		{
			name: "third party extension with empty manifest tier returns free",
			ext: &Extension{
				Manifest: Manifest{
					ID:   "custom_search_default",
					Tier: "",
				},
			},
			expected: "free",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetEffectiveTier(tc.ext)
			if got != tc.expected {
				t.Errorf("GetEffectiveTier() = %q, want %q", got, tc.expected)
			}
		})
	}
}
