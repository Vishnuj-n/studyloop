# Solution & Developer Guide: Adding a Pro Feature to StudyLoop

## Overview

In StudyLoop, subscription entitlements and Pro access control adhere strictly to our core architectural invariants:
- **Thin Frontend & Authoritative Backend**: The frontend reflects subscription state for UI polish (badges, disabling buttons, prompting upgrade modals), but **all authoritative gates and validation live strictly in the Go backend**.
- **Deterministic & Tamper-Proof**: Users cannot bypass Pro restrictions by modifying local files (e.g. `manifest.json`), mocking frontend state, or invoking Wails IPC endpoints directly.
- **Fail Fast / Explicit Denial**: Unauthorized attempts to run Pro features return an explicit structured error response (`"error"`, `"requires_pro": true` or `"is_pro_required": true`).

This guide details the complete end-to-end pattern for designating any capability—whether a Go backend endpoint, an extension, or a UI view—as a **Pro feature**.

---

## 1. Architecture of Pro Entitlements

The subscription lifecycle operates across three layers:

```
┌─────────────────────────────────────────────────────────┐
│              Clerk Auth & Loopback Server               │
│  - Hosted Clerk portal redirects to 127.0.0.1:<port>    │
│  - Browser extracts publicMetadata (role/plan == "pro") │
│  - Loopback server receives and verifies session        │
└───────────────────────────┬─────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│               Go Backend (internal/app)                 │
│  - App.SetSession(userID, email, isPro)                 │
│  - App.RestoreSession(userID, email, isPro, verifiedAt) │
│  - App.IsProUser() bool (Authoritative Check)           │
│  - officialExtensionTiers (Compiled In-Memory Gate)     │
└───────────────────────────┬─────────────────────────────┘
                            │ (Wails IPC Bridge)
                            ▼
┌─────────────────────────────────────────────────────────┐
│               Vue 3 Frontend Client                     │
│  - useClerkAuth() composable (isPro, isSignedIn)        │
│  - 10-day offline grace period cache in localStorage    │
│  - Reactive UI: Early Access tags, upgrade triggers     │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Backend Implementation (Go)

### A. Authoritative Check: `a.IsProUser()`

The `App` struct tracks the session thread-safely in `internal/app/app.go` and `internal/app/app_auth.go`:

```go
// IsProUser returns true only if the active session is an authenticated Pro user.
func (a *App) IsProUser() bool {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return a.sessionIsPro
}
```

### B. Pattern 1: Protecting a Regular Backend Endpoint

When creating or gating any backend method on `*App` (such as deep extraction, advanced exports, custom audio playback, etc.):

1. **Check `a.IsProUser()` immediately** after verifying initialization/database dependencies.
2. **Return a structured error** with a clear flag (`requires_pro: true` or `is_pro_required: true`).

#### Example: Gating Deep Structured PDF Ingestion ([internal/app/notebook_endpoints.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/notebook_endpoints.go))

```go
func (a *App) SelectAndUploadDeepStructuredPDF() map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}

	// 1. Authoritative Pro check
	if !a.IsProUser() {
		return map[string]interface{}{
			"error":        "Deep Structured PDF Ingestion is a Pro feature. Please upgrade your plan to unlock.",
			"requires_pro": true,
		}
	}

	// 2. Proceed with Pro execution
	...
}
```

#### Example: Gating Audio Folder Shuffling ([internal/app/app_pomodoro.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app_pomodoro.go))

```go
func (a *App) PomodoroPlayShuffleFolder(folder string) {
	if !a.IsProUser() {
		return
	}
	if a.pomoAudio != nil {
		a.pomoAudio.PlayShuffleFolder(folder)
	}
}
```

---

### C. Pattern 2: Protecting an Extension

For modular plugins or extensions (internal Go tools, Python scripts via uv, or external binaries):

#### 1. Register in Authoritative Compiled Tiers ([internal/extension/tiers.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/tiers.go))
Do **not** rely solely on the extension's local `manifest.json`, as users can manually edit local files on disk. Register official extensions in `officialExtensionTiers`:

```go
var officialExtensionTiers = map[string]string{
	"text_simplifier":  "free",
	"audio_overview":   "pro",
	"youtube":          "free",
	"deep_pdf":         "pro",
	"reading_simplify": "free",
	"your_new_feature": "pro", // <-- Add your extension ID here
}
```

`GetEffectiveTier(ext)` guarantees that even if a user alters their `manifest.json` to `"tier": "free"`, Go overrides it to `"pro"`.

#### 2. Authoritative Execution Gate ([internal/app/app_extension.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app_extension.go))
The extension execution pipeline automatically enforces this:

```go
effectiveTier := extension.GetEffectiveTier(ext)
if effectiveTier == "pro" && !a.IsProUser() {
	return map[string]interface{}{
		"error":           fmt.Sprintf("extension %q requires a Pro subscription", ext.Name()),
		"is_pro_required": true,
	}
}
```

---

## 3. Frontend Implementation (Vue 3)

### A. Accessing Pro Status in Components

Import `useClerkAuth` from `frontend/src/services/clerkAuth.js`:

```javascript
import { useClerkAuth } from '../services/clerkAuth'

