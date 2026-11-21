<template>
  <div class="assignments-page fade-in">
    <section class="section-card">
      <div style="margin-bottom: 1.5rem;">
        <h2 class="section-title" style="margin-bottom: 0.25rem;">
          Course Assignments
        </h2>
        <div class="micro-label">PUBLISH READINGS & PDF MATERIALS</div>
      </div>

      <form @submit.prevent="publishAssignment" style="margin-bottom: 2.25rem;" class="card">
        <span class="micro-label" style="display: block; margin-bottom: 1rem;">PUBLISH NEW PDF ASSIGNMENT</span>

        <div class="form-group">
          <label for="assign-title">Assignment Title</label>
          <input
            id="assign-title"
            v-model="newTitle"
            type="text"
            required
            placeholder="e.g. Chapter 4: Cell Division"
          />
        </div>

        <div class="form-group">
          <label>Upload Local PDF File (Max 75MB)</label>
          <input
            ref="fileInputRef"
            type="file"
            accept="application/pdf,.pdf"
            :disabled="uploadingPdf || publishing"
            style="display: none;"
            @change="onFileSelected"
          />
          <div
            class="custom-file-dropzone"
            :class="{ 'is-dragging': isDragging, 'is-uploading': uploadingPdf, 'has-file': !!selectedFileName }"
            @click="triggerBrowse"
            @dragover.prevent="isDragging = true"
            @dragleave.prevent="isDragging = false"
            @drop.prevent="onDrop"
          >
            <div class="file-dropzone-icon">
              <svg v-if="uploadingPdf" class="spinner-icon" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" stroke-opacity="0.25"></circle>
                <path d="M12 2a10 10 0 0 1 10 10" stroke-linecap="round"></path>
              </svg>
              <svg v-else-if="selectedFileName" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
                <line x1="16" y1="13" x2="8" y2="13"></line>
                <line x1="16" y1="17" x2="8" y2="17"></line>
              </svg>
              <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                <polyline points="17 8 12 3 7 8"></polyline>
                <line x1="12" y1="3" x2="12" y2="15"></line>
              </svg>
            </div>

            <div class="file-dropzone-info">
              <template v-if="uploadingPdf">
                <span class="file-title">Uploading to Supabase Storage...</span>
                <span class="file-sub">Generating secure signed URL...</span>
              </template>
              <template v-else-if="selectedFileName">
                <span class="file-title">{{ selectedFileName }}</span>
                <span class="file-sub">{{ selectedFileSize ? formatBytes(selectedFileSize) : 'Ready' }} &bull; Click or drop another to replace</span>
              </template>
              <template v-else>
                <span class="file-title">Choose a PDF file or drag & drop here</span>
                <span class="file-sub">PDF up to 75MB &bull; Automatically creates secure signed download link</span>
              </template>
            </div>

            <button
              type="button"
              class="btn-browse"
              :disabled="uploadingPdf || publishing"
              @click.stop="triggerBrowse"
            >
              {{ selectedFileName ? 'Change File' : 'Browse File' }}
            </button>
          </div>
        </div>

        <div class="form-group">
          <label for="assign-url">Or Direct PDF URL</label>
          <input
            id="assign-url"
            v-model="newUrl"
            type="url"
            required
            placeholder="https://example.com/files/cell_chap4.pdf"
          />
        </div>

        <div style="display: flex; gap: 1rem; flex-wrap: wrap;">
          <div class="form-group" style="flex: 1; min-width: 140px;">
            <label for="assign-start-page">Start Page (Optional)</label>
            <input
              id="assign-start-page"
              v-model.number="newStartPage"
              type="number"
              min="1"
              placeholder="e.g. 1"
            />
          </div>
          <div class="form-group" style="flex: 1; min-width: 140px;">
            <label for="assign-end-page">End Page (Optional)</label>
            <input
              id="assign-end-page"
              v-model.number="newEndPage"
              type="number"
              min="1"
              placeholder="e.g. 25"
            />
          </div>
        </div>

        <!-- PDF Preview trigger -->
        <div v-if="newUrl" style="margin-top: 0.75rem;">
          <button type="button" class="btn-ghost" style="font-size: 0.75rem;" @click="openPreview(newUrl, newStartPage || 1, newTitle || 'New Assignment')">
            📄 Preview PDF
          </button>
        </div>

        <button class="btn" style="width: 100%; margin-top: 0.75rem;" :disabled="publishing">
          <span v-if="publishing" class="loading-spinner" style="width: 12px; height: 12px; border-width: 2px;"></span>
          {{ publishing ? 'Publishing...' : 'Publish to Class' }}
        </button>
      </form>

      <div>
        <div style="display: flex; justify-content: space-between; align-items: center; border-top: 1px solid var(--ds-border); padding-top: 1.25rem; margin-bottom: 1rem;">
          <span class="micro-label">ACTIVE ASSIGNMENTS ({{ assignments.length }})</span>
        </div>

        <div v-if="loadingAssignments" class="text-center" style="padding: 2rem 0;">
          <div class="loading-spinner"></div>
        </div>

        <div v-else-if="assignments.length === 0" class="muted" style="font-size: 0.8rem; font-style: italic; text-align: center; padding: 2.5rem 1rem; border: 1px dashed var(--ds-border-hi); border-radius: 10px; background: var(--ds-surface-low);">
          No assignments published yet. Use the form above to post reading materials.
        </div>

        <div v-else class="assignments-list">
          <div
            v-for="asm in assignments"
            :key="asm.id"
            class="assignment-item"
            style="padding: 1rem 1.25rem;"
          >
            <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap;">
              <div style="min-width: 0; flex: 1;">
                <h4 style="margin: 0 0 0.35rem 0; font-size: 0.9rem; font-weight: 600; color: var(--ds-fg); display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap;">
                  <span>PDF: {{ asm.title }}</span>
                  <span v-if="asm.start_page || asm.end_page" style="font-size: 0.68rem; font-weight: 600; padding: 0.15rem 0.45rem; border-radius: 4px; background: var(--ds-primary-glow); border: 1px solid var(--ds-primary-ring); color: var(--ds-primary); font-family: var(--ds-mono);">
                    Pages {{ asm.start_page || 1 }}–{{ asm.end_page || 'End' }}
                  </span>
                </h4>
                <a :href="formatPdfUrl(asm.download_url)" target="_blank" rel="noopener noreferrer" style="font-size: 0.75rem; color: var(--ds-muted); text-decoration: none; word-break: break-all; display: block; margin-bottom: 0.25rem;" :title="formatPdfUrl(asm.download_url)">
                  {{ formatPdfUrl(asm.download_url) }}
                </a>
                <div style="font-size: 0.7rem; color: var(--ds-muted); font-family: var(--ds-mono);">Published {{ formatDate(asm.created_at) }}</div>
              </div>

              <div style="display: flex; gap: 0.5rem; align-items: center;">
                <button
                  class="btn-ghost"
                  style="font-size: 0.75rem; padding: 0.3rem 0.65rem;"
                  @click="openPreview(formatPdfUrl(asm.download_url), asm.start_page || 1)"
                >
                  Preview PDF
                </button>
                <button
                  class="btn-ghost danger"
                  style="font-size: 0.75rem; padding: 0.3rem 0.65rem;"
                  @click="deleteAssignment(asm.id)"
                  title="Remove assignment"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

  </div>
