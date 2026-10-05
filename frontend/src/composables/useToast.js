// ponytail: single reactive toast store for global error, notice popups, and 429 rate-limit remediation
import { ref } from 'vue'

const toast = ref({
  show: false,
  title: '',
  message: '',
  type: 'error', // 'error' | 'notice'
})
let timer = null

const rateLimitBanner = ref({
  show: false,
  title: '',
  message: '',
})

export function isRateLimitError(err) {
  if (!err) return false
  const str = String(err).toLowerCase()
  return (
    str.includes('429') ||
    str.includes('resource_exhausted') ||
    str.includes('rate limit') ||
    str.includes('rate_limit') ||
    str.includes('quota exceeded') ||
    str.includes('too many requests') ||
    str.includes('tpm limit') ||
    str.includes('rpm limit')
  )
}

export function useToast() {
  function showToast(message, title = 'Error', type = 'error', duration = 6000) {
    if (!message) return
    const msgStr = String(message)

    // Detect 429 / quota errors and trigger the actionable rate-limit banner if not permanently dismissed
    if (type === 'error' && isRateLimitError(msgStr)) {
      const isDismissed = localStorage.getItem('studyloop_dismiss_429_banner') === 'true'
      if (!isDismissed) {
        showRateLimitBanner(
          msgStr,
          title === 'Error' ? 'AI Rate Limit Reached (HTTP 429)' : title
        )
      }
    }

    if (timer) clearTimeout(timer)
    toast.value = { show: true, title, message: msgStr, type }
    if (duration > 0) {
      timer = setTimeout(() => {
        toast.value.show = false
        timer = null
      }, duration)
    }
  }

  function hideToast() {
    if (timer) clearTimeout(timer)
    toast.value.show = false
    timer = null
  }

  function showRateLimitBanner(message, title = 'AI Rate Limit Reached (HTTP 429)') {
    rateLimitBanner.value = {
      show: true,
      title: title || 'AI Rate Limit Reached (HTTP 429)',
      message: String(message || 'Your AI provider hit its quota or request rate limit.'),
    }
  }

  function hideRateLimitBanner(doNotShowAgain = false) {
    rateLimitBanner.value.show = false
    if (doNotShowAgain) {
      localStorage.setItem('studyloop_dismiss_429_banner', 'true')
    }
  }

  function resetRateLimitDismissal() {
    localStorage.removeItem('studyloop_dismiss_429_banner')
  }

  return {
    toast,
    rateLimitBanner,
    showToast,
    hideToast,
    showRateLimitBanner,
    hideRateLimitBanner,
    resetRateLimitDismissal,
    showError: (msg, title = 'Error') => showToast(msg, title, 'error'),
    showNotice: (msg, title = 'Notice') => showToast(msg, title, 'notice'),
  }
}
