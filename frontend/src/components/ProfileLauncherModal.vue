<template>
  <Teleport to="body">
    <Transition name="launcher-fade">
      <div
        v-if="isOpen"
        class="launcher-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="launcher-title"
        @click.self="handleOverlayClick"
      >
        <div ref="modalCardRef" class="launcher-card" tabindex="-1">
          <header class="launcher-header">
            <div class="launcher-badge">
              <BaseIcon name="users" size="16" />
              <span>Study Session Picker</span>
            </div>
            <h1 id="launcher-title">Who is studying today?</h1>
            <p class="launcher-subtitle">
              Choose your focus target to align your queue, daily pace, and reading agenda.
            </p>
          </header>

          <!-- Profile Cards Grid -->
          <div class="profiles-grid" role="listbox" aria-label="Available Study Profiles">
            <div
              v-for="(profile, idx) in profiles"
              :key="profile.id"
              class="profile-picker-card"
              :class="{
                'is-active': profile.id === activeProfileId,
                'is-focused': focusedIndex === idx
              }"
              role="option"
              :aria-selected="profile.id === activeProfileId"
              tabindex="0"
              @click="selectProfile(profile.id)"
              @keydown.enter.prevent="selectProfile(profile.id)"
              @keydown.space.prevent="selectProfile(profile.id)"
              @mouseenter="focusedIndex = idx"
            >
              <!-- Keyboard Shortcut Key Badge -->
              <span class="key-shortcut-tag" :title="`Press ${idx + 1} to choose`">
                {{ idx + 1 }}
              </span>

              <div class="profile-avatar">
                <span class="avatar-initial">{{ (profile.name || 'P').charAt(0).toUpperCase() }}</span>
              </div>

              <div class="profile-meta">
                <div class="profile-name-row">
                  <h2 class="profile-name">{{ profile.name }}</h2>
                  <span v-if="profile.id === activeProfileId" class="last-active-tag">Last active</span>
                </div>

                <div class="profile-indicators">
                  <span class="deadline-pill" :class="{ 'due-soon': isDueSoon(profile) }">
                    <BaseIcon name="calendar" size="12" />
                    {{ profileDeadlineLabel(profile) }}
                  </span>

                  <span v-if="profilePaceBadge(profile)" class="pace-pill" :class="profilePaceBadge(profile).class">
                    {{ profilePaceBadge(profile).text }}
                  </span>
                </div>

                <p class="tasks-summary">
                  <BaseIcon name="check-circle" size="13" />
                  <span>{{ profileTaskSubtitle(profile) }}</span>
                </p>
              </div>

              <button
                type="button"
                class="start-session-btn"
                :class="{ 'primary-btn': profile.id === activeProfileId }"
                @click.stop="selectProfile(profile.id)"
              >
                <span>Start Session</span>
                <BaseIcon name="arrow-right" size="14" />
              </button>
            </div>
          </div>

          <!-- Footer Options -->
          <footer class="launcher-footer">
            <label class="remember-toggle">
              <input
                v-model="dontAskAgain"
                type="checkbox"
                class="checkbox-input"
                @change="toggleDontAsk"
              />
              <span class="checkbox-label">
                Don't ask on startup <em>(always open last active profile)</em>
              </span>
            </label>

            <div class="footer-actions">
              <button
                type="button"
                class="manage-profiles-link"
                @click="openSettingsProfiles"
              >
                <BaseIcon name="settings" size="14" />
                <span>Manage Profiles</span>
              </button>

              <button
                v-if="canDismiss"
                type="button"
                class="dismiss-btn"
                @click="dismissModal"
              >
                Cancel
              </button>
            </div>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  isOpen: {
    type: Boolean,
    default: false,
  },
  profiles: {
    type: Array,
    default: () => [],
  },
  activeProfileId: {
    type: String,
    default: '',
  },
  askProfileOnStartup: {
    type: Boolean,
    default: true,
  },
  canDismiss: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits([
  'select-profile',
  'manage-profiles',
  'update-startup-preference',
  'dismiss',
])

const modalCardRef = ref(null)
const focusedIndex = ref(0)
const dontAskAgain = ref(!props.askProfileOnStartup)

watch(
  () => props.askProfileOnStartup,
  (val) => {
    dontAskAgain.value = !val
  }
)

watch(
  () => props.isOpen,
  async (open) => {
    if (open) {
      dontAskAgain.value = !props.askProfileOnStartup
      // Set initial focus index to active profile index if available
      const activeIdx = props.profiles.findIndex((p) => p.id === props.activeProfileId)
      focusedIndex.value = activeIdx >= 0 ? activeIdx : 0

      await nextTick()
      if (modalCardRef.value) {
        modalCardRef.value.focus()
      }
    }
  }
)

