# PDF Multimodal Image Processing Specification

## 1. Overview & Objective
Enhance the PDF ingestion pipeline with multimodal vision LLMs to perform intelligent image-to-text extraction, diagram interpretation, mathematical formula transcription, and structured document parsing.

Standard text extraction tools (`pdfcpu`, PyPDF, raw text extractors) fail when encountering:
1. Complex diagrams, flowcharts, circuits, and architectural graphs.
2. Formatted tables with merged cells and visual boundaries.
3. Mathematical formulas and scientific equations rendered as embedded vector graphics or raster bitmaps.
4. Scanned historical textbooks, handwritten margin annotations, or image-only slides.

This specification introduces a hybrid, cost-optimized pipeline that extracts visual elements, feeds page images or isolated crops to vision-capable LLMs (e.g., Gemini 1.5/2.0 Flash, GPT-4o-mini, DeepSeek-VL, Claude 3.5 Haiku/Sonnet, or local Ollama with Qwen2-VL / LLaVA), and converts them into searchable, high-yield Markdown.

---

## 2. Core Architecture Invariants & Rules

1. **Selective Vision Triggering (Zero Waste)**: Do not blindly send every clean, high-density digital text page to expensive vision APIs. Vision extraction is only triggered on pages meeting selective heuristics (e.g., low extractable character count, high image coverage, failed OCR, or user-selected "Deep Vision" ingestion).
2. **Deterministic Output Format**: The vision model must output standard GitHub Flavored Markdown (GFM), including KaTeX math blocks (`$$...$$` or `$...$`), ASCII/Markdown tables (`|---|`), and detailed figure captions inside blockquotes (`> [!NOTE] Diagram Analysis: ...`).
3. **No Hardcoded API Keys or Endpoints**: Vision calls must use the user's configured BYOK providers (OpenAI, Gemini, OpenRouter, Groq, Ollama) configured in Settings. Fail fast with clear error reporting if a vision model is required but the configured model lacks multimodal capabilities.
4. **Local Fallback Safety**: On offline or local-only configurations, vision processing integrates with local vision endpoints (e.g., Ollama running `qwen2-vl` or `llava`) without crashing the desktop app.

---

## 3. Pipeline Architecture

```
                          PDF Ingestion Workflow
                                    │
                                    ▼
                         Page-by-Page Inspection
                                    │
               ┌────────────────────┴────────────────────┐
               │                                         │
        [High Text Density]                     [Low Text / High Image /
      (Clean Digital Textbook)                    Diagrams / Scanned]
               │                                         │
               ▼                                         ▼
      Standard Text Chunker                     Rasterize Page to Image
  (pdfcpu / PyMuPDF text stream)             (PNG @ 150–200 DPI via fitz/pdftoppm)
               │                                         │
               │                                         ▼
               │                                Multimodal LLM Vision API
               │                               (Gemini Flash / GPT-4o / Qwen-VL)
               │                                         │
               │                                         ▼
               │                               Clean Structured Markdown
               │                              (Tables, LaTeX, Figure Analysis)
               │                                         │
               └────────────────────┬────────────────────┘
                                    │
                                    ▼
                         Semantic Chunker (~500w)
                                    │
                                    ▼
                       Local Vector Store (sqlite-vec)
```

---

## 4. Vision Prompt Construction

For rasterized pages or extracted figures sent to the multimodal model, the request uses a standardized system instruction:

```text
System:
You are an expert academic document digitizer and STEM transcriber.
Convert the provided document page image into clean, publication-grade Markdown text.

Rules:
1. Verbatim Text: Extract all body text accurately, preserving headings (#, ##, ###) and lists.
2. Tables: Transcribe tables into Markdown table syntax (|---|). Do not summarize table rows.
3. Math & Science: Convert all mathematical formulas, chemical equations, and notations into LaTeX syntax ($...$ for inline, $$...$$ for display equations).
4. Diagrams & Figures: When an image contains an illustration, chart, or architecture diagram:
   - Provide an explicit blockquote description:
     > **[Diagram: <Figure Title>]**
     > Detailed explanation of flow, components, axes, data trends, and conceptual meaning.
5. Clean Output: Do not include conversational preambles, meta-commentary, or code backticks wrapping the whole response. Return only the extracted markdown.
```

---

## 5. Cost & Performance Optimization

- **Image Resolution Downsampling**: Limit uploaded images to maximum dimension 1568px (adequate for fine typography while minimizing vision token costs).
- **Targeted Cropping**: If PyMuPDF detects a specific bounding box for an image/figure on an otherwise textual page, only the cropped bounding box is dispatched to the vision model, and the resulting explanation is inserted inline into the text stream.
- **Async Background Ingestion**: All vision calls run in the background worker queue with real-time UI progress updates (`"Transcribing Page 14 of 42 (Vision API)..."`).

---

## 6. Implementation Scope & Next Steps

- [ ] **Phase 1: Selective Heuristics & Page Rasterization**
  - Implement page inspection in `internal/notebook/` to flag scanned or diagram-heavy pages.
  - Implement pure-Go or lightweight extension rasterizer to export page PNG buffers.
- [ ] **Phase 2: Vision Client Wrapper**
  - Extend the LLM provider in `internal/study/` to support base64 image attachments in OpenAI-compatible and Gemini multimodal payloads.
  - Add model capability validation (verifying user's selected model supports vision inputs).
- [ ] **Phase 3: Markdown Assembly & RAG Indexing**
  - Splice vision-transcribed Markdown directly into the standard semantic chunker.
  - Verify embedded LaTeX formulas and diagram descriptions are queryable via hybrid RRF search.
