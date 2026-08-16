<template>
  <div class="upload-section">
    <div class="upload-card">
      <div v-if="isCloudProfile" class="cloud-locked-container">
        <div class="upload-icon">☁️</div>
        <h3>Cloud Classroom Active</h3>
        <p>
          Direct PDF uploads are disabled for Cloud Profiles. Study materials published by your
          teacher in classroom
          <strong v-if="classroomCode">{{ classroomCode }}</strong>
          will download automatically.
        </p>
      </div>

      <template v-else>
        <div class="upload-icon">📄</div>
        <h3>Upload Your Study Material</h3>
        <p class="upload-desc">
          Upload a PDF, Markdown (.md), or Text (.txt) file to create a notebook.
          <span class="upload-note"
            >You can also upload multiple files of the same type — they’ll be combined into one
            notebook.</span
          >
        </p>

        <input
          ref="fileInput"
          type="file"
          multiple
          accept=".pdf,.md,.txt"
          style="display: none"
          @change="handleFileSelect"
        />

        <div
          class="drop-zone"
          :class="{ dragging: isDragging }"
          @click="triggerFilePicker"
          @dragover.prevent="isDragging = true"
          @dragleave.prevent="isDragging = false"
          @drop.prevent="handleFileDrop"
        >
          <p class="drop-title">Drop files or a folder here</p>
          <button type="button" class="upload-cta">Choose Files</button>
          <p class="drop-hint">
            PDF, MD, TXT &bull; Up to 50 MB per file &bull; Multiple files must be the same type
          </p>
        </div>

        <div v-if="uploadProgress > 0 && uploadProgress < 100" class="progress">
          <div class="progress-bar" :style="{ width: uploadProgress + '%' }"></div>
          <span>{{ uploadProgress }}%</span>
          <p v-if="ingestionStatusMessage" class="progress-label">{{ ingestionStatusMessage }}</p>
        </div>

        <div v-if="indexingStatusMessage" class="progress indexing-progress">
          <p class="progress-label">{{ indexingStatusMessage }}</p>
        </div>

        <div v-if="uploadError || localError" class="error-message">
          {{ uploadError || localError }}
        </div>

        <div v-if="successMessage" class="success-message">{{ successMessage }}</div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useDialog } from '../composables/useDialog'

defineProps({
  isCloudProfile: { type: Boolean, default: false },
  classroomCode: { type: String, default: '' },
  uploadProgress: { type: Number, default: 0 },
  ingestionStatusMessage: { type: String, default: '' },
  indexingStatusMessage: { type: String, default: '' },
  uploadError: { type: String, default: '' },
  successMessage: { type: String, default: '' },
})

const emit = defineEmits(['upload-file'])
const { confirm } = useDialog()

const fileInput = ref(null)
const isDragging = ref(false)
const localError = ref('')

function triggerFilePicker() {
  localError.value = ''
  fileInput.value?.click()
}

async function processFiles(fileList) {
  localError.value = ''
  const files = Array.from(fileList || []).filter(
    (f) => !f.name.startsWith('.') && f.name !== 'Thumbs.db' && !f.name.includes('__MACOSX')
  )

  if (!files.length) return

  if (files.length === 1) {
    emit('upload-file', files[0])
    return
  }

  // Same-type check for folder/multi-file drops
  const exts = new Set(files.map((f) => f.name.split('.').pop().toLowerCase()))
  if (exts.has('pdf')) {
    localError.value =
      exts.size > 1
        ? 'Mixed file types detected. Please ensure all files in the folder are notes (.md or .txt).'
        : 'Folders of PDF files are not supported as a single book. Please upload PDFs individually.'
    return
  }

  const unsupported = Array.from(exts).filter(
    (e) => e !== 'md' && e !== 'markdown' && e !== 'txt' && e !== 'text'
  )
  if (unsupported.length > 0) {
    localError.value = 'Folders must contain only text/markdown (.md or .txt) chapter files.'
    return
  }

  // Sort files deterministically (natural order)
  files.sort((a, b) =>
    a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })
  )

  const folderName = files[0].webkitRelativePath?.split('/')[0] || 'Course Notes'

  // ponytail: reuse global useDialog() confirm modal instead of custom modal component
  const ok = await confirm({
    title: `Upload ${files.length} files to this site?`,
    message: `This will upload all files from "${folderName}". Do this only if you trust the site.`,
    confirmText: 'Upload',
    cancelText: 'Cancel',
    type: 'info',
  })
  if (!ok) return

  const sections = []
  for (const f of files) {
    const text = await f.text()
    const title = f.name
      .replace(/\.[^/.]+$/, '')
      .replace(/[-_]+/g, ' ')
      .trim()
    sections.push(`# ${title}\n\n${text.trim()}`)
  }

  emit(
    'upload-file',
    new File([sections.join('\n\n')], `${folderName}.md`, { type: 'text/markdown' })
  )
}

