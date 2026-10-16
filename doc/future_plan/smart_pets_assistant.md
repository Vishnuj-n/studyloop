# Smart Pets (Super Pet Assistant) Specification

## 1. Overview & Objective
Enhance the existing desktop pets (Dog, Cat, Fox) into optional, non-intrusive "Super Pets" that act as lightweight, in-app assistants. 

The feature strictly answers questions regarding **StudyLoop settings, design rationale, keyboard shortcuts, and workflow tips** without querying user books or personal study notes. This keeps the RAG context fast, bounded, and collision-free.

---

## 2. Core Architecture Invariants & Rules

1. **Zero DB Bloat:** No new SQLite tables or schema modifications. No persistent conversation history or autonomous memory.
2. **Ephemeral UI State:** Chat sessions live purely in frontend component state while open and are wiped on dismissal.
3. **Single Knowledge Source:** All answers are grounded strictly in a single, curated `app_manual.md` document (indexing app settings, reasons, shortcuts, and troubleshooting tips).
4. **Guardrail (Fail-Fast / Conditional Availability):**
   - Super Mode is only accessible when the LLM/RAG configuration (API key and embedding setup) is verified.
   - If unconfigured, the pet remains in ambient mode only, with an informative tooltip guiding the user to Settings.

---

## 3. Visual & Persona States

Each pet features a normal ambient mode and a toggleable "Super" state with unique visual accessories and signature dialogue sign-offs:

| Pet | Normal State | Super State Accessory | Signature Sign-off |
| :--- | :--- | :--- | :--- |
| **Dog** 🐶 | Ambient idle | **Superhero Cape / Mask** 🦸 | Ends with `... Bow!` (or `... Woof!`) |
| **Cat** 🐱 | Ambient idle | **Detective Monocle / Hat** 🕵️ | Ends with `... Meow!` |
| **Fox** 🦊 | Ambient idle | **Pilot Goggles / Bandana** 🦊🥽 | Ends with `... Yip!` (or `... Yip-yip!`) |

---

## 4. Interaction & UX Flow

1. **Activation:**
   - User clicks the pet (or presses keyboard shortcut `Ctrl/Cmd + Shift + P`).
   - If AI is configured: Super accessory attaches with a smooth animation and opens a compact chat drawer docked directly above/adjacent to the pet.
   - If AI is not configured: Shows an inline tooltip: *"Configure AI settings to awaken Super Pet powers!"*
2. **Deactivation (Zero Friction):**
   - Clicking `✕`, pressing `Esc`, or clicking outside closes the drawer immediately.
   - The accessory unequips, and the pet smoothly returns to normal ambient idle.

---

## 5. RAG Pipeline & System Prompt

### Document Source: `app_manual.md`
A dedicated markdown document containing:
- **Settings & Rationale:** Why defaults were chosen (e.g., FSRS retention = 0.9, interval weights, batch limits).
- **Shortcuts Cheatsheet:** Navigation, card flipping, ratings, queue hotkeys.
- **Queue Mechanics:** Deterministic priority hierarchy, task lifecycles, and queue operation rules.

### Prompt Construction
```text
System:
You are the user's helpful study companion in StudyLoop.
Your job is to answer questions strictly about StudyLoop settings, keyboard shortcuts, and app tips using the provided manual context.
Keep answers concise, direct, and helpful.

Rule:
- Always end your final response sentence with your signature sound: "{SIGNATURE_SOUND}".
- If the question cannot be answered from the manual, politely explain you only know StudyLoop tips and shortcuts.

Context:
{RETRIEVED_CHUNKS_FROM_APP_MANUAL}

User:
{USER_QUESTION}
```

---

## 6. Implementation Scope & Next Steps
- [ ] Curate and embed `app_manual.md` within the documentation/assets.
- [ ] Implement accessory overlay assets/CSS for Dog, Cat, and Fox.
- [ ] Add the floating chat drawer component attached to the Pet container.
- [ ] Add backend endpoint/handler for querying `app_manual.md` with the pet persona wrapper.
