<template>
  <div class="notebook-card" :class="variantClass">
    <button
      class="btn-edit-pen"
      title="Edit notebook and chapters"
      @click="$emit('edit-syllabus', notebook.id, notebook.title)"
    >
      ✎
    </button>

    <div class="notebook-header-card">
      <div class="file-icon" :class="{ 'active-icon': variant === 'active' }">
        {{ fileIcon }}
      </div>
      <div class="notebook-info">
        <h3>{{ notebook.title }}</h3>
        <p class="meta">{{ notebook.file_type.toUpperCase() }}</p>
        <p v-if="notebook.page_count > 0" class="meta">{{ notebook.page_count }} pages</p>
        <p class="meta">{{ notebook.chunk_count }} chunks</p>
        <p v-if="variant === 'dormant'" class="meta">Status: {{ formattedStatus }}</p>
      </div>
    </div>

    <div v-if="needsIngestion" class="notebook-topic">
      <span class="badge new-assignment-badge">{{ ingestionBadgeLabel }}</span>
    </div>
    <div
      v-else-if="notebook.topic_id"
      class="notebook-topic"
      style="display: flex; align-items: center; gap: 8px"
    >
      <span class="badge topic-badge">{{ topicTitle }}</span>
      <RouterLink
        v-if="ragEnabled && ragNotebookChapter"
        :to="`/tutor?topic_id=${notebook.topic_id}&notebook_id=${notebook.id}`"
        class="tutor-link-btn"
        title="Ask Tutor (RAG)"
      >
        <span class="tutor-icon">✨</span>
        <span>Ask Tutor</span>
      </RouterLink>
    </div>

    <div v-else-if="variant === 'dormant'" class="notebook-topic">
      <span class="badge muted">No topic linked</span>
    </div>

    <div class="notebook-priority">
      <label class="priority-label">Priority:</label>
      <select
        :value="notebook.priority || 5"
        class="priority-select"
        @change="(e) => $emit('update-priority', notebook.id, Number.parseInt(e.target.value))"
      >
        <option v-for="n in 10" :key="n" :value="n">{{ n }}</option>
      </select>
    </div>

    <div class="notebook-date">Uploaded: {{ formattedDate }}</div>

    <div class="notebook-actions">
      <button
        v-if="isProcessing"
        class="btn-processing"
        disabled
        :title="progressLabel"
      >
        <span class="spinner-small"></span>
        {{ progressLabel }}
      </button>
      <button
        v-else-if="notebook.status === 'draft_ready'"
        class="btn-ingest"
        title="Review structured chapter syllabus"
        @click="$emit('edit-syllabus', notebook.id, notebook.title)"
      >
        ✨ Review Chapters
      </button>
      <button
        v-else-if="needsIngestion"
        class="btn-ingest"
        title="Extract bookmarks and run AI cleanup"
        @click="$emit('edit-syllabus', notebook.id, notebook.title)"
      >
        ✨ Ingest Book
      </button>
      <button
        v-else-if="canUpgradeDeep"
        class="btn-upgrade-deep"
        title="Re-extract with deep structured analysis for rich tables, code blocks, and headings"
        @click="$emit('upgrade-deep', notebook.id)"
      >
        ⚡ Deep Extract
      </button>
      <button
        v-else-if="variant === 'active'"
        class="btn-sleep"
        @click="$emit('change-status', notebook.id, 'dormant')"
      >
        Sleep
      </button>
      <button
        v-else-if="variant === 'dormant'"
        class="btn-activate"
        :disabled="activeLimitReached"
        @click="$emit('change-status', notebook.id, 'active')"
      >
        Activate
      </button>
      <button
        class="btn-delete"
        :disabled="isCloudProfile"
        :title="
          isCloudProfile
            ? 'Classroom assignments cannot be deleted locally while linked to a cloud profile'
            : 'Delete notebook'
        "
        @click="!isCloudProfile && $emit('delete', notebook.id)"
      >
        Delete
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  notebook: { type: Object, required: true },
  availableTopics: { type: Array, default: () => [] },
  ragEnabled: { type: Boolean, default: false },
  ragNotebookChapter: { type: Boolean, default: true },
  isCloudProfile: { type: Boolean, default: false },
  isPro: { type: Boolean, default: false },
  extractionProgress: { type: Object, default: () => null },
  variant: {
    type: String,
    default: 'dormant',
    validator: (v) => ['active', 'dormant'].includes(v),
  },
  activeLimitReached: { type: Boolean, default: false },
})

defineEmits(['edit-syllabus', 'update-priority', 'change-status', 'delete', 'upgrade-deep'])

const FILE_ICONS = { pdf: '📕', txt: '📄', md: '📝' }

const fileIcon = computed(() => FILE_ICONS[props.notebook.file_type] || '📄')

const topicTitle = computed(() => {
  const topic = props.availableTopics.find((t) => t.id === props.notebook.topic_id)
  return topic ? topic.title : 'No topic'
})

const formattedStatus = computed(() => {
  const status = props.notebook.status
  if (!status) return 'uploaded'
  return status.replaceAll('_', ' ')
})

const formattedDate = computed(() => {
  return new Date(props.notebook.uploaded_at).toLocaleDateString()
})

const needsIngestion = computed(() => {
  return (
    (props.notebook.chunk_count === 0 || !props.notebook.chunk_count) &&
    (props.notebook.status === 'uploaded' ||
      props.notebook.status === 'draft_ready' ||
      !props.notebook.status)
  )
})

const isProcessing = computed(() => props.notebook.status === 'processing')

