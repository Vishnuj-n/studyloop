<template>
  <Teleport to="body">
    <Transition name="palette-fade">
      <div v-if="isOpen" class="palette-backdrop" @click.self="close" @mousedown.self="close">
        <div
          class="palette-shell"
          role="dialog"
          aria-modal="true"
          aria-label="Quick Profile Switcher"
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
              placeholder="Switch profile or search…"
              autocomplete="off"
              spellcheck="false"
              aria-label="Search profiles"
              @keydown.down.prevent="moveFocus(1)"
              @keydown.up.prevent="moveFocus(-1)"
              @keydown.enter.prevent="confirmSelection"
              @keydown.esc.prevent="close"
            />
            <kbd class="palette-esc-hint" aria-label="Press Escape to close">esc</kbd>
          </div>

          <!-- Profile list -->
          <ul
            class="palette-results"
            role="listbox"
            aria-label="Profiles"
            :aria-activedescendant="focusedId"
          >
            <li
              v-for="(p, idx) in filteredProfiles"
              :id="`palette-item-${p.id}`"
              :key="p.id"
              class="palette-item"
              :class="{
                'palette-item--active': p.id === activeProfileId,
                'palette-item--focused': idx === focusedIdx,
              }"
              role="option"
              :aria-selected="p.id === activeProfileId"
              @click="selectProfile(p.id)"
              @mouseenter="focusedIdx = idx"
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
                <kbd v-else-if="idx < 9" class="palette-item-digit" aria-label="Press {{ idx + 1 }}">
                  {{ idx + 1 }}
                </kbd>
              </span>
            </li>

            <li v-if="filteredProfiles.length === 0" class="palette-empty" role="alert">
              No profiles match "{{ query }}"
            </li>
          </ul>

          <!-- Footer actions -->
          <div class="palette-footer">
            <button class="palette-footer-btn" type="button" @click="goManage">
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
              Manage Profiles
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
import { ref, computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { getProfiles, setActiveProfileID } from '../services/appApi'
import { useShortcuts } from '../composables/useShortcuts'

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

// ── Derived ───────────────────────────────────────────────────────────────────

const filteredProfiles = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return profiles.value
  return profiles.value.filter((p) => p.name.toLowerCase().includes(q))
})

const focusedId = computed(() => {
  const p = filteredProfiles.value[focusedIdx.value]
  return p ? `palette-item-${p.id}` : undefined
})

// ── Palette shortcut scope (only active when open) ────────────────────────────

useShortcuts({
  scope: 'profile-palette',
  bindings: [
    // Digit shortcuts 1..9 pick the nth visible profile
    ...Array.from({ length: 9 }, (_, i) => ({
      key: String(i + 1),
      allowInInputs: true,
      handler(e) {
        if (!isOpen.value) return
        // Only fire when NO modifier is held (avoids eating Ctrl+1 etc)
        if (e.ctrlKey || e.metaKey || e.altKey || e.shiftKey) return
        const p = filteredProfiles.value[i]
        if (p) selectProfile(p.id)
      },
    })),
    { key: 'Escape', allowInInputs: true, handler: () => isOpen.value && close() },
  ],
})

// ── Watchers ──────────────────────────────────────────────────────────────────

watch(isOpen, async (opened) => {
  if (opened) {
    query.value = ''
    focusedIdx.value = 0
    await loadProfiles()
    await nextTick()
    searchInputRef.value?.focus()
  }
})

watch(filteredProfiles, () => {
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
  const len = filteredProfiles.value.length
  if (len === 0) return
  focusedIdx.value = (focusedIdx.value + delta + len) % len
}

function confirmSelection() {
  const p = filteredProfiles.value[focusedIdx.value]
  if (p) selectProfile(p.id)
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
    emit('profile-switched', profileId)
  } catch (err) {
    console.error('[ProfileCommandPalette] setActiveProfileID failed:', err)
  }
  close()
}

function close() {
  isOpen.value = false
}

function goManage() {
  close()
  router.push('/settings?category=profiles')
}

function goAddProfile() {
  close()
  router.push('/settings?category=profiles&action=add')
}

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Deterministic colour from profile id string */
function profileColor(p) {
  const PALETTE = [
    '#6c8ebf', '#82b366', '#d6a520', '#ae4132',
    '#7952b3', '#17a589', '#ca6f1e', '#2e86c1',
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
  max-height: 320px;
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
  transition: background 0.1s ease, color 0.1s ease;
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