function handleFileSelect(e) {
  if (e.target.files?.length) void processFiles(e.target.files)
  e.target.value = ''
}

function handleFileDrop(e) {
  isDragging.value = false
  if (e.dataTransfer?.files?.length) void processFiles(e.dataTransfer.files)
}
</script>

<style scoped>
.upload-section {
  display: block;
  margin-bottom: 48px;
}

.upload-card {
  background: var(--surface-container-low);
  border-radius: 16px;
  padding: 24px;
}

.upload-icon {
  font-size: 48px;
  text-align: center;
  margin-bottom: 16px;
}

.upload-card h3 {
  margin: 0 0 8px;
  font-size: 18px;
  color: var(--on-surface);
}

.upload-card p {
  margin: 0 0 16px;
  font-size: 14px;
  color: var(--muted-text);
}

.upload-desc {
  margin: 0 0 18px !important;
  line-height: 1.5;
}

.upload-desc strong {
  color: var(--on-surface);
  font-weight: 600;
}

.upload-note {
  display: block;
  margin-top: 4px;
  font-size: 12.5px;
  color: var(--muted-text);
  opacity: 0.9;
}

.drop-zone {
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 28px;
  text-align: center;
  cursor: pointer;
  background: var(--surface-container-lowest);
  min-height: 170px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.drop-zone:hover,
.drop-zone.dragging {
  background: rgba(0, 91, 193, 0.06);
  border-color: var(--primary);
}

.drop-title {
  margin: 0;
  font-size: 18px;
  font-family: 'Manrope', sans-serif;
  font-weight: 700;
  color: var(--on-surface);
}

.upload-cta {
  border: none;
  border-radius: 12px;
  padding: 12px 20px;
  font-size: 14px;
  font-family: 'Manrope', sans-serif;
  font-weight: 700;
  color: var(--on-primary);
  background: linear-gradient(15deg, var(--primary), var(--primary-dim));
  cursor: pointer;
  transition: all 0.15s ease;
}

.upload-cta:active {
  transform: scale(0.97);
}

.drop-hint {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

.progress {
  margin-top: 16px;
}

.progress-bar {
  height: 4px;
  background: var(--primary);
  border-radius: 2px;
  transition: width 0.3s;
}

.progress span {
  display: block;
  font-size: 12px;
  color: var(--muted-text);
  margin-top: 8px;
  text-align: center;
}

.progress-label {
  margin: 8px 0 0;
  text-align: center;
  font-size: 12px;
  color: var(--muted-text);
}

.indexing-progress {
  margin-top: 12px;
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 12px;
  background: var(--surface-container-low);
}

.indexing-progress .progress-bar {
  background: linear-gradient(15deg, #2e7d32, #4caf50);
}

.error-message {
  margin-top: 12px;
  padding: 12px;
  background: #ffebee;
  color: #c62828;
  border-radius: 6px;
  font-size: 14px;
}

.success-message {
  margin-top: 12px;
  padding: 12px;
  background: #e8f5e9;
  color: #2e7d32;
  border-radius: 6px;
  font-size: 14px;
}
</style>
