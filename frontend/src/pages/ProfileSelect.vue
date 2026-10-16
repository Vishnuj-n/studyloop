<template>
  <div ref="pageRef" class="profile-select-page">
    <main class="profile-select-container">
      <!-- Title & Subtitle in Google Chrome style -->
      <header class="select-header">
        <div class="logo-mark">
          <BaseIcon name="layers" size="28" />
        </div>
        <h1 id="select-title">Who is studying today?</h1>
        <p class="select-subtitle">
          Choose a profile to open your study queue, notebook targets, and personalized pace.
        </p>
      </header>

      <!-- Profile Cards Grid -->
      <div v-if="loading" class="loading-state">
        <div class="chrome-spinner"></div>
        <p>Loading study workspaces...</p>
      </div>

      <div v-else class="profiles-grid" role="listbox" aria-label="Available Study Profiles">
        <button
          v-for="(profile, idx) in profiles"
          :key="profile.id"
          type="button"
          class="profile-picker-card"
          :class="{
            'is-active': profile.id === activeProfileId,
            'is-focused': focusedIndex === idx,
            'is-submitting': isSubmitting && selectedTargetId === profile.id,
          }"
          :aria-label="`Select profile ${profile.name}`"
          :aria-selected="profile.id === activeProfileId"
          tabindex="0"
          @click="selectProfile(profile.id)"
        >
          <!-- Shortcut Key Pill -->
          <span
            class="card-shortcut-badge"
            :title="`Press ${idx + 1} to select ${profile.name}`"
            :aria-label="`Shortcut ${idx + 1}`"
          >
            <kbd>{{ idx + 1 }}</kbd>
          </span>

          <!-- Chrome-style Large Avatar Circle -->
          <div
            class="profile-avatar-circle"
            :style="{ background: getAvatarBg(profile, idx) }"
          >
            <span class="avatar-letter">
              {{ (profile.name || 'P').charAt(0).toUpperCase() }}
            </span>
            <span
              v-if="profile.id === activeProfileId"
              class="active-status-badge"
              title="Active Profile"
            >
              <BaseIcon name="check" size="11" />
            </span>
          </div>

          <!-- Profile Details -->
          <div class="profile-info">
            <h2 class="profile-name" :title="profile.name">
              {{ profile.name }}
            </h2>

            <div class="profile-subtext">
              <span v-if="profile.id === activeProfileId" class="active-label">
                Last active
              </span>
              <span v-else class="task-count-label">
                {{ profileTaskSubtitle(profile) }}
              </span>
            </div>

            <!-- Crisp Metadata Pills (Exam & Pace) -->
            <div class="profile-pills">
              <span class="pill exam-pill" :class="{ 'due-soon': isDueSoon(profile) }">
                <BaseIcon name="calendar" size="11" />
                <span>{{ profileDeadlineLabel(profile) }}</span>
              </span>

              <span
                v-if="profilePaceBadge(profile)"
                class="pill pace-pill"
                :class="profilePaceBadge(profile).class"
              >
                {{ profilePaceBadge(profile).text }}
              </span>
            </div>
          </div>
        </button>

        <!-- Add Profile Card (Chrome style "Add" slot) -->
        <button
          ref="addCardRef"
          type="button"
          class="profile-picker-card add-profile-card"
          :class="{ 'is-focused': focusedIndex === profiles.length }"
          tabindex="0"
          title="Create a new study profile"
          aria-label="Create a new study profile"
          @click="openAddProfile"
        >
          <div class="profile-avatar-circle add-avatar">
            <BaseIcon name="plus" size="26" />
          </div>
          <div class="profile-info">
            <h2 class="profile-name add-label">Add profile</h2>
            <div class="profile-subtext">Set up another workspace</div>
          </div>
        </button>
      </div>
    </main>

    <!-- Bottom Bar pinned cleanly at bottom -->
    <footer class="select-footer">
      <div class="footer-corner-left">
        <label class="remember-toggle">
          <input
            v-model="dontAskAgain"
            type="checkbox"
            class="chrome-checkbox"
            @change="handleToggleDontAsk"
          />
          <span class="checkbox-label">
            Don't ask on startup
            <span class="label-hint">(automatically open last active profile)</span>
          </span>
        </label>
      </div>

      <div class="footer-center-hint" title="You can also use number keys on your keyboard to switch workspace">
        <BaseIcon name="info" size="13" />
        <span>Press number keys to open quickly</span>
      </div>

      <div class="footer-corner-right">
        <button
          type="button"
          class="manage-profiles-link"
          title="Manage and configure all study profiles"
          @click="openSettingsProfiles"
        >
          <BaseIcon name="settings" size="14" />
          <span>Manage Profiles</span>
        </button>

        <button
          v-if="canCancel"
          type="button"
          class="dismiss-btn"
          @click="cancelSelection"
        >
          Cancel
        </button>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, nextTick, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import BaseIcon from '../components/BaseIcon.vue'
