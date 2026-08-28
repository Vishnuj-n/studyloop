package extension

import "strings"

// officialExtensionTiers defines the compiled authoritative tiers for official extensions.
// This prevents users from circumventing Pro requirements by editing local manifest.json files.
var officialExtensionTiers = map[string]string{
	"text_simplifier":  "free",
	"audio_overview":   "pro",
	"youtube":          "pro",
	"deep_pdf":         "pro",
	"reading_simplify": "free",
}

// GetEffectiveTier returns the authoritative tier for an extension.
// For official extensions, the compiled Go mapping overrides whatever is in manifest.json.
// For third-party/sideloaded extensions, it respects the manifest tier.
func GetEffectiveTier(ext *Extension) string {
	if ext == nil {
		return "free"
	}
	id := strings.ToLower(strings.TrimSpace(ext.ID()))
	if tier, ok := officialExtensionTiers[id]; ok {
		return tier
	}
	if strings.ToLower(strings.TrimSpace(ext.Manifest.Tier)) == "pro" {
		return "pro"
	}
	return "free"
}
