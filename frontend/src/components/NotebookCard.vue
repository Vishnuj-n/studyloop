<template>
  <div class="notebook-card" :class="variantClass">
    <!-- Top Bar: Type Indicator & Actions -->
    <div class="card-top-bar">
      <div class="file-type-chip" :class="{ 'active-chip': variant === 'active' }">
        <BaseIcon :name="fileIconName" size="14" />
        <span>{{ notebook.file_type.toUpperCase() }}</span>
      </div>

      <div class="card-top-actions">
        <!-- Certificate Badge / Button when completed -->
        <button
          v-if="completionPercent >= 100"
          class="btn-certificate-badge"
          title="View & Download Completion Certificate"
          @click="$emit('view-certificate', notebook.id)"
        >
          <BaseIcon name="award" size="13" />
          <span>Certificate</span>
        </button>

        <!-- Progress Ring -->
        <div
          class="completion-ring-container"
          :title="`${completionPercent}% syllabus completed`"
        >
          <svg class="progress-ring" width="32" height="32" viewBox="0 0 32 32">
            <circle
              class="progress-ring-bg"
              cx="16"
              cy="16"
              r="12"
              fill="transparent"
              stroke-width="2.5"
            />
            <circle
              class="progress-ring-fill"
              cx="16"
              cy="16"
              r="12"
              fill="transparent"
              stroke-width="2.5"
              transform="rotate(-90 16 16)"
              :stroke-dasharray="circleCircumference"
              :stroke-dashoffset="circleDashOffset"
            />
            <text
              x="16"
              y="16"
              class="progress-ring-text"
              text-anchor="middle"
              dominant-baseline="central"
            >
              {{ completionPercent }}
            </text>
          </svg>
        </div>

        <!-- Edit Pen -->
        <button
          class="btn-edit-pen"
          title="Edit notebook and chapters"
          @click="$emit('edit-syllabus', notebook.id, notebook.title)"
        >
          <BaseIcon name="edit" size="14" />
        </button>
      </div>
    </div>

    <!-- Title and Metadata Section -->
    <div class="notebook-title-section">
      <h3 :title="notebook.title">{{ notebook.title }}</h3>
      <div class="notebook-meta-row">
        <span v-if="notebook.page_count > 0" class="meta-item">{{ notebook.page_count }} pages</span>
        <span v-if="notebook.page_count > 0" class="meta-bullet">•</span>
        <span v-if="notebook.file_type === 'anki' || notebook.flashcard_count > 0" class="meta-item">{{ notebook.flashcard_count || 0 }} cards</span>
        <span v-else class="meta-item">{{ notebook.chunk_count }} chunks</span>
        <template v-if="isDeepExtracted">
          <span class="meta-bullet">•</span>
          <span class="meta-item deep-badge">
            <BaseIcon name="zap" size="12" />
            <span>Deep Extracted</span>
          </span>
        </template>
        <template v-if="variant === 'dormant' && formattedStatus !== 'uploaded'">
          <span class="meta-bullet">•</span>
          <span class="meta-item">Status: {{ formattedStatus }}</span>
        </template>
      </div>
    </div>

    <!-- Topic / Chapter / Ingestion row -->
    <div v-if="needsIngestion" class="notebook-topic">
      <span class="badge new-assignment-badge" style="display: inline-flex; align-items: center; gap: 4px;">
        <BaseIcon name="zap" size="12" />
        <span>{{ ingestionBadgeLabel }}</span>
      </span>
    </div>
    <div
      v-else-if="notebook.topic_id"
      class="notebook-topic"
    >
      <span class="badge topic-badge">{{ topicTitle }}</span>
      <RouterLink
        v-if="ragEnabled && ragNotebookChapter"
        :to="`/tutor?topic_id=${notebook.topic_id}&notebook_id=${notebook.id}`"
        class="tutor-link-btn"
        title="Ask Tutor (RAG)"
      >
        <BaseIcon name="sparkles" size="13" />
        <span>Ask Tutor</span>
      </RouterLink>
    </div>
    <div v-else-if="variant === 'dormant'" class="notebook-topic">
      <span class="badge muted">No topic linked</span>
    </div>

    <!-- Priority and Upload Date Row -->
    <div class="notebook-details-row">
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
    </div>

    <!-- Bottom Actions -->
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
        <BaseIcon name="sparkles" size="13" />
        <span>Review Chapters</span>
      </button>
      <button
        v-else-if="needsIngestion"
        class="btn-ingest"
        :title="isCloudProfile ? 'Extract bookmarks and run AI cleanup for cloud assignment' : 'Extract bookmarks and run AI cleanup'"
        @click="$emit('edit-syllabus', notebook.id, notebook.title)"
      >
        <BaseIcon name="sparkles" size="13" />
        <span>Ingest Book</span>
      </button>
      <template v-else-if="variant === 'active'">
        <button
          v-if="canUpgradeDeep"
          class="btn-upgrade-deep"
          title="Upgrade Deep: Re-extract with AI layout analysis to preserve 2-column text order, scanned pages, and tables"
          @click="$emit('upgrade-deep', notebook.id)"
        >
          <BaseIcon name="zap" size="13" />
          <span>Upgrade Deep</span>
        </button>
        <button
          class="btn-sleep"
          @click="$emit('change-status', notebook.id, 'dormant')"
        >
          Sleep
        </button>
      </template>
      <template v-else-if="variant === 'dormant'">
        <button
          v-if="canUpgradeDeep"
          class="btn-upgrade-deep"
          title="Upgrade Deep: Re-extract with AI layout analysis to preserve 2-column text order, scanned pages, and tables"
          @click="$emit('upgrade-deep', notebook.id)"
        >
          <BaseIcon name="zap" size="13" />
          <span>Upgrade Deep</span>
        </button>
        <button
          class="btn-activate"
          :disabled="activeLimitReached"
          @click="$emit('change-status', notebook.id, 'active')"
        >
          Activate
        </button>
      </template>
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
import BaseIcon from './BaseIcon.vue'

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