import {
  getProfiles,
  getUserSettings,
  setActiveProfileID,
  updateUserSettings,
} from '../services/appApi'

const router = useRouter()
const pageRef = ref(null)
const addCardRef = ref(null)

// Initialize profiles from cached storage so cards render instantly on mount
function loadCachedProfiles() {
  try {
    const raw = localStorage.getItem('studyloop_cached_profiles')
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed) && parsed.length > 0) return parsed
    }
  } catch (err) {
    console.warn('Failed to load cached profiles from localStorage:', err)
  }
  return []
}

const profiles = ref(loadCachedProfiles())
const activeProfileId = ref(localStorage.getItem('studyloop_cached_active_profile_id') || '')
const askProfileOnStartup = ref(localStorage.getItem('studyloop_ask_profile_on_startup') !== 'false')
const dontAskAgain = ref(!askProfileOnStartup.value)

const initialIdx = profiles.value.findIndex((p) => p.id === activeProfileId.value)
const focusedIndex = ref(initialIdx >= 0 ? initialIdx : 0)
const loading = ref(profiles.value.length === 0)
const canCancel = ref(false)
const isSubmitting = ref(false)
const selectedTargetId = ref('')

const AVATAR_PALETTES = [
  'linear-gradient(135deg, #4285F4 0%, #1A73E8 100%)',
  'linear-gradient(135deg, #0F9D58 0%, #0B8043 100%)',
  'linear-gradient(135deg, #8E24AA 0%, #6A1B9A 100%)',
  'linear-gradient(135deg, #FA7B17 0%, #E37400 100%)',
  'linear-gradient(135deg, #E91E63 0%, #C2185B 100%)',
  'linear-gradient(135deg, #0097A7 0%, #00838F 100%)',
  'linear-gradient(135deg, #3F51B5 0%, #283593 100%)',
]

function getAvatarBg(profile, idx) {
  const seed = (profile.id || profile.name || '').split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  return AVATAR_PALETTES[(seed + idx) % AVATAR_PALETTES.length]
}

// FIX #2: Include the "Add profile" card in the keyboard navigation loop.
// getCards() now returns ALL picker cards (profiles + Add), and the wrap math
// uses this full list so arrows can land on the Add card too.
function getCards() {
  const scope = pageRef.value || document
  return Array.from(scope.querySelectorAll('.profile-picker-card'))
}

// Total focusable cards (profiles + 1 for Add). Used by the wrap math.
function totalCardCount() {
  return profiles.value.length + 1
}

// FIX #6: Scroll the focused card into view. preventScroll:true was hiding
// the active card when the list overflowed on small viewports.
function syncCardFocus() {
  const cards = getCards()
  if (cards.length === 0) return
  const idx = Math.max(0, Math.min(focusedIndex.value, cards.length - 1))
  const target = cards[idx]
  if (!target) return
  target.focus({ preventScroll: true })
  target.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}