const { isPro, isSignedIn, signIn } = useClerkAuth()
```

- `isPro.value`: Boolean indicating whether user has an active Pro entitlement (verified online or within the 10-day offline grace period).
- `isSignedIn.value`: Boolean indicating whether user is logged in.

---

### B. UI Presentation Guidelines

1. **Badge / Labeling**:
   - Use badge styling with `EARLY ACCESS` or `★ PRO` tags.
   ```html
   <span v-if="isProFeature" class="tier-tag pro">EARLY ACCESS</span>
   ```

2. **Action Buttons**:
   - If user is NOT Pro, the action button should either prompt for upgrade or open the login/subscription flow:
   ```html
   <button 
     v-if="!isPro" 
     class="action-btn unlock-btn" 
     @click="router.push('/settings?tab=account')"
   >
     ⭐ Unlock with Pro
   </button>
   <button 
     v-else 
     class="action-btn primary-action" 
     @click="handleRunFeature"
   >
     Run Feature
   </button>
   ```

3. **Handling Backend Rejections Gracefully**:
   Always handle `{ requires_pro: true }` or `{ is_pro_required: true }` from backend API responses:
   ```javascript
   const res = await runMyFeature()
   if (res?.requires_pro || res?.is_pro_required) {
     showToast('This feature requires an active Pro subscription.')
     router.push('/settings?tab=account')
     return
   }
   if (res?.error) {
     showToast(`Error: ${res.error}`)
     return
   }
   ```

---

## 4. Offline & Grace Period Behavior

- When an online user authenticates via Clerk, the Go loopback server receives `publicMetadata` (`plan === "pro"` or `role === "pro"`).
- The desktop app stores this state in `localStorage` with a `lastVerifiedAt` timestamp.
- **Grace Period**: 10 days (`10 * 24 * 60 * 60 * 1000 ms`).
- On app launch, `RestoreSession` in Go validates:
  ```go
  if isPro && verifiedAt > 0 && (now-verifiedAt) > tenDaysSec {
      isPro = false // Expired offline grace period, revoke pro until online check
  }
  ```
- If expired, the app seamlessly falls back to the Free tier until the user reconnects and completes verification.

---

## 5. Development & Testing Checklist

### Testing as Free vs Pro Locally

1. **Dev Mode Simulation in Settings**:
   - Navigate to **Settings → Account**.
   - Use the **Dev Pro Simulator** toggle to instantly switch between Free and Pro states during local UI testing.
2. **Go Unit Tests**:
   - Always add unit tests ensuring unauthorized calls are rejected when `!a.IsProUser()`:
   ```go
   func TestMyFeature_ProEnforced(t *testing.T) {
       app := &App{}
       // Session starts as non-pro
       res := app.MyProEndpoint()
       if res["requires_pro"] != true {
           t.Fatalf("expected requires_pro=true, got %v", res)
       }

       // Set Pro session
       app.SetSession("test_uid", "user@example.com", true)
       res2 := app.MyProEndpoint()
       if res2["requires_pro"] == true {
           t.Fatalf("expected feature to proceed for pro user")
       }
   }
   ```
3. **Run Validation Commands**:
   - Instant compile/type check: `go test -run=^$ ./internal/...`
   - Rapid unit tests: `go test -short ./internal/...`
   - Full backend test suite: `go test ./internal/...`
   - Frontend production build: `cd frontend && npm run build`
