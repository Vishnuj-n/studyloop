<template>
  <Teleport to="body">
    <Transition name="palette-fade">
      <div v-if="isOpen" class="palette-backdrop" @click.self="close" @mousedown.self="close">
        <div
          class="palette-shell"
          role="dialog"
          aria-modal="true"
          aria-label="Quick Command and Profile Switcher"
          @click.stop
        >
          <!-- Header / Search -->
          <div class="palette-search-row">
            <svg
              class="palette-search-icon"
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
            <input
              ref="searchInputRef"
              v-model="query"
              type="text"
              class="palette-search-input"
              placeholder="Type a command or profile name…"
              autocomplete="off"
              spellcheck="false"
              aria-label="Search profiles and actions"
              @keydown.down.prevent="moveFocus(1)"
              @keydown.up.prevent="moveFocus(-1)"
              @keydown.enter.prevent="confirmSelection"
              @keydown.esc.prevent="close"
              @keydown="handleInputKeydown"
            />
            <kbd class="palette-esc-hint" aria-label="Press Escape to close">esc</kbd>
          </div>

          <!-- Command Items / Results list -->
          <ul
            class="palette-results"
            role="listbox"
            aria-label="Commands and Profiles"
            :aria-activedescendant="focusedId"
          >
            <!-- Quick Actions Section (Pomodoro & Settings) -->
            <template v-if="filteredActions.length > 0">
              <li class="palette-section-header">Actions</li>
              <li
                v-for="item in filteredActions"
                :id="`palette-item-${item.id}`"
                :key="item.id"
                class="palette-item palette-action-item"
                :class="{
                  'palette-item--focused': allItems[focusedIdx]?.id === item.id,
                }"
                role="option"
                @click="executeItem(item)"
                @mouseenter="setFocusedById(item.id)"
              >
                <span class="palette-action-icon" aria-hidden="true">
                  <!-- Pomodoro Icon -->
                  <svg
                    v-if="item.id === 'action-pomodoro'"
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <circle cx="12" cy="12" r="10" />
                    <polyline points="12 6 12 12 16 14" />
                  </svg>
                  <!-- Settings Icon -->
                  <svg
                    v-else-if="item.id === 'action-settings'"
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <circle cx="12" cy="12" r="3" />
                    <path
                      d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
                    />
                  </svg>
                </span>

                <span class="palette-item-body">
                  <span class="palette-item-name">{{ item.name }}</span>
                  <span class="palette-item-subtitle">{{ item.subtitle }}</span>
                </span>

                <span class="palette-item-end">
                  <span v-if="item.badge" class="badge badge--action">{{ item.badge }}</span>
                  <kbd class="palette-item-digit">↵</kbd>
                </span>
              </li>
            </template>

            <!-- Profiles Section -->
            <template v-if="filteredProfiles.length > 0">
              <li class="palette-section-header">Study Profiles</li>
              <li
                v-for="(p, pIdx) in filteredProfiles"
                :id="`palette-item-${p.id}`"
                :key="p.id"
                class="palette-item"
                :class="{
                  'palette-item--active': p.id === activeProfileId,
                  'palette-item--focused': allItems[focusedIdx]?.id === p.id,
                }"
                role="option"
                :aria-selected="p.id === activeProfileId"
                @click="selectProfile(p.id)"
                @mouseenter="setFocusedById(p.id)"
              >
                <!-- Colour dot / avatar -->
                <span
                  class="palette-item-dot"
                  :style="{ background: profileColor(p) }"
                  aria-hidden="true"
                />

                <!-- Name & subtitle -->
                <span class="palette-item-body">
                  <span class="palette-item-name">{{ p.name }}</span>
                  <span class="palette-item-subtitle">{{ profileSubtitle(p) }}</span>
                </span>

                <!-- Right-hand badges -->
                <span class="palette-item-end">
                  <span v-if="p.id === activeProfileId" class="badge badge--active">Active</span>
                  <kbd v-else-if="pIdx < 9" class="palette-item-digit">
                    {{ pIdx + 1 }}
                  </kbd>
                </span>
              </li>
            </template>

            <li v-if="allItems.length === 0" class="palette-empty" role="alert">
              No results match "{{ query }}"
            </li>
          </ul>

          <!-- Footer actions -->
          <div class="palette-footer">
            <button class="palette-footer-btn" type="button" @click="goSettings">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <circle cx="12" cy="12" r="3" />
                <path
                  d="M19.07 4.93A9.93 9.93 0 0 0 12 2a10 10 0 0 0-10 10 10 10 0 0 0 10 10 9.93 9.93 0 0 0 7.07-2.93"
                />
              </svg>
              Open Settings
            </button>
            <button class="palette-footer-btn" type="button" @click="togglePomodoro">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
              {{ pomoRunning ? 'Pause Pomodoro' : 'Start Pomodoro' }}
            </button>
            <button class="palette-footer-btn" type="button" @click="goAddProfile">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <line x1="12" y1="5" x2="12" y2="19" />
                <line x1="5" y1="12" x2="19" y2="12" />
              </svg>
              Add Profile
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { getProfiles, setActiveProfileID } from '../services/appApi'
import { getPomodoroTimerState } from '../services/pomodoroApi'
import { EventsOn } from '../../wailsjs/runtime/runtime'