const progressLabel = computed(() => {
  const prog = props.extractionProgress
  if (prog && typeof prog.percent === 'number' && prog.total > 0) {
    return `Extracting ${prog.percent}% (${prog.processed}/${prog.total} pgs)`
  }
  if (prog && typeof prog.percent === 'number' && prog.percent > 0) {
    return `Extracting ${prog.percent}%...`
  }
  return 'Deep Extracting...'
})

const canUpgradeDeep = computed(() => {
  return (
    props.isPro &&
    props.notebook.file_type === 'pdf' &&
    !needsIngestion.value &&
    !isProcessing.value
  )
})

const ingestionBadgeLabel = computed(() => {
  return props.isCloudProfile
    ? '⚡ New Assignment — Ingestion Needed'
    : '⚡ Ingestion Needed'
})

const variantClass = computed(() =>
  props.variant === 'active' ? 'active-notebook-card' : 'dormant-notebook-card'
)
</script>

<style scoped>
.notebook-card {
  background: var(--surface-container);
  border-radius: 12px;
  padding: 16px;
  border: 1px solid var(--outline-variant);
  transition: all 0.2s;
  position: relative;
}

.notebook-card:hover {
  box-shadow: 0 2px 8px rgba(45, 51, 56, 0.06);
}

.notebook-header-card {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.btn-edit-pen {
  position: absolute;
  top: 10px;
  right: 10px;
  border: 0;
  border-radius: 8px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  width: 30px;
  height: 30px;
  font-size: 15px;
  cursor: pointer;
}

.btn-edit-pen:hover {
  background: var(--surface-container-high, #e6e9ef);
}

.file-icon {
  font-size: 28px;
  flex-shrink: 0;
}

.notebook-info h3 {
  margin: 0;
  font-size: 16px;
  color: var(--on-surface);
  word-break: break-word;
}

.meta {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--muted-text);
}

.notebook-topic {
  margin-bottom: 12px;
}

.badge {
  display: inline-block;
  background: var(--surface-container-low);
  color: var(--primary);
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.badge.muted {
  color: var(--muted-text);
}

.new-assignment-badge {
  background: color-mix(in srgb, #3b82f6 18%, var(--surface-container-low));
  color: #60a5fa;
  border: 1px solid color-mix(in srgb, #3b82f6 30%, transparent);
}

.topic-badge {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tutor-link-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  text-decoration: none;
  background: color-mix(in srgb, #6366f1 22%, var(--surface-container-low));
  color: #818cf8;
  border: 1px solid color-mix(in srgb, #6366f1 45%, transparent);
  transition: all 0.2s ease;
  white-space: nowrap;
  flex-shrink: 0;
}

.tutor-link-btn:hover {
  background: #6366f1;
  color: #ffffff;
  border-color: #6366f1;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.35);
}

.tutor-icon {
  font-size: 13px;
  line-height: 1;
}

.btn-ingest {
  background: color-mix(in srgb, #3b82f6 20%, var(--surface-container-low));
  color: #60a5fa;
  border: 1px solid color-mix(in srgb, #3b82f6 40%, transparent);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 600;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-ingest:hover {
  background: #3b82f6;
  color: #ffffff;
  transform: translateY(-1px);
}

.notebook-date {
  font-size: 12px;
  color: var(--muted-text);
  margin-bottom: 12px;
}

.notebook-priority {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.priority-label {
  font-size: 12px;
  color: var(--muted-text);
}

.priority-select {
  padding: 4px 8px;
  border: 1px solid var(--outline-variant);
  border-radius: 4px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-size: 12px;
  cursor: pointer;
}

.priority-select:hover {
  border-color: var(--primary);
}

.notebook-actions {
  display: flex;
  gap: 8px;
}

.btn-delete {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid color-mix(in srgb, #ef4444 30%, transparent);
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  font-weight: 600;
  background: color-mix(in srgb, #ef4444 14%, var(--surface-container-low));
  color: #f87171;
}

.btn-delete:hover {
  background: color-mix(in srgb, #ef4444 28%, var(--surface-container-low));
  color: #ffffff;
}

/* ── Active variant ────────────────────────────────── */
.active-notebook-card {
  border-color: var(--primary);
  box-shadow:
    0 0 0 1px var(--primary),
    0 4px 12px rgba(0, 0, 0, 0.2);
}

.active-icon {
  color: var(--primary);
}

/* ── Action buttons ────────────────────────────────── */
.btn-activate {
  background: color-mix(in srgb, var(--primary) 18%, var(--surface-container-low));
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 35%, transparent);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-activate:hover:not(:disabled) {
  background: var(--primary);
  color: var(--on-primary);
  transform: translateY(-1px);
}

.btn-activate:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sleep {
  background: color-mix(in srgb, #f59e0b 14%, var(--surface-container-low));
  color: #fbbf24;
  border: 1px solid color-mix(in srgb, #f59e0b 30%, transparent);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-sleep:hover {
  background: color-mix(in srgb, #f59e0b 26%, var(--surface-container-low));
  color: #ffffff;
  transform: translateY(-1px);
}

.btn-upgrade-deep {
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.18), rgba(59, 130, 246, 0.18));
  color: #c4b5fd;
  border: 1px solid rgba(139, 92, 246, 0.4);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-upgrade-deep:hover {
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.32), rgba(59, 130, 246, 0.32));
  border-color: rgba(139, 92, 246, 0.65);
  color: #ffffff;
  transform: translateY(-1px);
}

.btn-processing {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 25%, transparent);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 600;
  cursor: wait;
}

.spinner-small {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid color-mix(in srgb, var(--primary) 30%, transparent);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
