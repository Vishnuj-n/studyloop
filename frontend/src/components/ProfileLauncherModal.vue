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
              <BaseIcon name="users" size="14" />
              <span>Workspace Session</span>
            </div>
            <h1 id="launcher-title">Who is studying today?</h1>
            <p class="launcher-subtitle">
              Select your focus target to align your queue, daily pace, and reading agenda.
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
              <div class="card-top-row">
                <span class="key-shortcut-tag" :title="`Press ${idx + 1} to select`">
                  <kbd>{{ idx + 1 }}</kbd>
                </span>
                <span v-if="profile.id === activeProfileId" class="last-active-tag">
                  <span class="active-dot"></span>
                  Last active
                </span>
              </div>

              <div class="profile-avatar">
                <span class="avatar-initial">{{ (profile.name || 'P').charAt(0).toUpperCase() }}</span>
              </div>

              <div class="profile-meta">
                <h2 class="profile-name" :title="profile.name">{{ profile.name }}</h2>

                <div class="profile-indicators">
                  <span class="deadline-pill" :class="{ 'due-soon': isDueSoon(profile) }">
                    <BaseIcon name="calendar" size="12" />
                    <span>{{ profileDeadlineLabel(profile) }}</span>
                  </span>

                  <span v-if="profilePaceBadge(profile)" class="pace-pill" :class="profilePaceBadge(profile).class">
                    {{ profilePaceBadge(profile).text }}
                  </span>
                </div>

                <div class="tasks-summary">
                  <BaseIcon name="check-circle" size="13" />
                  <span>{{ profileTaskSubtitle(profile) }}</span>
                </div>
              </div>

              <button
                type="button"
                class="start-session-btn"
                :class="{ 'primary-btn': profile.id === activeProfileId }"
                @click.stop="selectProfile(profile.id)"
              >
                <span>{{ profile.id === activeProfileId ? 'Resume Session' : 'Start Session' }}</span>
                <BaseIcon name="arrow-right" size="14" />
              </button>
            </div>
          </div>

          <!-- Footer Options -->
          <footer class="launcher-footer">
            <label class="remember-toggle">
              <span class="custom-switch">
                <input
                  v-model="dontAskAgain"
                  type="checkbox"
                  class="checkbox-input"
                  @change="toggleDontAsk"
                />
                <span class="switch-slider"></span>
              </span>
              <span class="checkbox-label">
                Don't ask on startup <span class="label-hint">(always open last active profile)</span>
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
import { ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
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
  background: rgba(8, 10, 14, 0.78);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  padding: 24px;
}

