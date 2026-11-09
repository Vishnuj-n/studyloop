# Unified Prompt Architecture: Multi-Variant Management, Profile Scoping & Book Overrides

**Date**: 2026-11-08  
**Status**: Implemented & Verified  
**Scope**: Go Backend (`internal/study`, `internal/db`, `internal/app`), Frontend Vue (`SettingsPromptStudio.vue`, `appApi.js`), SQLite Database Schema

---

## 1. Problem Statement

Previously, LLM prompt templates across the study system (Study Notes, Quiz Assessment, FSRS Flashcards, Socratic Duck, Text Simplifier, and Reader Q&A) suffered from several limitations:
1. **Hardcoded or Fragmented Prompts**: Prompt templates were scattered across service files (`study_notes_service.go`, `quiz_sync.go`, `flashcard.go`, `socratic.go`, `simplifier.go`, `reader_ai.go`) with differing injection styles, token counts, and no single source of truth.
2. **Lack of User Customization & Variant Experimentation**: Learners could not tailor prompt behavior (e.g., Feynman intuitive explanations vs. high-yield concise summaries vs. rigorous undergraduate proofs) or switch styles for different subjects.
3. **Absence of Book-Level Scoping**: Different subjects demand different prompt styles (e.g., a *Kannada Language* textbook benefits from step-by-step translation intuition, whereas an *Operating Systems* textbook requires rigorous technical definitions). Users had no way to assign custom variants to specific textbooks without globally changing all other subjects.
4. **Fragile AI Prompt Personalization**: When users attempted to copy system prompts into external LLMs (Claude, ChatGPT) for refinement, external models frequently altered critical runtime variables (`{CONTENT}`, `{TOPIC}`, `{NUM_QUESTIONS}`) or broke parser formatting rules (e.g., Quiz `Q: / + / -` syntax), causing silent runtime crashes.

---

## 2. Architecture Overview & Core Decisions

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Prompt Resolution Hierarchy                     │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ 1. Book-Specific Custom Override (Highest Priority)                    │
│    - Table: prompt_notebook_assignments(notebook_id, category)         │
│    - Evaluated if notebook_id != ""                                    │
│    - Only overrides the specific textbook for that category            │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ (If unassigned)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ 2. Global Active Variant (Default for All Unassigned Books)            │
│    - Table: custom_task_prompts(id, category, is_active = 1)          │
│    - Configured in Settings -> Prompt Studio                           │
│    - Applies seamlessly across all profiles and unassigned books       │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ (If table empty or unseeded)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ 3. Built-In Preset / Pristine System Template                          │
│    - Constants in internal/study/prompt_service.go                     │
│    - Fail-safe fallback ensuring zero broken study tasks               │
└────────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Invariants Enforced:
1. **Zero Orchestrator Bypass**: Task progression (`study_queue`) and session workflows do not orchestrate prompts; `PromptService` acts purely as a deterministic compiler and data provider.
2. **Deterministic Placeholder Validation**: Every template must retain mandatory category placeholders (`{CONTENT}` for Notes, `{CONTENT}` + `{NUM_QUESTIONS}` for Quizzes, `{CONTENT}` + `{MAX_CARDS}` for Flashcards). Missing variables fail fast before saving.
3. **Single-Pass Safe Interpolation**: Context insertion (`renderPlaceholders`) operates in a single pass without recursive token injection to prevent injection attacks or placeholder collisions.
4. **Parser Guardrail Enforcements**: Non-editable syntax footers (Quiz format rules `Q: / + / -` and Flashcard JSON schemas) are dynamically appended at runtime, guaranteeing that user customization cannot break parser deserialization.

---

## 3. Database Schema Updates

### Table: `custom_task_prompts`
Persists system presets and user-created custom prompt variants.

```sql
CREATE TABLE IF NOT EXISTS custom_task_prompts (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    name TEXT NOT NULL,
    template TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_custom_task_prompts_category ON custom_task_prompts(category);
```

### Table: `prompt_notebook_assignments`
Enables textbook-level prompt overrides.

```sql
CREATE TABLE IF NOT EXISTS prompt_notebook_assignments (
    notebook_id TEXT NOT NULL,
    category TEXT NOT NULL,
    variant_id TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (notebook_id, category),
    FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_prompt_notebook_assignments_category ON prompt_notebook_assignments(category);
```

---

## 4. Multi-Variant Library & Preset Styles

Each category ships with out-of-the-box academic presets that users can clone, edit, or set as global active:

