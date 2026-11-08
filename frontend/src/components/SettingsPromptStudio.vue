<template>
  <div class="prompt-studio" :class="{ 'is-disabled': disabled }">
    <!-- Top Controls: Language & Scope Context -->
    <div class="studio-top-bar">
      <div class="scope-info">
        <span class="scope-eyebrow">Prompt Architecture</span>
        <span class="scope-meta">Configure global system prompts, category presets, and notebook-level overrides</span>
      </div>

      <div class="header-right">
        <!-- Compact Language Picker -->
        <div class="language-pill-box">
          <BaseIcon name="globe" size="14" class="pill-icon" />
          <select
            id="prompt-studio-language"
            v-model="selectedLanguage"
            class="clean-lang-select"
            :disabled="disabled || savingLang"
            aria-label="Study Language"
            @change="handleLanguageChange"
          >
            <option value="english">English (Default)</option>
            <option value="kannada">Kannada (ಕನ್ನಡ)</option>
            <option value="hindi">Hindi (हिन्दी)</option>
            <option value="malayalam">Malayalam (മലയാളം)</option>
            <option value="tamil">Tamil (தமிழ்)</option>
          </select>
        </div>

        <div v-if="selectedLanguage !== 'english'" class="lang-active-pill" :title="`System injects translation and preservation instruction for ${languageDisplayName}`">
          <span class="pulse-dot"></span>
          <span>{{ languageDisplayName }} Active</span>
        </div>
      </div>
    </div>

    <!-- Category Switcher Nav -->
    <nav class="category-nav-bar" role="tablist" aria-label="Task Prompt Categories">
      <button
        v-for="cat in promptCategories"
        :key="cat.id"
        type="button"
        role="tab"
        class="nav-tab-btn"
        :class="{ active: activeCategory === cat.id }"
        :aria-selected="activeCategory === cat.id"
        @click="switchCategory(cat.id)"
      >
        <BaseIcon :name="cat.icon" size="14" class="tab-icon" />
        <span class="tab-label">{{ cat.label }}</span>
        <span v-if="templates[cat.id]?.is_custom" class="tab-status-dot" title="Custom prompt variant active"></span>
      </button>
    </nav>

    <!-- Loading State -->
    <div v-if="loadingTemplates" class="loading-container">
      <div class="spinner-dot"></div>
      <span>Loading Prompt Studio specifications...</span>
    </div>

    <!-- Main Studio Workspace (Split Grid) -->
    <main v-else-if="currentCategoryData" class="studio-layout">
      <!-- Left Column: Variant Manager + Monospace Editor -->
      <section class="studio-main-col">
        <!-- Variant / Preset Bar -->
        <div class="variant-shelf">
          <div class="variant-shelf-header">
            <span class="shelf-label">Prompt Variants &amp; Presets</span>
            <button
              type="button"
              class="add-variant-btn"
              :disabled="disabled"
              @click="openNewVariantModal"
            >
              <BaseIcon name="plus" size="12" />
              <span>New Variant</span>
            </button>
          </div>

          <!-- Horizontal Variant Pills -->
          <div class="variant-pills-row">
            <button
              v-for="v in currentCategoryData.variants || []"
              :key="v.id"
              type="button"
              class="variant-pill"
              :class="{
                'is-selected': selectedVariantId === v.id,
                'is-active-runtime': v.is_active,
                'is-system': v.is_system
              }"
              @click="selectVariant(v)"
            >
              <span v-if="v.is_active" class="active-badge-indicator" title="Global active variant in study sessions">
                <BaseIcon name="check" size="10" />
              </span>
              <span class="variant-name">{{ v.name }}</span>
              <span v-if="v.is_system" class="system-tag">Preset</span>
              <span v-else class="custom-tag">Custom</span>
            </button>
          </div>
        </div>

        <!-- Editor Toolbar & Chips -->
        <div class="editor-subhead">
          <div class="subhead-meta">
            <span class="editing-name">{{ activeVariantName }}</span>
            <span v-if="isEditorDirty" class="dirty-pill">Unsaved</span>
            <span v-else-if="isActiveVariantRuntime" class="runtime-pill">Global Active</span>
          </div>

          <div class="subhead-actions">
            <!-- Activate Variant Button -->
            <button
              v-if="!isActiveVariantRuntime && selectedVariant"
              type="button"
              class="action-btn-secondary"
              :disabled="disabled || activatingVariant"
              title="Set this prompt variant as the active one used by default for this category"
              @click="activateCurrentVariant"
            >
              <BaseIcon name="play" size="12" />
              <span>{{ activatingVariant ? 'Setting...' : 'Set as Global Active' }}</span>
            </button>

            <!-- Delete Variant Button -->
            <button
              v-if="selectedVariant && !selectedVariant.is_system"
              type="button"
              class="action-btn-danger"
              :disabled="disabled || deletingVariant"
              title="Delete this custom variant"
              @click="deleteCurrentVariant"
            >
              <BaseIcon name="trash-2" size="12" />
            </button>
          </div>
        </div>

        <!-- Insertable Placeholders -->
        <div class="token-tray">
          <span class="token-tray-title">Insert Context Variable:</span>
          <div class="token-chips">
            <button
              v-for="ph in currentCategoryData.mandatory_placeholders || []"
              :key="ph"
              type="button"
              class="chip-token mandatory"
              :class="{ missing: !editorText.includes(ph) }"
              :title="editorText.includes(ph) ? `Click to insert ${ph}` : `Required placeholder ${ph} is missing!`"
              @click="insertPlaceholder(ph)"
            >
              <code>{{ ph }}</code>
              <BaseIcon v-if="editorText.includes(ph)" name="check" size="10" />
              <BaseIcon v-else name="alert-triangle" size="10" />
            </button>

            <button
              v-for="ph in currentCategoryData.optional_placeholders || []"
              :key="ph"
              type="button"
              class="chip-token optional"
              :title="`Click to insert optional ${ph}`"
              @click="insertPlaceholder(ph)"
            >
              <code>{{ ph }}</code>
            </button>
          </div>
        </div>

        <!-- Monospace Editor Area -->
        <div class="editor-surface">
          <textarea
            ref="editorTextareaRef"
            v-model="editorText"
            class="studio-textarea"
            placeholder="Author prompt template..."
            spellcheck="false"
            :disabled="disabled || savingVariant"
            @input="validateCurrentEditor"
          ></textarea>
        </div>

        <!-- Feedback Messages -->
        <div v-if="validationError" class="status-banner error-banner">
          <BaseIcon name="alert-triangle" size="14" />
          <span>{{ validationError }}</span>
        </div>

        <div v-if="actionSuccess" class="status-banner success-banner">
          <BaseIcon name="check" size="14" />
          <span>{{ actionSuccess }}</span>
        </div>

        <!-- Bottom Actions -->
        <footer class="editor-footer">
          <div class="footer-left">
            <span class="metric-text">{{ editorText.length }} characters</span>
          </div>

          <div class="footer-right">
            <button
              type="button"
              class="save-pill-btn"
              :disabled="disabled || savingVariant || Boolean(validationError) || !isEditorDirty"
              @click="saveCurrentEditor"
            >
              <BaseIcon v-if="savingVariant" name="loader" size="13" class="spin-icon" />
              <BaseIcon v-else name="check" size="13" />
              <span>{{ savingVariant ? 'Saving...' : 'Save Variant' }}</span>
            </button>
          </div>
        </footer>
      </section>

      <!-- Right Column: Inspector & Notebook Scoping -->
      <aside class="studio-inspector-col">
        <div class="inspector-header">
          <div class="inspector-tabs">
            <button
              type="button"
              class="insp-tab-btn"
              :class="{ active: inspectorTab === 'compiled' }"
              @click="inspectorTab = 'compiled'"
            >
              <BaseIcon name="zap" size="12" />
              <span>LLM Stream Payload</span>
            </button>
            <button
              type="button"
              class="insp-tab-btn"
              :class="{ active: inspectorTab === 'books' }"
              @click="inspectorTab = 'books'"
            >
              <BaseIcon name="book" size="12" />
              <span>Books ({{ booksTabLabel }})</span>
            </button>
            <button
              type="button"
              class="insp-tab-btn"
              :class="{ active: inspectorTab === 'default' }"
              @click="inspectorTab = 'default'"
            >
              <BaseIcon name="file-text" size="12" />
              <span>Base Spec</span>
            </button>
          </div>

          <!-- Quick Copy Prompt Button for external AI tools -->
          <div class="inspector-header-actions">
            <button
              type="button"
              class="copy-prompt-btn"
              :title="copyTooltip"
              @click="copyActivePromptToClipboard"
            >
              <BaseIcon :name="copySuccess ? 'check' : 'copy'" size="12" />
              <span>{{ copySuccess ? 'Copied' : 'Copy for AI' }}</span>
            </button>
          </div>
        </div>

        <!-- Tab 1: Live Compiled Stream Payload -->
        <template v-if="inspectorTab === 'compiled'">
          <div class="inspector-sim-row">
            <label class="sim-checkbox-label">
              <input v-model="useSampleInterpolation" type="checkbox" />
              <span>Simulate Context Variables ({{ useSampleInterpolation ? 'Sample Injected' : 'Raw Tokens' }})</span>
            </label>
          </div>

          <div class="inspector-code-wrap">
            <pre class="code-viewport"><code>{{ compiledPromptPreview }}</code></pre>
          </div>
        </template>

        <!-- Tab 2: Books Scoped & Assigned to this Variant -->
        <template v-else-if="inspectorTab === 'books'">
          <div class="books-assignment-panel">
            <div class="assignment-summary-box">
              <span class="assignment-headline">
                {{ isActiveVariantRuntime ? 'Active Default for Books' : 'Book Overrides for "' + activeVariantName + '"' }}
              </span>
              <p class="assignment-caption">
                {{
                  isActiveVariantRuntime
                    ? 'This is the global active default prompt. All textbooks automatically use this prompt unless assigned a custom override.'
                    : 'Select textbooks that should use this variant instead of the global default. Unselected books keep using the global active default.'
                }}
              </p>
            </div>

            <div v-if="notebooks.length === 0" class="empty-books-state">
              <span>No textbooks found in the active profile. Textbooks uploaded in the active profile will appear here.</span>
            </div>

            <div v-else class="books-scope-list">
              <div
                v-for="nb in notebooks"
                :key="nb.id"
                class="book-assignment-row"
                :class="{
                  'is-assigned': isBookAssignedToCurrentVariant(nb.id),
                  'is-default-active': isActiveVariantRuntime && !isBookOverriddenAnywhere(nb.id)
                }"
              >
                <div class="book-info">
                  <span class="book-title">{{ nb.title }}</span>
                  <span class="book-status-tag" :class="getBookStatusClass(nb.id)">
                    {{ getBookStatusLabel(nb.id) }}
                  </span>
                </div>

                <div class="book-action">
                  <button
                    v-if="!isActiveVariantRuntime"
                    type="button"
                    class="assignment-toggle-btn"
                    :class="{ 'assigned-active': isBookAssignedToCurrentVariant(nb.id) }"
                    :disabled="disabled || assigningBookId === nb.id"
                    @click="toggleBookAssignment(nb.id)"
                  >
                    <BaseIcon :name="isBookAssignedToCurrentVariant(nb.id) ? 'check' : 'plus'" size="11" />
                    <span>{{ isBookAssignedToCurrentVariant(nb.id) ? 'Overriding Default' : 'Override for Book' }}</span>
                  </button>
                  <span v-else-if="!isBookOverriddenAnywhere(nb.id)" class="default-badge">
                    Active by Default
                  </span>
                  <button
                    v-else
                    type="button"
                    class="assignment-toggle-btn reset-btn"
                    :disabled="disabled || assigningBookId === nb.id"
                    title="Remove custom override and revert book to this active default"
                    @click="revertBookToDefault(nb.id)"
                  >
                    <span>Reset to Default</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- Tab 3: System Base Template Reference -->
        <template v-else>
          <div class="inspector-code-wrap">
            <pre class="code-viewport"><code>{{ currentCategoryData.default_template }}</code></pre>
          </div>
        </template>

        <!-- Guardrail Badges Footer -->
        <div class="guardrail-footer">
          <div class="guardrail-item">
            <BaseIcon name="shield" size="12" class="guard-icon" />
            <span class="guard-text">
              <strong>Runtime Enforcements:</strong>
              Syntax guardrails and regional language instructions are dynamically appended at runtime.
            </span>
          </div>
        </div>
      </aside>
    </main>

    <!-- Modal: Create New Variant -->
    <div v-if="showNewVariantModal" class="modal-overlay" @click.self="showNewVariantModal = false">
      <div class="modal-dialog">
        <header class="modal-header">
          <h3 class="modal-title">Create New Prompt Variant</h3>
          <button type="button" class="modal-close-btn" @click="showNewVariantModal = false">
            <BaseIcon name="x" size="14" />
          </button>
        </header>

        <div class="modal-body">
          <label class="modal-label" for="new-variant-name">Variant Name</label>
          <input
            id="new-variant-name"
            v-model="newVariantName"
            type="text"
            class="modal-input"
            placeholder="e.g. Rapid Socratic Recap, Custom Exam 1"
          />

          <label class="modal-label" for="new-variant-base">Start from</label>
          <select id="new-variant-base" v-model="newVariantBaseId" class="modal-select">
            <option value="current">Current Editor Contents</option>
            <option
              v-for="v in currentCategoryData?.variants || []"
              :key="v.id"
              :value="v.id"
            >
              {{ v.name }} ({{ v.is_system ? 'Preset' : 'Custom' }})
            </option>
          </select>
        </div>

        <footer class="modal-footer">
          <button type="button" class="btn-cancel" @click="showNewVariantModal = false">Cancel</button>
          <button
            type="button"
            class="btn-create"
            :disabled="!newVariantName.trim() || creatingVariant"
            @click="confirmCreateVariant"
          >
            {{ creatingVariant ? 'Creating...' : 'Create Variant' }}
          </button>
        </footer>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import BaseIcon from './BaseIcon.vue'
