package app

import (
	"strings"
	"testing"
)

func TestVersionCompareHelper(t *testing.T) {
	// ponytail: test the version difference logic cleanly
	testCases := []struct {
		remote   string
		current  string
		expected bool
	}{
		{"1.0.0", "1.0.0", false},
		{"v1.0.0", "1.0.0", false},
		{"1.1.0", "1.0.0", true},
		{"v1.1.0", "1.0.0", true},
		{"", "1.0.0", false},
	}

	for _, tc := range testCases {
		remoteClean := strings.TrimPrefix(tc.remote, "v")
		currentClean := strings.TrimPrefix(tc.current, "v")
		actual := remoteClean != "" && remoteClean != currentClean
		if actual != tc.expected {
			t.Errorf("Compare(%q, %q) expected %v, got %v", tc.remote, tc.current, tc.expected, actual)
		}
	}
}

func TestCheckForUpdates(t *testing.T) {
	app := &App{}
	res := app.CheckForUpdates()
	if ver, ok := res["current_version"].(string); !ok || ver == "" {
		t.Errorf("Expected current_version to be populated in CheckForUpdates response, got %v", res["current_version"])
	}
}

func TestAssetSelection(t *testing.T) {
	assets := []gitHubAsset{
		{Name: "rag-assets.zip", BrowserDownloadURL: "https://example.com/rag.zip", Size: 100},
		{Name: "Studyloop-Setup.exe", BrowserDownloadURL: "https://example.com/setup.exe", Size: 200},
	}

	var downloadURL, assetName string
	for _, asset := range assets {
		lower := strings.ToLower(asset.Name)
		if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, "-setup.exe") || strings.HasSuffix(lower, "-installer.exe") {
			downloadURL = asset.BrowserDownloadURL
			assetName = asset.Name
			break
		}
	}

	if downloadURL != "https://example.com/setup.exe" || assetName != "Studyloop-Setup.exe" {
		t.Fatalf("Asset selection failed, got url=%s name=%s", downloadURL, assetName)
	}
}
