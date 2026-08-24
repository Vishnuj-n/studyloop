import { ref, computed } from 'vue'

const isLoaded = ref(false)
const user = ref(null)
const isPro = ref(false)
const clerkInstance = ref(null)

const publishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY || ''

export async function initClerk() {
  if (isLoaded.value && clerkInstance.value) return clerkInstance.value

  try {
    const clerkVue = await import('@clerk/vue').catch(() => ({}))
    if (clerkVue && typeof clerkVue.useClerk === 'function') {
      const clerk = clerkVue.useClerk()
      if (clerk && clerk.value) {
        clerkInstance.value = clerk.value
        isLoaded.value = true

        const syncUser = async () => {
          if (clerk.value?.user) {
            user.value = {
              id: clerk.value.user.id,
              email: clerk.value.user.primaryEmailAddress?.emailAddress || 'User',
              fullName: clerk.value.user.fullName,
            }
            await refreshSubscriptionStatus()
          } else {
            user.value = null
            isPro.value = false
          }
        }

        await syncUser()

        if (typeof clerk.value.addListener === 'function') {
          clerk.value.addListener(syncUser)
        }

        return clerk.value
      }
    }
  } catch (err) {
    console.warn('[CLERK] Initialization note (running in local mode):', err)
  }

  isLoaded.value = true
  return null
}

export async function refreshSubscriptionStatus() {
  if (!clerkInstance.value || !clerkInstance.value.user) {
    isPro.value = false
    return false
  }

  try {
    const clerk = clerkInstance.value
    // 1. Check Clerk Billing API if available
    if (clerk.billing && typeof clerk.billing.getSubscription === 'function') {
      const sub = await clerk.billing.getSubscription()
      if (sub && (sub.status === 'active' || sub.plan?.name?.toLowerCase() === 'pro')) {
        isPro.value = true
        return true
      }
    }

    // 2. Check user publicMetadata / plan claims
    const metadata = clerk.user.publicMetadata || {}
    if (metadata.is_pro || metadata.tier === 'pro' || metadata.plan === 'pro') {
      isPro.value = true
      return true
    }

    isPro.value = false
    return false
  } catch (err) {
    console.warn('[CLERK] Billing subscription check error:', err)
    isPro.value = false
    return false
  }
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
      // For local testing & verification
      isPro.value = !!val
    },
    signIn: async () => {
      try {
        const res = await startBrowserAuth('sign-in')
        if (res?.url) {
          window.open(res.url, '_blank')
          return
        }
      } catch (err) {
        console.warn('[AUTH] Loopback auth bridge not available, falling back:', err)
      }
      const redirect = encodeURIComponent(window.location.origin)
      window.open(`https://innocent-orca-5605.accounts.dev/sign-in?redirect_url=${redirect}`, '_blank')
    },
    signOut: async () => {
      if (clerkInstance.value) {
        try {
          await clerkInstance.value.signOut()
        } catch (err) {
          console.warn('[AUTH] Clerk instance sign-out failed, continuing with local cleanup:', err)
        }
      }
      user.value = null
      isPro.value = false
      localStorage.removeItem('studyloop_user_session')
    },
    openBilling: async () => {
      try {
        const res = await startBrowserAuth('billing')
        if (res?.url) {
          window.open(res.url, '_blank')
          return
        }
      } catch (err) {
        console.warn('[AUTH] Loopback auth bridge not available, falling back:', err)
      }
      const redirect = encodeURIComponent(window.location.origin)
      window.open(`https://innocent-orca-5605.accounts.dev/user?redirect_url=${redirect}`, '_blank')
    },
  }
}