import {
  getPromptStudioTemplates,
  savePromptVariant,
  setActivePromptVariant,
  deletePromptVariant,
  setPromptNotebookAssignment,
  getPreferredLanguage,
  setPreferredLanguage
} from '../services/appApi.js'

const props = defineProps({
  disabled: {
    type: Boolean,
    default: false
  },
  settings: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update-settings'])

const promptCategories = [
  { id: 'notes', label: 'Study Notes', icon: 'book-open' },
  { id: 'quiz', label: 'Quiz Assessment', icon: 'help-circle' },
  { id: 'flashcard', label: 'FSRS Flashcards', icon: 'layers' },
  { id: 'tutor_duck', label: 'Socratic Duck', icon: 'message-circle' },
  { id: 'simplify', label: 'Text Simplifier', icon: 'sparkles' },
  { id: 'reader_qa', label: 'Reader Q&A', icon: 'search' }
]

const activeCategory = ref('notes')
const templates = ref({})
const notebooks = ref([])
const loadingTemplates = ref(true)
const selectedVariantId = ref('')
const editorText = ref('')
const initialLoadedText = ref('')
const selectedLanguage = ref('english')
const savingLang = ref(false)
const savingVariant = ref(false)
const activatingVariant = ref(false)
const deletingVariant = ref(false)
const assigningBookId = ref('')
const actionSuccess = ref('')
const validationError = ref('')
const inspectorTab = ref('compiled')
const useSampleInterpolation = ref(true)
const editorTextareaRef = ref(null)
const copySuccess = ref(false)

// Modal states
const showNewVariantModal = ref(false)
const newVariantName = ref('')
const newVariantBaseId = ref('current')
const creatingVariant = ref(false)

const currentCategoryData = computed(() => {
  return templates.value[activeCategory.value] || null
})

const selectedVariant = computed(() => {
  const vars = currentCategoryData.value?.variants || []
  return vars.find(v => v.id === selectedVariantId.value) || vars[0] || null
})

const activeVariantName = computed(() => {
  return selectedVariant.value?.name || 'Standard Template'
})

const isActiveVariantRuntime = computed(() => {
  return selectedVariant.value?.is_active === true
})

const isEditorDirty = computed(() => {
  return editorText.value !== initialLoadedText.value
})

const copyTooltip = computed(() => {
  return 'Copy full compiled prompt to clipboard to run in Claude, ChatGPT, or external models'
})

const languageDisplayName = computed(() => {
  switch (selectedLanguage.value) {
    case 'kannada': return 'Kannada (ಕನ್ನಡ)'
    case 'hindi': return 'Hindi (हिन्दी)'
    case 'malayalam': return 'Malayalam (മലയാളം)'
    case 'tamil': return 'Tamil (தமிழ்)'
    default: return 'English'
  }
})

const assignedBookCount = computed(() => {
  const assignments = currentCategoryData.value?.assignments || {}
  const targetId = selectedVariant.value?.id
  if (!targetId) return 0
  return Object.values(assignments).filter(vID => vID === targetId).length
})

const overriddenBookCount = computed(() => {
  const assignments = currentCategoryData.value?.assignments || {}
  return Object.keys(assignments).length
})

const booksTabLabel = computed(() => {
  if (isActiveVariantRuntime.value) {
    const unoverridden = notebooks.value.length - overriddenBookCount.value
    return `${unoverridden}/${notebooks.value.length} on Default`
  }
  return `${assignedBookCount.value} Overrides`
})

const sampleContext = {
  '{CONTENT}': 'Operating systems manage computer hardware and software resources. The kernel provides core services including process management, memory allocation, virtual memory paging, and device I/O abstraction.',
  '{TOPIC}': 'Operating Systems Architecture',
  '{NUM_QUESTIONS}': '3',
  '{DIFFICULTY}': 'Undergraduate',
  '{MAX_CARDS}': '4',
  '{QUESTION}': 'How does virtual memory paging prevent external memory fragmentation?'
}

const quizTokenSyntaxFooter = `CRITICAL FORMAT RULES:
Format each question EXACTLY like this:
Q: <question text>
+ <correct answer (4-10 words)>
- <plausible incorrect distractor 1 (4-10 words)>
- <plausible incorrect distractor 2 (4-10 words)>
- <plausible incorrect distractor 3 (4-10 words)>`

const flashcardSchemaWrapper = `CRITICAL: Return ONLY valid JSON.
=== JSON FORMAT ===
{"cards":[{"prompt":"...","answer":"..."}]}`

const languageInstructionMap = {
  kannada: '[LANGUAGE INSTRUCTION] Write the entire response in Kannada; preserve established English technical terms, formulas, and code in parentheses or unchanged.',
  hindi: '[LANGUAGE INSTRUCTION] Write the entire response in Hindi; preserve established English technical terms, formulas, and code in parentheses or unchanged.',
  malayalam: '[LANGUAGE INSTRUCTION] Write the entire response in Malayalam; preserve established English technical terms, formulas, and code in parentheses or unchanged.',
  tamil: '[LANGUAGE INSTRUCTION] Write the entire response in Tamil; preserve established English technical terms, formulas, and code in parentheses or unchanged.'
}

const compiledPromptPreview = computed(() => {
  let text = editorText.value
  if (!text && selectedVariant.value?.template) {
    text = selectedVariant.value.template
  }
  if (!text && currentCategoryData.value?.default_template) {
    text = currentCategoryData.value.default_template
  }
  if (!text) {
    text = ''
  }

  if (useSampleInterpolation.value && text) {
    for (const [k, v] of Object.entries(sampleContext)) {
      text = text.replaceAll(k, v)
    }
  }

  if (activeCategory.value === 'quiz') {
    text = text.trim() + '\n\n' + quizTokenSyntaxFooter
  } else if (activeCategory.value === 'flashcard') {
    text = text.trim() + '\n\n' + flashcardSchemaWrapper
  }

  if (selectedLanguage.value !== 'english' && languageInstructionMap[selectedLanguage.value]) {
    text = text.trim() + '\n\n' + languageInstructionMap[selectedLanguage.value]
  }

  return text
})

function isBookAssignedToCurrentVariant(bookId) {
  const assignments = currentCategoryData.value?.assignments || {}
  return assignments[bookId] === selectedVariant.value?.id
}

function isBookOverriddenAnywhere(bookId) {
  const assignments = currentCategoryData.value?.assignments || {}
  return Boolean(assignments[bookId])
}

function getBookStatusClass(bookId) {
  const assignments = currentCategoryData.value?.assignments || {}
  const assignedID = assignments[bookId]
  if (!assignedID) {
    return 'status-default'
  }
  if (assignedID === selectedVariant.value?.id) {
    return 'status-dedicated'
  }
  return 'status-other'
}

function getBookStatusLabel(bookId) {
  const assignments = currentCategoryData.value?.assignments || {}
  const assignedID = assignments[bookId]
  if (!assignedID) {
    return 'Using Global Default'
  }
  if (assignedID === selectedVariant.value?.id) {
    return 'Overridden for this Variant'
  }
  const otherVar = currentCategoryData.value?.variants?.find(v => v.id === assignedID)
  return `Overridden to: ${otherVar?.name || 'Other Custom'}`
}

async function revertBookToDefault(bookId) {
  assigningBookId.value = bookId
  actionSuccess.value = ''
  try {
    const res = await setPromptNotebookAssignment(bookId, activeCategory.value, '')
    if (res?.error) {
      validationError.value = 'Failed reverting book assignment: ' + res.error
      return
    }
    await loadTemplates()
    actionSuccess.value = 'Book reverted to Global Active default'
  } catch (err) {
    validationError.value = 'Failed reverting book assignment: ' + (err.message || err)
  } finally {
    assigningBookId.value = ''
  }
}

async function toggleBookAssignment(bookId) {
  if (!selectedVariant.value) return
  assigningBookId.value = bookId
  actionSuccess.value = ''
  try {
    const isCurrentlyAssigned = isBookAssignedToCurrentVariant(bookId)
    // If currently assigned, toggle off (pass empty string to revert to global default)
    const newVariantTarget = isCurrentlyAssigned ? '' : selectedVariant.value.id
    const res = await setPromptNotebookAssignment(bookId, activeCategory.value, newVariantTarget)
    if (res?.error) {
      validationError.value = 'Failed updating book assignment: ' + res.error
      return
    }
    await loadTemplates()
    actionSuccess.value = isCurrentlyAssigned
      ? 'Book reverted to Global Active default'
      : `Book dedicated to "${selectedVariant.value.name}"`
  } catch (err) {
    validationError.value = 'Failed updating book assignment: ' + (err.message || err)
  } finally {
    assigningBookId.value = ''
  }
}

async function copyActivePromptToClipboard() {
  const rawPrompt = editorText.value || selectedVariant.value?.template || ''
  if (!rawPrompt) return

  const mandatoryVars = (currentCategoryData.value?.mandatory_placeholders || []).join(', ')
  const optionalVars = (currentCategoryData.value?.optional_placeholders || []).join(', ')

  const aiCoPilotBrief = `You are an expert prompt engineer specializing in educational AI.

I am refining a system prompt for my study application. Below is my current prompt draft, along with critical runtime constraints that CANNOT be changed:

=== SYSTEM PROMPT DRAFT ===
${rawPrompt}
===========================

STRICT SYSTEM CONSTRAINTS (DO NOT VIOLATE):
1. MANDATORY VARIABLES: You must preserve all runtime context placeholders in curly brackets exactly as written: ${mandatoryVars || 'none'}${optionalVars ? ' (Optional supported: ' + optionalVars + ')' : ''}. Do not delete or rename them.
2. SYNTAX RULES: Do not alter or omit any structured response markers or JSON formatting instructions required by the parser.

GOAL:
I want to upgrade, elevate, or personalize this prompt to make it significantly more effective for learning.

BEFORE writing or revising the prompt, please ask me these 3 specific questions so we can tailor it perfectly:
1. "What cognitive/teaching style do you prefer? (e.g., Socratic inquiry, first-principles Feynman breakdown, high-yield bulleted cheat sheet, rigorous academic depth)?"
2. "Who is the learner, and what is their academic level or background (e.g., beginner, high school, university STEM, professional certification)?"
3. "Are there specific shortcomings in your current AI responses that you want to eliminate (e.g., too verbose, too generic, misses edge cases, lacks practical examples)?"

Please stop here and wait for my answers before generating the updated prompt.`

  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(aiCoPilotBrief)
    } else {
      const ta = document.createElement('textarea')
      ta.value = aiCoPilotBrief
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    copySuccess.value = true
    actionSuccess.value = 'AI Prompt Engineering Brief copied! Paste it directly into Claude or ChatGPT.'
    setTimeout(() => {
      copySuccess.value = false
    }, 2500)
  } catch (err) {
    console.error('Failed copying to clipboard:', err)
  }
}

function validateCurrentEditor() {
  actionSuccess.value = ''
  validationError.value = ''
  if (!currentCategoryData.value) return true

  for (const ph of currentCategoryData.value.mandatory_placeholders || []) {
    if (!editorText.value.includes(ph)) {
      validationError.value = `Mandatory variable ${ph} is missing from template.`
      return false
    }
  }
  return true
}

function insertPlaceholder(placeholder) {
  const el = editorTextareaRef.value
  if (!el) {
    editorText.value += (editorText.value ? ' ' : '') + placeholder
    validateCurrentEditor()
    return
  }

  const start = el.selectionStart ?? el.value.length
  const end = el.selectionEnd ?? el.value.length
  const current = editorText.value
  editorText.value = current.substring(0, start) + placeholder + current.substring(end)
  validateCurrentEditor()

  nextTick(() => {
    el.focus()
    const nextCursor = start + placeholder.length
    el.setSelectionRange(nextCursor, nextCursor)
  })
}

function selectVariant(v) {
  if (!v) return
  selectedVariantId.value = v.id
  editorText.value = v.template || ''
  initialLoadedText.value = v.template || ''
  actionSuccess.value = ''
  validationError.value = ''
  validateCurrentEditor()
}

function switchCategory(catId) {
  activeCategory.value = catId
  actionSuccess.value = ''
  validationError.value = ''
  const cat = templates.value[catId]
  if (cat) {
    if (cat.variants?.length) {
      const activeV = cat.variants.find(v => v.id === cat.active_variant_id) || cat.variants[0]
      selectVariant(activeV)
    } else if (cat.default_template) {
      editorText.value = cat.default_template
      initialLoadedText.value = cat.default_template
    }
  }
}

async function loadTemplates() {
  loadingTemplates.value = true
  try {
    const res = await getPromptStudioTemplates()
    if (res && res.templates) {
      templates.value = res.templates
      if (res.notebooks) {
        notebooks.value = res.notebooks
      }
      const cat = templates.value[activeCategory.value]
      if (cat) {
        if (cat.variants?.length) {
          const activeV = cat.variants.find(v => v.id === selectedVariantId.value) ||
            cat.variants.find(v => v.id === cat.active_variant_id) ||
            cat.variants[0]
          selectVariant(activeV)
        } else if (cat.default_template) {
          editorText.value = cat.default_template
          initialLoadedText.value = cat.default_template
        }
      }
    }
  } catch (err) {
    validationError.value = 'Failed to load templates: ' + (err.message || err)
  } finally {
    loadingTemplates.value = false
  }
}

async function loadLanguage() {
  try {
    const res = await getPreferredLanguage()
    if (res && res.preferred_language) {
      selectedLanguage.value = res.preferred_language
    }
  } catch (err) {
    console.error('Failed reading preferred language:', err)
  }
}

async function handleLanguageChange() {
  savingLang.value = true
  try {
    const res = await setPreferredLanguage(selectedLanguage.value)
    if (res?.error) {
      validationError.value = 'Failed saving language: ' + res.error
      return
    }
    const savedLang = res?.preferred_language || selectedLanguage.value
    emit('update-settings', { preferred_language: savedLang })
    actionSuccess.value = 'Study language updated to ' + languageDisplayName.value
  } catch (err) {
    validationError.value = 'Failed saving language: ' + (err.message || err)
  } finally {
    savingLang.value = false
  }
}

async function saveCurrentEditor() {
  if (!validateCurrentEditor()) return
  if (!selectedVariant.value) return

  savingVariant.value = true
  actionSuccess.value = ''
  try {
    const variantToSave = {
      ...selectedVariant.value,
      template: editorText.value
    }

    if (variantToSave.is_system) {
      variantToSave.id = ''
      variantToSave.name = `${variantToSave.name} (Custom)`
      variantToSave.is_system = false
      variantToSave.is_active = true
    }

    const res = await savePromptVariant(variantToSave)
    if (res?.error) {
      validationError.value = 'Failed saving variant: ' + res.error
      return
    }
    if (res && res.variant) {
      await loadTemplates()
      selectedVariantId.value = res.variant.id
      initialLoadedText.value = res.variant.template
      actionSuccess.value = `Saved variant: ${res.variant.name}`
    }
  } catch (err) {
    validationError.value = 'Failed saving variant: ' + (err.message || err)
  } finally {
    savingVariant.value = false
  }
}

async function activateCurrentVariant() {
  if (!selectedVariant.value) return
  activatingVariant.value = true
  actionSuccess.value = ''
  try {
    const res = await setActivePromptVariant(activeCategory.value, selectedVariant.value.id)
    if (res?.error) {
      validationError.value = 'Failed activating variant: ' + res.error
      return
    }
    await loadTemplates()
    actionSuccess.value = `Activated "${selectedVariant.value.name}" globally for ${promptCategories.find(c => c.id === activeCategory.value)?.label}`
  } catch (err) {
    validationError.value = 'Failed activating variant: ' + (err.message || err)
  } finally {
    activatingVariant.value = false
  }
}

async function deleteCurrentVariant() {
  if (!selectedVariant.value || selectedVariant.value.is_system) return
  if (!window.confirm(`Delete variant "${selectedVariant.value.name}"?`)) return

  deletingVariant.value = true
  actionSuccess.value = ''
  try {
    const res = await deletePromptVariant(selectedVariant.value.id)
    if (res?.error) {
      validationError.value = 'Failed deleting variant: ' + res.error
      return
    }
    await loadTemplates()
    actionSuccess.value = 'Variant deleted successfully'
  } catch (err) {
    validationError.value = 'Failed deleting variant: ' + (err.message || err)
  } finally {
    deletingVariant.value = false
  }
}

function openNewVariantModal() {
  newVariantName.value = ''
  newVariantBaseId.value = selectedVariantId.value || 'current'
  showNewVariantModal.value = true
}

async function confirmCreateVariant() {
  if (!newVariantName.value.trim()) return
  creatingVariant.value = true
  try {
    let baseTemplate = editorText.value
    if (newVariantBaseId.value !== 'current') {
      const bVar = currentCategoryData.value?.variants?.find(v => v.id === newVariantBaseId.value)
      if (bVar) baseTemplate = bVar.template
    }

    const newV = {
      id: '',
      category: activeCategory.value,
      name: newVariantName.value.trim(),
      template: baseTemplate,
      is_active: true,
      is_system: false
    }

    const res = await savePromptVariant(newV)
    if (res?.error) {
      validationError.value = 'Failed creating variant: ' + res.error
      return
    }
    if (res && res.variant) {
      showNewVariantModal.value = false
      await loadTemplates()
      selectedVariantId.value = res.variant.id
      initialLoadedText.value = res.variant.template
      actionSuccess.value = `Created variant: ${res.variant.name}`
    }
  } catch (err) {
    alert('Failed creating variant: ' + (err.message || err))
  } finally {
    creatingVariant.value = false
  }
}

onMounted(() => {
  loadTemplates()
  loadLanguage()
})
</script>

<style scoped>
.prompt-studio {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  color: var(--on-surface);
  font-family: inherit;
}

.prompt-studio.is-disabled {
  opacity: 0.6;
  pointer-events: none;
}

/* Studio Top Bar */
.studio-top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--outline-variant);
}

