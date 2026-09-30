<template>
  <Teleport to="body">
    <div v-if="visible" class="cert-backdrop" @click.self="closeModal">
      <div class="cert-modal-card">
        <!-- Close Button -->
        <button class="close-btn" type="button" aria-label="Close" @click="closeModal">
          <BaseIcon name="x" size="18" />
        </button>

        <!-- Ambient glow -->
        <div class="cert-ambient-glow"></div>

        <div class="cert-header">
          <div class="cert-badge-pill">
            <BaseIcon name="sparkles" size="14" />
            <span>STUDYLOOP MASTERY CERTIFICATE</span>
            <BaseIcon name="sparkles" size="14" />
          </div>
          <h2 class="cert-title">Textbook Completed!</h2>
          <p class="cert-subtitle">
            You have mastered the syllabus and completed all guided study milestones.
          </p>
        </div>

        <!-- Recipient Name Input Option -->
        <div class="cert-name-customization">
          <div class="cert-name-input-group">
            <label for="cert-recipient-name" class="cert-name-label">
              <BaseIcon name="user" size="14" />
              <span>Recipient Name:</span>
            </label>
            <div class="cert-name-input-wrapper">
              <input
                id="cert-recipient-name"
                v-model="recipientName"
                type="text"
                maxlength="50"
                placeholder="Enter recipient name (e.g. Alex Morgan)"
                class="cert-name-input"
                @input="handleNameInput"
              />
              <button
                v-if="recipientName.trim() && recipientName !== defaultRecipientName"
                type="button"
                class="cert-name-reset-btn"
                title="Reset to default title"
                @click="resetRecipientName"
              >
                Reset
              </button>
            </div>
          </div>
        </div>

        <!-- Certificate Canvas Preview Frame -->
        <div class="cert-preview-frame">
          <div class="cert-document" ref="certDocumentRef">
            <div class="cert-inner-border">
              <div class="cert-watermark">STUDYLOOP</div>

              <!-- Top Row / Seal -->
              <div class="doc-header">
                <div class="doc-brand">
                  <span class="brand-name">Studyloop</span>
                  <span class="brand-tag">Autonomous Learning Environment</span>
                </div>
                <div class="doc-seal">
                  <BaseIcon name="award" size="24" />
                  <span>VERIFIED</span>
                </div>
              </div>

              <!-- Main Certificate Text -->
              <div class="doc-body">
                <p class="doc-present">This is to certify that</p>
                <h3 class="doc-user-title">{{ displayRecipientName }}</h3>
                <p class="doc-subtext">has successfully studied and demonstrated mastery in</p>
                <h1 class="doc-notebook-title">{{ stats?.notebook_title || 'Course Textbook' }}</h1>
              </div>

              <!-- Metrics Row -->
              <div class="doc-metrics-grid">
                <div class="metric-box">
                  <span class="metric-val">{{ stats?.milestones_cleared || 0 }}</span>
                  <span class="metric-lbl">Milestones Cleared</span>
                </div>
                <div class="metric-box">
                  <span class="metric-val">
                    {{ stats?.quizzes_passed || 0 }}
                    <small v-if="stats?.average_quiz_score">({{ Math.round(stats.average_quiz_score) }}%)</small>
                  </span>
                  <span class="metric-lbl">Quizzes Passed</span>
                </div>
                <div class="metric-box">
                  <span class="metric-val">{{ stats?.flashcards_mastered || 0 }}</span>
                  <span class="metric-lbl">Cards Mastered</span>
                </div>
                <div class="metric-box">
                  <span class="metric-val">{{ stats?.total_study_minutes || 0 }}m</span>
                  <span class="metric-lbl">Time Studied</span>
                </div>
              </div>

              <!-- Footer Signature / Timestamp -->
              <div class="doc-footer">
                <div class="doc-date">
                  <span class="footer-lbl">Date Completed</span>
                  <span class="footer-val">{{ formattedDate }}</span>
                </div>
                <div class="doc-signature">
                  <div class="signature-line"></div>
                  <span class="footer-lbl">Studyloop Verification System</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Hidden High-Res Canvas for PNG Export -->
        <canvas ref="exportCanvasRef" style="display: none;"></canvas>

        <!-- Action Controls -->
        <div class="cert-actions">
          <button
            class="export-btn primary"
            type="button"
            :disabled="isExporting"
            @click="exportCertificatePNG"
          >
            <BaseIcon :name="isExporting ? 'loader' : 'download'" size="16" />
            <span>{{ isExporting ? 'Generating PNG...' : 'Download PNG Certificate' }}</span>
          </button>
          <button
            class="export-btn secondary"
            type="button"
            :disabled="isExporting"
            @click="copyCertificateToClipboard"
          >
            <BaseIcon name="copy" size="16" />
            <span>{{ copySuccess ? 'Copied to Clipboard!' : 'Copy to Clipboard' }}</span>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { playChestOpenFanfare } from '../utils/audioJuice'