| Category | Presets Included |
| :--- | :--- |
| **Study Notes (`notes`)** | • *Executive Summary* (high-yield bulleted takeaway)<br>• *High-Yield Concept Card* (definitions, mechanisms, common traps)<br>• *Minimalist Cheatsheet* (dense formulas and syntax)<br>• *Feynman Intuition* (plain analogies and first-principles breakdowns)<br>• *Comprehensive Deep Dive* (exhaustive textbook-level chapter coverage) |
| **Quiz Assessment (`quiz`)** | • *Standard Diagnostic* (mixed conceptual & application)<br>• *Adversarial Distractors* (focuses on common misconceptions)<br>• *First-Principles Depth* (rigorous scenario analysis) |
| **FSRS Flashcards (`flashcard`)**| • *Atomic Concept* (strict single-fact question/answer pairs)<br>• *Scenario-Based Application* (clinical or engineering problem prompt) |
| **Socratic Duck (`tutor_duck`)** | • *Guiding Inquirer* (pure questions without giving direct answers)<br>• *Step-by-Step Proof Partner* (breaks down complex problems iteratively) |
| **Text Simplifier (`simplify`)** | • *Analogy-First* (translates jargon into everyday metaphors)<br>• *Plain Technical English* (removes fluff while retaining exact definitions) |
| **Reader Q&A (`reader_qa`)** | • *Grounded Source Citation* (strictly answers using textbook text with page references) |

---

## 5. Profile Scoping & Book Overrides

### Profile Isolation Semantics:
- **Global Tooling, Profile-Scoped Books**: Prompt variants are stored globally (a user's custom prompt is available across all study profiles without having to recreate it).
- **Active Profile Textbooks**: When opening Prompt Studio, the **Books tab** queries books strictly for the active study profile (`repo.GetNotebooks("", activeProfileID)`). Textbooks from other profiles are not intermingled.
- **Inheritance vs. Overrides**:
  - Every book by default has **no override** (`Inheriting Global Default`).
  - Setting a variant as **Global Active** immediately applies to all books across all profiles that do not have an explicit override.
  - Clicking **"Override for Book"** locks that specific book to a dedicated variant (e.g., dedicating a Kannada grammar book to the Feynman prompt).
  - Explicit book overrides survive global active switches: if the user later changes the global active default, overridden books stay anchored to their custom selection.

---

## 6. Co-Pilot "Copy for AI" (Prompt Engineering Meta-Prompt)

To eliminate broken prompt customizations when working with external models (Claude, ChatGPT), the **Copy for AI** button generates a complete prompt engineering brief:

```text
You are an expert prompt engineer specializing in educational AI.

I am refining a system prompt for my study application. Below is my current prompt draft, along with critical runtime constraints that CANNOT be changed:

=== SYSTEM PROMPT DRAFT ===
{PROMPT_CONTENT}
===========================

STRICT SYSTEM CONSTRAINTS (DO NOT VIOLATE):
1. MANDATORY VARIABLES: You must preserve all runtime context placeholders in curly brackets (e.g., {CONTENT}, {TOPIC}, {NUM_QUESTIONS}) exactly as spelled. Do not delete or rename them.
2. SYNTAX RULES: Do not alter any structured response markers or JSON formatting instructions required by the parser.

GOAL:
I want to upgrade, elevate, or personalize this prompt to make it significantly more effective for learning.

BEFORE writing or revising the prompt, please ask me these 3 specific questions so we can tailor it perfectly:
1. "What cognitive/teaching style do you prefer? (e.g., Socratic inquiry, first-principles Feynman breakdown, high-yield bulleted cheat sheet, rigorous academic depth)?"
2. "Who is the learner, and what is their academic level or background (e.g., beginner, high school, university STEM, professional certification)?"
3. "Are there specific shortcomings in your current AI responses that you want to eliminate (e.g., too verbose, too generic, misses edge cases, lacks practical examples)?"

Please stop here and wait for my answers before generating the updated prompt.
```

When pasted into an external LLM, the model does not prematurely guess the user's intent. Instead, it asks the 3 diagnostic questions and returns an upgraded prompt guaranteed to preserve `{CONTENT}` and parser syntax.

---

## 7. Verification & Test Coverage

- **Backend Verification**:
  - `go test -short ./internal/...`: All unit tests in `internal/study` and `internal/app` pass.
  - Compilation: `go build -o nul ./internal/...` compiles with zero warnings or package circularities.
- **Frontend Verification**:
  - Vitest test suite (`npm --prefix frontend run test`): 25/25 test files, 113/113 tests passed.
  - Validated template selection, book assignment toggling, and clean styling without emojis.
