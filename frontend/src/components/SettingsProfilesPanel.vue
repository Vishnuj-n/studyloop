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
        <div class="profile-info">
          <div class="profile-title-row">
            <h3>{{ profile.name }}</h3>
            <span
              v-if="profile.classroom_code"
              class="cloud-badge"
              :title="`Synced to classroom ${profile.classroom_code}`"
            >
              <BaseIcon name="cloud" size="12" />
              <span>{{ profile.classroom_code }}</span>
            </span>
          </div>
          <p class="deadline">
            Deadline: <strong>{{ formatUnixDate(profile.deadline_at) }}</strong>
          </p>

          <!-- Retention Inheritance Indicator -->
          <div
            v-if="getInheritedProfileNames(profile).length > 0"
            class="profile-inherited-box"
            title="Due flashcards from these profiles appear in daily reviews"
          >
            <BaseIcon name="layers" size="13" class="inherited-box-icon" />
            <span class="inherited-box-label">Reviews linked:</span>
            <div class="inherited-box-tags">
              <span
                v-for="name in getInheritedProfileNames(profile)"
                :key="name"
                class="inherited-tag-hash"
              >
                #{{ name }}
              </span>
            </div>
          </div>
          <button
            v-else-if="profiles.length > 1"
            type="button"
            class="link-inheritance-btn"
            title="Configure shared flashcard reviews with other profiles"
            @click="$emit('edit', profile)"
          >
            <BaseIcon name="plus" size="11" />
            <span>Link other profiles' flashcards</span>
          </button>
        </div>
        <div class="profile-actions">
          <button
            v-if="activeProfileId !== profile.id"
            class="select-btn"
            @click="$emit('select', profile.id)"
          >
            Select Active
          </button>
          <button
            v-else
            class="deselect-btn"
            title="Click to deselect active profile"
            @click="$emit('select', '')"
          >
            <BaseIcon name="check" size="12" />
            <span>Active (Deselect)</span>
          </button>

          <button class="edit-btn" @click="$emit('edit', profile)">Edit</button>
          <button class="delete-btn" @click="$emit('delete', profile.id)">Delete</button>
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

.startup-picker-preference {
  background: var(--surface-container, #181b22);
  border: 1px solid var(--outline-variant, #2a2f3d);
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
  accent-color: var(--primary, #88c0d0);
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
  color: var(--on-surface, #eceff4);
}

.preference-desc {
  font-size: 0.8rem;
  color: var(--on-surface-variant, #94a3b8);
}

.panel-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-popover-trigger {
  background: var(--surface-container-high, #e2e8f0);
  border: 1px solid var(--outline-variant, rgba(0, 0, 0, 0.08));
  color: var(--on-surface-variant, #64748b);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 9999px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  transition: all 0.2s ease;
  user-select: none;
}

.info-popover-trigger:hover,
.info-popover-trigger.active {
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
}

.panel-info-banner {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  background: var(--surface-container-low, #f8fafc);
  border: 1px solid var(--outline-variant, #e2e8f0);
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
  color: var(--muted-text, #64748b);
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.banner-close-btn:hover {
  color: var(--on-surface);
}

.profile-inherited-box {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  padding: 6px 12px;
  border-radius: 8px;
  background: var(--surface-container-lowest, #ffffff);
  border: 1px solid var(--outline-variant, #e2e8f0);
  font-size: 11px;
  max-width: 100%;
  box-sizing: border-box;
  flex-wrap: wrap;
}

.inherited-box-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.inherited-box-label {
  color: var(--muted-text, #64748b);
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
}

.inherited-box-tags {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  min-width: 0;
  max-width: 100%;
}

.inherited-tag-hash {
  display: inline-flex;
  align-items: center;
  font-family: monospace;
  font-weight: 700;
  font-size: 11px;
  color: var(--primary, #4f46e5);
  background: color-mix(in srgb, var(--primary, #4f46e5) 8%, transparent);
  border: 1px solid var(--outline-variant);
  padding: 2px 7px;
  border-radius: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}

.link-inheritance-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 5px 12px;
  background: var(--surface-container-lowest, #ffffff);
  border: 1px dashed var(--outline-variant, #cbd5e1);
  border-radius: 8px;
  color: var(--muted-text, #64748b);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  max-width: 100%;
  box-sizing: border-box;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.link-inheritance-btn:hover {
  color: var(--primary);
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 8%, transparent);
  box-shadow: 0 2px 6px color-mix(in srgb, var(--primary) 12%, transparent);
}

.panel-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
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
}

.profiles-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.profile-card {
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 18px;
  background: var(--surface-container-low);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
}

.profile-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.profile-card.active {
  border-color: var(--primary);
  box-shadow: 0 0 10px rgba(108, 92, 231, 0.15);
}

.profile-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.cloud-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border: 1px solid var(--outline-variant);
  padding: 2px 8px;
  border-radius: 999px;
}

.profile-info h3 {
  margin: 0;
  font-size: 16px;
}

.profile-info .deadline {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

.profile-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.select-btn {
  background: rgba(108, 92, 231, 0.1);
  color: var(--primary);
  border: none;
  border-radius: 8px;
  padding: 6px 12px;
  font-weight: 700;
  font-size: 12px;
  cursor: pointer;
}

.deselect-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.deselect-btn:hover {
  opacity: 0.85;
}

.edit-btn,
.delete-btn {
  background: none;
  border: none;
  color: var(--muted-text);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
}

.edit-btn:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--on-surface);
}

.delete-btn:hover {
  background: rgba(235, 94, 85, 0.1);
  color: #eb5e55;
}
</style>