</template>

<script setup>
import { ref } from 'vue';
import { useDashboard } from '../composables/useDashboard';

// ponytail: auto-repair missing /storage/v1 prefix on existing stored URLs
function formatPdfUrl(url) {
  if (!url) return '';
  if (url.includes('.supabase.co/object/sign/')) {
    return url.replace('.supabase.co/object/sign/', '.supabase.co/storage/v1/object/sign/');
  }
  return url;
}

// ponytail: native browser tab over broken iframe embed; avoids CORS and X-Frame-Options DENY
function openPreview(url, page = 1) {
  if (!url) return;
  const target = formatPdfUrl(url);
  window.open(`${target}#page=${page}`, '_blank', 'noopener,noreferrer');
}

const {
  newTitle,
  newUrl,
  newStartPage,
  newEndPage,
  publishing,
  uploadingPdf,
  assignments,
  loadingAssignments,
  handleFileUpload,
  publishAssignment,
  deleteAssignment,
  formatDate
} = useDashboard();

const fileInputRef = ref(null);
const isDragging = ref(false);
const selectedFileName = ref('');
const selectedFileSize = ref(0);

function triggerBrowse() {
  if (uploadingPdf.value || publishing.value) return;
  fileInputRef.value?.click();
}

async function onFileSelected(e) {
  const file = e.target.files?.[0];
  if (!file) return;
  selectedFileName.value = file.name;
  selectedFileSize.value = file.size;
  await handleFileUpload(e);
}

