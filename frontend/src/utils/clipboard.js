/**
 * Resilient clipboard copy that handles iframe de-focus, WebView2 restrictions,
 * and environments where navigator.clipboard.writeText fails.
 *
 * @param {string} text - The text to copy to clipboard.
 * @returns {Promise<boolean>} - Resolves to true if successful, false otherwise.
 */
export async function copyTextToClipboard(text) {
  if (typeof text !== 'string') {
    text = String(text ?? '')
  }

  // 1. Try modern navigator.clipboard if available
  if (typeof navigator !== 'undefined' && navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch (err) {
      console.warn('[clipboard] navigator.clipboard.writeText failed, attempting execCommand fallback:', err)
    }
  }

  // 2. Fallback: Create off-screen textarea, refocus window, and use execCommand('copy')
  if (typeof document !== 'undefined' && typeof document.createElement === 'function') {
    try {
      if (typeof window !== 'undefined' && typeof window.focus === 'function') {
        window.focus()
      }
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.setAttribute('readonly', '')
      textarea.style.position = 'fixed'
      textarea.style.top = '0'
      textarea.style.left = '-9999px'
      textarea.style.opacity = '0'
      textarea.style.pointerEvents = 'none'

      document.body.appendChild(textarea)
      textarea.focus()
      textarea.select()
      textarea.setSelectionRange(0, text.length)

      const successful = document.execCommand('copy')
      document.body.removeChild(textarea)

      if (successful) {
        return true
      }
    } catch (fallbackErr) {
      console.error('[clipboard] execCommand fallback failed:', fallbackErr)
    }
  }

  return false
}
