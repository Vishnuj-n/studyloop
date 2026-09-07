<template>
  <!-- Pills Variant (e.g. Socratic Tutor header) -->
  <div v-if="variant === 'pills'" class="selector-pills">
    <div class="selector-pill">
      <span class="pill-icon" aria-hidden="true">📖</span>
      <select
        id="notebook-select"
        :value="notebookId"
        :disabled="disabled || internalLoading"
        aria-label="Select notebook"
        @change="onNotebookSelect($event.target.value)"
      >
        <option value="" disabled>Choose notebook</option>
        <option
          v-for="nb in tree"
          :key="nb.notebook_id"
          :value="nb.notebook_id"
        >
          {{ nb.title }}
        </option>
      </select>
    </div>

    <div class="selector-pill">
      <span class="pill-icon" aria-hidden="true">🎯</span>
      <select
        id="topic-select"
        :value="topicId"
        :disabled="disabled || internalLoading || !notebookId || availableTopics.length === 0"
        aria-label="Select topic"
        @change="onTopicSelect($event.target.value)"
      >
        <option v-if="!notebookId" value="" disabled>Choose notebook first</option>
        <option v-else-if="allowEntireNotebook" value="">
          Entire book (No topic filter)
        </option>
        <option v-else-if="availableTopics.length === 0" value="" disabled>
          No topics available
        </option>
        <option v-else value="" disabled>Choose topic</option>
        <option
          v-for="topic in availableTopics"
          :key="topic.topic_id"
          :value="topic.topic_id"
        >
          {{ topic.title }}
        </option>
      </select>
    </div>
  </div>

  <!-- Fields Variant (e.g. Reader controls panel) -->
  <div v-else class="selector-fields">
    <label class="field">
      <span>Notebook</span>
      <select
        :value="notebookId"
        :disabled="disabled || internalLoading || tree.length === 0"
        aria-label="Select notebook"
        @change="onNotebookSelect($event.target.value)"
      >
        <option disabled value="">Select notebook</option>
        <option
          v-for="nb in tree"
          :key="nb.notebook_id"
          :value="nb.notebook_id"
        >
          {{ nb.title }}
        </option>
      </select>
    </label>

    <label class="field">
      <span>Topic</span>
      <select
        :value="topicId"
        :disabled="disabled || internalLoading || !notebookId || availableTopics.length === 0"
        aria-label="Select topic"
        @change="onTopicSelect($event.target.value)"
      >
        <option v-if="!notebookId" disabled value="">Choose notebook first</option>
        <option v-else-if="allowEntireNotebook" value="">
          Entire book (No topic filter)
        </option>
        <option v-else-if="availableTopics.length === 0" disabled value="">
          No topics available
        </option>
        <option v-else disabled value="">Select topic</option>
        <option
          v-for="topic in availableTopics"
          :key="topic.topic_id"
          :value="topic.topic_id"
        >
          {{ topic.title }}
        </option>
      </select>
    </label>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { getNotebookTopicTree } from '../services/appApi'
import { sortNotebookTopics } from '../composables/useReaderBase'

const props = defineProps({
  notebookId: {
    type: String,
    default: '',
  },
  topicId: {
    type: String,
    default: '',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  allowEntireNotebook: {
    type: Boolean,
    default: false,
  },
  variant: {
    type: String,
    default: 'fields',
    validator: (v) => ['fields', 'pills'].includes(v),
  },
  notebookTree: {
    type: Array,
    default: null,
  },
})

const emit = defineEmits([
  'update:notebookId',
  'update:topicId',
  'changeNotebook',
  'changeTopic',
])

const internalTree = ref([])
const internalLoading = ref(false)

const tree = computed(() => {
  return props.notebookTree !== null ? props.notebookTree : internalTree.value
})

const selectedNotebook = computed(() => {
  return tree.value.find((n) => n.notebook_id === props.notebookId) || null
})

const availableTopics = computed(() => {
  if (!selectedNotebook.value) return []
  return sortNotebookTopics(selectedNotebook.value.topics || [])
})

async function fetchTree() {
  if (props.notebookTree !== null) return
  internalLoading.value = true
  try {
    const data = await getNotebookTopicTree()
    internalTree.value = Array.isArray(data) ? data : []
  } catch (err) {
    console.error('[NotebookTopicSelector] Failed to load topic tree:', err)
    internalTree.value = []
  } finally {
    internalLoading.value = false
  }
}

onMounted(() => {
  fetchTree()
})

function onNotebookSelect(newNotebookId) {
  emit('update:notebookId', newNotebookId)
  emit('changeNotebook', newNotebookId)

  emit('update:topicId', '')
  emit('changeTopic', '')
}

function onTopicSelect(newTopicId) {
  emit('update:topicId', newTopicId)
  emit('changeTopic', newTopicId)
}
</script>

<style scoped>
/* Fields variant styles */
.selector-fields {
  display: flex;
  gap: 16px;
  align-items: center;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field span {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--on-surface-variant);
}

.field select {
  min-width: 220px;
  max-width: 320px;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--on-surface);
  font-family: inherit;
  font-size: 13px;
  outline: none;
  cursor: pointer;
  text-overflow: ellipsis;
  transition: border-color 0.15s ease;
}

.field select:focus {
  border-color: var(--primary);
}

.field select:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Pills variant styles */
.selector-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.selector-pill {
  display: inline-flex;
  align-items: center;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  padding: 4px 12px 4px 10px;
  font-family: inherit;
  font-size: 13px;
  transition: all 0.2s ease;
}

.selector-pill:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px rgba(0, 91, 193, 0.1);
}

.pill-icon {
  font-size: 14px;
  margin-right: 6px;
}

.selector-pill select {
  border: none;
  background: transparent;
  color: var(--on-surface);
  font-family: inherit;
  font-size: 13px;
  font-weight: 500;
  outline: none;
  padding: 2px 4px;
  cursor: pointer;
  max-width: 260px;
  text-overflow: ellipsis;
}

.selector-pill select:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