.scope-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.scope-eyebrow {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted-text);
}

.scope-meta {
  font-size: 0.8rem;
  color: var(--on-surface-variant);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.language-pill-box {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 0.3rem 0.6rem;
  transition: all 0.15s ease;
}

.language-pill-box:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 20%, transparent);
}

.pill-icon {
  color: var(--muted-text);
  flex-shrink: 0;
}

.clean-lang-select {
  background: transparent;
  border: none;
  color: var(--on-surface);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  outline: none;
  font-family: inherit;
}

.clean-lang-select option {
  background: var(--surface-container);
  color: var(--on-surface);
}

.lang-active-pill {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 0.65rem;
  border-radius: 999px;
  background: color-mix(in srgb, #10b981 15%, transparent);
  color: var(--green-high-contrast, #059669);
  border: 1px solid var(--outline-variant);
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
}

/* Category Nav Bar */
.category-nav-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  background: var(--surface-container);
  padding: 0.35rem;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
}

.nav-tab-btn {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.45rem 0.85rem;
  background: transparent;
  border: none;
  border-radius: 7px;
  color: var(--muted-text);
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.nav-tab-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container-high, color-mix(in srgb, var(--on-surface) 5%, transparent));
}

.nav-tab-btn.active {
  color: var(--on-primary);
  background: var(--primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
}

.tab-icon {
  flex-shrink: 0;
}

.tab-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
}