.launcher-card {
  width: 100%;
  max-width: 800px;
  background: var(--surface-container-lowest, #0e1117);
  border: 1px solid var(--outline-variant, #222734);
  border-radius: 20px;
  padding: 36px 34px 28px;
  box-shadow:
    0 24px 60px -12px rgba(0, 0, 0, 0.75),
    0 0 0 1px rgba(255, 255, 255, 0.04),
    0 8px 24px -4px rgba(0, 0, 0, 0.4);
  display: flex;
  flex-direction: column;
  gap: 28px;
  color: var(--on-surface, #f0f3f6);
  outline: none;
  position: relative;
  overflow: hidden;
}

/* Subtle top glow bar */
.launcher-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 20%;
  right: 20%;
  height: 1px;
  background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--primary, #88c0d0) 40%, white), transparent);
  pointer-events: none;
}

.launcher-header {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.launcher-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 9999px;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  background: color-mix(in srgb, var(--primary, #88c0d0) 10%, transparent);
  color: var(--primary, #88c0d0);
  border: 1px solid color-mix(in srgb, var(--primary, #88c0d0) 24%, transparent);
}

.launcher-header h1 {
  font-size: 1.85rem;
  font-weight: 700;
  letter-spacing: -0.03em;
  margin: 0;
  color: var(--on-surface, #f8fafc);
  line-height: 1.2;
}

.launcher-subtitle {
  font-size: 0.92rem;
  color: var(--on-surface-variant, #94a3b8);
  margin: 0;
  max-width: 520px;
  line-height: 1.5;
}

/* Profile cards grid */
.profiles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
  max-height: 52vh;
  overflow-y: auto;
  padding: 4px 2px;
}

.profile-picker-card {
  position: relative;
  background: var(--surface-container, #131722);
  border: 1px solid var(--outline-variant, #202636);
  border-radius: 16px;
  padding: 18px 20px 18px;
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
  background: var(--surface-container-high, #181e2b);
  border-color: color-mix(in srgb, var(--primary, #88c0d0) 50%, var(--outline-variant, #202636));
  transform: translateY(-2px);
  box-shadow:
    0 12px 28px -6px rgba(0, 0, 0, 0.45),
    0 0 0 1px color-mix(in srgb, var(--primary, #88c0d0) 25%, transparent);
}

.profile-picker-card.is-active {
  border-color: color-mix(in srgb, var(--primary, #88c0d0) 75%, transparent);
  background: linear-gradient(180deg, color-mix(in srgb, var(--primary, #88c0d0) 10%, #131722) 0%, #131722 100%);
  box-shadow:
    0 12px 30px -8px rgba(0, 0, 0, 0.5),
    0 0 0 1px color-mix(in srgb, var(--primary, #88c0d0) 40%, transparent);
}

.card-top-row {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.key-shortcut-tag kbd {
  font-family: inherit;
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--on-surface-variant, #7e8b9b);
  border: 1px solid var(--outline-variant, #2d3446);
  border-radius: 5px;
  padding: 2px 7px;
  background: var(--surface-container-lowest, #090b0e);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
}

.last-active-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.68rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 2px 8px;
  border-radius: 9999px;
  background: color-mix(in srgb, var(--primary, #88c0d0) 15%, transparent);
  color: var(--primary, #88c0d0);
  border: 1px solid color-mix(in srgb, var(--primary, #88c0d0) 30%, transparent);
}

.active-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--primary, #88c0d0);
  box-shadow: 0 0 6px var(--primary, #88c0d0);
}

.profile-avatar {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--primary, #88c0d0) 80%, #3b82f6) 0%, #1e293b 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.avatar-initial {
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
}

.profile-meta {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
}

.profile-name {
  font-size: 1.12rem;
  font-weight: 600;
  margin: 0;
  color: var(--on-surface, #f8fafc);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.profile-indicators {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: center;
  align-items: center;
}

.deadline-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.72rem;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--surface-container-high, #1b2130);
  color: var(--on-surface-variant, #94a3b8);
  border: 1px solid var(--outline-variant, #242c3e);
}

.deadline-pill.due-soon {
  background: color-mix(in srgb, #f59e0b 14%, transparent);
  color: #f59e0b;
  border-color: color-mix(in srgb, #f59e0b 28%, transparent);
  font-weight: 600;
}

.pace-pill {
  font-size: 0.72rem;
  padding: 2px 8px;
  border-radius: 6px;
  font-weight: 600;
}

.pace-ontrack {
  background: color-mix(in srgb, #10b981 14%, transparent);
  color: #10b981;
  border: 1px solid color-mix(in srgb, #10b981 25%, transparent);
}

.pace-behind {
  background: color-mix(in srgb, #ef4444 14%, transparent);
  color: #ef4444;
  border: 1px solid color-mix(in srgb, #ef4444 25%, transparent);
}

.pace-ahead {
  background: color-mix(in srgb, var(--primary, #88c0d0) 14%, transparent);
  color: var(--primary, #88c0d0);
  border: 1px solid color-mix(in srgb, var(--primary, #88c0d0) 25%, transparent);
}

.tasks-summary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.78rem;
  color: var(--on-surface-variant, #8b99ac);
  margin-top: 2px;
}

.start-session-btn {
  width: 100%;
  padding: 9px 14px;
  border-radius: 9px;
  border: 1px solid var(--outline-variant, #2d3548);
  background: var(--surface-container-lowest, #0d1016);
  color: var(--on-surface, #e2e8f0);
  font-size: 0.84rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.start-session-btn:hover {
  background: var(--surface-container-high, #1e2433);
  border-color: color-mix(in srgb, var(--primary, #88c0d0) 40%, var(--outline-variant));
  color: #ffffff;
}

.start-session-btn.primary-btn {
  background: var(--primary, #88c0d0);
  color: #0b1118;
  border-color: var(--primary, #88c0d0);
  font-weight: 650;
  box-shadow: 0 4px 12px color-mix(in srgb, var(--primary, #88c0d0) 25%, transparent);
}

.start-session-btn.primary-btn:hover {
  background: color-mix(in srgb, var(--primary, #88c0d0) 90%, white);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--primary, #88c0d0) 40%, transparent);
}

/* Footer layout */
.launcher-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 20px;
  border-top: 1px solid var(--outline-variant, #1c2230);
  flex-wrap: wrap;
  gap: 16px;
}

.remember-toggle {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: 0.84rem;
  color: var(--on-surface-variant, #94a3b8);
  cursor: pointer;
  user-select: none;
}

/* Custom switch toggle */
.custom-switch {
  position: relative;
  display: inline-block;
  width: 32px;
  height: 18px;
}

.custom-switch input {
  opacity: 0;
  width: 0;
  height: 0;
  position: absolute;
}

.switch-slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background-color: var(--surface-container-high, #242c3e);
  border: 1px solid var(--outline-variant, #333d54);
  transition: 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  border-radius: 9999px;
}

.switch-slider::before {
  position: absolute;
  content: "";
  height: 12px;
  width: 12px;
  left: 2px;
  bottom: 2px;
  background-color: #8b99ac;
  transition: 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  border-radius: 50%;
}

.custom-switch input:checked + .switch-slider {
  background-color: var(--primary, #88c0d0);
  border-color: var(--primary, #88c0d0);
}

.custom-switch input:checked + .switch-slider::before {
  transform: translateX(14px);
  background-color: #0b1118;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 4px;
}

.label-hint {
  color: var(--on-surface-variant, #64748b);
  font-size: 0.78rem;
}

.footer-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.manage-profiles-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--on-surface-variant, #94a3b8);
  font-size: 0.82rem;
  font-weight: 500;
  cursor: pointer;
  padding: 6px 10px;
  border-radius: 7px;
  transition: all 0.15s ease;
}

.manage-profiles-link:hover {
  color: var(--on-surface, #e2e8f0);
  background: var(--surface-container, #141824);
  border-color: var(--outline-variant, #242c3e);
}

.dismiss-btn {
  background: transparent;
  border: 1px solid var(--outline-variant, #2d3548);
  color: var(--on-surface-variant, #94a3b8);
  padding: 6px 14px;
  border-radius: 7px;
  font-size: 0.82rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.dismiss-btn:hover {
  background: var(--surface-container-high, #1d2331);
  color: var(--on-surface, #eceff4);
}

/* Modal Transition */
.launcher-fade-enter-active,
.launcher-fade-leave-active {
  transition: opacity 0.18s cubic-bezier(0.16, 1, 0.3, 1), transform 0.18s cubic-bezier(0.16, 1, 0.3, 1);
}

.launcher-fade-enter-from,
.launcher-fade-leave-to {
  opacity: 0;
  transform: scale(0.98);
}
</style>
