package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"ai-tutor/internal/utils"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type AuthCallbackResult struct {
	Success bool   `json:"success"`
	UserID  string `json:"userId"`
	Email   string `json:"email"`
	IsPro   bool   `json:"isPro"`
	Error   string `json:"error,omitempty"`
}

type authServerState struct {
	mu     sync.Mutex
	server *http.Server
}

var activeAuthServer = &authServerState{}

// StartBrowserAuth spins up an ephemeral HTTP server on 127.0.0.1:0 and returns the browser login URL.
func (a *App) StartBrowserAuth(mode string) map[string]interface{} {
	activeAuthServer.mu.Lock()
	if activeAuthServer.server != nil {
		_ = activeAuthServer.server.Close()
		activeAuthServer.server = nil
	}
	activeAuthServer.mu.Unlock()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		utils.Warnf("[AUTH] Failed to create local loopback listener: %v", err)
		return map[string]interface{}{"error": "Failed to start local auth listener"}
	}

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
	}

	activeAuthServer.mu.Lock()
	activeAuthServer.server = server
	activeAuthServer.mu.Unlock()

	mux.HandleFunc("/callback", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		userID := q.Get("user_id")
		email := q.Get("email")
		isProStr := q.Get("is_pro")
		isPro := isProStr == "true" || isProStr == "1"

		if email == "" {
			email = q.Get("primary_email_address")
		}

		result := AuthCallbackResult{
			Success: true,
			UserID:  userID,
			Email:   email,
			IsPro:   isPro,
		}

		// Emit event back to Wails frontend window
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "clerk_auth_success", result)
			wailsruntime.WindowUnminimise(a.ctx)
			wailsruntime.Show(a.ctx)
		}

		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = rw.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>StudyLoop Authenticated</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
    .card { background: #1e293b; padding: 32px; border-radius: 12px; text-align: center; max-width: 400px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }
    h1 { color: #38bdf8; font-size: 24px; margin-bottom: 12px; }
    p { color: #94a3b8; font-size: 15px; margin-bottom: 24px; line-height: 1.5; }
    .badge { display: inline-block; background: #0284c7; color: white; padding: 6px 16px; border-radius: 9999px; font-weight: bold; font-size: 14px; }
  </style>
</head>
<body>
  <div class="card">
    <div class="badge">StudyLoop Connected</div>
    <h1>Authentication Complete</h1>
    <p>You can close this tab and return to your StudyLoop desktop app.</p>
  </div>
  <script>setTimeout(function(){ window.close(); }, 2000);</script>
</body>
</html>`))

		// Asynchronously close listener after handling request
		go func() {
			time.Sleep(500 * time.Millisecond)
			activeAuthServer.mu.Lock()
			if activeAuthServer.server != nil {
				_ = activeAuthServer.server.Close()
				activeAuthServer.server = nil
			}
			activeAuthServer.mu.Unlock()
		}()
	})

	go func() {
		_ = server.Serve(listener)
	}()

	// Auto shutdown listener after 5 minutes if unused
	go func() {
		time.Sleep(5 * time.Minute)
		activeAuthServer.mu.Lock()
		if activeAuthServer.server == server {
			_ = server.Shutdown(context.Background())
			activeAuthServer.server = nil
		}
		activeAuthServer.mu.Unlock()
	}()

	var targetURL string
	escapedCallback := url.QueryEscape(callbackURL)
	if mode == "billing" {
		targetURL = fmt.Sprintf("https://innocent-orca-5605.accounts.dev/user?redirect_url=%s&force_redirect_url=%s&after_sign_in_url=%s", escapedCallback, escapedCallback, escapedCallback)
	} else {
		targetURL = fmt.Sprintf("https://innocent-orca-5605.accounts.dev/sign-in?redirect_url=%s&force_redirect_url=%s&after_sign_in_url=%s", escapedCallback, escapedCallback, escapedCallback)
	}

	return map[string]interface{}{
		"url": targetURL,
	}
}
