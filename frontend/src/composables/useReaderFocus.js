import { ref } from 'vue'

export const isReaderFocusMode = ref(false)

export function toggleReaderFocusMode() {
  isReaderFocusMode.value = !isReaderFocusMode.value
}

export function exitReaderFocusMode() {
  isReaderFocusMode.value = false
}