defineEmits(['edit-syllabus', 'update-priority', 'change-status', 'delete', 'upgrade-deep', 'view-certificate', 'dev-unlock-certificate'])

const FILE_ICON_NAMES = { pdf: 'book', txt: 'file-text', md: 'file-text', youtube: 'video', anki: 'cards' }

const fileIconName = computed(() => FILE_ICON_NAMES[props.notebook.file_type] || 'file-text')

const topicTitle = computed(() => {
  const topic = props.availableTopics.find((t) => t.id === props.notebook.topic_id)
  if (topic) return topic.title
  if (props.notebook.file_type === 'anki') return props.notebook.title
  return 'No topic'
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
  if (prog && typeof prog.percent === 'number') {
    if (prog.percent === 0) return 'Initializing...'
    return `Extracting ${prog.percent}%...`
  }
  return 'Initializing...'
})

const isDeepExtracted = computed(() => {
  return props.notebook.file_type === 'pdf' && props.notebook.extraction_engine === 'deep_structured'
})

const canUpgradeDeep = computed(() => {
  return (
    props.isPro &&
    props.notebook.file_type === 'pdf' &&
    !isDeepExtracted.value &&
    !needsIngestion.value &&
    !isProcessing.value
  )
})

const ingestionBadgeLabel = computed(() => {
  return props.isCloudProfile
    ? 'New Assignment — Ingestion Needed'
    : 'Ingestion Needed'
})

const variantClass = computed(() =>
  props.variant === 'active' ? 'active-notebook-card' : 'dormant-notebook-card'
)

const completionPercent = computed(() => {
  const pct = props.notebook.completion_percent
  if (typeof pct === 'number' && !isNaN(pct)) {
    return Math.min(100, Math.max(0, Math.round(pct)))
  }
  return 0
})

const circleRadius = 12
const circleCircumference = computed(() => 2 * Math.PI * circleRadius)
const circleDashOffset = computed(() => {
  const c = circleCircumference.value
  return c - (c * completionPercent.value) / 100
})
</script>

