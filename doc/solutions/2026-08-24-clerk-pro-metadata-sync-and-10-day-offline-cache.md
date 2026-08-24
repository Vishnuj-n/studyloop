# Solution: Clerk Pro Metadata Synchronization & 10-Day Offline Cache

## Date: 2026-08-24

---

## 1. Problem Overview

When integrating **Clerk Authentication & Entitlements** with a desktop application (`.exe`), two issues occurred:
1. **Metadata Loss during Browser Loopback**:
   - Clerk's hosted Account Portal (`accounts.dev/sign-in`) redirects back to the desktop loopback URL (`http://127.0.0.1:<port>/callback`) without appending custom `publicMetadata` (`plan: "pro"`, `role: "pro"`, `is_pro: true`) to the URL query string.
   - As a result, the Go backend defaulted `isPro` to `false`, overwriting the user's local session.
2. **Offline & Periodic Entitlement Verification**:
   - Desktop apps require deterministic offline behavior without constant server pings while preventing expired subscriptions from persisting indefinitely.

---

## 2. Technical Implementation

### A. Authoritative Clerk JS Metadata Extraction ([internal/app/app_auth.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app_auth.go))
- When the browser finishes sign-in and reaches `http://127.0.0.1:<port>/callback`:
  - The callback HTML loads standard Clerk JS in the browser context where Clerk's active session and cookies reside.
  - The script executes `await window.Clerk.load()`, retrieves `window.Clerk.user.publicMetadata`, and extracts:
    - `isPro = meta.role === "pro" || meta.plan === "pro" || meta.is_pro === true`
  - The browser script then POSTs `{ userId, email, isPro, plan, role }` to `/api/auth-confirm` on the Go loopback server.
- The Go backend receives the verified metadata, emits `clerk_auth_success` via Wails runtime, and brings the desktop app window into focus (`wailsruntime.WindowUnminimise`).

### B. 10-Day Offline Grace Period Cache ([frontend/src/services/clerkAuth.js](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/services/clerkAuth.js))
- When `clerk_auth_success` is received, the session is persisted to `localStorage` alongside `lastVerifiedAt: Date.now()`.
- On application launch:
  ```javascript
  const TEN_DAYS_MS = 10 * 24 * 60 * 60 * 1000
  const isWithinGracePeriod = (Date.now() - savedTime) < TEN_DAYS_MS
  if (parsed.isPro && !isWithinGracePeriod) {
    // Grace period expired, requires fresh online check
    isPro.value = false
  } else {
    isPro.value = !!parsed.isPro
  }
  ```
- If the user is offline within 10 days of their last online verification, Pro features and extensions remain fully accessible.

---

## 3. Verification

1. **Automated Backend Tests**:
   ```powershell
   go test ./internal/...
   ```
   All test suites in `internal/app`, `internal/db`, `internal/extension`, `internal/study`, and `internal/runtime` passed with 0 failures.

2. **Frontend Production Build**:
   ```powershell
   cd frontend && npm run build
   ```
   Vite compiled all 426 modules and generated the production bundle cleanly.