import { triggerConfettiCelebration } from '../utils/confettiCelebration'

const props = defineProps({
  stats: {
    type: Object,
    required: true,
  },
  freshUnlock: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['close'])

const visible = ref(true)
const isExporting = ref(false)
const copySuccess = ref(false)
const certDocumentRef = ref(null)
const exportCanvasRef = ref(null)
let confettiCleanup = null

const defaultRecipientName = computed(() => props.stats?.user_title || 'The Scholar')
const storedName = typeof window !== 'undefined' ? localStorage.getItem('studyloop_cert_recipient_name') : null
const recipientName = ref(storedName !== null ? storedName : defaultRecipientName.value)

function handleNameInput() {
  if (typeof window !== 'undefined') {
    localStorage.setItem('studyloop_cert_recipient_name', recipientName.value)
  }
}

function resetRecipientName() {
  recipientName.value = defaultRecipientName.value
  if (typeof window !== 'undefined') {
    localStorage.removeItem('studyloop_cert_recipient_name')
  }
}

const displayRecipientName = computed(() => {
  return recipientName.value.trim() || defaultRecipientName.value
})

const formattedDate = computed(() => {
  if (!props.stats?.completed_at) {
    return new Date().toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    })
  }
  const d = new Date(props.stats.completed_at)
  if (isNaN(d.getTime())) return props.stats.completed_at
  return d.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
})

function closeModal() {
  visible.value = false
  emit('close')
}

