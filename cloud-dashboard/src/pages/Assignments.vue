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
          <label for="assign-file">Upload Local PDF File (Max 75MB)</label>
          <input
            id="assign-file"
            type="file"
            accept="application/pdf,.pdf"
            :disabled="uploadingPdf || publishing"
            @change="handleFileUpload"
          />
          <span v-if="uploadingPdf" class="muted" style="font-size: 0.78rem; margin-top: 0.25rem; display: block;">
            Uploading PDF to Supabase Storage...
          </span>
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
                <a :href="asm.download_url" target="_blank" rel="noopener noreferrer" style="font-size: 0.75rem; color: var(--ds-muted); text-decoration: none; word-break: break-all; display: block; margin-bottom: 0.25rem;" :title="asm.download_url">
                  {{ asm.download_url }}
                </a>
                <div style="font-size: 0.7rem; color: var(--ds-muted); font-family: var(--ds-mono);">Published {{ formatDate(asm.created_at) }}</div>
              </div>

              <div style="display: flex; gap: 0.5rem; align-items: center;">
                <button
                  class="btn-ghost"
                  style="font-size: 0.75rem; padding: 0.3rem 0.65rem;"
                  @click="openPreview(asm.download_url, asm.start_page || 1, asm.title)"
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

    <!-- Centered PDF Preview Modal -->
    <Teleport to="body">
      <div v-if="previewUrl || loadingPreview" class="modal-backdrop" @click.self="closePreview">
        <div class="modal-card">
          <div class="modal-head">
            <span>📄 {{ previewTitle }}</span>
            <button class="btn-ghost" style="font-size: 0.8rem; padding: 0.2rem 0.5rem;" @click="closePreview">✕ Close</button>
          </div>
          <div v-if="loadingPreview" style="display: flex; flex-direction: column; align-items: center; justify-content: center; flex: 1; color: var(--ds-fg); gap: 1rem;">
            <div class="loading-spinner"></div>
            <span style="font-size: 0.85rem; color: var(--ds-muted);">Downloading PDF for preview...</span>
          </div>
          <iframe v-else-if="previewUrl" :src="previewUrl" class="modal-pdf" title="PDF Preview" sandbox="allow-scripts allow-same-origin"></iframe>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { useDashboard } from '../composables/useDashboard';

const previewUrl = ref(null);
const previewTitle = ref('');
const loadingPreview = ref(false);
let activeObjectUrl = null;

async function openPreview(url, page = 1, title = 'PDF Preview') {
  previewTitle.value = title;
  loadingPreview.value = true;
  
  if (activeObjectUrl) {
    URL.revokeObjectURL(activeObjectUrl);
    activeObjectUrl = null;
  }
  previewUrl.value = null;

  try {
    const res = await fetch(url);
    if (!res.ok) {
      throw new Error(`Failed to fetch file (HTTP ${res.status})`);
    }
    const blob = await res.blob();
    const blobPdf = new Blob([blob], { type: 'application/pdf' });
    activeObjectUrl = URL.createObjectURL(blobPdf);
    previewUrl.value = `${activeObjectUrl}#page=${page}`;
  } catch (err) {
    console.error('Error fetching PDF for preview:', err);
    alert(`Could not load PDF preview: ${err.message}. You can try downloading it directly by clicking its link.`);
  } finally {
    loadingPreview.value = false;
  }
}

function closePreview() {
  if (activeObjectUrl) {
    URL.revokeObjectURL(activeObjectUrl);
    activeObjectUrl = null;
  }
  previewUrl.value = null;
  loadingPreview.value = false;
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
</style>

