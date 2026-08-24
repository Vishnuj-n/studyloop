import { createApp } from 'vue'
import { clerkPlugin } from '@clerk/vue'
import App from './App.vue'
import router from './router'
import { useToast } from './composables/useToast'
import './style.css'
import './assets/shared.css'
import 'katex/dist/katex.min.css'

const publishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY

const app = createApp(App)
app.use(router)
if (publishableKey) {
  app.use(clerkPlugin, {
    publishableKey,
  })
}

// ponytail: catch unhandled runtime errors globally into toast
const { showError } = useToast()
app.config.errorHandler = (err) => {
  console.error('[GLOBAL_ERROR]', err)
  showError(err?.message || String(err), 'Runtime Error')
}
window.addEventListener('unhandledrejection', (event) => {
  console.error('[UNHANDLED_REJECTION]', event.reason)
  showError(event.reason?.message || String(event.reason), 'Unhandled Error')
})

app.mount('#app')