// ── Props / Emits ─────────────────────────────────────────────────────────────

const props = defineProps({
  /** Controlled open state from parent (App.vue) */
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'profile-switched'])

// ── Local state ───────────────────────────────────────────────────────────────

const router = useRouter()

const isOpen = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const profiles = ref([])
const activeProfileId = ref('')
const query = ref('')
const focusedIdx = ref(0)
const searchInputRef = ref(null)

// ── Pomodoro state for palette item ──────────────────────────────────────────
const pomoRunning = ref(false)
const pomoRemainSec = ref(25 * 60)
let unsubs = []

function formatTime(sec) {
  const m = Math.floor(sec / 60).toString().padStart(2, '0')
  const s = (sec % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

async function syncPomodoroState() {
  try {
    const s = await getPomodoroTimerState()
    if (s) {
      pomoRunning.value = !!s.running
      if (typeof s.remainingSec === 'number' && s.remainingSec > 0) {
        pomoRemainSec.value = s.remainingSec
      }
    }
  } catch {
    // Non-blocking
  }
}

// ── Built-in Command Actions ──────────────────────────────────────────────────
const commandActions = computed(() => {
  return [
    {
      id: 'action-pomodoro',
      type: 'action',
      name: pomoRunning.value ? 'Pause Pomodoro Focus' : 'Start 25m Pomodoro Focus',
      subtitle: pomoRunning.value ? `Timer active (${formatTime(pomoRemainSec.value)} left)` : 'Toggle focus timer',
      badge: pomoRunning.value ? 'Running' : '25m',
      handler: togglePomodoro,
    },
    {
      id: 'action-settings',
      type: 'action',
      name: 'Open Settings',
      subtitle: 'System, profiles, AI providers & study limits',
      badge: 'Config',
      handler: goSettings,
    },
  ]
})

const filteredActions = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return commandActions.value
  return commandActions.value.filter(
    (a) =>
      a.name.toLowerCase().includes(q) ||
      a.subtitle.toLowerCase().includes(q) ||
      (a.id === 'action-pomodoro' && (q.includes('pomo') || q.includes('focus') || q.includes('timer'))) ||
      (a.id === 'action-settings' && (q.includes('set') || q.includes('conf') || q.includes('pref')))
  )
})

// ── Derived ───────────────────────────────────────────────────────────────────

const filteredProfiles = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return profiles.value
  return profiles.value.filter((p) => p.name.toLowerCase().includes(q))
})

// Unified list of all navigable items (Actions first, then Profiles)
const allItems = computed(() => {
  return [...filteredActions.value, ...filteredProfiles.value]
})

const focusedId = computed(() => {
  const item = allItems.value[focusedIdx.value]
  return item ? `palette-item-${item.id}` : undefined
})

function setFocusedById(id) {
  const idx = allItems.value.findIndex((it) => it.id === id)
  if (idx !== -1) focusedIdx.value = idx
}

// ── Palette shortcut scope (only active when open) ────────────────────────────