function handleOverlayClick() {
  if (props.canDismiss) {
    dismissModal()
  }
}

function dismissModal() {
  emit('dismiss')
}

function selectProfile(profileId) {
  emit('select-profile', profileId)
}

function openSettingsProfiles() {
  emit('manage-profiles')
}

function toggleDontAsk() {
  emit('update-startup-preference', !dontAskAgain.value)
}

function profileDeadlineLabel(profile) {
  const deadlineAt = Number(profile?.deadline_at)
  if (!Number.isFinite(deadlineAt) || deadlineAt <= 0) return 'No target exam'

  const daysLeft = Math.ceil((deadlineAt * 1000 - Date.now()) / 86400000)
  if (daysLeft < 0) return 'Exam passed'
  if (daysLeft === 0) return 'Exam is today'
  return `${daysLeft}d left`
}

function isDueSoon(profile) {
  const deadlineAt = Number(profile?.deadline_at)
  if (!Number.isFinite(deadlineAt) || deadlineAt <= 0) return false
  const daysLeft = Math.ceil((deadlineAt * 1000 - Date.now()) / 86400000)
  return daysLeft >= 0 && daysLeft <= 14
}

function profilePaceBadge(profile) {
  const pace = profile?.pace
  if (!pace || pace.error) return null

  const status = pace.feasibility_status
  const gap = pace.days_gap
  const gapFormatted = gap !== undefined ? (gap % 1 === 0 ? Math.abs(gap) : Math.abs(Number(gap.toFixed(1)))) : 0

  if (status === 'BEHIND' && pace.days_gap !== undefined) {
    return {
      text: `${gapFormatted}d behind`,
      class: 'pace-behind',
    }
  }
  if (status === 'AHEAD' && pace.days_gap !== undefined) {
    return {
      text: `${gapFormatted}d ahead`,
      class: 'pace-ahead',
    }
  }
  if (status === 'ON_TRACK') {
    return {
      text: 'On track',
      class: 'pace-ontrack',
    }
  }
  return null
}

function profileTaskSubtitle(profile) {
  const count = typeof profile?.pending_tasks === 'number' ? profile.pending_tasks : null
  if (count === null) return 'Ready to study'
  if (count === 0) return 'All caught up today'
  return `${count} pending study task${count === 1 ? '' : 's'}`
}

function handleKeydown(e) {
  if (!props.isOpen) return

  // Digits 1..9 directly select profile
  if (e.key >= '1' && e.key <= '9') {
    const idx = parseInt(e.key, 10) - 1
    if (idx >= 0 && idx < props.profiles.length) {
      e.preventDefault()
      selectProfile(props.profiles[idx].id)
      return
    }
  }

  // Left / Right / Up / Down arrow navigation
  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
    e.preventDefault()
    if (props.profiles.length > 0) {
      focusedIndex.value = (focusedIndex.value + 1) % props.profiles.length
    }
    return
  }
  if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (props.profiles.length > 0) {
      focusedIndex.value = (focusedIndex.value - 1 + props.profiles.length) % props.profiles.length
    }
    return
  }

  // Enter triggers selected/focused card
  if (e.key === 'Enter') {
    const target = props.profiles[focusedIndex.value]
    if (target) {
      e.preventDefault()
      selectProfile(target.id)
      return
    }
  }

  // Escape dismisses if allowed
  if (e.key === 'Escape' && props.canDismiss) {
    e.preventDefault()
    dismissModal()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.launcher-overlay {
  position: fixed;
  inset: 0;
  z-index: 100000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 12, 16, 0.75);
  backdrop-filter: blur(10px);
  padding: 24px;
}

