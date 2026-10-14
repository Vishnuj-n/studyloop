<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-card">
      <div class="modal-header">
        <h2>{{ isEdit ? 'Edit Study Profile' : 'Create Study Profile' }}</h2>
        <button class="close-icon-btn" @click="$emit('close')">&times;</button>
      </div>

      <div class="modal-body">
        <div v-if="!isEdit" class="clone-info-card">
          <BaseIcon name="check" size="14" />
          <span>Auto-inherits reading pacing, quiz rules, focus intervals, and theme from your active profile.</span>
        </div>

        <div class="form-group">
          <label for="profile-name">Profile Name</label>
          <input
            id="profile-name"
            v-model="localName"
            type="text"
            :placeholder="isEdit ? '' : 'e.g. UPSC, Semester Finals'"
            required
          />
        </div>

        <div class="form-group">
          <label for="target-deadline">Target Deadline</label>
          <input id="target-deadline" v-model="localDeadline" type="date" required />
        </div>

        <!-- Flashcard Retention & Inheritance -->
        <div v-if="isEdit && otherProfiles.length > 0" class="inheritance-section">
          <div class="inheritance-header">
            <div class="inheritance-title-row">
              <BaseIcon name="layers" size="14" />
              <span class="inheritance-title">Flashcard Retention &amp; Inheritance</span>
            </div>
            <p class="inheritance-desc">
              Choose which other profiles' due flashcards appear in this profile's daily review queue:
            </p>
          </div>
          <div class="inheritance-list">
            <label
              v-for="prof in otherProfiles"
              :key="prof.id"
              class="inheritance-item"
              :class="{ selected: selectedInherited.includes(prof.id) }"
            >
              <input
                type="checkbox"
                class="inheritance-checkbox"
                :value="prof.id"
                :checked="selectedInherited.includes(prof.id)"
                @change="toggleInherited(prof.id)"
              />
              <span class="profile-name-bold">{{ prof.name }}</span>
              <span class="due-count-pill" :class="{ 'has-due': getDueCount(prof.id) > 0 }">
                {{ getDueCount(prof.id) }} cards due
              </span>
            </label>
          </div>
        </div>
      </div>

      <div class="modal-actions">
        <button class="cancel-btn" @click="$emit('close')">Cancel</button>
        <button
          class="save-btn"
          :disabled="!localName || !localDeadline"
          @click="handleSave"
        >
          {{ isEdit ? 'Save Changes' : 'Create Profile' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { getProfileFlashcardDueBreakdown } from '../services/appApi'

const props = defineProps({
  isEdit: { type: Boolean, default: false },
  initialName: { type: String, default: '' },
  initialDeadline: { type: String, default: '' },
  currentProfileId: { type: String, default: '' },
  allProfiles: { type: Array, default: () => [] },
  initialInherited: { type: Array, default: () => [] },
})

const emit = defineEmits(['close', 'save'])

const localName = ref(props.initialName)
const localDeadline = ref(props.initialDeadline)
const selectedInherited = ref([...(props.initialInherited || [])])
const profileDueMap = ref({})

const otherProfiles = computed(() => {
  if (!props.allProfiles || !props.currentProfileId) return []
  return props.allProfiles.filter((p) => p.id !== props.currentProfileId)
})

function getDueCount(profileId) {
  return profileDueMap.value[profileId] ?? 0
}

function toggleInherited(profileId) {
  const idx = selectedInherited.value.indexOf(profileId)
  if (idx > -1) {
    selectedInherited.value.splice(idx, 1)
  } else {
    selectedInherited.value.push(profileId)
  }
}

function handleSave() {
  emit('save', {
    name: localName.value,
    deadline: localDeadline.value,
    inheritedProfileIds: [...selectedInherited.value],
  })
}

onMounted(async () => {
  try {
    const res = await getProfileFlashcardDueBreakdown()
    if (res && res.profiles) {
      const map = {}
      for (const p of res.profiles) {
        map[p.profile_id] = p.due_cards || 0
      }
      profileDueMap.value = map
    }
  } catch (err) {
    console.warn('Failed to load profile flashcard due breakdown:', err)
  }
})

watch(
  () => props.initialName,
  (v) => {
    localName.value = v
  }
)
watch(
  () => props.initialDeadline,
  (v) => {
    localDeadline.value = v
  }
)
watch(
  () => props.initialInherited,
  (v) => {
    selectedInherited.value = [...(v || [])]
  }
)
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.4);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-card {
  background: var(--surface-container-lowest, #ffffff);
  border: 1px solid var(--outline-variant, rgba(0, 0, 0, 0.08));
  border-radius: 24px;
  padding: 28px;
  width: 100%;
  max-width: 440px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  box-shadow:
    0 4px 6px -1px rgba(0, 0, 0, 0.05),
    0 20px 40px -4px rgba(0, 0, 0, 0.12);
  animation: slideUp 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h2 {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--on-surface);
}

.close-icon-btn {
  background: none;
  border: none;
  font-size: 28px;
  line-height: 1;
  color: var(--muted-text, #64748b);
  cursor: pointer;
  padding: 0 4px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.close-icon-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container-low, #f1f5f9);
}

.modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.clone-info-card {
  display: flex;
  align-items: center;
  gap: 8px;
  background: color-mix(in srgb, var(--primary, #4f46e5) 10%, transparent);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 10px 14px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--on-surface, #0f172a);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

label {
  font-weight: 700;
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted-text, #64748b);
}

input {
  font-family: 'Inter', sans-serif;
  font-size: 15px;
  padding: 12px 16px;
  border-radius: 12px;
  background: var(--surface-container-low, #f8fafc);
  border: 1px solid var(--outline-variant, #e2e8f0);
  color: var(--on-surface, #0f172a);
  outline: none;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

input:hover {
  border-color: color-mix(in srgb, var(--primary, #4f46e5) 40%, var(--outline-variant));
}

input:focus {
  border-color: var(--primary, #4f46e5);
  background: var(--surface-container-lowest, #ffffff);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--primary, #4f46e5) 15%, transparent);
}

input::placeholder {
  color: var(--muted-text, #94a3b8);
  opacity: 0.8;
}

.inheritance-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--surface-container-low, #f8fafc);
  border: 1px solid var(--outline-variant, #e2e8f0);
  border-radius: 16px;
  padding: 16px;
}

.inheritance-header {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.inheritance-title-row {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--primary, #4f46e5);
}

.inheritance-title {
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.info-badge-btn {
  background: var(--surface-container-high, #e2e8f0);
  border: 1px solid var(--outline-variant, rgba(0, 0, 0, 0.08));
  color: var(--on-surface-variant, #64748b);
  cursor: pointer;
  padding: 2px 7px;
  border-radius: 9999px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  transition: all 0.2s ease;
  user-select: none;
}

.info-badge-btn:hover,
.info-badge-btn.active {
  color: var(--primary, #4f46e5);
  background: color-mix(in srgb, var(--primary, #4f46e5) 12%, transparent);
  border-color: color-mix(in srgb, var(--primary, #4f46e5) 30%, transparent);
}

.inheritance-info-box {
  padding: 10px 12px;
  background: var(--surface-container-lowest, #ffffff);
  border: 1px solid var(--outline-variant, #e2e8f0);
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.45;
  color: var(--on-surface);
  animation: fadeIn 0.2s ease;
}

.inheritance-desc {
  margin: 0;
  font-size: 12px;
  color: var(--muted-text, #64748b);
  line-height: 1.4;
}

.inheritance-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 4px;
}

.inheritance-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: var(--surface-container-lowest, #ffffff);
  border: 1px solid var(--outline-variant, #e2e8f0);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
  user-select: none;
}

.inheritance-item:hover {
  border-color: color-mix(in srgb, var(--primary, #4f46e5) 40%, var(--outline-variant));
  background: var(--surface-container-lowest, #ffffff);
}

.inheritance-item.selected {
  border-color: var(--primary, #4f46e5);
  background: color-mix(in srgb, var(--primary, #4f46e5) 4%, var(--surface-container-lowest));
}

.inheritance-checkbox {
  width: 16px;
  height: 16px;
  accent-color: var(--primary, #4f46e5);
  cursor: pointer;
}

.profile-name-bold {
  flex: 1;
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #0f172a);
}

.due-count-pill {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 9999px;
  background: var(--surface-container-high, #e2e8f0);
  color: var(--muted-text, #64748b);
}

.due-count-pill.has-due {
  background: color-mix(in srgb, var(--tertiary, #d97706) 12%, transparent);
  color: var(--tertiary, #b45309);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}

.cancel-btn {
  background: none;
  border: 1px solid var(--outline-variant, #e2e8f0);
  padding: 12px 20px;
  border-radius: 12px;
  font-weight: 700;
  font-size: 14px;
  cursor: pointer;
  color: var(--on-surface, #0f172a);
  transition: all 0.2s ease;
}

.cancel-btn:hover {
  background: var(--surface-container-low, #f1f5f9);
  border-color: var(--outline-variant);
}

.save-btn {
  border: 0;
  border-radius: 12px;
  padding: 12px 24px;
  color: var(--on-primary, #ffffff);
  font-weight: 700;
  font-size: 14px;
  background: linear-gradient(135deg, var(--primary, #4f46e5), var(--primary-dim, #6366f1));
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 4px 12px color-mix(in srgb, var(--primary, #4f46e5) 20%, transparent);
}

.save-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 16px color-mix(in srgb, var(--primary, #4f46e5) 30%, transparent);
}

.save-btn:active:not(:disabled) {
  transform: translateY(0);
}

.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(16px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
</style>