function handleInputKeydown(e) {
  if (e.ctrlKey || e.metaKey || e.altKey || e.shiftKey) return
  const isDigit = /^[1-9]$/.test(e.key) || /^Digit[1-9]$/.test(e.code) || /^Numpad[1-9]$/.test(e.code)
  if (isDigit) {
    let digit = parseInt(e.key, 10)
    if (isNaN(digit) && e.code) {
      digit = parseInt(e.code.replace(/^(Digit|Numpad)/, ''), 10)
    }
    const idx = digit - 1
    if (idx >= 0 && idx < filteredProfiles.value.length) {
      e.preventDefault()
      e.stopPropagation()
      selectProfile(filteredProfiles.value[idx].id)
    }
  }
}

// ponytail: modal navigation is cleanly handled by input @keydown without global stack competition

// ── Lifecycle & Listeners ─────────────────────────────────────────────────────

onMounted(() => {
  try {
    const unsubTick = EventsOn('timerTicked', (data) => {
      if (data && typeof data.remainingSec === 'number') {
        pomoRemainSec.value = data.remainingSec
        pomoRunning.value = true
      }
    })
    if (typeof unsubTick === 'function') unsubs.push(unsubTick)

    const unsubDone = EventsOn('timerCompleted', () => {
      pomoRunning.value = false
    })
    if (typeof unsubDone === 'function') unsubs.push(unsubDone)
  } catch {
    // Non-blocking
  }
})

onUnmounted(() => {
  unsubs.forEach((fn) => {
    try {
      fn()
    } catch {}
  })
})

// ── Watchers ──────────────────────────────────────────────────────────────────

watch(
  isOpen,
  async (opened) => {
    if (opened) {
      query.value = ''
      focusedIdx.value = 0
      await Promise.all([loadProfiles(), syncPomodoroState()])
      await nextTick()
      searchInputRef.value?.focus()
    }
  },
  { immediate: true }
)

watch(allItems, () => {
  focusedIdx.value = 0
})

// ── Data loading ──────────────────────────────────────────────────────────────

async function loadProfiles() {
  try {
    const res = await getProfiles()
    if (res?.error) return
    const list = res?.profiles ?? []
    profiles.value = list
    const active = list.find((p) => p.is_active)
    if (active) activeProfileId.value = active.id
  } catch {
    // Silent: palette is optional chrome, not critical
  }
}

// ── Keyboard navigation ───────────────────────────────────────────────────────

function moveFocus(delta) {
  const len = allItems.value.length
  if (len === 0) return
  focusedIdx.value = (focusedIdx.value + delta + len) % len
}

function confirmSelection() {
  const item = allItems.value[focusedIdx.value]
  if (!item) return
  executeItem(item)
}

function executeItem(item) {
  if (item.type === 'action' && typeof item.handler === 'function') {
    item.handler()
  } else if (item.id) {
    selectProfile(item.id)
  }
}

function togglePomodoro() {
  window.dispatchEvent(new CustomEvent('pomodoro-toggle'))
  close()
}

function goSettings() {
  close()
  router.push('/settings')
}

// ── Actions ───────────────────────────────────────────────────────────────────

async function selectProfile(profileId) {
  if (profileId === activeProfileId.value) {
    close()
    return
  }
  try {
    await setActiveProfileID(profileId)
    // Persist selection locally (mirrors ProfileSelect.vue behaviour)
    localStorage.setItem('studyloop_cached_active_profile_id', profileId)
    activeProfileId.value = profileId
    window.dispatchEvent(new CustomEvent('profile-switched', { detail: { profileId } }))
    window.dispatchEvent(new CustomEvent('settings-updated'))
    emit('profile-switched', profileId)
  } catch (err) {
    console.error('[ProfileCommandPalette] setActiveProfileID failed:', err)
  }
  close()
}

function close() {
  isOpen.value = false
}

function goAddProfile() {
  close()
  router.push('/settings?category=profiles&action=add')
}

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Deterministic colour from profile id string */
function profileColor(p) {
  const PALETTE = [
    '#6c8ebf',
    '#82b366',
    '#d6a520',
    '#ae4132',
    '#7952b3',
    '#17a589',
    '#ca6f1e',
    '#2e86c1',
  ]
  let hash = 0
  const str = p.id || p.name || ''
  for (let i = 0; i < str.length; i++) hash = str.charCodeAt(i) + ((hash << 5) - hash)
  return PALETTE[Math.abs(hash) % PALETTE.length]
}

