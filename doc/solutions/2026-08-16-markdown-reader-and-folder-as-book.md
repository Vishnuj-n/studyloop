# Open-Source Markdown/Text Reader & Multi-File Course Ingestion

**Date:** 2026-08-16  
**Module:** Reader UI (`Reader.vue`, `useReaderBase.js`), Notebook Ingestion (`NotebookUpload.vue`, `Notebook.vue`, `upload.go`, `syllabus.go`)

---

## Problem & Architectural Goals

1. **Native Markdown & Note Reader**: Enable reading and studying Markdown (`.md`) and plain-text (`.txt`) documents with full code-block, heading, table, and list rendering without needing a separate complex reader subsystem.
2. **Zero Changes to PDF Reader**: Ensure the existing `vue-pdf-embed` PDF viewer, zoom scaling, page scroll, view filters, and edge controls remain 100% untouched and isolated from markdown changes.
3. **Multi-File / Chapter Ingestion as 1 Unified Book**: Support selecting multiple chapter files or dropping note folders to create 1 unified notebook where each file becomes a discrete chapter, avoiding SQLite write-lock contention, embedding thrashing, and conflicting syllabus modals.
4. **Deterministic Chapter Boundaries**: Prevent subheadings (`##`, `###`, etc.) in revision notes from fragmenting into dozens of tiny micro-chapters by enforcing `# ` (H1) chapter boundaries.
5. **No Browser Security Popups**: Use standard `<input type="file" multiple>` with explicit copy instead of browser-intercepted `webkitdirectory` inputs.

---

## Key Solutions Implemented

### 1. Unified Text/Markdown Viewport (`Reader.vue` & `useReaderBase.js`)
- Added `isMarkdown` computed flag in `useReaderBase.js` for non-PDF notebooks (`.md`, `.txt`).
- When `fileType !== 'pdf'`, `useReaderBase.js` fetches the raw text from `notebookUrl` (with section-content fallback).
- In `Reader.vue`, rendered a sibling `<div v-else-if="reader.isMarkdown.value" class="markdown-viewport" v-html="renderedMarkdown">` using `renderMarkdown()` from `markdown.js`.
- The PDF viewer (`ref="pdfViewportRef"`, zoom, controls) remains 100% untouched and only renders when `isPdf === true`.

### 2. Multi-File Ingestion (`NotebookUpload.vue`)
- Single unified **"Choose Files"** CTA with `<input type="file" multiple accept=".pdf,.md,.txt">`.
- Clear, honest guidance: *"supports multiple pdf/md/text but all be considered as a single notebook and only same files type allowed"*.
- Reuses global `useDialog().confirm()` for confirmation when multiple chapter files are selected.
- Natural-sorts files and compiles them into a single `.md` notebook with `# <ChapterName>` headings, creating 1 document and opening 1 syllabus verification modal.
- Same-type guard blocks mixing PDFs with note files.

### 3. Top-Level H1 Chapter Boundaries (`upload.go` & `syllabus.go`)
- **`upload.go` (`splitMarkdownByHeadings`)**: Updated to only split on top-level `# ` (H1) headings (`strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "##")`). Sub-headings (`##`, `###`, etc.) remain intact inside their chapter.
- **`syllabus.go` (`DraftSyllabusChapters`)**: For Markdown documents, uses extracted H1 sections to deterministically generate chapter drafts without requiring LLM calls, while supporting optional AI cleanup.

---

## Design Tradeoff Rationale

### Multi-File Picker vs. `webkitdirectory` Folder Picker
- **Tradeoff Choice**: Standard multi-file selection (`<input type="file" multiple>`) over `webkitdirectory`.
- **Rationale**: `webkitdirectory` triggers a native Chromium black security dialog warning the user about uploading folder contents. Standard multi-file selection allows users to select chapters (or drag-and-drop folders from Explorer) with zero browser popups, zero custom staging modal complexity, and 100% design consistency.

---

## Verification

- **Go Backend Tests**: `go test ./...` → **PASS** across all packages.
- **Frontend Vitest Tests**: `npm test` → **PASS** (9 test files, 23/23 tests passing).