<style scoped>
.notebook-card {
  background: var(--surface-container-lowest, #ffffff);
  border-radius: 12px;
  padding: 16px;
  border: 1px solid var(--outline-variant);
  transition: all 0.2s ease;
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
}

.notebook-card:hover {
  border-color: color-mix(in srgb, var(--primary) 35%, var(--outline-variant));
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}

.active-notebook-card {
  border-color: var(--primary);
  box-shadow:
    0 0 0 1px var(--primary),
    0 4px 14px rgba(0, 0, 0, 0.07);
}

/* ── Top Bar ────────────────────────────────────────── */
.card-top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.file-type-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 8px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  color: var(--primary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.03em;
}

.file-type-chip.active-chip {
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  border-color: color-mix(in srgb, var(--primary) 25%, transparent);
}

.card-top-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.btn-certificate-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 9px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
  background: color-mix(in srgb, #d97706 14%, var(--surface-container-low));
  border: 1px solid var(--outline-variant);
  color: #b45309;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  white-space: nowrap;
}

[data-theme^='dark'] .btn-certificate-badge {
  background: color-mix(in srgb, #f59e0b 16%, var(--surface-container-low));
  border-color: var(--outline-variant);
  color: #f59e0b;
}

.btn-certificate-badge:hover {
  background: #d97706;
  color: #ffffff;
  border-color: #d97706;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(217, 119, 6, 0.25);
}

[data-theme^='dark'] .btn-certificate-badge:hover {
  background: #f59e0b;
  color: #1a1a1a;
  border-color: #f59e0b;
}

.completion-ring-container {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  user-select: none;
}

.progress-ring-bg {
  stroke: var(--outline-variant, rgba(0, 0, 0, 0.1));
}

.progress-ring-fill {
  stroke: var(--primary, #6366f1);
  stroke-linecap: round;
  transition: stroke-dashoffset 0.4s ease;
}

.progress-ring-text {
  font-size: 8.5px;
  font-weight: 700;
  fill: var(--on-surface, #ffffff);
}

.btn-edit-pen {
  border: 0;
  border-radius: 8px;
  background: var(--surface-container-low);
  color: var(--on-surface-variant, var(--on-surface));
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-edit-pen:hover {
  background: var(--surface-container-high, #e6e9ef);
  color: var(--primary);
  transform: translateY(-1px);
}

/* ── Title & Meta ───────────────────────────────────── */
.notebook-title-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.notebook-title-section h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--on-surface);
  word-break: break-word;
}

.notebook-meta-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--muted-text);
}

.meta-bullet {
  opacity: 0.5;
  font-size: 10px;
}

.deep-badge {
  color: var(--primary);
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

/* ── Topic ─────────────────────────────────────────── */
.notebook-topic {
  display: flex;
  align-items: center;
  gap: 8px;
}

.badge {
  display: inline-block;
  background: var(--surface-container-low);
  color: var(--on-surface-variant, var(--primary));
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid var(--outline-variant);
}

.badge.muted {
  color: var(--muted-text);
}

.new-assignment-badge {
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  color: var(--primary);
  font-weight: 700;
  border-color: color-mix(in srgb, var(--primary) 25%, transparent);
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
  font-weight: 700;
  text-decoration: none;
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  color: var(--primary);
  border: 1px solid var(--outline-variant);
  transition: all 0.2s ease;
  white-space: nowrap;
  flex-shrink: 0;
}

.tutor-link-btn:hover {
  background: var(--primary);
  color: var(--on-primary, #ffffff);
  border-color: var(--primary);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--primary) 35%, transparent);
}

/* ── Details row (Priority + Date) ─────────────────── */
.notebook-details-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-top: 2px;
}

.notebook-priority {
  display: flex;
  align-items: center;
  gap: 6px;
}

.priority-label {
  font-size: 12px;
  color: var(--muted-text);
  font-weight: 500;
}

.priority-select {
  padding: 3px 8px;
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s ease;
}

.priority-select:hover {
  border-color: var(--primary);
}

.notebook-date {
  font-size: 11.5px;
  color: var(--muted-text);
}

/* ── Action buttons ────────────────────────────────── */
.notebook-actions {
  display: flex;
  gap: 8px;
  margin-top: auto;
  padding-top: 4px;
}

.btn-delete {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  font-weight: 700;
  background: color-mix(in srgb, #dc2626 10%, var(--surface-container-low));
  color: #dc2626;
}

.btn-delete:hover {
  background: #dc2626;
  color: #ffffff;
}

.btn-ingest {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: color-mix(in srgb, var(--primary) 14%, var(--surface-container-low));
  color: var(--primary);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 700;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-ingest:hover {
  background: var(--primary);
  color: var(--on-primary, #ffffff);
  transform: translateY(-1px);
}

.btn-activate {
  background: color-mix(in srgb, var(--primary) 18%, var(--surface-container-low));
  color: var(--primary);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 700;
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
  background: color-mix(in srgb, #d97706 12%, var(--surface-container-low));
  color: #b45309;
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-sleep:hover {
  background: #d97706;
  color: #ffffff;
  transform: translateY(-1px);
}

.btn-upgrade-deep {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: color-mix(in srgb, var(--primary) 14%, var(--surface-container-low));
  color: var(--primary);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-upgrade-deep:hover {
  background: var(--primary);
  border-color: var(--primary);
  color: var(--on-primary, #ffffff);
  transform: translateY(-1px);
}

.btn-processing {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  color: var(--primary);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 8px 14px;
  font-weight: 600;
  cursor: wait;
}

.spinner-small {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--outline-variant);
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