async function onDrop(e) {
  isDragging.value = false;
  if (uploadingPdf.value || publishing.value) return;
  const file = e.dataTransfer?.files?.[0];
  if (!file) return;
  selectedFileName.value = file.name;
  selectedFileSize.value = file.size;
  await handleFileUpload(file);
}

function formatBytes(bytes) {
  if (!bytes) return '';
  if (bytes < 1024) return bytes + ' B';
  const kb = bytes / 1024;
  if (kb < 1024) return kb.toFixed(1) + ' KB';
  return (kb / 1024).toFixed(1) + ' MB';
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed; inset: 0; background: rgba(0, 0, 0, 0.75);
  display: flex; align-items: center; justify-content: center;
  z-index: 9999; backdrop-filter: blur(4px); padding: 1rem;
}
.modal-card {
  width: 90vw; max-width: 1000px; height: 85vh;
  background: var(--ds-surface, #1e1e24); border: 1px solid var(--ds-border-hi);
  border-radius: 12px; display: flex; flex-direction: column; overflow: hidden;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
}
.modal-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 0.75rem 1rem; background: var(--ds-surface-low); border-bottom: 1px solid var(--ds-border);
  font-weight: 600; color: var(--ds-fg); font-size: 0.9rem;
}
.modal-pdf { width: 100%; height: 100%; border: none; background: #fff; flex: 1; }
.loading-spinner {
  border: 3px solid var(--ds-border-hi);
  border-top: 3px solid var(--ds-primary);
  border-radius: 50%;
  width: 2.25rem;
  height: 2.25rem;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Custom PDF File Dropzone */
.custom-file-dropzone {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.85rem 1.15rem;
  border-radius: 10px;
  background: var(--ds-input-bg);
  border: 1.5px dashed var(--ds-border-hi);
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  user-select: none;
}

.custom-file-dropzone:hover {
  border-color: var(--ds-primary);
  background: var(--ds-surface-hi);
}

.custom-file-dropzone.is-dragging {
  border-color: var(--ds-primary);
  background: var(--ds-primary-glow);
  box-shadow: 0 0 0 3px var(--ds-primary-ring);
  transform: scale(1.005);
}

.custom-file-dropzone.has-file {
  border-style: solid;
  border-color: var(--ds-border-hi);
}

.custom-file-dropzone.is-uploading {
  cursor: wait;
  opacity: 0.85;
}

.file-dropzone-icon {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  background: var(--ds-surface-hi);
  color: var(--ds-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.custom-file-dropzone:hover .file-dropzone-icon {
  background: var(--ds-primary-glow);
  transform: scale(1.05);
}

.file-dropzone-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.file-title {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--ds-fg);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-sub {
  font-size: 0.72rem;
  color: var(--ds-muted);
}

.btn-browse {
  font-family: inherit;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.45rem 0.85rem;
  border-radius: 6px;
  border: 1px solid var(--ds-border-hi);
  background: var(--ds-surface-hi);
  color: var(--ds-fg);
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-browse:hover:not(:disabled) {
  border-color: var(--ds-primary);
  color: var(--ds-primary);
  background: var(--ds-primary-glow);
}

.btn-browse:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinner-icon {
  animation: spin 0.8s linear infinite;
}
</style>

