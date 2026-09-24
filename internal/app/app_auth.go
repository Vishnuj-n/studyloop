package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-tutor/internal/runtime"
	"ai-tutor/internal/utils"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ClerkPublishableKey can be set at compile time via:
// -ldflags "-X ai-tutor/internal/app.ClerkPublishableKey=pk_live_..."
var ClerkPublishableKey string

const (
	tenDaysSec        = 10 * 24 * 60 * 60
	sessionSecretSalt = "studyloop-desktop-auth-salt-v1-wails"
)

type AuthCallbackResult struct {
	Success bool   `json:"success"`
	UserID  string `json:"userId"`
	Email   string `json:"email"`
	IsPro   bool   `json:"isPro"`
	Error   string `json:"error,omitempty"`
}

type persistentSession struct {
	UserID     string `json:"userId"`
	Email      string `json:"email"`
	IsPro      bool   `json:"isPro"`
	VerifiedAt int64  `json:"verifiedAt"`
	Signature  string `json:"signature"`
}

type authServerState struct {
	mu         sync.Mutex
	server     *http.Server
	stateNonce string
}

var activeAuthServer = &authServerState{}

func getMachineID() string {
	h, _ := os.Hostname()
	u := os.Getenv("USERNAME")
	if u == "" {
		u = os.Getenv("USER")
	}
	return strings.TrimSpace(h + "::" + u)
}

