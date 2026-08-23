import { ref, computed } from 'vue'

const isLoaded = ref(false)
const user = ref(null)
const isPro = ref(false)
const clerkInstance = ref(null)

const publishableKey =
  import.meta.env.VITE_CLERK_PUBLISHABLE_KEY ||
  'pk_test_placeholder_for_studyloop'

// Initialize Clerk
export async function initClerk() {
  if (isLoaded.value) return clerkInstance.value

  try {
    // Dynamic import to support offline / zero-network bundling
    const { Clerk } = await import('@clerk/clerk-js')
    const clerk = new Clerk(publishableKey)
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
        alert('Clerk publishable key not configured. Set VITE_CLERK_PUBLISHABLE_KEY.')
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
        window.open('https://clerk.com', '_blank')
      }
    },
  }
}