async function loadData() {
  if (profiles.value.length === 0) {
    loading.value = true
  }
  try {
    const [settingsRes, profilesRes] = await Promise.all([
      getUserSettings(),
      getProfiles(),
    ])

    const profs = Array.isArray(profilesRes?.profiles) ? profilesRes.profiles : []
    profiles.value = profs
    try {
      localStorage.setItem('studyloop_cached_profiles', JSON.stringify(profs))
    } catch (err) {
      console.warn('Failed to cache profiles to localStorage:', err)
    }

    if (settingsRes && !settingsRes.error) {
      if (settingsRes.active_profile_id) {
        activeProfileId.value = settingsRes.active_profile_id
        localStorage.setItem('studyloop_cached_active_profile_id', settingsRes.active_profile_id)
      }
      askProfileOnStartup.value = settingsRes.ask_profile_on_startup !== false
      dontAskAgain.value = !askProfileOnStartup.value
      localStorage.setItem('studyloop_ask_profile_on_startup', String(askProfileOnStartup.value))
    }

    // FIX #3: Always normalize focusedIndex against the fresh profile list.
    // If the active profile was deleted on another device, fall back to 0
    // instead of keeping a stale out-of-range index.
    const activeIdx = profs.findIndex((p) => p.id === activeProfileId.value)
    focusedIndex.value = activeIdx >= 0 ? activeIdx : 0

    if (sessionStorage.getItem('studyloop_profile_picker_shown') && activeProfileId.value) {
      canCancel.value = true
    }
  } catch (err) {
    console.error('Failed to load profile selection data:', err)
  } finally {
    loading.value = false
    await nextTick()
    syncCardFocus()
  }
}

async function selectProfile(profileId) {
  if (isSubmitting.value) return
  isSubmitting.value = true
  selectedTargetId.value = profileId

  try {
    const res = await setActiveProfileID(profileId)
    if (res?.error) {
      console.error('Failed to set active profile:', res.error)
      isSubmitting.value = false
      return
    }

    activeProfileId.value = profileId
    localStorage.setItem('studyloop_cached_active_profile_id', profileId)
    sessionStorage.setItem('studyloop_profile_picker_shown', 'true')
    window.dispatchEvent(new CustomEvent('settings-updated'))
    window.dispatchEvent(new CustomEvent('profile-switched', { detail: { profileId } }))

    await router.replace('/dashboard')
  } catch (err) {
    console.error('Error selecting profile:', err)
  } finally {
    isSubmitting.value = false
  }
}

async function handleToggleDontAsk() {
  const enabled = !dontAskAgain.value
  askProfileOnStartup.value = enabled
  localStorage.setItem('studyloop_ask_profile_on_startup', String(enabled))
  try {
    const current = await getUserSettings()
    if (current && !current.error) {
      await updateUserSettings({ ...current, ask_profile_on_startup: enabled })
    }
  } catch (err) {
    console.error('Failed to persist startup profile preference:', err)
  }
}

function openSettingsProfiles() {
  sessionStorage.setItem('studyloop_profile_picker_shown', 'true')
  router.push('/settings?category=profiles')
}

function openAddProfile() {
  sessionStorage.setItem('studyloop_profile_picker_shown', 'true')
  router.push('/settings?category=profiles&action=add')
}

function cancelSelection() {
  router.replace('/dashboard')
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
    return { text: `${gapFormatted}d behind`, class: 'pace-behind' }
  }
  if (status === 'AHEAD' && pace.days_gap !== undefined) {
    return { text: `${gapFormatted}d ahead`, class: 'pace-ahead' }
  }
  if (status === 'ON_TRACK') {
    return { text: 'On track', class: 'pace-ontrack' }
  }
  return null
}

function profileTaskSubtitle(profile) {
  const count = typeof profile?.pending_tasks === 'number' ? profile.pending_tasks : null
  if (count === null) return 'Ready to study'
  if (count === 0) return 'All caught up today'
  return `${count} pending study task${count === 1 ? '' : 's'}`
}

