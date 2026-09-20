<template>
  <div class="profile-selector-container" @click.stop>
    <span class="profile-context-label">Profile</span>
    <button
      id="active-profile-select"
      type="button"
      class="profile-switcher-trigger"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      @click="isOpen = !isOpen"
    >
      <span class="profile-trigger-status" aria-hidden="true"></span>
      <span class="profile-trigger-name">{{ activeProfileName }}</span>
      <span class="profile-trigger-chevron" aria-hidden="true">
        <BaseIcon name="chevron-down" size="14" />
      </span>
    </button>

    <div v-if="isOpen" class="profile-switcher-menu" role="listbox" aria-label="Switch profile">
      <button
        v-for="p in profiles"
        :key="p.id"
        type="button"
        class="profile-menu-item"
        :class="{ active: p.id === activeProfileId }"
        role="option"
        :aria-selected="p.id === activeProfileId"
        @click="handleSelect(p.id)"
      >
        <span class="profile-menu-dot" aria-hidden="true"></span>
        <span class="profile-menu-copy">
          <span class="profile-menu-title-row">
            <strong class="profile-menu-name">{{ p.name }}</strong>
            <span v-if="profilePaceBadge(p)" class="profile-pace-badge" :class="profilePaceBadge(p).class">
              {{ profilePaceBadge(p).text }}
            </span>
          </span>
          <small class="profile-menu-subtitle">{{ profileTaskSubtitle(p) }}</small>
        </span>
        <span v-if="p.id === activeProfileId" class="profile-menu-current">Active</span>
      </button>

      <div class="profile-menu-divider" aria-hidden="true"></div>
      <button type="button" class="profile-menu-action" @click="handleManage">
        <BaseIcon name="settings" size="14" />
        Manage profiles &amp; notebooks
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  profiles: {
    type: Array,
    default: () => [],
  },
  activeProfileId: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['select-profile', 'manage-profiles'])

const isOpen = ref(false)

const activeProfileName = computed(() => {
  const p = props.profiles.find((pr) => pr.id === props.activeProfileId)
  return p ? p.name : 'Unknown'
})

function closeMenu() {
  isOpen.value = false
}

onMounted(() => {
  window.addEventListener('click', closeMenu)
})

onUnmounted(() => {
  window.removeEventListener('click', closeMenu)
})

function handleSelect(profileId) {
  isOpen.value = false
  emit('select-profile', profileId)
}

function handleManage() {
  isOpen.value = false
  emit('manage-profiles')
}

function profileDeadlineLabel(profile) {
  const deadlineAt = Number(profile?.deadline_at)
  if (!Number.isFinite(deadlineAt) || deadlineAt <= 0) return 'No deadline'

  const daysLeft = Math.ceil((deadlineAt * 1000 - Date.now()) / 86400000)
  if (daysLeft < 0) return 'Deadline passed'
  if (daysLeft === 0) return 'Due today'
  return `${daysLeft}d left`
}

function profilePaceBadge(profile) {
  const pace = profile?.pace
  if (!pace) return null
  if (pace.error) {
    return {
      text: 'Pace unavailable',
      class: 'pace-unavailable',
    }
  }

  const status = pace.feasibility_status
  if (status === 'BEHIND' && pace.days_gap !== undefined) {
    const lateDays = Math.abs(pace.days_gap)
    return {
      text: `${lateDays}d behind`,
      class: 'pace-behind',
    }
  }
  if (status === 'AHEAD' && pace.days_gap !== undefined) {
    return {
      text: `${pace.days_gap}d ahead`,
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
  const deadline = profileDeadlineLabel(profile)
  const count = typeof profile?.pending_tasks === 'number' ? profile.pending_tasks : null
  if (count === null) return deadline
  if (count === 0) return `${deadline} · 0 tasks left`
  return `${deadline} · ${count} task${count === 1 ? '' : 's'} left`
}
</script>

<style scoped>
.profile-selector-container {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.profile-context-label {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted-text, #94a3b8);
}

.profile-switcher-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  color: var(--text-color, #f8fafc);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.profile-switcher-trigger:hover {
  background: var(--surface-hover, rgba(255, 255, 255, 0.06));
  border-color: var(--border-hover, rgba(255, 255, 255, 0.15));
}

.profile-trigger-status {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--brand-primary, #6366f1);
}

.profile-trigger-name {
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-trigger-chevron {
  display: inline-flex;
  align-items: center;
  color: var(--muted-text, #94a3b8);
}

.profile-switcher-menu {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 50;
  min-width: 240px;
  max-width: 320px;
  background: var(--surface-dropdown, #1e1e24);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: 12px;
  padding: 6px;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.profile-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-color, #f8fafc);
  text-align: left;
  cursor: pointer;
  width: 100%;
  transition: background 0.15s ease;
}

.profile-menu-item:hover {
  background: var(--surface-hover, rgba(255, 255, 255, 0.06));
}

.profile-menu-item.active {
  background: rgba(99, 102, 241, 0.12);
}

.profile-menu-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--muted-text, #64748b);
  flex-shrink: 0;
}

.profile-menu-item.active .profile-menu-dot {
  background: var(--brand-primary, #6366f1);
}

.profile-menu-copy {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.profile-menu-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.profile-menu-name {
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-menu-subtitle {
  font-size: 11px;
  color: var(--muted-text, #94a3b8);
}

.profile-menu-current {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--brand-primary, #6366f1);
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(99, 102, 241, 0.15);
}

.profile-pace-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 4px;
}

.profile-pace-badge.pace-ontrack {
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
}

.profile-pace-badge.pace-behind {
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
}

.profile-pace-badge.pace-ahead {
  background: rgba(59, 130, 246, 0.15);
  color: #60a5fa;
}

.profile-pace-badge.pace-unavailable {
  background: rgba(148, 163, 184, 0.15);
  color: #94a3b8;
}

.profile-menu-divider {
  height: 1px;
  background: var(--border-color, rgba(255, 255, 255, 0.08));
  margin: 4px 0;
}

.profile-menu-action {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--muted-text, #94a3b8);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  width: 100%;
  transition: all 0.15s ease;
}

.profile-menu-action:hover {
  background: var(--surface-hover, rgba(255, 255, 255, 0.06));
  color: var(--text-color, #f8fafc);
}
</style>
