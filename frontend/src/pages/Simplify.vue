<template>
  <div class="simplify-page">
    <header class="simplify-page-header">
      <div class="header-left">
        <button class="back-nav-btn" @click="handleBack">
          {{ backButtonText }}
        </button>
        <div class="title-meta">
          <div class="badge-row">
            <span class="sparkle-badge">✨ AI Simplified Mode</span>
            <span v-if="bookTitle" class="book-badge">{{ bookTitle }}</span>
          </div>
          <h1 class="topic-heading">{{ displayTitle }}</h1>
        </div>
      </div>

      <div class="header-actions">
        <button
          v-if="simplifiedText"
          class="copy-btn"
          @click="copyMarkdown"
        >
          {{ copied ? 'Copied! ✓' : '📋 Copy Markdown' }}
        </button>
        <button
          v-if="!loading && rawContent"
          class="resimplify-btn"
          :disabled="loading"
          @click="generateSimplification"
        >
          🔄 Refresh
        </button>
      </div>
    </header>

    <!-- Error Banner -->
    <div v-if="errorMessage" class="error-banner">
      <div class="error-text">⚠️ {{ errorMessage }}</div>
      <button class="retry-btn" @click="generateSimplification">Retry</button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <h3>Generating Intuitive Breakdown...</h3>
      <p>Using AI to simplify technical explanations, formulas, and key concepts into clean Markdown.</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="!simplifiedText && !errorMessage" class="empty-state">
      <p>No content available to simplify. Please return to the Reader and select a topic.</p>
      <button class="primary-btn" @click="handleBack">Return to Reading</button>
    </div>

    <!-- Simplified Content View -->
    <main v-else-if="simplifiedText" class="simplified-content-card">
      <MarkdownReader
        :content="simplifiedText"
        :topic-title="displayTitle"
        :is-task-flow="false"
      />
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { simplifyReadingContent, getTopicSectionsContent } from '../services/appApi'
import MarkdownReader from '../components/MarkdownReader.vue'

const route = useRoute()
const router = useRouter()

const topicId = computed(() => (route.query.topicId || route.query.topic_id || '').trim())
const notebookId = computed(() => (route.query.notebookId || route.query.notebook_id || '').trim())

const bookTitle = ref('')
const topicTitle = ref('')
const rawContent = ref('')
const simplifiedText = ref('')
const loading = ref(false)
const errorMessage = ref('')
const copied = ref(false)

const backButtonText = computed(() => {
  return window.history.length > 1 ? '← Back' : '← Back to Dashboard'
})

const displayTitle = computed(() => {
  return topicTitle.value || 'Topic Simplification'
})

function handleBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/dashboard')
  }
}

async function copyMarkdown() {
  if (!simplifiedText.value) return
  try {
    await navigator.clipboard.writeText(simplifiedText.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy markdown:', err)
  }
}

async function generateSimplification() {
  if (!rawContent.value.trim()) {
    errorMessage.value = 'No text content available to simplify.'
    return
  }

  loading.value = true
  errorMessage.value = ''

  try {
    const res = await simplifyReadingContent(rawContent.value)
    if (res?.error) {
      errorMessage.value = res.error
    } else if (res?.simplified) {
      simplifiedText.value = res.simplified
    } else {
      errorMessage.value = 'Failed to generate simplified breakdown.'
    }
  } catch (err) {
    errorMessage.value = String(err?.message || err)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  // 1. Try reading passed session data from sessionStorage
  try {
    const saved = sessionStorage.getItem('simplify_session_data')
    if (saved) {
      const parsed = JSON.parse(saved)
      if (parsed) {
        if (parsed.text) {
          rawContent.value = parsed.text
        }
        if (parsed.bookTitle) bookTitle.value = parsed.bookTitle
        if (parsed.topicTitle) topicTitle.value = parsed.topicTitle
      }
    }
  } catch (e) {
    console.warn('[Simplify] Failed to read simplify_session_data from sessionStorage:', e)
  }

  // 2. If raw content is still empty, load topic content via API if topicId is provided
  if (!rawContent.value.trim() && topicId.value) {
    try {
      const bundle = await getTopicSectionsContent(topicId.value, notebookId.value)
      if (bundle?.content || bundle?.sections_content) {
        rawContent.value = bundle.content || bundle.sections_content
        topicTitle.value = bundle.topic_title || topicTitle.value
        bookTitle.value = bundle.notebook_title || bookTitle.value
      }
    } catch (e) {
      console.warn('Failed to load topic sections content:', e)
    }
  }

  // 3. Generate if content is available
  if (rawContent.value.trim()) {
    await generateSimplification()
  } else {
    errorMessage.value = 'No reading session text found. Please open a reading session first.'
  }
})
</script>

<style scoped>
.simplify-page {
  max-width: 1000px;
  margin: 0 auto;
  padding: 32px 24px 60px 24px;
  color: var(--on-surface);
}

.simplify-page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 28px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--outline-variant);
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.back-nav-btn {
  align-self: flex-start;
  background: var(--surface-container-high);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface-variant);
  font-size: 13px;
  font-weight: 600;
  padding: 6px 14px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.back-nav-btn:hover {
  background: var(--primary);
  color: var(--on-primary);
  border-color: var(--primary);
}

.title-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.badge-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sparkle-badge {
  font-size: 12px;
  font-weight: 700;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  padding: 3px 10px;
  border-radius: 6px;
  letter-spacing: 0.02em;
}

.book-badge {
  font-size: 12px;
  color: var(--muted-text);
  background: var(--surface-container);
  padding: 3px 10px;
  border-radius: 6px;
}

.topic-heading {
  font-size: 24px;
  font-weight: 700;
  margin: 0;
  letter-spacing: -0.02em;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.copy-btn {
  background: var(--surface-container-high);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  font-size: 13.5px;
  font-weight: 600;
  padding: 8px 16px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.copy-btn:hover {
  background: var(--surface-container-highest);
}

.resimplify-btn {
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  font-size: 13.5px;
  font-weight: 600;
  padding: 8px 14px;
  border-radius: 10px;
  cursor: pointer;
}

.resimplify-btn:hover:not(:disabled) {
  color: var(--on-surface);
  border-color: var(--on-surface-variant);
}

.error-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid var(--outline-variant);
  color: #ef4444;
  padding: 14px 18px;
  border-radius: 12px;
  margin-bottom: 24px;
}

.error-text {
  font-size: 14px;
  font-weight: 500;
}

.retry-btn {
  background: #ef4444;
  color: #ffffff;
  border: none;
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  text-align: center;
}

.spinner {
  width: 42px;
  height: 42px;
  border: 4px solid var(--outline-variant);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.loading-state h3 {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.loading-state p {
  color: var(--muted-text);
  font-size: 14px;
  max-width: 480px;
  margin: 0;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
}

.primary-btn {
  margin-top: 16px;
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  padding: 10px 20px;
  border-radius: 10px;
  font-weight: 600;
  cursor: pointer;
}

.simplified-content-card {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 32px 36px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.04);
}
</style>
