<template>
  <div class="dashboard-container ds-root" :class="{ light: isLight }">
    <Navbar :is-light="isLight" @toggle-theme="toggleTheme" />

    <main class="main-content">
      <!-- Custom Toast Notification System -->
      <Teleport to="body">
        <transition name="toast-slide">
          <div
            v-if="toast.show"
            class="custom-toast"
            :class="`toast-${toast.type}`"
            role="alert"
          >
            <div class="toast-indicator-icon">
              <!-- Success check icon -->
              <svg v-if="toast.type === 'success'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
              <!-- Error exclamation icon -->
              <svg v-else-if="toast.type === 'error'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="8" x2="12" y2="12"></line>
                <line x1="12" y1="16" x2="12.01" y2="16"></line>
              </svg>
              <!-- Info icon -->
              <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="16" x2="12" y2="12"></line>
                <line x1="12" y1="8" x2="12.01" y2="8"></line>
              </svg>
            </div>
            <div class="toast-body">{{ toast.message }}</div>
            <button
              type="button"
              class="toast-close"
              aria-label="Dismiss notification"
              @click="dismissToast"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
        </transition>
      </Teleport>

      <!-- Error Bar -->
      <div v-if="error" class="error-message" style="margin-bottom: 1rem;">
        <div style="flex: 1;">{{ error }}</div>
      </div>

      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { ref, provide, onMounted, onUnmounted } from 'vue';
import Navbar from './components/Navbar.vue';
import { useDashboard } from './composables/useDashboard';

const isLight = ref(localStorage.getItem('cloud_dashboard_theme') === 'light');

function toggleTheme() {
  isLight.value = !isLight.value;
  localStorage.setItem('cloud_dashboard_theme', isLight.value ? 'light' : 'dark');
}

provide('theme', { isLight, toggleTheme });

const {
  toast,
  toastMessage,
  dismissToast,
  error,
  searchInputRef,
  initSession,
  stopPolling
} = useDashboard();

const handleGlobalKeydown = (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault();
    searchInputRef.value?.focus();
  } else if (e.key === '/' && document.activeElement !== searchInputRef.value && !['INPUT', 'TEXTAREA'].includes(document.activeElement?.tagName)) {
    e.preventDefault();
    searchInputRef.value?.focus();
  }
};

onMounted(() => {
  initSession();
  window.addEventListener('keydown', handleGlobalKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown);
  stopPolling();
});
</script>

<style>
/* Global dashboard styles loaded from style.css */
</style>