// High-DPI Canvas Rendering Engine for crisp PNG output
function renderCertificateToCanvas(canvas) {
  const width = 1920
  const height = 1200
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')

  // Background Gradient
  const bgGrad = ctx.createLinearGradient(0, 0, width, height)
  bgGrad.addColorStop(0, '#12141a')
  bgGrad.addColorStop(0.5, '#181b24')
  bgGrad.addColorStop(1, '#0e1015')
  ctx.fillStyle = bgGrad
  ctx.fillRect(0, 0, width, height)

  // Outer Border & Gold Accent
  ctx.strokeStyle = '#3a4150'
  ctx.lineWidth = 12
  ctx.strokeRect(40, 40, width - 80, height - 80)

  ctx.strokeStyle = '#d4af37'
  ctx.lineWidth = 4
  ctx.strokeRect(56, 56, width - 112, height - 112)

  // Corner Ornaments
  const drawCorner = (x, y, rot) => {
    ctx.save()
    ctx.translate(x, y)
    ctx.rotate(rot)
    ctx.strokeStyle = '#d4af37'
    ctx.lineWidth = 3
    ctx.beginPath()
    ctx.moveTo(0, 0)
    ctx.lineTo(40, 0)
    ctx.moveTo(0, 0)
    ctx.lineTo(0, 40)
    ctx.stroke()
    ctx.restore()
  }
  drawCorner(70, 70, 0)
  drawCorner(width - 70, 70, Math.PI / 2)
  drawCorner(width - 70, height - 70, Math.PI)
  drawCorner(70, height - 70, -Math.PI / 2)

  // Watermark
  ctx.save()
  ctx.font = '900 130px sans-serif'
  ctx.fillStyle = 'rgba(255, 255, 255, 0.02)'
  ctx.textAlign = 'center'
  ctx.fillText('STUDYLOOP', width / 2, height / 2 + 50)
  ctx.restore()

  // Header Brand
  ctx.font = '800 36px "Manrope", sans-serif'
  ctx.fillStyle = '#f0f3fa'
  ctx.textAlign = 'left'
  ctx.fillText('Studyloop', 100, 130)

  ctx.font = '500 18px sans-serif'
  ctx.fillStyle = '#9aa4b2'
  ctx.fillText('Autonomous Learning & Mastery Environment', 100, 162)

  // Seal / Badge (Right)
  ctx.save()
  ctx.beginPath()
  ctx.arc(width - 160, 140, 45, 0, Math.PI * 2)
  ctx.fillStyle = 'rgba(212, 175, 55, 0.15)'
  ctx.fill()
  ctx.strokeStyle = '#d4af37'
  ctx.lineWidth = 3
  ctx.stroke()

  ctx.font = '800 14px sans-serif'
  ctx.fillStyle = '#d4af37'
  ctx.textAlign = 'center'
  ctx.fillText('VERIFIED', width - 160, 142)
  ctx.font = '600 11px sans-serif'
  ctx.fillStyle = '#f0f3fa'
  ctx.fillText('MASTERY', width - 160, 158)
  ctx.restore()

  // Subtitle / Presenting
  ctx.font = '600 24px sans-serif'
  ctx.fillStyle = '#8f9bb3'
  ctx.textAlign = 'center'
  ctx.fillText('THIS IS PROUDLY PRESENTED TO', width / 2, 330)

  // Student Title / Name with dynamic font size scaling
  const nameToRender = displayRecipientName.value
  let fontSize = 52
  if (nameToRender.length > 32) {
    fontSize = 36
  } else if (nameToRender.length > 22) {
    fontSize = 44
  }
  ctx.font = `800 ${fontSize}px "Manrope", sans-serif`
  ctx.fillStyle = '#d4af37'
  ctx.fillText(nameToRender, width / 2, 400)

  // Text
  ctx.font = '500 22px sans-serif'
  ctx.fillStyle = '#8f9bb3'
  ctx.fillText('for successfully mastering all chapters, quizzes, and milestones in', width / 2, 465)

  // Book Title
  ctx.font = '800 48px "Manrope", sans-serif'
  ctx.fillStyle = '#ffffff'
  const bookTitle = props.stats?.notebook_title || 'Course Textbook'
  ctx.fillText(bookTitle, width / 2, 535)

  // Metrics Grid (4 columns)
  const metrics = [
    { lbl: 'Milestones Cleared', val: String(props.stats?.milestones_cleared || 0) },
    {
      lbl: 'Quizzes Passed',
      val: props.stats?.average_quiz_score
        ? `${props.stats.quizzes_passed || 0} (${Math.round(props.stats.average_quiz_score)}%)`
        : String(props.stats?.quizzes_passed || 0),
    },
    { lbl: 'Cards Mastered', val: String(props.stats?.flashcards_mastered || 0) },
    { lbl: 'Time Studied', val: `${props.stats?.total_study_minutes || 0} mins` },
  ]

  const startX = 220
  const boxWidth = 340
  const gap = 40
  const boxY = 660
  const boxH = 130

  metrics.forEach((m, idx) => {
    const bx = startX + idx * (boxWidth + gap)
    ctx.fillStyle = 'rgba(255, 255, 255, 0.04)'
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.roundRect(bx, boxY, boxWidth, boxH, 16)
    ctx.fill()
    ctx.stroke()

    ctx.font = '800 36px "Manrope", sans-serif'
    ctx.fillStyle = '#f0f3fa'
    ctx.textAlign = 'center'
    ctx.fillText(m.val, bx + boxWidth / 2, boxY + 58)

    ctx.font = '600 16px sans-serif'
    ctx.fillStyle = '#8f9bb3'
    ctx.fillText(m.lbl, bx + boxWidth / 2, boxY + 96)
  })

  // Footer / Date
  ctx.textAlign = 'left'
  ctx.font = '600 18px sans-serif'
  ctx.fillStyle = '#8f9bb3'
  ctx.fillText('Date Completed', 160, 980)
  ctx.font = '800 22px sans-serif'
  ctx.fillStyle = '#f0f3fa'
  ctx.fillText(formattedDate.value, 160, 1015)

  // Signature
  ctx.textAlign = 'right'
  ctx.strokeStyle = '#4a5568'
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.moveTo(width - 460, 990)
  ctx.lineTo(width - 160, 990)
  ctx.stroke()

  ctx.font = '600 16px sans-serif'
  ctx.fillStyle = '#8f9bb3'
  ctx.fillText('Studyloop Verification System', width - 160, 1020)
}

