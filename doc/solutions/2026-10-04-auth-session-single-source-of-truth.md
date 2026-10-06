# Solution: Go Backend Single Source of Truth for Session & Entitlements

## Date: 2026-10-04

---

## 1. Problem Overview

When managing user authentication, offline session persistence, and Pro entitlement verification across a Wails desktop application (Go backend + Vue frontend), a critical dual-source state drift and security vulnerability was identified:

1. **Dual Source of Truth & Vulnerability**:
   - `localStorage` in the browser webview cached `user`, `isPro`, and `lastVerifiedAt`.
   - If `session.json` was deleted or missing, the backend fell back to trusting frontend-supplied arguments (`isPro: true`), re-hydrating a Pro session into `session.json`.
   - In addition, the frontend `clerkAuth.js` preserved `isPro = true` if `localStorage` had a valid timestamp, overriding Go's `false` response.
   - **Exploit Vector**: Any user could edit `localStorage` in browser DevTools or manipulate frontend bridge calls to bypass Pro checks, re-signing and unlocking Pro extensions locally without purchasing a subscription.

2. **State Inconsistency ("Zombie" Session)**:
   - Upon signing out or deleting `session.json`, `user.value` survived in `localStorage` while backend reported `isPro: false`.
   - The UI showed a contradictory state: **"Connected + FREE PLAN"** instead of clean sign-out / reset.

3. **Violation of Core Architecture Invariant**:
   - Violated [AGENTS.md](file:///c:/Users/vishn/PROJECT/ai-tutor/AGENTS.md#L9-L15) Invariant 2: *"Thin Frontend: UI holds only ephemeral state. All business logic, state mutations, and validation live in the Go backend."*

---

## 2. Technical Implementation

### A. Go Backend as the Authoritative Single Source of Truth ([internal/app/app_auth.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app_auth.go))

1. **Strict Session File Verification**:
   - `RestoreSession()` no longer accepts or trusts client-provided entitlement claims or fallback parameters.
   - Session restoration reads **strictly** from the machine-bound, HMAC-SHA256 signed `session.json` on disk:
     - Machine ID + User ID + Email + IsPro + VerifiedAt are validated against `Signature`.
     - 10-day offline grace period (`(now - sess.VerifiedAt) <= 10 * 24 * 3600`) is computed and enforced strictly within Go.
2. **Fail Closed**:
   - If `session.json` is missing, unreadable, or tampered with, `RestoreSession()` immediately resets in-memory session to Free/Unauthenticated (`a.applyRestoredSession("", "", false, 0)`) and returns `false`.

```go
func (a *App) RestoreSession(userID, email string, isPro bool, verifiedAt int64) bool {
    filePath, err := runtime.ResolveSessionPath()
    if err != nil {
        a.applyRestoredSession("", "", false, 0)
        return false
    }

    data, err := os.ReadFile(filePath)
    if err != nil {
        utils.Warnf("[AUTH] Session file not found at %s (err=%v). Resetting active session to Free.", filePath, err)
        a.applyRestoredSession("", "", false, 0)
        return false
    }

    // Unmarshal and verify HMAC signature + 10-day expiry purely within Go
    ...
}
```

### B. Thin Frontend State & Clean Synchronization ([frontend/src/services/clerkAuth.js](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/services/clerkAuth.js))

1. **Unidirectional State Synchronization**:
   - On initialization or sync, `clerkAuth.js` invokes `restoreSession('', '', false, 0)` followed by `getUserSession()`.
   - If the backend reports a valid user session, `user.value` and `isPro.value` are hydrated from backend data.
   - If the backend reports no valid session (empty `userId` or `isPro: false`):
     - The frontend immediately clears `user.value = null`, `isPro.value = false`, and purges `localStorage`.
2. **Elimination of Client-Side Fallback Overrides**:
   - Removed all logic where `localStorage` grace periods could override backend `false` decisions.

```javascript
// 1. Ask Go backend to verify/restore signed session.json
await restoreSession('', '', false, 0)

// 2. Retrieve active session state verified strictly by Go backend
const backendSession = await getUserSession()
if (backendSession && backendSession.userId && backendSession.email) {
  user.value = {
    id: backendSession.userId,
    email: backendSession.email,
    fullName: 'Authenticated User',
  }
  if (backendSession.verifiedAt) {
    lastVerifiedAt.value = backendSession.verifiedAt * 1000
  }
  isPro.value = Boolean(backendSession.isPro)
  saveLocalSession(user.value, isPro.value, lastVerifiedAt.value)
} else {
  // Go backend session is missing, invalid, or expired -> reset frontend auth state
  user.value = null
  isPro.value = false
  lastVerifiedAt.value = 0
  saveLocalSession(null, false, 0)
}
```

---

## 3. Security & Architecture Invariants Upheld

| Invariant / Requirement | Previous Status | Refactored Status |
| :--- | :--- | :--- |
| **Thin Frontend** | Frontend could override entitlement state | UI is purely reflective of Go backend state |
| **Single Source of Truth** | Split between `localStorage` and `session.json` | Solely `session.json` + Go memory |
| **Tamper Resistance** | Client could fabricate `isPro: true` in `localStorage` | Client inputs ignored; HMAC-signed disk cache required |
| **Offline Grace Period** | Evaluated independently in frontend and backend | Evaluated deterministically inside Go |
| **Sign-Out Cleanliness** | Partial state remained in frontend (`user` without Pro) | Both backend `session.json` and frontend memory/storage wiped |

---

## 4. Verification

1. **Go Unit Test Suite**:
   ```powershell
   go test -short ./internal/...
   ```
   All packages (`internal/app`, `internal/db`, `internal/extension`, `internal/study`, `internal/runtime`, `internal/pomodoro/...`, etc.) compiled and passed without regression.
