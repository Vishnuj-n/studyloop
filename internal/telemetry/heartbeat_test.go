package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
)

func TestSendHeartbeat_Success(t *testing.T) {
	received := make(chan HeartbeatPayload, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload HeartbeatPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		received <- payload
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	os.Setenv("TELEMETRY_ENDPOINT_URL", server.URL)
	os.Setenv("TELEMETRY_ANON_KEY", "test-anon-key")
	defer func() {
		os.Unsetenv("TELEMETRY_ENDPOINT_URL")
		os.Unsetenv("TELEMETRY_ANON_KEY")
	}()

	tempDB := "test_heartbeat.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := db.Init(tempDB, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// Ensure user settings exists with an anonymous_user_id
	settings := models.UserSettings{
		AnonymousUserID: "test-uuid-12345",
	}
	if err := repo.UpdateUserSettings(settings); err != nil {
		t.Fatalf("failed to update user settings: %v", err)
	}

	SendHeartbeat(repo, "1.4.1")

	select {
	case p := <-received:
		if p.InstallationID != "test-uuid-12345" {
			t.Errorf("expected installation_id 'test-uuid-12345', got %q", p.InstallationID)
		}
		if p.Event != "app_started" {
			t.Errorf("expected event 'app_started', got %q", p.Event)
		}
		if p.AppVersion != "1.4.1" {
			t.Errorf("expected version '1.4.1', got %q", p.AppVersion)
		}
		if p.Platform == "" || p.Architecture == "" {
			t.Errorf("expected non-empty platform and architecture, got %s/%s", p.Platform, p.Architecture)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for telemetry ping")
	}
}

func TestSendHeartbeat_ServerDownFailSilent(t *testing.T) {
	// Point to closed/invalid server port
	os.Setenv("TELEMETRY_ENDPOINT_URL", "http://127.0.0.1:54321/rest/v1/app_telemetry")
	defer os.Unsetenv("TELEMETRY_ENDPOINT_URL")

	tempDB := "test_heartbeat_silent.db"
	_ = os.Remove(tempDB)
	defer func() { _ = os.Remove(tempDB) }()

	repo, err := db.Init(tempDB, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// Should not panic or block
	SendHeartbeat(repo, "1.4.1")
	time.Sleep(50 * time.Millisecond)
}
