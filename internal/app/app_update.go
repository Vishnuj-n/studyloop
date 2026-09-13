package app

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed VERSION
var embeddedVersion string

//go:embed RELEASE_NOTES.md
var embeddedReleaseNotes string

// getAppVersion retrieves current app version dynamically from the embedded VERSION file.
func getAppVersion() string {
	return strings.TrimPrefix(strings.TrimSpace(embeddedVersion), "v")
}

type gitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
}

// fetchGitHubReleaseNotes attempts to fetch the release body for a given version from GitHub API.
func fetchGitHubReleaseNotes(version string) string {
	client := http.Client{Timeout: 4 * time.Second}
	tag := "v" + strings.TrimPrefix(version, "v")
	url := "https://api.github.com/repos/Vishnuj-n/studyloop/releases/tags/" + tag

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Studyloop-App")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// Fallback to latest release if tag specific release fails
		latestURL := "https://api.github.com/repos/Vishnuj-n/studyloop/releases/latest"
		reqLatest, err := http.NewRequest("GET", latestURL, nil)
		if err != nil {
			return ""
		}
		reqLatest.Header.Set("User-Agent", "Studyloop-App")
		respLatest, err := client.Do(reqLatest)
		if err != nil || respLatest.StatusCode != http.StatusOK {
			if respLatest != nil {
				_ = respLatest.Body.Close()
			}
			return ""
		}
		resp = respLatest
		defer func() { _ = respLatest.Body.Close() }()
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var rel gitHubRelease
	if err := json.Unmarshal(bodyBytes, &rel); err != nil {
		return ""
	}

	return strings.TrimSpace(rel.Body)
}

// GetReleaseNotes returns application version and release notes (dynamically fetched from GitHub with local fallback).
func (a *App) GetReleaseNotes() map[string]interface{} {
	appVer := getAppVersion()
	remoteNotes := fetchGitHubReleaseNotes(appVer)
	if remoteNotes != "" {
		return map[string]interface{}{
			"version": appVer,
			"notes":   remoteNotes,
			"source":  "github",
		}
	}
	return map[string]interface{}{
		"version": appVer,
		"notes":   strings.TrimSpace(embeddedReleaseNotes),
		"source":  "embedded",
	}
}

// CheckForUpdates checks the remote version file and returns if an update is available.
func (a *App) CheckForUpdates() map[string]interface{} {
	appVer := getAppVersion()

	// ponytail: simple HTTP GET to check raw text version, minimal overhead
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get("https://raw.githubusercontent.com/Vishnuj-n/studyloop/main/internal/app/VERSION")
	if err == nil && resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		resp, err = client.Get("https://raw.githubusercontent.com/Vishnuj-n/studyloop/main/VERSION")
	}
	if err != nil {
		return map[string]interface{}{
			"update_available": false,
			"current_version":  appVer,
			"error":            err.Error(),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return map[string]interface{}{
			"update_available": false,
			"current_version":  appVer,
			"error":            "unexpected status code: " + resp.Status,
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"update_available": false,
			"current_version":  appVer,
			"error":            err.Error(),
		}
	}

	remoteVersion := strings.TrimSpace(string(bodyBytes))
	remoteVersionClean := strings.TrimPrefix(remoteVersion, "v")
	currentVersionClean := strings.TrimPrefix(appVer, "v")

	// ponytail: simple string comparison. Since we are doing sequential releases,
	// if the remote tag differs from current, we flag update available.
	if remoteVersionClean != "" && remoteVersionClean != currentVersionClean {
		return map[string]interface{}{
			"update_available": true,
			"latest_version":   remoteVersionClean,
			"current_version":  currentVersionClean,
			"url":              "https://github.com/Vishnuj-n/studyloop/releases",
		}
	}

	return map[string]interface{}{
		"update_available": false,
		"latest_version":   remoteVersionClean,
		"current_version":  currentVersionClean,
	}
}

// OpenRepoURL opens the GitHub repository releases page in the user's default system browser.
func (a *App) OpenRepoURL() {
	// ponytail: use native OS browser via Wails runtime wrapper
	wailsruntime.BrowserOpenURL(a.ctx, "https://github.com/Vishnuj-n/studyloop/releases")
}

// OpenURLInBrowser opens any external URL in the user's default system browser.
func (a *App) OpenURLInBrowser(urlStr string) {
	if a.ctx != nil && strings.TrimSpace(urlStr) != "" {
		wailsruntime.BrowserOpenURL(a.ctx, strings.TrimSpace(urlStr))
	}
}