// ---- Keyboard handler ----
function handleKeydown(e) {
  if (loading.value) return

  // FIX #4: Bail if no cards are rendered yet (avoids firing selectProfile
  // on a card that hasn't mounted). Use the DOM as the source of truth.
  const cards = getCards()
  if (cards.length === 0) return

  // Ignore typing in inputs / textareas / contentEditable
  const tag = e.target?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target?.isContentEditable) return

  // Ignore modified keys (let browser/system shortcuts through)
  if (e.ctrlKey || e.metaKey || e.altKey) return

  // --- Digits 1..9 select a profile ---
  // Note: digits map ONLY to profiles (1..N), never to the "Add profile" card.
  let digit = null
  if (/^[1-9]$/.test(e.key)) {
    digit = parseInt(e.key, 10)
  } else if (/^(?:Digit|Numpad)[1-9]$/.test(e.code)) {
    digit = parseInt(e.code.replace(/^(?:Digit|Numpad)/, ''), 10)
  }

  if (digit !== null) {
    const idx = digit - 1
    if (idx >= 0 && idx < profiles.value.length) {
      e.preventDefault()
      e.stopPropagation()
      focusedIndex.value = idx
      selectProfile(profiles.value[idx].id)
      return
    }
  }

  // FIX #2: Wrap against the full card count (profiles + Add card) so arrows
  // can land on the Add card too. This matches what getCards() returns.
  const total = totalCardCount()

  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
    e.preventDefault()
    e.stopPropagation()
    focusedIndex.value = (focusedIndex.value + 1) % total
    syncCardFocus()
    return
  }
  if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
    e.preventDefault()
    e.stopPropagation()
    focusedIndex.value = (focusedIndex.value - 1 + total) % total
    syncCardFocus()
    return
  }

  // FIX #1 & #5: Space / Enter handled ONLY here. The card no longer has a
  // local @keydown.space.prevent handler, so there's exactly one code path
  // and no double-trigger race with the native button click.
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    e.stopPropagation()

    // If the currently focused DOM element is a card (native focus landed on
    // one), let it be the authority — but we still preventDefault to stop the
    // browser's implicit Space→click on keyup.
    const target = cards[Math.min(focusedIndex.value, cards.length - 1)]
    if (!target) return

    if (target.classList.contains('add-profile-card')) {
      openAddProfile()
    } else {
      const profile = profiles.value[focusedIndex.value]
      if (profile) selectProfile(profile.id)
    }
  }
}

