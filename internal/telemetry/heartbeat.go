package telemetry

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/utils"
)

// Default Production Target for Anonymous Heartbeat Telemetry (can be overridden at build via -ldflags)
var (
	DefaultTelemetryEndpoint = "https://rptpauakhdsqinpcnebw.supabase.co/rest/v1/app_telemetry"
	DefaultTelemetryAnonKey  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InJwdHBhdWFraGRzcWlucGNuZWJ3Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3ODM5NTE1NjgsImV4cCI6MjA5OTUyNzU2OH0.yj7dpHIHcL3eEHo5NHCnjGmtxr9rGWjfvxNKcGBFzw8"
	telemetryHTTPClient      = &http.Client{Timeout: 3 * time.Second}
)

type HeartbeatPayload struct {
	InstallationID string `json:"installation_id"`
	Event          string `json:"event"`
	AppVersion     string `json:"app_version"`
	Platform       string `json:"platform"`
	Architecture   string `json:"architecture"`
}

// SendHeartbeat sends a minimal anonymous startup ping in a non-blocking goroutine.
// ponytail: Stdlib only, fire-and-forget, zero locks or retry storms on failure.
func SendHeartbeat(repo *db.Repository, appVersion string) {
	go func() {
		if repo == nil {
			return
		}

		settings, err := repo.GetUserSettings()
		if err != nil || settings == nil || settings.AnonymousUserID == "" {
			return
		}

		endpoint := os.Getenv("TELEMETRY_ENDPOINT_URL")
		if endpoint == "" {
			// Never transmit telemetry to default production targets during tests or CI
			if flag.Lookup("test.v") != nil || os.Getenv("CI") != "" {
				return
			}
			endpoint = DefaultTelemetryEndpoint
		}
		if endpoint == "" {
			if sbURL := os.Getenv("SUPABASE_URL"); sbURL != "" {
				endpoint = strings.TrimSuffix(sbURL, "/") + "/rest/v1/app_telemetry"
			}
		}

		if endpoint == "" {
			return
		}

		// If endpoint is a general Supabase URL, ensure it targets /rest/v1/app_telemetry
		if !strings.HasSuffix(endpoint, "/app_telemetry") && !strings.Contains(endpoint, "/rpc/") {
			if strings.HasSuffix(endpoint, "/rest/v1") {
				endpoint += "/app_telemetry"
			}
		}

		token := os.Getenv("TELEMETRY_ANON_KEY")
		if token == "" {
			token = os.Getenv("SUPABASE_PUBLISHABLE_KEY_LOG")
		}
		if token == "" {
			token = os.Getenv("RESEARCH_ANALYTICS_ANON_KEY")
		}
		if token == "" {
			token = os.Getenv("SUPABASE_ANON_KEY")
		}
		if token == "" {
			token = DefaultTelemetryAnonKey
		}

		ver := strings.TrimPrefix(strings.TrimSpace(appVersion), "v")
		if ver == "" {
			ver = "1.0.0"
		}

		payload := HeartbeatPayload{
			InstallationID: settings.AnonymousUserID,
			Event:          "app_started",
			AppVersion:     ver,
			Platform:       runtime.GOOS,
			Architecture:   runtime.GOARCH,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}

		client := telemetryHTTPClient
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
		if err != nil {
			return
		}

		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("apikey", token)
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err != nil {
			// Fail silent - telemetry should never interrupt app experience
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			utils.Infof("[TELEMETRY] Anonymous heartbeat sent successfully (v%s, %s/%s)", ver, runtime.GOOS, runtime.GOARCH)
		}
	}()
}