/* Loading state */
.loading-container {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 3rem 1rem;
  color: var(--muted-text);
  font-size: 0.88rem;
  background: var(--surface-container-low);
  border-radius: 12px;
  border: 1px dashed var(--outline-variant);
}

.spinner-dot {
  width: 18px;
  height: 18px;
  border: 2px solid var(--outline-variant);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Studio Layout (Split Columns) */
.studio-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(0, 1.05fr);
  gap: 1.25rem;
  min-height: 580px;
}

@media (max-width: 1024px) {
  .studio-layout {
    grid-template-columns: 1fr;
  }
}

.studio-main-col,
.studio-inspector-col {
  display: flex;
  flex-direction: column;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.04);
}

/* Variant Shelf */
.variant-shelf {
  padding: 0.75rem 1rem;
  background: var(--surface-container-low);
  border-bottom: 1px solid var(--outline-variant);
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.variant-shelf-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.shelf-label {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted-text);
}

.add-variant-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: var(--surface-container);
  border: 1px dashed var(--outline-variant);
  color: var(--on-surface-variant);
  border-radius: 6px;
  padding: 0.25rem 0.6rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.add-variant-btn:hover {
  color: var(--primary);
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, transparent);
}

.variant-pills-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.variant-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 0.3rem 0.65rem;
  font-size: 0.76rem;
  font-weight: 500;
  color: var(--on-surface-variant);
  cursor: pointer;
  transition: all 0.15s ease;
}