async function exportCertificatePNG() {
  if (isExporting.value) return
  isExporting.value = true
  await nextTick()

  try {
    const canvas = exportCanvasRef.value || document.createElement('canvas')
    renderCertificateToCanvas(canvas)

    const dataURL = canvas.toDataURL('image/png')
    const link = document.createElement('a')
    const safeTitle = (props.stats?.notebook_title || 'Studyloop-Mastery')
      .replace(/[^a-zA-Z0-9_-]/g, '_')
    link.download = `Studyloop-Certificate-${safeTitle}.png`
    link.href = dataURL
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  } catch (err) {
    console.error('Failed exporting certificate PNG:', err)
  } finally {
    isExporting.value = false
  }
}

async function copyCertificateToClipboard() {
  try {
    const canvas = exportCanvasRef.value || document.createElement('canvas')
    renderCertificateToCanvas(canvas)

    canvas.toBlob(async (blob) => {
      if (!blob) return
      try {
        await navigator.clipboard.write([
          new ClipboardItem({ 'image/png': blob })
        ])
        copySuccess.value = true
        setTimeout(() => {
          copySuccess.value = false
        }, 3000)
      } catch (clipErr) {
        console.warn('Clipboard write failed:', clipErr)
      }
    })
  } catch (err) {
    console.error('Failed copying certificate image:', err)
  }
}

onMounted(() => {
  if (props.freshUnlock) {
    try {
      playChestOpenFanfare()
    } catch (e) {
      console.debug('Audio skipped:', e)
    }

    try {
      confettiCleanup = triggerConfettiCelebration({
        tier: 'gold',
        particleCount: 140,
      })
    } catch (e) {
      console.debug('Confetti skipped:', e)
    }
  }
})

onUnmounted(() => {
  if (confettiCleanup) {
    confettiCleanup()
  }
})
</script>

