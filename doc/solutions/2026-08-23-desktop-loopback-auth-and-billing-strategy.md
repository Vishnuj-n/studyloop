# Desktop Authentication Architecture & Freemium Billing Strategy

## Date: 2026-08-23

---

## 1. Problem Overview

In a Wails desktop application (`.exe`), the app frontend runs inside an embedded WebView2/WebKit environment. When implementing cloud authentication and billing (via Clerk / Stripe), two critical architectural problems arise:

1. **In-App Modal Freezes**:
   - Loading heavy remote auth bundles (like Clerk's modal iframe) inside a desktop webview can cause cold-load freezes (3–5s dark screens) and block hardware-accelerated passkeys or browser autofill.
2. **The Desktop vs. Browser Cookie Sandbox**:
   - When launching an external browser (Chrome/Edge) to complete OAuth or payment, Chrome holds the session cookie, but the desktop `.exe` cannot access Chrome's private cookie jar.
   - Without a handshake mechanism, the desktop app cannot know who just logged in.

---

## 2. Technical Solution: Ephemeral Local Loopback Server

To achieve instant browser loading and 100% reliable session transfer back to the desktop application, StudyLoop implements the **Ephemeral Local Loopback Handshake** (the same standard used by Google Cloud CLI, GitHub CLI, Spotify, and VS Code).

```
┌─────────────────┐    1. Starts Loopback Listener (127.0.0.1:0)    ┌───────────────────────────────────┐
│ StudyLoop App   │ ──────────────────────────────────────────────> │ Browser (Chrome/Edge)             │
│ (Desktop .exe)  │    2. Opens Browser with dynamic redirect_url   │ User logs in / Stripe checkout    │
└────────┬────────┘                                                 └─────────────────┬─────────────────┘
         ▲                                                                            │
         │             3. Redirects to http://127.0.0.1:<port>/callback               │
         └────────────────────────────────────────────────────────────────────────────┘
                       4. Go Backend parses session -> Emits Wails Event
                       5. Desktop window unminimizes & activates Pro!
```

### Key Implementation Details:

1. **Dynamic Kernel-Assigned Port (`127.0.0.1:0`)**:
   - Binding to port `0` instructs the operating system kernel to assign a guaranteed free, collision-free TCP port.
   - Avoids hardcoded port collisions across developer and user machines.

2. **Backend Handler (`internal/app/app_auth.go`)**:
   - `StartBrowserAuth(mode string)` creates the temporary HTTP server and returns the target URL:
     `https://<app-id>.accounts.dev/sign-in?redirect_url=http://127.0.0.1:<port>/callback`
   - Listens on `/callback`, extracts `user_id`, `email`, and `is_pro` status.
   - Emits `clerk_auth_success` event via `wailsruntime.EventsEmit` and calls `wailsruntime.WindowUnminimise`.
   - Shuts down cleanly after handling the callback or after a 5-minute timeout.

3. **Frontend Integration (`frontend/src/services/clerkAuth.js`)**:
   - `signIn()` and `openBilling()` call `startBrowserAuth()` to launch the user's default browser.
   - Listens to `clerk_auth_success` to update reactive `user.value` and `isPro.value`, persisting session to `localStorage`.

---

## 3. Production Readiness with Live Keys (`pk_live_...`)

When switching from development (`pk_test_...`) to production (`pk_live_...`):
- **No Warning Banners**: Clerk removes all development banners (`"Development mode..."`).
- **Custom Branding**: Routes seamlessly to `accounts.yourdomain.com`.
- **Payment Verification**: Stripe webhooks inside Clerk automatically update user metadata to active Pro, which flows back through the loopback callback.

---

## 4. StudyLoop Billing Strategy: One-Time vs. Monthly

### Comparison:

| Strategy | Advantages | Disadvantages | Suitability |
| :--- | :--- | :--- | :--- |
| **Monthly Subscription**<br>*(e.g., $8–$12/month)* | Predictable recurring revenue to cover ongoing hosted cloud LLM token costs. | Higher friction; consumer "subscription fatigue" lowers initial conversion. | Best if StudyLoop pays for all user LLM inferences on private cloud servers. |
| **One-Time Lifetime License**<br>*(e.g., $29–$49 one-off)* | Highest conversion rate for desktop software (similar to Obsidian, Raycast, Sublime Text, Typora). | No recurring income to subsidize ongoing third-party API costs. | **Ideal when users provide their own API keys** (Gemini/OpenAI/Ollama). |
| **The Hybrid Freemium Model** ⭐ | Core app + standard features are Free / One-Time Pro; hosted cloud AI tokens or heavy extensions are billed by usage or cloud subscription. | Requires two distinct tiers in product marketing. | **Recommended for StudyLoop**. |

### Recommended Decision for StudyLoop:
- **Bring-Your-Own-Key (BYOK) Model**: Charge a **One-Time Pro Lifetime Unlock ($29–$39)** or **Annual Pass ($24/year)** for Pro extensions (e.g., AI Audio Overview, Advanced Analytics, Classroom Multi-Profile).
- **Zero Server Liability**: Users configure their own Gemini API key or local Ollama runtime, meaning 100% of Pro revenue is pure margin with zero ongoing cloud inference bills.
