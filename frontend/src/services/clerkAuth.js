import { ref, computed } from 'vue'

const isLoaded = ref(false)
const user = ref(null)
const isPro = ref(false)
const authError = ref('')
export async function initClerk() {
  isLoaded.value = true
  return null
}

import { startBrowserAuth, openURLInBrowser } from './appApi'

const TEN_DAYS_MS = 10 * 24 * 60 * 60 * 1000
const lastVerifiedAt = ref(Date.now())

// Listen for loopback authentication callback from Go backend
if (typeof window !== 'undefined' && window?.runtime?.EventsOn) {
  window.runtime.EventsOn('clerk_auth_success', (data) => {
    if (data && data.success) {
      user.value = {
        id: data.userId || 'user_' + Date.now(),
        email: data.email || 'Pro User',
        fullName: 'Authenticated User',
      }
      isPro.value = !!data.isPro
      lastVerifiedAt.value = Date.now()
      authError.value = ''
      localStorage.setItem('studyloop_user_session', JSON.stringify({
        user: user.value,
        isPro: isPro.value,
        lastVerifiedAt: lastVerifiedAt.value,
      }))
    }
  })
}

// Restore saved session if present with 10-day validity check
try {
  const saved = localStorage.getItem('studyloop_user_session')
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
    }
  }
} catch (err) {
  console.warn('[AUTH] Could not restore saved local user session:', err)
}

export function useClerkAuth() {
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
    setMockPro: (val) => {
      if (!import.meta.env.DEV) {
        console.warn('[CLERK_AUTH] setMockPro is disabled in production builds.')
        return
      }
      console.log('[CLERK_AUTH] setMockPro called, setting isPro to:', val)
      isPro.value = !!val
      if (user.value) {
        localStorage.setItem(
          'studyloop_user_session',
          JSON.stringify({
            user: user.value,
            isPro: isPro.value,
          })
        )
      }
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
    },
    openBilling: () => {
      // ponytail: direct to pricing section for early access / pro support
      openURLInBrowser('https://studyloop-landing.vercel.app/#pricing')
    },
  }
}
