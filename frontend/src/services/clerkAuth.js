import { ref, computed } from 'vue'
import { startBrowserAuth, openURLInBrowser, restoreSession, getUserSession, clearSession, logFrontendEvent } from './appApi'

const isLoaded = ref(false)
const user = ref(null)
const isPro = ref(false)
const authError = ref('')
const TEN_DAYS_MS = 10 * 24 * 60 * 60 * 1000
const lastVerifiedAt = ref(Date.now())

function saveLocalSession(u, proStatus, verifiedTime) {
  if (u) {
    localStorage.setItem('studyloop_user_session', JSON.stringify({
      user: u,
      isPro: !!proStatus,
      lastVerifiedAt: verifiedTime || Date.now(),
    }))
  } else {
    localStorage.removeItem('studyloop_user_session')
  }
}

async function waitForBridge(timeoutMs = 3000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    if (typeof window !== 'undefined' && (window?.go?.app?.App || window?.go?.main?.App)) {
      return true
    }
    await new Promise((r) => setTimeout(r, 50))
  }
  return false
}

let inFlightSync = null
let lastSyncSuccessTime = 0
const SYNC_THROTTLE_MS = 15000

async function syncWithBackend(force = false) {
  if (!force && lastSyncSuccessTime > 0 && Date.now() - lastSyncSuccessTime < SYNC_THROTTLE_MS) {
    return
  }
  if (inFlightSync) {
    return inFlightSync
  }

  inFlightSync = (async () => {
    try {
      const bridgeReady = await waitForBridge(2000)
      if (!bridgeReady) {
        return
      }

      // First call backend restoreSession to validate / hydrate session from disk
      const verifiedPro = await restoreSession(
        user.value?.id || '',
        user.value?.email || '',
        isPro.value,
        Math.floor((lastVerifiedAt.value || Date.now()) / 1000)
      )

      // Retrieve active session details from Go backend (persisted session.json)
      const backendSession = await getUserSession()
      if (backendSession && backendSession.email) {
        if (!user.value) {
          user.value = {
            id: backendSession.userId || 'user_' + Date.now(),
            email: backendSession.email,
            fullName: 'Authenticated User',
          }
        }
        if (backendSession.verifiedAt) {
          lastVerifiedAt.value = backendSession.verifiedAt * 1000
        }
        isPro.value = !!backendSession.isPro
        saveLocalSession(user.value, isPro.value, lastVerifiedAt.value)
      } else if (typeof verifiedPro === 'boolean') {
        if (isPro.value !== verifiedPro) {
          console.warn(`[AUTH] isPro status changed by backend verification: ${isPro.value} -> ${verifiedPro}`)
          logFrontendEvent('warn', 'Auth', 'is_pro_transition', { from: isPro.value, to: verifiedPro })
        }
        isPro.value = verifiedPro
        if (user.value) {
          saveLocalSession(user.value, isPro.value, lastVerifiedAt.value)
        }
      }
      lastSyncSuccessTime = Date.now()
    } catch (err) {
      console.warn('[AUTH] Could not sync session with backend:', err)
    } finally {
      inFlightSync = null
    }
  })()

  return inFlightSync
}

export async function initClerk() {
  console.log('[AUTH] initClerk() called. Initial User:', user.value?.email, 'isPro:', isPro.value)
  isLoaded.value = true
  await syncWithBackend()
  return null
}

// Listen for loopback authentication callback from Go backend
if (typeof window !== 'undefined' && window?.runtime?.EventsOn) {
  window.runtime.EventsOn('clerk_auth_success', (data) => {
    console.log('[AUTH] Received clerk_auth_success event from Go backend:', data)
    logFrontendEvent('info', 'Auth', 'clerk_auth_success_received', data)
    if (data && data.success) {
      user.value = {
        id: data.userId || 'user_' + Date.now(),
        email: data.email || 'Pro User',
        fullName: 'Authenticated User',
      }
      isPro.value = !!data.isPro
      lastVerifiedAt.value = Date.now()
      authError.value = ''
      saveLocalSession(user.value, isPro.value, lastVerifiedAt.value)
      console.log('[AUTH] Updated localStorage and memory with Pro status:', isPro.value)
      syncWithBackend()
    }
  })
}

// Restore saved session if present with 10-day validity check
try {
  const saved = localStorage.getItem('studyloop_user_session')
  console.log('[AUTH] Checking localStorage for saved user session...')
  if (saved) {
    const parsed = JSON.parse(saved)
    if (parsed && parsed.user) {
      user.value = parsed.user
      const savedTime = parsed.lastVerifiedAt || 0
      const isWithinGracePeriod = (Date.now() - savedTime) < TEN_DAYS_MS

      if (parsed.isPro && !isWithinGracePeriod) {
        console.warn('[AUTH] 10-day offline Pro grace period expired. Re-verification required.')
        isPro.value = false
      } else {
        isPro.value = !!parsed.isPro
      }
      lastVerifiedAt.value = savedTime || Date.now()
      console.log('[AUTH] Restored session from localStorage: email =', parsed.user?.email, 'isPro =', isPro.value, 'withinGrace =', isWithinGracePeriod)
    }
  } else {
    console.log('[AUTH] No saved session found in localStorage')
  }
} catch (err) {
  console.warn('[AUTH] Could not restore saved local user session:', err)
}

export function useClerkAuth() {
  // Ensure backend is in sync whenever composable is accessed
  syncWithBackend()

  return {
    isLoaded: computed(() => isLoaded.value),
    isSignedIn: computed(() => !!user.value),
    user: computed(() => user.value),
    isPro: computed(() => isPro.value),
    lastVerifiedAt: computed(() => lastVerifiedAt.value),
    authError: computed(() => authError.value),
    clearAuthError: () => {
      authError.value = ''
    },
    signIn: async () => {
      console.log('[CLERK_AUTH] signIn() triggered, calling backend startBrowserAuth...')
      authError.value = ''
      try {
        const res = await startBrowserAuth('sign-in')
        console.log('[CLERK_AUTH] startBrowserAuth response:', res)
        if (res?.error) {
          throw new Error(res.error)
        }
        if (!res?.url) {
          throw new Error('Failed to start local auth listener')
        }
        return { success: true }
      } catch (err) {
        const errMsg = err?.message || String(err) || 'Failed to start authentication'
        console.error('[CLERK_AUTH] signIn error:', errMsg)
        authError.value = errMsg
        return { success: false, error: errMsg }
      }
    },
    signOut: () => {
      console.log('[CLERK_AUTH] signOut() triggered')
      user.value = null
      isPro.value = false
      authError.value = ''
      localStorage.removeItem('studyloop_user_session')
      clearSession()
    },
    openBilling: () => {
      // ponytail: direct to pricing section for early access / pro support
      openURLInBrowser('https://studyloop-landing.vercel.app/#pricing')
    },
  }
}