<style scoped>
.cert-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.82);
  backdrop-filter: blur(14px);
  z-index: 99999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.cert-modal-card {
  position: relative;
  background: var(--surface-container-low, #181b22);
  border: 1px solid var(--outline-variant, #2e3644);
  border-radius: 24px;
  width: 100%;
  max-width: 780px;
  padding: 2rem 2.25rem;
  color: var(--on-surface, #f0f3fa);
  box-shadow: 0 32px 64px -12px rgba(0, 0, 0, 0.45);
  overflow: hidden;
  animation: scaleUp 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes scaleUp {
  from { transform: scale(0.9) translateY(20px); opacity: 0; }
  to { transform: scale(1) translateY(0); opacity: 1; }
}

.cert-ambient-glow {
  position: absolute;
  top: -80px;
  left: 50%;
  transform: translateX(-50%);
  width: 400px;
  height: 300px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(212, 175, 55, 0.22) 0%, transparent 70%);
  pointer-events: none;
  z-index: 0;
}

.close-btn {
  position: absolute;
  top: 18px;
  right: 20px;
  background: none;
  border: none;
  color: var(--muted-text, #9aa4b2);
  cursor: pointer;
  padding: 6px;
  border-radius: 8px;
  transition: color 0.15s ease;
  z-index: 2;
}

.close-btn:hover {
  color: #fff;
}

.cert-header {
  position: relative;
  text-align: center;
  margin-bottom: 1.25rem;
  z-index: 1;
}

.cert-badge-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  padding: 4px 14px;
  border-radius: 999px;
  margin-bottom: 0.5rem;
  background: rgba(212, 175, 55, 0.15);
  border: 1px solid rgba(212, 175, 55, 0.4);
  color: #d4af37;
}

.cert-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.6rem;
  font-weight: 800;
  margin: 0 0 0.25rem;
}

.cert-subtitle {
  font-size: 0.85rem;
  color: var(--muted-text, #9aa4b2);
  margin: 0;
}

.cert-name-customization {
  margin-bottom: 1.25rem;
  position: relative;
  z-index: 1;
}

.cert-name-input-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.cert-name-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  font-weight: 700;
  color: #d4af37;
  letter-spacing: 0.02em;
}

.cert-name-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.cert-name-input {
  width: 100%;
  background: #12141a;
  border: 1px solid var(--outline-variant, #2e3644);
  border-radius: 10px;
  padding: 0.6rem 4rem 0.6rem 0.85rem;
  font-size: 0.95rem;
  font-weight: 600;
  color: #ffffff;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.cert-name-input:focus {
  outline: none;
  border-color: #d4af37;
  box-shadow: 0 0 0 3px rgba(212, 175, 55, 0.18);
}

.cert-name-input::placeholder {
  color: var(--muted-text, #6b7280);
  font-weight: 400;
}

.cert-name-reset-btn {
  position: absolute;
  right: 8px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 6px;
  color: var(--muted-text, #9aa4b2);
  font-size: 0.72rem;
  font-weight: 600;
  padding: 3px 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.cert-name-reset-btn:hover {
  background: rgba(212, 175, 55, 0.2);
  border-color: rgba(212, 175, 55, 0.4);
  color: #d4af37;
}

.cert-preview-frame {
  position: relative;
  background: #101217;
  border-radius: 16px;
  padding: 1.25rem;
  margin-bottom: 1.5rem;
  border: 1px solid var(--outline-variant, #2a313d);
  z-index: 1;
}

.cert-document {
  background: #151821;
  border: 2px solid #d4af37;
  border-radius: 12px;
  padding: 1.5rem 1.75rem;
  position: relative;
  overflow: hidden;
}

.cert-inner-border {
  border: 1px dashed rgba(212, 175, 55, 0.4);
  border-radius: 8px;
  padding: 1.25rem;
  position: relative;
}

.cert-watermark {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 4.5rem;
  font-weight: 900;
  color: rgba(255, 255, 255, 0.02);
  letter-spacing: 0.2em;
  pointer-events: none;
  user-select: none;
}

.doc-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
}

.brand-name {
  font-size: 1.1rem;
  font-weight: 800;
  color: #fff;
  display: block;
}

.brand-tag {
  font-size: 0.68rem;
  color: var(--muted-text, #8f9bb3);
}

.doc-seal {
  display: flex;
  flex-direction: column;
  align-items: center;
  color: #d4af37;
  font-size: 0.65rem;
  font-weight: 800;
  gap: 2px;
}

.doc-body {
  text-align: center;
  margin-bottom: 1.5rem;
}

.doc-present {
  font-size: 0.8rem;
  color: var(--muted-text, #8f9bb3);
  margin: 0 0 0.25rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.doc-user-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.4rem;
  font-weight: 800;
  color: #d4af37;
  margin: 0 0 0.25rem;
}

.doc-subtext {
  font-size: 0.8rem;
  color: var(--muted-text, #8f9bb3);
  margin: 0 0 0.4rem;
}

.doc-notebook-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
  margin: 0;
}

.doc-metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  margin-bottom: 1.5rem;
}

.metric-box {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 0.6rem 0.4rem;
  text-align: center;
}

.metric-val {
  display: block;
  font-size: 1.05rem;
  font-weight: 800;
  color: #f0f3fa;
}

.metric-lbl {
  font-size: 0.65rem;
  color: var(--muted-text, #8f9bb3);
  font-weight: 600;
}

.doc-footer {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  font-size: 0.72rem;
}

.footer-lbl {
  display: block;
  color: var(--muted-text, #8f9bb3);
  font-size: 0.65rem;
  margin-bottom: 2px;
}

.footer-val {
  font-weight: 700;
  color: #f0f3fa;
}

.signature-line {
  width: 140px;
  height: 1px;
  background: #3e4756;
  margin-bottom: 4px;
}

.cert-actions {
  display: flex;
  gap: 12px;
  position: relative;
  z-index: 1;
}

.export-btn {
  flex: 1;
  padding: 0.75rem 1rem;
  border-radius: 12px;
  font-weight: 700;
  font-size: 0.9rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  transition: transform 0.15s ease, background-color 0.15s ease;
}

.export-btn.primary {
  background: #d4af37;
  color: #12141a;
}

.export-btn.primary:hover {
  background: #e5be42;
  transform: translateY(-1px);
}

.export-btn.secondary {
  background: var(--surface-container, #222631);
  color: var(--on-surface, #f0f3fa);
  border: 1px solid var(--outline-variant, #323b49);
}

.export-btn.secondary:hover {
  background: #2a303e;
  transform: translateY(-1px);
}

.export-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
