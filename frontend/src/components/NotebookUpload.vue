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
        <!-- Core File & Folder Dropzone -->
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
          <div class="drop-icon-wrapper">
            <span class="drop-main-icon">📂</span>
          </div>
          <p class="drop-title">Drag & drop your study material</p>
          <p class="drop-subtitle">
            Upload a PDF, Markdown (.md), or Text (.txt) file or folder
          </p>
          <button type="button" class="upload-cta">Browse Files</button>
          <p class="drop-hint">
            PDF, MD, TXT &bull; Up to 50 MB per file &bull; Multi-files combine into one notebook
          </p>
        </div>

        <!-- Dynamic Extension Importers Tray -->
        <div v-if="activeImporters.length > 0" class="extension-importers-tray">
          <span class="tray-label">Import from extensions:</span>
          <div class="importer-buttons">
            <button
              v-for="importer in activeImporters"
              :key="importer.id"
              type="button"
              class="importer-pill-btn"
              :title="importer.description"
              @click="openImporter(importer)"
            >
              <span class="importer-icon">{{ importer.icon }}</span>
              <span class="importer-name">{{ importer.name }}</span>
            </button>
          </div>
        </div>

        <!-- Progress and Alerts -->
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

        <!-- Dynamic Importer Modals -->
        <YoutubeImportModal
          :show="activeModal === 'youtube'"
          :is-loading="uploadProgress > 0 && uploadProgress < 100"
          :status-message="ingestionStatusMessage"
          :error="uploadError || localError"
          @close="closeActiveModal"
          @submit="handleYoutubeSubmit"
        />
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useDialog } from '../composables/useDialog'
import { useExtensions } from '../composables/useExtensions'
import { getAvailableImporters } from '../services/importerRegistry'
import YoutubeImportModal from './YoutubeImportModal.vue'

defineProps({
  isCloudProfile: { type: Boolean, default: false },
  classroomCode: { type: String, default: '' },
  uploadProgress: { type: Number, default: 0 },
  ingestionStatusMessage: { type: String, default: '' },
  indexingStatusMessage: { type: String, default: '' },
  uploadError: { type: String, default: '' },
  successMessage: { type: String, default: '' },
})

const emit = defineEmits(['upload-file', 'upload-youtube'])
const { confirm } = useDialog()
const { isExtensionActive } = useExtensions()

const fileInput = ref(null)
const isDragging = ref(false)
const localError = ref('')
const activeModal = ref(null)

const activeImporters = computed(() => getAvailableImporters(isExtensionActive))

function openImporter(importer) {
  localError.value = ''
  activeModal.value = importer.modalName
}

function closeActiveModal() {
  activeModal.value = null
}

function handleYoutubeSubmit(url) {
  localError.value = ''
  emit('upload-youtube', url)
}

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
  margin-bottom: 40px;
}

.upload-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.cloud-locked-container {
  text-align: center;
  padding: 24px;
}

.cloud-locked-container .upload-icon {
  font-size: 40px;
  margin-bottom: 12px;
}

/* File Drop Zone */
.drop-zone {
  border: 1.5px dashed var(--outline-variant);
  border-radius: 12px;
  padding: 32px 24px;
  text-align: center;
  cursor: pointer;
  background: var(--surface-container-lowest);
  min-height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.2s ease;
}

.drop-zone:hover,
.drop-zone.dragging {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 6%, var(--surface-container-lowest));
  transform: translateY(-1px);
}

.drop-icon-wrapper {
  margin-bottom: 2px;
}

.drop-main-icon {
  font-size: 32px;
}

.drop-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface);
}

.drop-subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

.upload-cta {
  margin: 8px 0 4px;
  border: none;
  border-radius: 8px;
  padding: 9px 18px;
  font-size: 13.5px;
  font-weight: 600;
  font-family: inherit;
  color: var(--on-primary);
  background: var(--primary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.upload-cta:hover {
  filter: brightness(1.08);
}

.upload-cta:active {
  transform: scale(0.97);
}

.drop-hint {
  margin: 0;
  font-size: 12px;
  color: var(--muted-text);
  opacity: 0.8;
}

/* Extension Importers Tray */
.extension-importers-tray {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 14px;
  padding: 10px 14px;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
}

.tray-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--muted-text);
}

.importer-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.importer-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--on-surface);
  font-size: 12.5px;
  font-family: inherit;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.importer-pill-btn:hover {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, var(--surface-container));
  transform: translateY(-1px);
}

.importer-pill-btn:active {
  transform: scale(0.97);
}

.importer-icon {
  font-size: 13px;
}

/* Progress & Feedback */
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
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  border-radius: 8px;
  font-size: 13.5px;
}

.success-message {
  margin-top: 12px;
  padding: 12px;
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #10b981;
  border-radius: 8px;
  font-size: 13.5px;
}
</style>

