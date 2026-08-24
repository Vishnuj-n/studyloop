import { ref, computed } from 'vue'

const isLoaded = ref(false)
const user = ref(null)
const isPro = ref(false)
export async function initClerk() {
  isLoaded.value = true
  return null
}

import { startBrowserAuth } from './appApi'

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
      localStorage.setItem('studyloop_user_session', JSON.stringify({
        user: user.value,
        isPro: isPro.value,
      }))
    }
  })
}

// Restore saved session if present
try {
  const saved = localStorage.getItem('studyloop_user_session')
  if (saved) {
    const parsed = JSON.parse(saved)
    if (parsed && parsed.user) {
      user.value = parsed.user
      isPro.value = !!parsed.isPro
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
    setMockPro: (val) => {
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
      try {
        const res = await startBrowserAuth('sign-in')
        console.log('[CLERK_AUTH] startBrowserAuth response:', res)
        if (res?.url) {
          if (window?.runtime?.BrowserOpenURL) {
            window.runtime.BrowserOpenURL(res.url)
          } else {
            window.open(res.url, '_blank')
          }
        }
      } catch (err) {
        console.error('[CLERK_AUTH] signIn error:', err)
        const fallback = 'https://innocent-orca-5605.accounts.dev/sign-in'
        if (window?.runtime?.BrowserOpenURL) {
          window.runtime.BrowserOpenURL(fallback)
        } else {
          window.open(fallback, '_blank')
        }
      }
    },
    signOut: () => {
      console.log('[CLERK_AUTH] signOut() triggered')
      user.value = null
      isPro.value = false
      localStorage.removeItem('studyloop_user_session')
    },
    openBilling: async () => {
      console.log('[CLERK_AUTH] openBilling() triggered, calling backend startBrowserAuth...')
      try {
        const res = await startBrowserAuth('billing')
        console.log('[CLERK_AUTH] openBilling response:', res)
        if (res?.url) {
          if (window?.runtime?.BrowserOpenURL) {
            window.runtime.BrowserOpenURL(res.url)
          } else {
            window.open(res.url, '_blank')
          }
        }
      } catch (err) {
        console.error('[CLERK_AUTH] openBilling error:', err)
        const fallback = 'https://innocent-orca-5605.accounts.dev/user'
        if (window?.runtime?.BrowserOpenURL) {
          window.runtime.BrowserOpenURL(fallback)
        } else {
          window.open(fallback, '_blank')
        }
      }
    },
  }
}
