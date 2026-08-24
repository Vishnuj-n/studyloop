package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
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
func (a *App) StartBrowserAuth(mode string) (map[string]interface{}, error) {
	utils.Infof("[AUTH] StartBrowserAuth requested with mode: %s", mode)
	activeAuthServer.mu.Lock()
	if activeAuthServer.server != nil {
		_ = activeAuthServer.server.Close()
		activeAuthServer.server = nil
	}
	activeAuthServer.mu.Unlock()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		utils.Warnf("[AUTH] Failed to create local loopback listener: %v", err)
		return nil, fmt.Errorf("failed to start local auth listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	utils.Infof("[AUTH] Created loopback server on port %d with callback: %s", port, callbackURL)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
	}

	activeAuthServer.mu.Lock()
	activeAuthServer.server = server
	activeAuthServer.mu.Unlock()

	clerkKey := os.Getenv("VITE_CLERK_PUBLISHABLE_KEY")
	if clerkKey == "" {
		clerkKey = os.Getenv("CLERK_PUBLISHABLE_KEY")
	}
	if clerkKey == "" {
		clerkKey = "pk_test_aW5ub2NlbnQtb3JjYS01NjA1LmNsZXJrLmFjY291bnRzLmRldiQ="
	}

	mux.HandleFunc("/api/auth-confirm", func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload struct {
			UserID string `json:"userId"`
			Email  string `json:"email"`
			IsPro  bool   `json:"isPro"`
			Plan   string `json:"plan"`
			Role   string `json:"role"`
		}

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			utils.Warnf("[AUTH] Failed to decode /api/auth-confirm body: %v", err)
			http.Error(rw, "Invalid body", http.StatusBadRequest)
			return
		}

		isPro := payload.IsPro || payload.Plan == "pro" || payload.Role == "pro"
		utils.Infof("[AUTH] Authoritative Clerk JS sync: user=%s email=%s plan=%s role=%s -> isPro=%v",
			payload.UserID, payload.Email, payload.Plan, payload.Role, isPro)

		result := AuthCallbackResult{
			Success: true,
			UserID:  payload.UserID,
			Email:   payload.Email,
			IsPro:   isPro,
		}

		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "clerk_auth_success", result)
			wailsruntime.WindowUnminimise(a.ctx)
			utils.Infof("[AUTH] Emitted clerk_auth_success event to frontend from /api/auth-confirm")
		}

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]bool{"ok": true})

		// Gracefully close listener shortly after receiving client confirm
		go func() {
			time.Sleep(2 * time.Second)
			activeAuthServer.mu.Lock()
			if activeAuthServer.server == server {
				_ = activeAuthServer.server.Close()
				activeAuthServer.server = nil
			}
			activeAuthServer.mu.Unlock()
		}()
	})

	mux.HandleFunc("/callback", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		userID := q.Get("user_id")
		email := q.Get("email")
		isProStr := q.Get("is_pro")
		isPro := isProStr == "true" || isProStr == "1"

		if email == "" {
			email = q.Get("primary_email_address")
		}
		if email == "" {
			email = "Authenticated User"
		}
		if userID == "" {
			userID = fmt.Sprintf("user_%d", time.Now().Unix())
		}
		utils.Infof("[AUTH] Received initial callback for user %s (%s), isPro: %v", userID, email, isPro)

		result := AuthCallbackResult{
			Success: true,
			UserID:  userID,
			Email:   email,
			IsPro:   isPro,
		}

		// Emit initial event back to Wails frontend window
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "clerk_auth_success", result)
			utils.Infof("[AUTH] Emitted clerk_auth_success initial event to frontend")
		} else {
			utils.Warnf("[AUTH] Wails app context is nil, unable to emit clerk_auth_success")
		}

		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		userDisplay := email
		if userDisplay == "" {
			userDisplay = "Pro User"
		}
		responseHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>StudyLoop — Authenticated</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=Manrope:wght@600;700;800&display=swap" rel="stylesheet">
  <script
    async
    crossorigin="anonymous"
    data-clerk-publishable-key="%s"
    src="https://innocent-orca-5605.accounts.dev/npm/@clerk/clerk-js@5/dist/clerk.browser.js"
    type="text/javascript"
  ></script>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
      background: #0b0d16;
      color: #e2e8f0;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
      position: relative;
      overflow: hidden;
    }
    .bg-glow-1 {
      position: absolute;
      top: -20%%;
      left: 30%%;
      width: 500px;
      height: 500px;
      background: radial-gradient(circle, rgba(99, 102, 241, 0.18) 0%%, transparent 70%%);
      pointer-events: none;
      filter: blur(40px);
    }
    .bg-glow-2 {
      position: absolute;
      bottom: -10%%;
      right: 20%%;
      width: 450px;
      height: 450px;
      background: radial-gradient(circle, rgba(16, 185, 129, 0.12) 0%%, transparent 70%%);
      pointer-events: none;
      filter: blur(40px);
    }
    .card {
      position: relative;
      background: rgba(23, 26, 43, 0.85);
      border: 1px solid rgba(255, 255, 255, 0.1);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border-radius: 24px;
      padding: 44px 36px;
      text-align: center;
      max-width: 440px;
      width: 100%%;
      box-shadow:
        0 25px 50px -12px rgba(0, 0, 0, 0.6),
        0 0 0 1px rgba(255, 255, 255, 0.05),
        0 0 40px rgba(99, 102, 241, 0.12);
      animation: cardAppear 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    }
    @keyframes cardAppear {
      from { opacity: 0; transform: translateY(16px) scale(0.98); }
      to { opacity: 1; transform: translateY(0) scale(1); }
    }
    .brand-row {
      display: inline-flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 28px;
    }
    .brand-logo {
      width: 32px;
      height: 32px;
      border-radius: 9px;
      background: linear-gradient(135deg, #6366f1, #3b82f6);
      color: #ffffff;
      display: flex;
      align-items: center;
      justify-content: center;
      font-family: 'Manrope', sans-serif;
      font-weight: 800;
      font-size: 17px;
      box-shadow: 0 4px 12px rgba(99, 102, 241, 0.35);
    }
    .brand-name {
      font-family: 'Manrope', sans-serif;
      font-size: 15px;
      font-weight: 700;
      letter-spacing: -0.01em;
      color: #f8fafc;
    }
    .success-icon-wrap {
      width: 72px;
      height: 72px;
      margin: 0 auto 20px auto;
      border-radius: 50%%;
      background: rgba(16, 185, 129, 0.12);
      border: 1px solid rgba(16, 185, 129, 0.3);
      display: flex;
      align-items: center;
      justify-content: center;
      position: relative;
      animation: pulseGlow 2.5s infinite;
    }
    @keyframes pulseGlow {
      0%% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4); }
      70%% { box-shadow: 0 0 0 16px rgba(16, 185, 129, 0); }
      100%% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
    }
    .check-svg {
      width: 36px;
      height: 36px;
      stroke: #10b981;
      stroke-width: 2.5;
      stroke-linecap: round;
      stroke-linejoin: round;
      fill: none;
    }
    h1 {
      font-family: 'Manrope', sans-serif;
      font-size: 24px;
      font-weight: 800;
      letter-spacing: -0.02em;
      color: #ffffff;
      margin-bottom: 8px;
    }
    .user-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: rgba(255, 255, 255, 0.06);
      border: 1px solid rgba(255, 255, 255, 0.1);
      padding: 5px 14px;
      border-radius: 9999px;
      font-size: 13px;
      color: #cbd5e1;
      margin-bottom: 18px;
    }
    .user-dot {
      width: 7px;
      height: 7px;
      border-radius: 50%%;
      background: #10b981;
    }
    p {
      color: #94a3b8;
      font-size: 14.5px;
      line-height: 1.55;
      margin-bottom: 28px;
    }
    .action-badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      width: 100%%;
      background: linear-gradient(135deg, rgba(99, 102, 241, 0.15), rgba(59, 130, 246, 0.15));
      border: 1px solid rgba(99, 102, 241, 0.35);
      color: #cbd5e1;
      border-radius: 12px;
      padding: 14px 20px;
      font-size: 14.5px;
      font-weight: 600;
      font-family: inherit;
    }
    .action-icon {
      width: 18px;
      height: 18px;
      stroke: #818cf8;
      flex-shrink: 0;
    }
  </style>
