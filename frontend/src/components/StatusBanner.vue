<template>
  <article :class="['banner', `banner--${variant}`, 'card']">
    <div class="banner-content">
      <span class="banner-icon">
        <slot name="icon">
          <BaseIcon v-if="isKnownIcon" :name="icon" size="20" />
          <span v-else>{{ icon }}</span>
        </slot>
      </span>
      <div class="banner-text">
        <p class="banner-title">{{ title }}</p>
        <p v-if="subtitle" class="banner-subtitle">{{ subtitle }}</p>
      </div>
      <div class="banner-actions">
        <button v-if="actionLabel" class="banner-action-btn" @click="$emit('action')">
          {{ actionLabel }}
        </button>
        <button
          v-if="dismissable"
          type="button"
          class="banner-close-btn"
          title="Dismiss"
          aria-label="Dismiss banner"
          @click="$emit('dismiss')"
        >
          ✕
        </button>
      </div>
    </div>
  </article>
</template>

<script setup>
import { computed } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { icons } from '../assets/icons/index.js'

const props = defineProps({
  /** 'info' | 'rescue' | 'success' | 'error' | 'warning' | 'star' */
  variant: { type: String, required: true },
  icon: { type: String, required: true },
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
  actionLabel: { type: String, default: '' },
  dismissable: { type: Boolean, default: false },
})

defineEmits(['action', 'dismiss'])

const isKnownIcon = computed(() => Boolean(icons[props.icon]))
</script>

<style scoped>
.banner {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
}

.banner-icon {
  width: 40px;
  height: 40px;
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  flex-shrink: 0;
}

.banner-text {
  margin-right: auto;
}

.banner-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.banner-action-btn {
  background: var(--primary, #4f46e5);
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  font-weight: 700;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.banner-action-btn:hover {
  opacity: 0.92;
}

.banner-close-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  font-size: 15px;
  font-weight: 700;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.banner-close-btn:hover {
  background: var(--surface-container-high, rgba(255, 255, 255, 0.08));
  color: var(--on-surface);
}

.banner-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 700;
}

.banner-subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

/* --- Variant: star (GitHub Star callout - 100% dynamic theme token system) --- */
.banner--star {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--on-surface) 4%, transparent);
}
.banner--star .banner-icon {
  background: linear-gradient(135deg, var(--primary-dim, var(--primary)), var(--primary));
  color: var(--on-primary, #ffffff);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--primary) 25%, transparent);
}
.banner--star .banner-title {
  color: var(--on-surface);
}
.banner--star .banner-subtitle {
  color: var(--muted-text);
}
.banner--star .banner-action-btn {
  background: linear-gradient(135deg, var(--primary-dim, var(--primary)), var(--primary));
  color: var(--on-primary, #ffffff);
  font-weight: 700;
  box-shadow: 0 2px 8px color-mix(in srgb, var(--primary) 20%, transparent);
}
.banner--star .banner-action-btn:hover {
  opacity: 0.92;
  transform: translateY(-1px);
}

/* --- Variant: info (escape hatch) --- */
.banner--info {
  background: rgba(230, 126, 34, 0.1);
  border: 1px solid var(--outline-variant);
}
.banner--info .banner-icon {
  background: #e67e22;
}
.banner--info .banner-title {
  color: #e67e22;
}

/* --- Variant: rescue (socratic rescue) --- */
.banner--rescue {
  background: rgba(211, 84, 0, 0.1);
  border: 1px solid var(--outline-variant);
}
.banner--rescue .banner-icon {
  background: #d35400;
}
.banner--rescue .banner-title {
  color: #d35400;
}

/* --- Variant: success (flashcard creation) --- */
.banner--success {
  background: rgba(46, 204, 113, 0.1);
  border-color: rgba(46, 204, 113, 0.2);
}
.banner--success .banner-icon {
  background: #2ecc71;
}
.banner--success .banner-title {
  color: #2ecc71;
}

/* --- Variant: error --- */
.banner--error {
  background: rgba(235, 94, 85, 0.1);
  border-color: rgba(235, 94, 85, 0.2);
}
.banner--error .banner-icon {
  background: #eb5e55;
}
.banner--error .banner-title {
  color: #eb5e55;
}
</style>
