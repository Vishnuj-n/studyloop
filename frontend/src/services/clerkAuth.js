import { ref, computed } from 'vue'

const isLoaded = ref(false)
const user = ref(null)
const isPro = ref(false)
const clerkInstance = ref(null)

const publishableKey =
  import.meta.env.VITE_CLERK_PUBLISHABLE_KEY ||
  'pk_test_aW5ub2NlbnQtb3JjYS01NjA1LmNsZXJrLmFjY291bnRzLmRldiQ'

function getClerkAccountsUrl(pubKey) {
  try {
    const parts = (pubKey || '').split('_')
    if (parts.length >= 3) {
      const decoded = atob(parts[2])
      const domain = decoded.endsWith('$') ? decoded.slice(0, -1) : decoded
      if (domain) {
        return `https://${domain}`
      }
    }
  } catch {
    // fallback
  }
  return 'https://clerk.com'
}

// Initialize Clerk
export async function initClerk() {
  if (isLoaded.value && clerkInstance.value) return clerkInstance.value

  try {
    let ClerkClass = null
    try {
      const mod = await import('@clerk/clerk-js')
      ClerkClass = mod.Clerk || mod.default?.Clerk || mod.default
    } catch {
      // If npm import isn't bundled, load from Clerk CDN
      if (window.Clerk) {
        ClerkClass = window.Clerk
      } else {
        const domain = getClerkAccountsUrl(publishableKey).replace('https://', '')
        await new Promise((resolve, reject) => {
          const script = document.createElement('script')
          script.src = `https://${domain}/npm/@clerk/clerk-js@5/dist/clerk.browser.js`
          script.async = true
          script.crossOrigin = 'anonymous'
          script.onload = () => resolve()
          script.onerror = () => reject(new Error('Failed to load Clerk script from CDN'))
          document.head.appendChild(script)
        })
        ClerkClass = window.Clerk
      }
    }

    if (!ClerkClass) {
      throw new Error('Clerk class could not be resolved')
    }

    const clerk = new ClerkClass(publishableKey)
    await clerk.load()

    clerkInstance.value = clerk
    isLoaded.value = true

    if (clerk.user) {
      user.value = {
        id: clerk.user.id,
        email: clerk.user.primaryEmailAddress?.emailAddress || 'User',
        fullName: clerk.user.fullName,
      }
      await refreshSubscriptionStatus()
    }

    clerk.addListener(async ({ user: updatedUser }) => {
      if (updatedUser) {
        user.value = {
          id: updatedUser.id,
          email: updatedUser.primaryEmailAddress?.emailAddress || 'User',
          fullName: updatedUser.fullName,
        }
        await refreshSubscriptionStatus()
      } else {
        user.value = null
        isPro.value = false
      }
    })

    return clerk
  } catch (err) {
    console.warn('[CLERK] Initialization note (running in local mode):', err)
    isLoaded.value = true
    return null
  }
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
      if (clerkInstance.value) {
        clerkInstance.value.openSignIn()
      } else {
        window.open(getClerkAccountsUrl(publishableKey), '_blank')
      }
    },
    signOut: async () => {
      if (clerkInstance.value) {
        await clerkInstance.value.signOut()
      }
      user.value = null
      isPro.value = false
    },
    openBilling: () => {
      if (clerkInstance.value) {
        clerkInstance.value.openUserProfile()
      } else {
        window.open(getClerkAccountsUrl(publishableKey), '_blank')
      }
    },
  }
}