function profileSubtitle(p) {
  const deadlineAt = Number(p?.deadline_at)
  let deadlineLabel = ''
  if (Number.isFinite(deadlineAt) && deadlineAt > 0) {
    const daysLeft = Math.ceil((deadlineAt * 1000 - Date.now()) / 86400000)
    if (daysLeft < 0) deadlineLabel = 'Deadline passed'
    else if (daysLeft === 0) deadlineLabel = 'Due today'
    else deadlineLabel = `${daysLeft}d left`
  }
  const tasks =
    typeof p?.pending_tasks === 'number'
      ? `${p.pending_tasks} task${p.pending_tasks === 1 ? '' : 's'} pending`
      : ''
  return [deadlineLabel, tasks].filter(Boolean).join(' · ') || 'No deadline set'
}
</script>

<style scoped>
/* ── Backdrop & shell ─────────────────────────────────────────── */
.palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 9000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 15vh;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

.palette-shell {
  width: min(560px, 92vw);
  border-radius: 14px;
  overflow: hidden;
  background: var(--surface-elevated, #1e1e2e);
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.1));
  box-shadow:
    0 24px 64px rgba(0, 0, 0, 0.5),
    0 4px 12px rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
}

/* ── Search row ──────────────────────────────────────────────── */
.palette-search-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.08));
}

.palette-search-icon {
  flex-shrink: 0;
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.4));
}

.palette-search-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  font-size: 15px;
  color: var(--on-surface, #e0e0e0);
  caret-color: var(--primary, #7c9ef5);
}

.palette-search-input::placeholder {
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.35));
}

.palette-esc-hint {
  flex-shrink: 0;
  padding: 2px 6px;
  border-radius: 5px;
  font-size: 11px;
  font-family: inherit;
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.35));
  background: var(--surface-variant, rgba(255, 255, 255, 0.06));
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.1));
  user-select: none;
}

/* ── Results list ────────────────────────────────────────────── */
.palette-results {
  list-style: none;
  margin: 0;
  padding: 6px;
  max-height: 340px;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.palette-results::-webkit-scrollbar {
  width: 6px;
}

.palette-results::-webkit-scrollbar-thumb {
  background: var(--outline-variant, rgba(255, 255, 255, 0.1));
  border-radius: 99px;
}

.palette-section-header {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.4));
  padding: 8px 12px 3px;
  user-select: none;
}

.palette-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.1s ease;
  user-select: none;
}

.palette-item:hover,
.palette-item--focused {
  background: var(--surface-variant, rgba(255, 255, 255, 0.07));
}

.palette-item--active {
  background: var(--primary-container, rgba(124, 158, 245, 0.12));
}

.palette-action-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-variant, rgba(255, 255, 255, 0.06));
  color: var(--primary, #7c9ef5);
  flex-shrink: 0;
}

.badge--action {
  font-size: 11px;
  padding: 2px 7px;
  border-radius: 6px;
  background: var(--surface-variant, rgba(255, 255, 255, 0.08));
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.6));
  font-weight: 500;
}

.palette-item-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.palette-item-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.palette-item-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--on-surface, #e0e0e0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.palette-item-subtitle {
  font-size: 11px;
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.45));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.palette-item-end {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.badge--active {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 99px;
  background: var(--primary, #7c9ef5);
  color: #fff;
  font-weight: 600;
}

.palette-item-digit {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 5px;
  font-family: inherit;
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.35));
  background: var(--surface-variant, rgba(255, 255, 255, 0.06));
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.1));
}

.palette-empty {
  padding: 20px 12px;
  text-align: center;
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.4));
  font-size: 13px;
}

/* ── Footer ──────────────────────────────────────────────────── */
.palette-footer {
  display: flex;
  gap: 4px;
  padding: 8px;
  border-top: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.08));
}

.palette-footer-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--on-surface-muted, rgba(255, 255, 255, 0.5));
  font-size: 12px;
  cursor: pointer;
  transition:
    background 0.1s ease,
    color 0.1s ease;
}

.palette-footer-btn:hover {
  background: var(--surface-variant, rgba(255, 255, 255, 0.07));
  color: var(--on-surface, #e0e0e0);
}

/* ── Transition ──────────────────────────────────────────────── */
.palette-fade-enter-active,
.palette-fade-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}

.palette-fade-enter-from,
.palette-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.98);
}
</style>
