<template>
  <article class="panel profiles-panel">
    <div class="panel-header">
      <div class="panel-header-left">
        <h2>Profiles &amp; Targets</h2>
        <button
          type="button"
          class="info-popover-trigger"
          :class="{ active: showPanelInfo }"
          :title="showPanelInfo ? 'Hide explanation' : 'How study profiles & shared retention work'"
          @click="showPanelInfo = !showPanelInfo"
        >
          <BaseIcon name="info" size="13" />
          <span>Info</span>
        </button>
      </div>
      <button class="add-profile-btn" @click="$emit('add')">+ Create Profile</button>
    </div>

    <!-- Explanatory Banner -->
    <div v-if="showPanelInfo" class="panel-info-banner">
      <BaseIcon name="info" size="16" class="banner-icon" />
      <div class="banner-text">
        <strong>Cross-Profile Retention:</strong> Each profile strictly isolates textbooks, reading progress, and quizzes. To prevent memory decay across exams, you can configure any profile to inherit due flashcards from other profiles into your daily queue.
      </div>
      <button
        type="button"
        class="banner-close-btn"
        aria-label="Dismiss banner"
        title="Dismiss"
        @click="showPanelInfo = false"
      >
        <BaseIcon name="x" size="13" />
      </button>
    </div>

    <!-- Startup Profile Picker Preference -->
    <div class="startup-picker-preference">
      <label class="preference-switch-row">
        <input
          type="checkbox"
          :checked="askProfileOnStartup"
          class="switch-checkbox"
          @change="$emit('update-startup-preference', $event.target.checked)"
        />
        <div class="preference-text">
          <span class="preference-title">Show "Who is studying?" launcher on startup</span>
          <span class="preference-desc">Prompt to choose a study profile upon launch when multiple profiles exist.</span>
        </div>
      </label>
    </div>

    <div v-if="profiles.length === 0" class="empty-state">
      No profiles defined. Create one to organize textbooks.
    </div>

    <div v-else class="profiles-list">
      <div
        v-for="profile in profiles"
        :key="profile.id"
        class="profile-card"
        :class="{ active: activeProfileId === profile.id }"
      >
        <!-- Card Header: Title & Identity on Left, Actions on Right -->
        <div class="profile-card-header">
          <div class="profile-identity">
            <h3 class="profile-name">{{ profile.name }}</h3>
            <span
              v-if="profile.classroom_code"
              class="cloud-badge"
              :title="`Synced to classroom ${profile.classroom_code}`"
            >
              <BaseIcon name="cloud" size="12" />
              <span>{{ profile.classroom_code }}</span>
            </span>
            <span
              v-if="activeProfileId === profile.id"
              class="active-status-badge"
              title="Currently active study profile"
            >
              <span class="status-pulse-dot"></span>
              <span>Active</span>
            </span>
          </div>

          <div class="profile-card-actions">
            <button
              v-if="activeProfileId !== profile.id"
              type="button"
              class="action-btn select-btn"
              title="Set as active study profile"
              @click="$emit('select', profile.id)"
            >
              Select Active
            </button>
            <button
              v-else
              type="button"
              class="action-btn deselect-btn"
              title="Click to deselect active profile"
              @click="$emit('select', '')"
            >
              <BaseIcon name="check" size="12" />
              <span>Active (Deselect)</span>
            </button>

            <button
              type="button"
              class="action-btn icon-text-btn edit-btn"
              title="Edit profile & retention links"
              @click="$emit('edit', profile)"
            >
              <BaseIcon name="edit" size="12" />
              <span>Edit</span>
            </button>
            <button
              type="button"
              class="action-btn icon-text-btn delete-btn"
              title="Delete profile"
              @click="$emit('delete', profile.id)"
            >
              <BaseIcon name="trash" size="12" />
              <span>Delete</span>
            </button>
          </div>
        </div>

        <!-- Card Body: Metadata & Retention Links -->
        <div class="profile-card-body">
          <div class="profile-meta-row">
            <div class="meta-item deadline-meta">
              <BaseIcon name="calendar" size="13" class="meta-icon" />
              <span class="meta-label">Deadline:</span>
              <strong class="meta-val">{{ formatUnixDate(profile.deadline_at) }}</strong>
            </div>
          </div>

          <!-- Retention Inheritance Row -->
          <div
            v-if="getInheritedProfileNames(profile).length > 0"
            class="profile-retention-row"
            title="Daily queue includes due flashcards from these profiles"
          >
            <div class="retention-label-wrap">
              <BaseIcon name="layers" size="13" class="retention-icon" />
              <span class="retention-label">Reviews linked:</span>
            </div>
            <div class="retention-tags-wrap">
              <span
                v-for="name in getInheritedProfileNames(profile)"
                :key="name"
                class="retention-tag-pill"
                :title="`Includes flashcards from #${name}`"
              >
                #{{ name }}
              </span>
              <button
                v-if="profiles.length > 1"
                type="button"
                class="retention-manage-chip"
                title="Edit linked profiles"
                @click="$emit('edit', profile)"
              >
                <BaseIcon name="plus" size="10" />
                <span>Link more</span>
              </button>
            </div>
          </div>

          <div
            v-else-if="profiles.length > 1"
            class="profile-retention-row empty"
          >
            <button
              type="button"
              class="link-retention-ghost-btn"
              title="Configure shared flashcard reviews with other profiles"
              @click="$emit('edit', profile)"
            >
              <BaseIcon name="layers" size="12" />
              <span>+ Link other profiles' flashcards</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </article>
