package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppTestLLMConnectionAndLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {"content": "Hello!"},
					"finish_reason": "stop"
				}
			]
		}`))
	}))
	defer server.Close()

	app := &App{}

	connRes := app.TestLLMConnection("fast", "custom", server.URL, "test-model", "test-key")
	if connRes["ok"] != true {
		t.Fatalf("expected TestLLMConnection ok: true, got: %v", connRes)
	}

	limitsRes := app.TestLLMLimits("fast", "custom", server.URL, "test-model", "test-key", 4000)
	if limitsRes["ok"] != true {
		t.Fatalf("expected TestLLMLimits ok: true, got: %v", limitsRes)
	}
	statusStr, _ := limitsRes["status"].(string)
	if !strings.Contains(statusStr, "4000 token input prompt limit") {
		t.Fatalf("unexpected status output from TestLLMLimits: %s", statusStr)
	}
}