</head>
<body>
  <div class="bg-glow-1"></div>
  <div class="bg-glow-2"></div>
  <div class="card">
    <div class="brand-row">
      <div class="brand-logo">S</div>
      <span class="brand-name">The StudyLoop</span>
    </div>

    <div class="success-icon-wrap">
      <svg class="check-svg" viewBox="0 0 24 24">
        <polyline points="20 6 9 17 4 12"></polyline>
      </svg>
    </div>

    <h1>Authentication Complete</h1>
    <div class="user-badge">
      <span class="user-dot"></span>
      <span id="user-display-email">%s</span>
    </div>

    <p>Your StudyLoop desktop workspace is now connected and ready.</p>

    <div class="action-badge">
      <svg class="action-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M15 3h6v6"></path>
        <path d="M10 14L21 3"></path>
        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
      </svg>
      <span>Switch back to StudyLoop app</span>
    </div>
  </div>

  <script>
    window.addEventListener('load', async () => {
      try {
        if (window.Clerk) {
          await window.Clerk.load();
          const user = window.Clerk.user;
          if (user) {
            const meta = user.publicMetadata || {};
            const isPro = meta.role === 'pro' || meta.plan === 'pro' || meta.is_pro === true;
            const email = user.primaryEmailAddress ? user.primaryEmailAddress.emailAddress : '';
            const userId = user.id;

            const emailEl = document.getElementById('user-display-email');
            if (emailEl && email) {
              emailEl.textContent = email + (isPro ? ' (★ Pro Plan)' : ' (Free Plan)');
            }

            await fetch('/api/auth-confirm', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                userId: userId,
                email: email,
                isPro: isPro,
                plan: meta.plan || '',
                role: meta.role || ''
              })
            });
          }
        }
      } catch (err) {
        console.warn('[AUTH] Clerk metadata sync error:', err);
      }
    });
  </script>
</body>
</html>`, clerkKey, userDisplay)

		_, _ = rw.Write([]byte(responseHTML))

		// Auto close server after 10 seconds if no further confirm received
		go func() {
			time.Sleep(10 * time.Second)
			activeAuthServer.mu.Lock()
			if activeAuthServer.server == server {
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
		targetURL = fmt.Sprintf("https://innocent-orca-5605.accounts.dev/user?redirect_url=%s&force_redirect_url=%s&after_sign_in_url=%s&after_sign_up_url=%s", escapedCallback, escapedCallback, escapedCallback, escapedCallback)
	} else {
		targetURL = fmt.Sprintf("https://innocent-orca-5605.accounts.dev/sign-in?redirect_url=%s&force_redirect_url=%s&after_sign_in_url=%s&after_sign_up_url=%s", escapedCallback, escapedCallback, escapedCallback, escapedCallback)
	}

	if a.ctx != nil {
		wailsruntime.BrowserOpenURL(a.ctx, targetURL)
	}

	return map[string]interface{}{
		"url": targetURL,
	}, nil
}