func computeSessionSignature(machineID, userID, email string, isPro bool, verifiedAt int64) string {
	payload := fmt.Sprintf("%s|%s|%s|%t|%d", machineID, userID, email, isPro, verifiedAt)
	mac := hmac.New(sha256.New, []byte(sessionSecretSalt))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func getSessionFilePath() (string, error) {
	dir, err := runtime.ResolveAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

func (a *App) persistSession(userID, email string, isPro bool, verifiedAt int64) {
	filePath, err := getSessionFilePath()
	if err != nil {
		utils.Warnf("[AUTH] Failed to resolve session file path: %v", err)
		return
	}

	sig := computeSessionSignature(getMachineID(), userID, email, isPro, verifiedAt)
	sess := persistentSession{
		UserID:     userID,
		Email:      email,
		IsPro:      isPro,
		VerifiedAt: verifiedAt,
		Signature:  sig,
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		utils.Warnf("[AUTH] Failed to marshal session: %v", err)
		return
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		utils.Warnf("[AUTH] Failed to write session file: %v", err)
	}
}

// setSession sets the active session in memory and persists signed session to disk.
func (a *App) setSession(userID, email string, isPro bool) {
	now := time.Now().Unix()
	a.sessionMu.Lock()
	a.sessionUserID = userID
	a.sessionEmail = email
	a.sessionIsPro = isPro
	a.sessionVerifiedAt = now
	a.sessionMu.Unlock()

	a.persistSession(userID, email, isPro, now)
}

// RestoreSession allows frontend to re-hydrate offline session on launch within grace period.
// Untrusted frontend arguments are discarded to prevent DevTools/localStorage tampering;
// session is verified from machine-bound signed storage on disk.
func (a *App) RestoreSession(userID, email string, isPro bool, verifiedAt int64) bool {
	filePath, err := getSessionFilePath()
	if err != nil {
		utils.Warnf("[AUTH] Could not resolve session file path: %v", err)
		a.applyRestoredSession("", "", false, 0)
		return false
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		// File does not exist or unreadable -> no offline session
		a.applyRestoredSession("", "", false, 0)
		return false
	}

	var sess persistentSession
	if err := json.Unmarshal(data, &sess); err != nil {
		utils.Warnf("[AUTH] Invalid session JSON format: %v", err)
		a.applyRestoredSession("", "", false, 0)
		return false
	}

	// Verify HMAC signature
	expectedSig := computeSessionSignature(getMachineID(), sess.UserID, sess.Email, sess.IsPro, sess.VerifiedAt)
	if !hmac.Equal([]byte(sess.Signature), []byte(expectedSig)) {
		utils.Warnf("[AUTH] Session signature mismatch or tampering detected. Downgrading to free.")
		a.applyRestoredSession(sess.UserID, sess.Email, false, 0)
		return false
	}

	// Check 10-day offline grace period
	now := time.Now().Unix()
	actualIsPro := sess.IsPro
	if actualIsPro && sess.VerifiedAt > 0 && (now-sess.VerifiedAt) > tenDaysSec {
		utils.Warnf("[AUTH] Session 10-day grace period expired. Re-verification required.")
		actualIsPro = false
	}

	a.applyRestoredSession(sess.UserID, sess.Email, actualIsPro, sess.VerifiedAt)
	return actualIsPro
}

func (a *App) applyRestoredSession(userID, email string, isPro bool, verifiedAt int64) {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	a.sessionUserID = userID
	a.sessionEmail = email
	a.sessionIsPro = isPro
	a.sessionVerifiedAt = verifiedAt
}

// ClearSession clears the current in-memory session and removes disk cache.
func (a *App) ClearSession() {
	a.sessionMu.Lock()
	a.sessionUserID = ""
	a.sessionEmail = ""
	a.sessionIsPro = false
	a.sessionVerifiedAt = 0
	a.sessionMu.Unlock()

	if filePath, err := getSessionFilePath(); err == nil {
		_ = os.Remove(filePath)
	}
}

// IsProUser returns true only if the active session is an authenticated Pro user.
func (a *App) IsProUser() bool {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return a.sessionIsPro
}

// getUserSession returns the current active session state.
func (a *App) getUserSession() map[string]interface{} {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return map[string]interface{}{
		"userId":     a.sessionUserID,
		"email":      a.sessionEmail,
		"isPro":      a.sessionIsPro,
		"verifiedAt": a.sessionVerifiedAt,
	}
}

const (
	DefaultClerkPublishableKey = "pk_test_aW5ub2NlbnQtb3JjYS01NjA1LmNsZXJrLmFjY291bnRzLmRldiQ"
)

func resolveClerkPublishableKey() string {
	if ClerkPublishableKey != "" {
		return ClerkPublishableKey
	}
	if k := os.Getenv("VITE_CLERK_PUBLISHABLE_KEY"); k != "" {
		return k
	}
	if k := os.Getenv("CLERK_PUBLISHABLE_KEY"); k != "" {
		return k
	}
	return DefaultClerkPublishableKey
}

// StartBrowserAuth spins up an ephemeral HTTP server on 127.0.0.1:0 and returns the browser login URL.
func (a *App) StartBrowserAuth(mode string) (map[string]interface{}, error) {
	utils.Infof("[AUTH] StartBrowserAuth requested with mode: %s", mode)
	activeAuthServer.mu.Lock()
	if activeAuthServer.server != nil {
		_ = activeAuthServer.server.Close()
		activeAuthServer.server = nil
		activeAuthServer.stateNonce = ""
	}
	activeAuthServer.mu.Unlock()

	// Generate a 16-byte random hex nonce (32 hex characters) for CSRF/loopback protection
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		utils.Warnf("[AUTH] Failed to generate crypto nonce: %v", err)
	}
	stateNonce := hex.EncodeToString(nonceBytes)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		utils.Warnf("[AUTH] Failed to create local loopback listener: %v", err)
		return nil, fmt.Errorf("failed to start local auth listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?state=%s", port, stateNonce)
	utils.Infof("[AUTH] Created loopback server on port %d with callback: %s", port, callbackURL)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
	}

	activeAuthServer.mu.Lock()
	activeAuthServer.server = server
	activeAuthServer.stateNonce = stateNonce
	activeAuthServer.mu.Unlock()

	clerkKey := resolveClerkPublishableKey()
	if clerkKey == "" {
		// In development, log notice
		utils.Warnf("[AUTH] No Clerk publishable key configured via ldflags or environment variables")
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
			State  string `json:"state"`
		}

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			utils.Warnf("[AUTH] Failed to decode /api/auth-confirm body: %v", err)
			http.Error(rw, "Invalid body", http.StatusBadRequest)
			return
		}

		// Verify state nonce
		activeAuthServer.mu.Lock()
		expectedState := activeAuthServer.stateNonce
		activeAuthServer.mu.Unlock()

		headerState := req.Header.Get("X-Auth-State")
		incomingState := payload.State
		if incomingState == "" {
			incomingState = headerState
		}

		if expectedState == "" || incomingState != expectedState {
			utils.Warnf("[AUTH] /api/auth-confirm rejected: invalid state nonce")
			http.Error(rw, "Forbidden: invalid state nonce", http.StatusForbidden)
			return
		}

		plan := strings.ToLower(strings.TrimSpace(payload.Plan))
		role := strings.ToLower(strings.TrimSpace(payload.Role))
		isPro := payload.IsPro || plan == "pro" || role == "pro"
		utils.Infof("[AUTH] Authoritative Clerk JS sync: user=%s email=%s plan=%s role=%s -> isPro=%v",
			payload.UserID, payload.Email, payload.Plan, payload.Role, isPro)

		a.setSession(payload.UserID, payload.Email, isPro)

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
				activeAuthServer.stateNonce = ""
			}
			activeAuthServer.mu.Unlock()
		}()
	})

	mux.HandleFunc("/callback", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		givenState := q.Get("state")

		activeAuthServer.mu.Lock()
		expectedState := activeAuthServer.stateNonce
		activeAuthServer.mu.Unlock()

		utils.Infof("[AUTH] GET /callback hit with full URL: %s | Query: %v | Headers: %v", req.URL.String(), req.URL.RawQuery, req.Header)
		fmt.Printf("[AUTH] GET /callback hit with query: %s\n", req.URL.RawQuery)

		if expectedState == "" || givenState != expectedState {
			utils.Warnf("[AUTH] /callback rejected: state nonce mismatch. expected=%s, given=%s", expectedState, givenState)
			fmt.Printf("[AUTH] /callback rejected: state nonce mismatch. expected=%s, given=%s\n", expectedState, givenState)
			http.Error(rw, "Forbidden: invalid state nonce", http.StatusForbidden)
			return
		}

		userID := q.Get("user_id")
		if userID == "" {
			userID = q.Get("userId")
		}
		if userID == "" {
			userID = q.Get("id")
		}

		email := q.Get("email")
		if email == "" {
			email = q.Get("primary_email_address")
		}

		planStr := strings.ToLower(strings.TrimSpace(q.Get("plan")))
		roleStr := strings.ToLower(strings.TrimSpace(q.Get("role")))
		isProStr := strings.ToLower(strings.TrimSpace(q.Get("is_pro")))
		if isProStr == "" {
			isProStr = strings.ToLower(strings.TrimSpace(q.Get("isPro")))
		}

		isPro := isProStr == "true" || isProStr == "1" || isProStr == "pro" || isProStr == "yes" || planStr == "pro" || roleStr == "pro"

		// Only emit session immediately if email/user were explicitly provided in query params;
		// otherwise, let the browser Clerk JS script authoritatively confirm via /api/auth-confirm.
		if email != "" && userID != "" {
			utils.Infof("[AUTH] Query param session detected on /callback: user=%s email=%s isPro=%v", userID, email, isPro)
			fmt.Printf("[AUTH] Query param session detected on /callback: user=%s email=%s isPro=%v\n", userID, email, isPro)
			a.setSession(userID, email, isPro)
			result := AuthCallbackResult{
				Success: true,
				UserID:  userID,
				Email:   email,
				IsPro:   isPro,
			}
			if a.ctx != nil {
				wailsruntime.EventsEmit(a.ctx, "clerk_auth_success", result)
			}
		} else {
			utils.Infof("[AUTH] Awaiting client-side Clerk JS metadata confirmation from browser...")
			fmt.Println("[AUTH] Awaiting client-side Clerk JS metadata confirmation from browser...")
		}

		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		userDisplay := email
		if userDisplay == "" {
			userDisplay = "Pro User"
		}

		escapedClerkKey := html.EscapeString(clerkKey)
		escapedUserDisplay := html.EscapeString(userDisplay)
		escapedState := html.EscapeString(expectedState)

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
        inset 0 1px 0 rgba(255, 255, 255, 0.1);
    }
    .brand-row {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      margin-bottom: 24px;
    }
    .brand-logo {
      width: 32px;
      height: 32px;
      border-radius: 8px;
      background: linear-gradient(135deg, #6366f1, #8b5cf6);
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: 800;
      font-size: 16px;
      color: #fff;
    }
    .brand-name {
      font-family: 'Manrope', sans-serif;
      font-weight: 700;
      font-size: 18px;
      color: #f8fafc;
      letter-spacing: -0.3px;
    }
    .success-icon-wrap {
      width: 72px;
      height: 72px;
      border-radius: 50%%;
      background: rgba(16, 185, 129, 0.12);
      border: 1.5px solid rgba(16, 185, 129, 0.35);
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 0 auto 20px;
      animation: popIn 0.5s cubic-bezier(0.175, 0.885, 0.32, 1.275) both;
    }
    @keyframes popIn {
      0%% { transform: scale(0.5); opacity: 0; }
      100%% { transform: scale(1); opacity: 1; }
    }
    .check-svg {
      width: 36px;
      height: 36px;
      stroke: #10b981;
      stroke-width: 2.5;
      fill: none;
      stroke-linecap: round;
      stroke-linejoin: round;
    }
    h1 {
      font-family: 'Manrope', sans-serif;
      font-size: 22px;
      font-weight: 700;
      color: #f8fafc;
      margin-bottom: 8px;
      letter-spacing: -0.4px;
    }
    .user-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: rgba(255, 255, 255, 0.06);
      border: 1px solid rgba(255, 255, 255, 0.08);
      border-radius: 20px;
      padding: 4px 12px;
      font-size: 13px;
      color: #94a3b8;
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

    <p id="status-desc">Connecting your account to the desktop workspace...</p>

    <div class="action-badge">
      <span>Switch back to StudyLoop app</span>
    </div>
  </div>

  <script>
    function logMsg(msg) {
      console.log('[AUTH_CALLBACK]', msg);
    }

    async function runVerification() {
      const stateNonce = "%s";
      const clerkKey = "%s";
      logMsg('Init auth verification for state: ' + stateNonce.substring(0, 8) + '...');

      let clerk = window.Clerk;
      for (let i = 0; i < 50 && !clerk; i++) {
        await new Promise(r => setTimeout(r, 100));
        clerk = window.Clerk;
      }

      if (!clerk) {
        logMsg('ERROR: Clerk JS failed to mount on window. Check internet / adblocker.');
        return;
      }

      logMsg('Clerk JS mounted. Loading instance...');
      try {
        if (!clerk.loaded) {
          await clerk.load({ publishableKey: clerkKey });
        }
        logMsg('Clerk load() complete.');
      } catch (loadErr) {
        logMsg('Clerk load() error: ' + (loadErr.message || loadErr));
      }

      const currentUser = clerk.user;
      if (!currentUser) {
        logMsg('No active Clerk session found in browser. Please sign in below:');
        const emailEl = document.getElementById('user-display-email');
        if (emailEl) emailEl.textContent = 'Not Signed In';
        
        // Provide in-place sign in modal trigger
        if (clerk.openSignIn) {
          clerk.openSignIn();
        }
        return;
      }

      const userId = currentUser.id || '';
      const email = currentUser.primaryEmailAddress?.emailAddress || 
                    (currentUser.emailAddresses && currentUser.emailAddresses[0]?.emailAddress) || 
                    'User';
      
      const pubMetadata = currentUser.publicMetadata || {};
      const unsafeMetadata = currentUser.unsafeMetadata || {};
      
      logMsg('User identified: ' + email + ' (ID: ' + userId + ')');
      logMsg('Public Metadata: ' + JSON.stringify(pubMetadata));
      logMsg('Unsafe Metadata: ' + JSON.stringify(unsafeMetadata));

      const plan = String(pubMetadata.plan || pubMetadata.tier || pubMetadata.subscription || unsafeMetadata.plan || unsafeMetadata.tier || unsafeMetadata.subscription || '').toLowerCase();
      const role = String(pubMetadata.role || unsafeMetadata.role || '').toLowerCase();
      
      const isProFlag = pubMetadata.isPro === true || pubMetadata.is_pro === true || pubMetadata.pro === true ||
                        unsafeMetadata.isPro === true || unsafeMetadata.is_pro === true || unsafeMetadata.pro === true ||
                        pubMetadata.isPro === 'true' || pubMetadata.is_pro === 'true' || pubMetadata.pro === 'true' ||
                        unsafeMetadata.isPro === 'true' || unsafeMetadata.is_pro === 'true';
                        
      const isProPlan = plan.includes('pro') || plan.includes('premium') || plan.includes('lifetime') || plan.includes('active') || plan.includes('supporter') || plan.includes('early_access');
      const isProRole = role.includes('pro') || role.includes('admin') || role.includes('owner');
      
      let isPro = Boolean(isProFlag || isProPlan || isProRole);

      if (!isPro && Array.isArray(currentUser.organizationMemberships)) {
        for (const org of currentUser.organizationMemberships) {
          const orgRole = String(org.role || '').toLowerCase();
          logMsg('Org membership: ' + org.organization?.name + ' (role: ' + orgRole + ')');
          if (orgRole.includes('pro') || orgRole.includes('admin') || orgRole.includes('member') || orgRole.includes('owner')) {
            isPro = true;
            break;
          }
        }
      }

      logMsg('Resolved Entitlement: isPro=' + isPro + ' (plan=' + plan + ', role=' + role + ')');

      // Update UI badge
      const emailEl = document.getElementById('user-display-email');
      if (emailEl) {
        emailEl.textContent = email + (isPro ? ' (★ Pro Plan)' : ' (Free Plan)');
      }

      // Send authoritative confirmation to Go backend
      try {
        logMsg('Sending confirmation to desktop backend (/api/auth-confirm)...');
        const res = await fetch('/api/auth-confirm', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Auth-State': stateNonce
          },
          body: JSON.stringify({
            userId: userId,
            email: email,
            isPro: isPro,
            plan: plan,
            role: role,
            state: stateNonce
          })
        });
        const json = await res.json();
        logMsg('Backend confirmed sync: ' + JSON.stringify(json));
        const statusDesc = document.getElementById('status-desc');
        if (statusDesc) {
          statusDesc.textContent = 'Your StudyLoop desktop workspace is now connected and ready.';
        }
      } catch (fetchErr) {
        logMsg('Backend confirm fetch error: ' + (fetchErr.message || fetchErr));
      }
    }

    window.addEventListener('DOMContentLoaded', runVerification);
    window.addEventListener('load', runVerification);
  </script>
</body>
</html>`, escapedClerkKey, escapedUserDisplay, escapedState, escapedClerkKey)

		_, _ = rw.Write([]byte(responseHTML))

		// Auto close server after 10 seconds if no further confirm received
		go func() {
			time.Sleep(10 * time.Second)
			activeAuthServer.mu.Lock()
			if activeAuthServer.server == server {
				_ = activeAuthServer.server.Close()
				activeAuthServer.server = nil
				activeAuthServer.stateNonce = ""
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
			activeAuthServer.stateNonce = ""
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
