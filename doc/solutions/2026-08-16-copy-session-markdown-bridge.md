# 1-Click "Copy Session" Markdown Bridge in Reader

**Date:** 2026-08-16  
**Module:** Reader (`Reader.vue`, `Reader.spec.js`)

---

## Problem & Context

1. **Studying Technical Literature with Frontier AI**: When reading complex technical literature (e.g. distributed systems, algorithms), students frequently want to export their current reading session excerpt to their preferred AI chat client (ChatGPT Plus, Claude Pro, DeepSeek, Ollama) for freeform deep-dives, ASCII diagrams, and active questioning.
2. **Avoiding In-App LLM Bottlenecks**: Forcing all AI interactions through an in-app reader chat adds backend maintenance, token latency, and confines the user. Providing a 1-click clipboard export bridge respects student agency and existing AI subscriptions.
3. **Clean Markdown Format vs. Rigid Prompts**: Rather than hardcoding long, opinionated prompt templates that conflict with user custom instructions or vary across different learning intents, copying clean, structured Markdown (with book metadata, topic name, page range, and section text) provides the highest flexibility and utility.

---

## Key Solutions Implemented

### 1. Minimal UI & Clipboard Action (`Reader.vue`)
- Added a `📋 Copy Session` button in the stage header (`.stage-head-left`).
- Extracts session bounds dynamically from `useReaderBase` (`startPage` to `endPage`).
- Slices `reader.sections.value` for sections matching the active page range (with fallback to `reader.textContent.value`).
- Formats structured Markdown:
  ```markdown
  # [Book Title]
  ## [Topic Title] (Pages [startPage]–[endPage])

  [Extracted Session Text]
  ```
- Writes directly to `navigator.clipboard.writeText()` with a 2-second in-button confirmation (`Copied to Clipboard! ✓`).

### 2. Automated Integration Testing (`Reader.spec.js`)
- Added a Vitest test verifying the button renders, triggers `navigator.clipboard.writeText()`, and formats the expected Markdown schema with book and topic headers.

---

## Ponytail Design Rationale

- **Zero Backend Changes**: All section content and page boundary metadata are already loaded in memory by `useReaderBase`.
- **Zero Extra Dependencies**: Replaced toast notification libraries with transient in-button text swap.
- **Unconstrained User Agency**: Outputs clean structured Markdown without imposing a rigid prompt, letting users type their own questions or use custom GPT/Claude project instructions.

---

## Verification

- **Frontend Vitest Tests**: `npm test -- src/pages/Reader.spec.js` → **PASS** (3/3 tests passing).
