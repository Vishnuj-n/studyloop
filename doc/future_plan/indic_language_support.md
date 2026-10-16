# Indic Language Support Specification

## 1. Overview & Objective
Enable native support for major Indic languages (Hindi, Tamil, Telugu, Kannada, Bengali, Malayalam, Marathi, Gujarati, etc.) across the AI Tutor / StudyLoop learning lifecycle. 

This enables learners to ingest Indic textbooks and lecture materials, generate native-script quizzes and flashcards, interact with Socratic and Reader tutors in their preferred language (or conversational bilingual code-mixing like Hinglish/Tanglish), and review spaced-repetition cards with correct typography and rendering.

---

## 2. Core Architecture Invariants & Rules

1. **Deterministic Processing**: Queue ordering and FSRS intervals remain purely mathematical and language-agnostic; language settings govern content generation and rendering without altering task state machines.
2. **Fail Fast on Missing Encodings/Models**: If an Indic notebook is ingested and requires a multilingual embedding model or font that is missing or unconfigured, fail fast with a descriptive error rather than generating corrupted text or silent garbled fallback vectors.
3. **No Database Schema Bloat**: Notebooks and topics store language metadata (`language_code`, e.g., `hi`, `ta`, `te`, `en-IN`) within existing JSON metadata fields (`meta`) without requiring destructive schema migrations.
4. **Local-First Verification**: Local ONNX embedding inference must verify tokenizer vocabulary support for Brahmic script Unicode blocks (Devanagari, Tamil, Telugu, etc.) before vector indexing.

---

## 3. Technical Architecture & Challenges

### 3.1 Script & Ligature Ingestion (The Brahmic Script Problem)
- **Problem**: Standard text-layer extractors (`pdfcpu`, raw Poppler text dump) frequently break Brahmic conjuncts (halant/virama, matras, pre-base vowels), resulting in broken rendering (e.g., `क + ् + ष` separating instead of forming `क्ष`).
- **Solution**:
  - Integrate UTF-8 normalization (`NFC` form) upon extraction.
  - Pair with PyMuPDF4LLM or Vision OCR fallback (see `pdf_multimodal_image_processing.md`) for scanned Indic documents or PDFs using legacy non-Unicode fonts (e.g., Kruti Dev, Shree Lipi).

### 3.2 Multilingual Tokenization & Embeddings
- **Current Model**: Nomic Embed Text v1.5 (English-centric, 768-d).
- **Indic Upgrade Path**:
  - Support pluggable ONNX multilingual models (e.g., `paraphrase-multilingual-MiniLM-L12-v2` or `bge-m3` INT8 ONNX).
  - Enable cloud embedding endpoints (OpenAI `text-embedding-3-small`, Google Gemini embeddings, or Sarvam AI Indic embeddings) via user-configured BYOK API keys.
  - Hybrid lexical search: Configure SQLite FTS5 with unicode61 tokenizer or Indic trigram tokenization to support partial-word matching on conjugated Indic terms.

### 3.3 Prompt Engineering & Persona Tuning
- **Native & Code-Mixed Generation**:
  - System prompts dynamically inject the target language instruction:
    ```text
    Language Directive:
    - Respond strictly in {TARGET_LANGUAGE} (Script: {SCRIPT_NAME}).
    - For technical and scientific terminology, include the standard English term in parentheses upon first introduction (e.g., "प्रकाश संश्लेषण (Photosynthesis)").
    - Ensure grammatical agreement and natural phrasing; avoid unnatural verbatim machine translation.
    ```
  - Support user-selected "Bilingual / Code-Mixed" mode for regional colloquial study (e.g., conversational Hinglish).

### 3.4 Frontend Rendering & Typography
- **Font Stack**: Ensure frontend CSS loads Google Noto Sans Indic font families (`Noto Sans Devanagari`, `Noto Sans Tamil`, `Noto Sans Telugu`, etc.) with proper system fallbacks to prevent "tofu" glyphs (□) on Windows and Linux.
- **Bi-directional & Complex Text Shaping**: Verify CSS `text-rendering: optimizeLegibility` and standard Unicode ligature rendering inside `Reader.vue`, `Flashcards.vue`, and `Quiz.vue`.

---

## 4. Implementation Scope & Milestones

- [ ] **Phase 1: Ingestion & Typography Validation**
  - Add language code detection on uploaded text/PDFs.
  - Configure frontend font fallbacks for Devanagari, Dravidian, and Eastern Indic scripts.
- [ ] **Phase 2: Multilingual Embedder & RAG Indexing**
  - Add optional multilingual ONNX model bundle or cloud embedding provider support in `internal/embeddings/`.
  - Validate reciprocal rank fusion (RRF) with Indic lexical tokens.
- [ ] **Phase 3: Generation & Evaluation Prompts**
  - Update Quiz, Flashcard, Socratic, and Examiner prompt generators in `internal/study/` to support language directives.
  - Calibrate evaluation prompts for short-answer viva responses in native Indic scripts.