.launcher-card {
  width: 100%;
  max-width: 820px;
  background: var(--surface-container-lowest, #12141a);
  border: 1px solid var(--outline-variant, #2e3440);
  border-radius: 20px;
  padding: 36px 32px 28px;
  box-shadow:
    0 24px 48px -12px rgba(0, 0, 0, 0.5),
    0 4px 16px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
  gap: 28px;
  color: var(--on-surface, #e5e9f0);
  outline: none;
}

.launcher-header {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.launcher-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  background: color-mix(in srgb, var(--primary, #88c0d0) 12%, transparent);
  color: var(--primary, #88c0d0);
  border: 1px solid color-mix(in srgb, var(--primary, #88c0d0) 25%, transparent);
}

.launcher-header h1 {
  font-size: 1.75rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0;
  color: var(--on-surface, #eceff4);
}

.launcher-subtitle {
  font-size: 0.95rem;
  color: var(--on-surface-variant, #94a3b8);
  margin: 0;
  max-width: 520px;
}

/* Profile cards grid */
.profiles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  max-height: 52vh;
  overflow-y: auto;
  padding: 4px 2px;
}

.profile-picker-card {
  position: relative;
  background: var(--surface-container, #181b22);
  border: 1.5px solid var(--outline-variant, #2a2f3d);
  border-radius: 16px;
  padding: 20px 18px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  outline: none;
}

.profile-picker-card:hover,
.profile-picker-card.is-focused {
  background: var(--surface-container-high, #212631);
  border-color: var(--primary, #88c0d0);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
}

.profile-picker-card.is-active {
  border-color: color-mix(in srgb, var(--primary, #88c0d0) 70%, transparent);
  background: color-mix(in srgb, var(--surface-container-high, #212631) 90%, var(--primary, #88c0d0) 10%);
}

.key-shortcut-tag {
  position: absolute;
  top: 10px;
  left: 12px;
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--on-surface-variant, #64748b);
  border: 1px solid var(--outline-variant, #334155);
  border-radius: 6px;
  padding: 1px 6px;
  background: var(--surface-container-low, #101216);
}

.profile-avatar {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary, #88c0d0) 0%, #5e81ac 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  box-shadow: 0 4px 12px color-mix(in srgb, var(--primary, #88c0d0) 30%, transparent);
}

.avatar-initial {
  font-size: 1.35rem;
  font-weight: 700;
  color: #0b0f14;
}

.profile-meta {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  margin-bottom: 16px;
}

.profile-name-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.profile-name {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0;
  color: var(--on-surface, #eceff4);
  word-break: break-word;
}

.last-active-tag {
  font-size: 0.68rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 1px 6px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--primary, #88c0d0) 18%, transparent);
  color: var(--primary, #88c0d0);
}

.profile-indicators {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: center;
  margin-top: 4px;
}

.deadline-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--surface-container-highest, #2e3440);
  color: var(--on-surface-variant, #d8dee9);
}

.deadline-pill.due-soon {
  background: color-mix(in srgb, #d08770 18%, transparent);
  color: #d08770;
  font-weight: 600;
}

.pace-pill {
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 6px;
  font-weight: 600;
}

.pace-ontrack {
  background: color-mix(in srgb, #a3be8c 18%, transparent);
  color: #a3be8c;
}

.pace-behind {
  background: color-mix(in srgb, #bf616a 18%, transparent);
  color: #bf616a;
}

.pace-ahead {
  background: color-mix(in srgb, #88c0d0 18%, transparent);
  color: #88c0d0;
}

.tasks-summary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  color: var(--on-surface-variant, #94a3b8);
  margin: 4px 0 0;
}

.start-session-btn {
  width: 100%;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant, #3b4252);
  background: var(--surface-container-lowest, #14171f);
  color: var(--on-surface, #eceff4);
  font-size: 0.85rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.start-session-btn:hover {
  background: var(--surface-container-highest, #3b4252);
  border-color: var(--primary, #88c0d0);
}

.start-session-btn.primary-btn {
  background: var(--primary, #88c0d0);
  color: #0d1117;
  border-color: var(--primary, #88c0d0);
}

.start-session-btn.primary-btn:hover {
  background: color-mix(in srgb, var(--primary, #88c0d0) 88%, white);
}

/* Footer layout */
.launcher-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 16px;
  border-top: 1px solid var(--outline-variant, #2a2f3d);
  flex-wrap: wrap;
  gap: 12px;
}

.remember-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--on-surface-variant, #94a3b8);
  cursor: pointer;
  user-select: none;
}

.checkbox-input {
  width: 16px;
  height: 16px;
  accent-color: var(--primary, #88c0d0);
  cursor: pointer;
}

.checkbox-label em {
  font-style: normal;
  color: var(--on-surface-variant, #64748b);
}

.footer-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.manage-profiles-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: none;
  color: var(--primary, #88c0d0);
  font-size: 0.85rem;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
  transition: opacity 0.15s ease;
}

.manage-profiles-link:hover {
  opacity: 0.8;
  text-decoration: underline;
}

.dismiss-btn {
  background: transparent;
  border: 1px solid var(--outline-variant, #3b4252);
  color: var(--on-surface-variant, #94a3b8);
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 0.85rem;
  cursor: pointer;
}

.dismiss-btn:hover {
  background: var(--surface-container-high, #212631);
  color: var(--on-surface, #eceff4);
}

/* Transitions */
.launcher-fade-enter-active,
.launcher-fade-leave-active {
  transition: opacity 0.2s ease;
}

.launcher-fade-enter-from,
.launcher-fade-leave-to {
  opacity: 0;
}
</style>