</template>

<script setup>
import { ref } from 'vue'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  profiles: { type: Array, required: true },
  activeProfileId: { type: String, default: '' },
  formatUnixDate: { type: Function, required: true },
  askProfileOnStartup: { type: Boolean, default: true },
})

defineEmits(['add', 'select', 'edit', 'delete', 'update-startup-preference'])

const showPanelInfo = ref(false)

function getInheritedProfileNames(profile) {
  if (!profile || !profile.inherited_flashcard_profile_ids) return []
  return profile.inherited_flashcard_profile_ids
    .map((id) => props.profiles.find((p) => p.id === id)?.name)
    .filter(Boolean)
}
</script>

<style scoped>
.profiles-panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 28px;
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--on-surface) 3%, transparent);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.panel-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.panel-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--on-surface);
}

.info-popover-trigger {
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface-variant);
  cursor: pointer;
  padding: 4px 10px;
  border-radius: 9999px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 600;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  user-select: none;
}

.info-popover-trigger:hover,
.info-popover-trigger.active {
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-highest));
  border-color: color-mix(in srgb, var(--primary) 35%, transparent);
}

.add-profile-btn {
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  border-radius: 8px;
  padding: 8px 16px;
  font-weight: 700;
  font-size: 13px;
  cursor: pointer;
  transition: opacity 0.18s ease;
}

.add-profile-btn:hover {
  opacity: 0.9;
}

.panel-info-banner {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  margin-bottom: 20px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--on-surface);
}

.banner-icon {
  color: var(--primary);
  flex-shrink: 0;
  margin-top: 2px;
}

.banner-text {
  flex: 1;
}

.banner-close-btn {
  background: none;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s ease;
}

.banner-close-btn:hover {
  color: var(--on-surface);
}

.startup-picker-preference {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 14px 18px;
  margin-bottom: 20px;
}

.preference-switch-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  cursor: pointer;
}

.switch-checkbox {
  margin-top: 3px;
  width: 18px;
  height: 18px;
  accent-color: var(--primary);
  cursor: pointer;
}

.preference-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.preference-title {
  font-size: 0.92rem;
  font-weight: 600;
  color: var(--on-surface);
}

.preference-desc {
  font-size: 0.8rem;
  color: var(--muted-text);
}

.empty-state {
  padding: 32px;
  text-align: center;
  color: var(--muted-text);
  font-size: 13px;
}

.profiles-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.profile-card {
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 16px 20px;
  background: var(--surface-container-low);
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.profile-card:hover {
  border-color: color-mix(in srgb, var(--primary) 30%, var(--outline-variant));
}

.profile-card.active {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 4%, var(--surface-container-low));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--primary) 40%, transparent), 0 4px 16px color-mix(in srgb, var(--primary) 10%, transparent);
}

.profile-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.profile-identity {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex-wrap: wrap;
}

.profile-name {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: -0.01em;
}

.cloud-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent);
  padding: 2px 8px;
  border-radius: 999px;
}

.active-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent);
  padding: 2px 8px;
  border-radius: 999px;
}

.status-pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--primary);
  box-shadow: 0 0 6px var(--primary);
}

.profile-card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.action-btn {
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.18s cubic-bezier(0.16, 1, 0.3, 1);
}

.select-btn {
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 25%, transparent);
  padding: 5px 12px;
}

.select-btn:hover {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
  border-color: var(--primary);
}

.deselect-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: var(--primary);
  color: var(--on-primary);
  border: 1px solid transparent;
  padding: 5px 12px;
}

.deselect-btn:hover {
  opacity: 0.9;
}

.icon-text-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface-variant);
  padding: 5px 10px;
}

.icon-text-btn:hover {
  background: color-mix(in srgb, var(--on-surface) 8%, var(--surface-container));
  color: var(--on-surface);
}

.delete-btn:hover {
  background: color-mix(in srgb, #ef4444 14%, var(--surface-container));
  border-color: color-mix(in srgb, #ef4444 30%, transparent);
  color: #ef4444;
}

.profile-card-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.profile-meta-row {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.deadline-meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--muted-text);
}

.meta-icon {
  color: var(--muted-text);
  opacity: 0.8;
}

.meta-val {
  color: var(--on-surface);
  font-weight: 600;
}

.profile-retention-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding-top: 8px;
  border-top: 1px solid color-mix(in srgb, var(--outline-variant) 50%, transparent);
}

.retention-label-wrap {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 600;
  color: var(--muted-text);
  white-space: nowrap;
}

.retention-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.retention-tags-wrap {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.retention-tag-pill {
  display: inline-flex;
  align-items: center;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-weight: 600;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, var(--surface-container-lowest));
  border: 1px solid color-mix(in srgb, var(--primary) 25%, transparent);
  padding: 2px 8px;
  border-radius: 6px;
  white-space: nowrap;
}

.retention-manage-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  font-weight: 600;
  color: var(--on-surface-variant);
  background: var(--surface-container);
  border: 1px dashed var(--outline-variant);
  padding: 2px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.18s ease;
}

.retention-manage-chip:hover {
  color: var(--primary);
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 8%, var(--surface-container));
}

.link-retention-ghost-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: transparent;
  border: 1px dashed var(--outline-variant);
  border-radius: 6px;
  color: var(--muted-text);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.18s ease;
}

.link-retention-ghost-btn:hover {
  color: var(--primary);
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 8%, transparent);
}
</style>