.variant-pill:hover {
  background: var(--surface-container-high, color-mix(in srgb, var(--on-surface) 6%, transparent));
  color: var(--on-surface);
}

.variant-pill.is-selected {
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
  color: var(--primary);
  font-weight: 600;
}

.variant-pill.is-active-runtime {
  border-left: 3px solid #10b981;
}

.active-badge-indicator {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #10b981;
}

.system-tag,
.custom-tag {
  font-size: 0.65rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  padding: 0.08rem 0.35rem;
  border-radius: 4px;
}

.system-tag {
  background: var(--surface-container-low);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.custom-tag {
  background: color-mix(in srgb, #f59e0b 14%, transparent);
  color: var(--amber-high-contrast, #b45309);
  border: 1px solid var(--outline-variant);
}

/* Editor Subhead */
.editor-subhead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.65rem 1rem;
  background: var(--surface-container-low);
  border-bottom: 1px solid var(--outline-variant);
}

.subhead-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.editing-name {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--on-surface);
}

.dirty-pill {
  font-size: 0.68rem;
  font-weight: 600;
  padding: 0.15rem 0.45rem;
  background: color-mix(in srgb, #f59e0b 15%, transparent);
  color: var(--amber-high-contrast, #b45309);
  border-radius: 4px;
  border: 1px solid var(--outline-variant);
}

.runtime-pill {
  font-size: 0.68rem;
  font-weight: 600;
  padding: 0.15rem 0.45rem;
  background: color-mix(in srgb, #10b981 15%, transparent);
  color: var(--green-high-contrast, #059669);
  border-radius: 4px;
  border: 1px solid var(--outline-variant);
}

.subhead-actions {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.action-btn-secondary {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  border-radius: 6px;
  padding: 0.3rem 0.65rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn-secondary:hover {
  background: var(--surface-container-high, color-mix(in srgb, var(--on-surface) 6%, transparent));
  border-color: var(--primary);
  color: var(--primary);
}

.action-btn-danger {
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, #ef4444 10%, transparent);
  border: 1px solid var(--outline-variant);
  color: var(--red-high-contrast, #dc2626);
  border-radius: 6px;
  padding: 0.3rem 0.5rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn-danger:hover {
  background: color-mix(in srgb, #ef4444 20%, transparent);
}

/* Token Tray */
.token-tray {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.5rem 1rem;
  background: var(--surface-container);
  border-bottom: 1px solid var(--outline-variant);
}

.token-tray-title {
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--muted-text);
  white-space: nowrap;
}

.token-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.chip-token {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 5px;
  padding: 0.2rem 0.45rem;
  font-size: 0.72rem;
  font-weight: 500;
  color: var(--on-surface-variant);
  cursor: pointer;
  transition: all 0.15s ease;
}

.chip-token:hover {
  border-color: var(--primary);
  color: var(--primary);
}

.chip-token.mandatory {
  border-color: color-mix(in srgb, var(--primary) 35%, transparent);
  color: var(--primary);
}

.chip-token.missing {
  border-color: color-mix(in srgb, #ef4444 40%, transparent);
  color: var(--red-high-contrast, #dc2626);
  background: color-mix(in srgb, #ef4444 8%, transparent);
}

/* Editor Surface */
.editor-surface {
  flex: 1;
  padding: 0.75rem 1rem;
  display: flex;
  flex-direction: column;
  background: var(--surface-container-lowest);
}

.studio-textarea {
  width: 100%;
  flex: 1;
  min-height: 380px;
  background: transparent;
  border: none;
  outline: none;
  resize: vertical;
  color: var(--on-surface);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.85rem;
  line-height: 1.6;
}

/* Status Banners */
.status-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0.5rem 1rem;
  padding: 0.45rem 0.85rem;
  border-radius: 8px;
  font-size: 0.78rem;
  font-weight: 500;
}

.error-banner {
  background: color-mix(in srgb, #ef4444 12%, transparent);
  color: var(--red-high-contrast, #dc2626);
  border: 1px solid var(--outline-variant);
}

.success-banner {
  background: color-mix(in srgb, #10b981 12%, transparent);
  color: var(--green-high-contrast, #059669);
  border: 1px solid var(--outline-variant);
}

/* Editor Footer */
.editor-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.65rem 1rem;
  background: var(--surface-container-low);
  border-top: 1px solid var(--outline-variant);
}

.metric-text {
  font-size: 0.74rem;
  color: var(--muted-text);
  font-weight: 500;
}

.save-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: var(--primary);
  border: none;
  border-radius: 8px;
  padding: 0.4rem 0.95rem;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--on-primary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.save-pill-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.save-pill-btn:not(:disabled):hover {
  filter: brightness(1.08);
}

/* Right Column (Inspector) */
.inspector-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.55rem 0.85rem;
  background: var(--surface-container-low);
  border-bottom: 1px solid var(--outline-variant);
}

.inspector-tabs {
  display: flex;
  gap: 0.3rem;
}

.inspector-header-actions {
  display: flex;
  align-items: center;
}

.copy-prompt-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 0.25rem 0.6rem;
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--primary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.copy-prompt-btn:hover {
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border-color: var(--primary);
}

.insp-tab-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.7rem;
  background: transparent;
  border: none;
  border-radius: 6px;
  font-size: 0.76rem;
  font-weight: 600;
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.insp-tab-btn:hover {
  color: var(--on-surface);
  background: color-mix(in srgb, var(--on-surface) 5%, transparent);
}

.insp-tab-btn.active {
  background: var(--surface-container-highest);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
}

.inspector-sim-row {
  display: flex;
  align-items: center;
  padding: 0.45rem 0.95rem;
  background: var(--surface-container);
  border-bottom: 1px solid var(--outline-variant);
}

.sim-checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.76rem;
  color: var(--on-surface-variant);
  cursor: pointer;
}

.inspector-code-wrap {
  flex: 1;
  padding: 0.85rem;
  overflow: auto;
  background: var(--surface-container-lowest);
}

.code-viewport {
  margin: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.8rem;
  line-height: 1.6;
  color: var(--on-surface);
  white-space: pre-wrap;
  word-break: break-word;
}

/* Books Assignment Panel */
.books-assignment-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0.85rem;
  background: var(--surface-container-lowest);
  overflow-y: auto;
  gap: 0.85rem;
}

.assignment-summary-box {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 0.65rem 0.85rem;
}

.assignment-headline {
  display: block;
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--on-surface);
  margin-bottom: 0.2rem;
}

.assignment-caption {
  margin: 0;
  font-size: 0.73rem;
  color: var(--muted-text);
  line-height: 1.4;
}

.empty-books-state {
  padding: 1.5rem;
  text-align: center;
  font-size: 0.78rem;
  color: var(--muted-text);
}

.books-scope-list {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.book-assignment-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.55rem 0.85rem;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  transition: all 0.15s ease;
}

.book-assignment-row:hover {
  background: var(--surface-container);
}

.book-assignment-row.is-assigned {
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
  background: color-mix(in srgb, var(--primary) 8%, transparent);
}

.book-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
}

.book-title {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.book-status-tag {
  font-size: 0.7rem;
  color: var(--muted-text);
}

.assignment-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 0.3rem 0.65rem;
  font-size: 0.74rem;
  font-weight: 600;
  color: var(--on-surface-variant);
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.assignment-toggle-btn:hover {
  background: var(--surface-container-high, color-mix(in srgb, var(--on-surface) 6%, transparent));
  color: var(--on-surface);
}

.assignment-toggle-btn.assigned-active {
  background: color-mix(in srgb, #10b981 15%, transparent);
  border-color: color-mix(in srgb, #10b981 35%, transparent);
  color: var(--green-high-contrast, #059669);
}

.default-badge {
  font-size: 0.74rem;
  font-weight: 500;
  color: var(--muted-text);
  white-space: nowrap;
  padding: 0.3rem 0.65rem;
}

.guardrail-footer {
  padding: 0.65rem 0.85rem;
  background: var(--surface-container-low);
  border-top: 1px solid var(--outline-variant);
}

.guardrail-item {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
}

.guard-icon {
  margin-top: 2px;
  color: var(--primary);
  flex-shrink: 0;
}

.guard-text {
  font-size: 0.74rem;
  color: var(--muted-text);
  line-height: 1.4;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.65);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-dialog {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  width: 90%;
  max-width: 440px;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.12);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.85rem 1rem;
  border-bottom: 1px solid var(--outline-variant);
}

.modal-title {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--on-surface);
}

.modal-close-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border-radius: 4px;
}

.modal-close-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container);
}

.modal-body {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.modal-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--muted-text);
}

.modal-input,
.modal-select {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 0.5rem 0.75rem;
  color: var(--on-surface);
  font-size: 0.82rem;
  outline: none;
  font-family: inherit;
}

.modal-input:focus,
.modal-select:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 20%, transparent);
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 0.85rem 1rem;
  border-top: 1px solid var(--outline-variant);
}

.btn-cancel {
  background: transparent;
  border: none;
  color: var(--muted-text);
  padding: 0.4rem 0.8rem;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  border-radius: 6px;
}

.btn-cancel:hover {
  color: var(--on-surface);
  background: var(--surface-container);
}

.btn-create {
  background: var(--primary);
  border: none;
  border-radius: 6px;
  color: var(--on-primary);
  padding: 0.4rem 0.85rem;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-create:hover:not(:disabled) {
  filter: brightness(1.08);
}

.btn-create:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
