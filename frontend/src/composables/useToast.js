// ponytail: single reactive toast store for global error and notice popups
import { ref } from 'vue'

const toast = ref({
  show: false,
  title: '',
  message: '',
  type: 'error', // 'error' | 'notice'
})
let timer = null

export function useToast() {
  function showToast(message, title = 'Error', type = 'error', duration = 6000) {
    if (!message) return
    if (timer) clearTimeout(timer)
    toast.value = { show: true, title, message: String(message), type }
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

  return {
    toast,
    showToast,
    hideToast,
    showError: (msg, title = 'Error') => showToast(msg, title, 'error'),
    showNotice: (msg, title = 'Notice') => showToast(msg, title, 'notice'),
  }
}