onMounted(() => {
  // Single listener on window, bubble phase.
  window.addEventListener('keydown', handleKeydown)
  // Ensure the window is focused so key events reach us in WebView2.
  window.focus()
  syncCardFocus()
  loadData()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.profile-select-page {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: center;
  background-color: var(--background);
  background-image: radial-gradient(circle at 50% 12%, color-mix(in srgb, var(--primary) 7%, transparent) 0%, transparent 65%);
  padding: 40px 36px 20px;
  box-sizing: border-box;
  overflow-y: auto;
  font-family: inherit;
  color: var(--on-surface);
  outline: none;
}

.profile-select-container {
  width: 100%;
  max-width: 920px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 32px;
  margin: auto 0;
  animation: fadeIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* Header */
.select-header {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.logo-mark {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 4px;
  border: 1px solid color-mix(in srgb, var(--primary) 20%, transparent);
}

.select-header h1 {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  margin: 0;
  color: var(--on-surface);
}

.select-subtitle {
  font-size: 15px;
  color: var(--muted-text);
  margin: 0;
  max-width: 560px;
  line-height: 1.5;
}

/* Loading */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64px;
  gap: 16px;
  color: var(--muted-text);
  font-size: 14px;
}

.chrome-spinner {
  width: 38px;
  height: 38px;
  border: 3px solid color-mix(in srgb, var(--primary) 18%, transparent);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s cubic-bezier(0.4, 0, 0.2, 1) infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* Profiles Grid */
.profiles-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: stretch;
  justify-content: center;
  gap: 24px;
  width: 100%;
}

/* Chrome-Style Individual Profile Card */
.profile-picker-card {
  position: relative;
  width: 200px;
  min-height: 230px;
  padding: 28px 16px 20px;
  background: var(--surface-container-low);
  border: 1px solid color-mix(in srgb, var(--outline-variant) 45%, transparent);
  border-radius: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  font-family: inherit;
  color: inherit;
  appearance: none;
  cursor: pointer;
  outline: none;
  box-sizing: border-box;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  user-select: none;
}

/* FIX #7: Only .is-focused (set by keyboard handler) draws the "keyboard
   focus" ring. :hover gets a distinct, softer treatment so mouse users are
   never confused about which card Enter will activate. The mouseenter listener
   that used to mutate focusedIndex has been removed from the template. */
.profile-picker-card.is-focused {
  background: var(--surface-container);
  border-color: var(--primary);
  outline: 2px solid var(--primary);
  outline-offset: 2px;
  transform: translateY(-4px);
  box-shadow:
    0 12px 30px -8px rgba(0, 0, 0, 0.16),
    0 0 0 1px color-mix(in srgb, var(--primary) 25%, transparent);
}

.profile-picker-card:hover:not(.is-focused) {
  background: var(--surface-container);
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
  transform: translateY(-2px);
  box-shadow: 0 8px 20px -6px rgba(0, 0, 0, 0.14);
}

.profile-picker-card:active,
.profile-picker-card.is-submitting {
  transform: translateY(-1px) scale(0.98);
  filter: brightness(0.95);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.profile-picker-card.is-active:not(.is-focused) {
  background: color-mix(in srgb, var(--primary) 3%, var(--surface-container-low));
  border-color: color-mix(in srgb, var(--outline-variant) 65%, transparent);
  box-shadow: none;
}

.profile-picker-card.is-active.is-focused {
  background: color-mix(in srgb, var(--primary) 6%, var(--surface-container-low));
}

/* Keyboard Shortcut Badge Button */
.card-shortcut-badge {
  position: absolute;
  top: 12px;
  left: 14px;
  background: transparent;
  border: none;
  padding: 0;
  margin: 0;
  cursor: pointer;
  outline: none;
  z-index: 2;
  transition: transform 0.15s ease;
}

.card-shortcut-badge:hover kbd {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  border-color: var(--primary);
  color: var(--primary);
}

.card-shortcut-badge:active { transform: scale(0.9); }

.card-shortcut-badge kbd {
  font-family: inherit;
  font-size: 10.5px;
  font-weight: 700;
  color: var(--muted-text);
  background: color-mix(in srgb, var(--on-surface) 6%, transparent);
  border: 1px solid color-mix(in srgb, var(--outline-variant) 50%, transparent);
  border-radius: 6px;
  padding: 2px 6px;
  display: inline-block;
  transition: all 0.15s ease;
}

/* Avatar Circle */
.profile-avatar-circle {
  position: relative;
  width: 84px;
  height: 84px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  box-shadow: 0 8px 18px -4px rgba(0, 0, 0, 0.18);
  transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);
}

.profile-picker-card:hover .profile-avatar-circle,
.profile-picker-card.is-focused .profile-avatar-circle {
  transform: scale(1.06);
}

.avatar-letter {
  font-size: 36px;
  font-weight: 600;
  color: #ffffff;
  line-height: 1;
}

.active-status-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #10b981;
  color: #ffffff;
  border: 2.5px solid var(--surface-container-low);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.25);
}

.profile-picker-card:hover .active-status-badge {
  border-color: var(--surface-container);
}

/* Profile Info */
.profile-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  gap: 4px;
  flex: 1;
}

.profile-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface);
  margin: 0;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.profile-subtext {
  font-size: 12px;
  line-height: 1.4;
  color: var(--muted-text);
  min-height: 18px;
}

.active-label {
  color: #10b981;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.active-label::before {
  content: '';
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
}

.task-count-label { color: var(--muted-text); }

/* Subtle Chrome Pills */
.profile-pills {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  width: 100%;
}

.pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 9999px;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.exam-pill {
  background: color-mix(in srgb, var(--on-surface) 6%, transparent);
  color: var(--muted-text);
  border: 1px solid color-mix(in srgb, var(--outline-variant) 40%, transparent);
}

.exam-pill.due-soon {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.25);
  font-weight: 600;
}

.pace-pill.pace-ontrack {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.25);
}

.pace-pill.pace-ahead {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
  border: 1px solid rgba(59, 130, 246, 0.25);
}

.pace-pill.pace-behind {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.25);
}

/* Add Profile Card */
.add-profile-card {
  border-style: dashed;
  background: transparent;
  border-color: color-mix(in srgb, var(--outline-variant) 70%, transparent);
}

.add-profile-card:hover,
.add-profile-card.is-focused {
  background: var(--surface-container-low);
  border-style: solid;
  border-color: var(--primary);
}

.add-avatar {
  background: var(--surface-container-high) !important;
  color: var(--muted-text);
  box-shadow: none;
  border: 1px dashed color-mix(in srgb, var(--outline-variant) 70%, transparent);
}

.add-profile-card:hover .add-avatar,
.add-profile-card.is-focused .add-avatar {
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-high)) !important;
  color: var(--primary);
  border-style: solid;
  border-color: var(--primary);
}

.add-label {
  color: var(--on-surface);
  font-weight: 500;
}

/* Chrome Bottom Bar */
.select-footer {
  width: 100%;
  max-width: 1100px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0 0;
  box-sizing: border-box;
  margin-top: auto;
  flex-shrink: 0;
}

.footer-corner-left,
.footer-corner-right {
  display: inline-flex;
  align-items: center;
  white-space: nowrap;
}

.footer-corner-right { gap: 12px; }

.footer-center-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--muted-text);
  opacity: 0.65;
  user-select: none;
  white-space: nowrap;
  transition: opacity 0.2s ease, color 0.2s ease;
}

.footer-center-hint:hover {
  opacity: 1;
  color: var(--on-surface);
}

.remember-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}

.chrome-checkbox {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  accent-color: var(--primary);
  cursor: pointer;
}

.checkbox-label {
  font-size: 13px;
  color: var(--on-surface);
  white-space: nowrap;
}

.label-hint {
  color: var(--muted-text);
  font-size: 12px;
}

.manage-profiles-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--on-surface);
  background: var(--surface-container-low);
  border: 1px solid color-mix(in srgb, var(--outline-variant) 50%, transparent);
  cursor: pointer;
  transition: all 0.16s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.manage-profiles-link:hover {
  background: var(--surface-container);
  border-color: color-mix(in srgb, var(--outline-variant) 80%, transparent);
}

.dismiss-btn {
  padding: 8px 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--muted-text);
  background: var(--surface-container-low);
  border: 1px solid color-mix(in srgb, var(--outline-variant) 40%, transparent);
  cursor: pointer;
  transition: all 0.16s ease;
}

.dismiss-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container);
}

@media (max-width: 640px) {
  .profile-select-page { padding-bottom: 90px; }
  .profile-select-container { gap: 24px; }
  .select-header h1 { font-size: 26px; }
  .profiles-grid { gap: 16px; }
  .profile-picker-card {
    width: 100%;
    min-height: auto;
    flex-direction: row;
    padding: 16px;
    text-align: left;
    gap: 16px;
  }
  .card-shortcut-badge { position: static; }
  .profile-avatar-circle {
    width: 56px;
    height: 56px;
    margin-bottom: 0;
    flex-shrink: 0;
  }
  .avatar-letter { font-size: 24px; }
  .profile-info { align-items: flex-start; }
  .profile-pills {
    flex-direction: row;
    flex-wrap: wrap;
    justify-content: flex-start;
  }
  .select-footer {
    position: static;
    flex-direction: column;
    align-items: flex-start;
    padding: 24px 0 0;
    gap: 16px;
    background: transparent;
    pointer-events: auto;
  }
  .footer-corner-right {
    width: 100%;
    justify-content: space-between;
  }
}
</style>