package app

import (
	"strings"
	"testing"

	"ai-tutor/internal/extension"
)

func TestListExtensions_Uninitialized(t *testing.T) {
	a := &App{}
	_, err := a.ListExtensions()
	if err == nil {
		t.Fatal("expected error when extManager is uninitialized, got nil")
	}
}

func TestListExtensions_WithInitError(t *testing.T) {
	a := &App{
		extInitError: "directory permission denied",
	}
	_, err := a.ListExtensions()
	if err == nil {
		t.Fatal("expected error when extInitError is set, got nil")
	}
	if !strings.Contains(err.Error(), "directory permission denied") {
		t.Fatalf("expected error to contain init error message, got %v", err)
	}
}

func TestListExtensions_WithManager(t *testing.T) {
	tempDir := t.TempDir()
	mgr := extension.NewManager(tempDir)
	a := &App{
		extManager: mgr,
	}

	exts, err := a.ListExtensions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exts) != 0 {
		t.Fatalf("expected 0 extensions, got %d", len(exts))
	}
}

func TestRunExtension_ProEntitlementEnforced(t *testing.T) {
	a := &App{}
	// When not pro, uninitialized check or Pro check stops unauthorized calls
	res := a.RunExtension("unknown_ext", "test")
	if res["error"] == nil {
		t.Fatal("expected error when running extension without initialization")
	}

	// Verify session Pro tracking
	if a.IsProUser() {
		t.Fatal("expected IsProUser to be false by default")
	}

	a.setSession("user_123", "pro@studyloop.dev", true)
	if !a.IsProUser() {
		t.Fatal("expected IsProUser to be true after SetSession")
	}

	session := a.GetUserSession()
	if session["userId"] != "user_123" || session["isPro"] != true {
		t.Fatalf("unexpected session state: %#v", session)
	}

	a.ClearSession()
	if a.IsProUser() {
		t.Fatal("expected IsProUser to be false after ClearSession")
	}
}

func TestRunExtension_InputSizeCap(t *testing.T) {
	a := &App{}
	bigInput := strings.Repeat("a", maxExtensionInputBytes+10)
	res := a.RunExtension("test_id", bigInput)
	if res["error"] == nil || !strings.Contains(res["error"].(string), "exceeds maximum allowable size") {
		t.Fatalf("expected input size cap error, got: %#v", res)
	}
	if res["id"] != "test_id" {
		t.Fatalf("expected id to be preserved in error map, got: %#v", res)
	}
}

func TestCancelExtensionSetup_Tracking(t *testing.T) {
	a := &App{}
	res := a.CancelExtensionSetup()
	if res["success"] != true || res["was_running"] != false {
		t.Fatalf("unexpected cancel result when no setup running: %#v", res)
	}
}

func TestSaveExtensionConfig_Validation(t *testing.T) {
	a := &App{}
	err := a.SaveExtensionConfig("{invalid json")
	if err == nil || !strings.Contains(err.Error(), "repository not initialized") {
		// repo is nil first
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	}
}

