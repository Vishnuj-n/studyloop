package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRestoreSession_HMACAndGracePeriod(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("APP_ENV", "dev")

	// Ensure clean working directory environment
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	_ = os.Chdir(tempDir)

	a := &App{}

	// 1. Valid Session Persistence & Restoration
	a.setSession("user_123", "pro@example.com", true)
	if !a.IsProUser() {
		t.Fatalf("expected IsProUser to be true after setSession")
	}

	// Create fresh instance and restore from disk
	a2 := &App{}
	if !a2.RestoreSession("", "", false, 0) {
		t.Fatalf("expected RestoreSession to return true for freshly persisted session")
	}
	if !a2.IsProUser() {
		t.Fatalf("expected a2 to be pro")
	}
	sess := a2.getUserSession()
	if sess["userId"] != "user_123" || sess["email"] != "pro@example.com" {
		t.Fatalf("unexpected restored session: %#v", sess)
	}

	// 2. Tampering Attack: modify session.json directly on disk to gain Pro
	filePath, err := getSessionFilePath()
	if err != nil {
		t.Fatalf("failed to get session file path: %v", err)
	}

	// Create non-pro session legitimately
	a.setSession("free_user", "free@example.com", false)

	// User tampers with disk file: edits isPro from false to true without valid HMAC
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read session file: %v", err)
	}
	tamperedData := strings.Replace(string(fileBytes), `"isPro": false`, `"isPro": true`, 1)
	if err := os.WriteFile(filePath, []byte(tamperedData), 0o600); err != nil {
		t.Fatalf("failed to write tampered file: %v", err)
	}

	// Restore tampered session -> must reject and set isPro = false
	a3 := &App{}
	if a3.RestoreSession("attacker", "hacker@evil.com", true, time.Now().Unix()) {
		t.Fatalf("expected RestoreSession to reject tampered file signature")
	}
	if a3.IsProUser() {
		t.Fatalf("expected a3.IsProUser() to be false after tampering attempt")
	}

	// 3. Grace Period Expiration (>10 days)
	elevenDaysAgo := time.Now().Unix() - (11 * 24 * 60 * 60)
	sig := computeSessionSignature(getMachineID(), "user_old", "old@example.com", true, elevenDaysAgo)
	expiredSess := persistentSession{
		UserID:     "user_old",
		Email:      "old@example.com",
		IsPro:      true,
		VerifiedAt: elevenDaysAgo,
		Signature:  sig,
	}
	expiredBytes, _ := json.Marshal(expiredSess)
	_ = os.WriteFile(filePath, expiredBytes, 0o600)

	a4 := &App{}
	if a4.RestoreSession("", "", true, 0) {
		t.Fatalf("expected RestoreSession to return false for expired grace period")
	}
	if a4.IsProUser() {
		t.Fatalf("expected a4.IsProUser() to be false for expired session")
	}

	// 4. ClearSession removes file and clears memory
	a.setSession("user_clear", "clear@example.com", true)
	a.ClearSession()
	if a.IsProUser() {
		t.Fatalf("expected IsProUser to be false after ClearSession")
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected session file to be deleted on ClearSession")
	}
}

func TestAuthConfirm_StateNonceEnforcement(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("APP_ENV", "dev")
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	_ = os.Chdir(tempDir)

	a := &App{}

	// Start auth server to initialize listener and state nonce
	res, err := a.StartBrowserAuth("sign-in")
	if err != nil {
		t.Fatalf("StartBrowserAuth failed: %v", err)
	}
	if res["url"] == "" {
		t.Fatalf("expected non-empty url in auth start response")
	}

	activeAuthServer.mu.Lock()
	validNonce := activeAuthServer.stateNonce
	srv := activeAuthServer.server
	activeAuthServer.mu.Unlock()

	if validNonce == "" || srv == nil {
		t.Fatalf("expected active server and stateNonce to be set")
	}

	// Test 1: Request with wrong / missing state nonce -> 403 Forbidden
	payloadBad := map[string]interface{}{
		"userId": "evil_user",
		"email":  "evil@domain.com",
		"isPro":  true,
		"state":  "invalid-state-token",
	}
	bodyBad, _ := json.Marshal(payloadBad)
	reqBad := httptest.NewRequest(http.MethodPost, "/api/auth-confirm", bytes.NewReader(bodyBad))
	reqBad.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for bad nonce, got %d", wBad.Code)
	}

	// Test 2: Request with valid state nonce in header -> 200 OK
	payloadGood := map[string]interface{}{
		"userId": "valid_user",
		"email":  "valid@domain.com",
		"isPro":  true,
		"state":  validNonce,
	}
	bodyGood, _ := json.Marshal(payloadGood)
	reqGood := httptest.NewRequest(http.MethodPost, "/api/auth-confirm", bytes.NewReader(bodyGood))
	reqGood.Header.Set("Content-Type", "application/json")
	reqGood.Header.Set("X-Auth-State", validNonce)
	wGood := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wGood, reqGood)

	if wGood.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid nonce, got %d", wGood.Code)
	}

	if !a.IsProUser() {
		t.Fatalf("expected user to be Pro after successful auth confirm")
	}

	// Clean up
	activeAuthServer.mu.Lock()
	if activeAuthServer.server != nil {
		_ = activeAuthServer.server.Close()
		activeAuthServer.server = nil
	}
	activeAuthServer.mu.Unlock()
}
