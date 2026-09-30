# Floating Study Pet Companion (Mochi)

## Overview & Background
To improve study engagement and provide a delightful, non-intrusive companion experience, StudyLoop includes a desktop floating companion widget (`Mochi` the Cat). 

This document details the architectural decisions, interaction rules, click vs ambient state separation, speech bubble mechanics, and how to add additional companion pets (e.g. Dog, Capybara, Owl).

---

## Architectural Principles & Invariants

1. **Purely Client-Side Ephemeral UI**:
   - The pet is a purely visual floating widget.
   - It does not control study queue progression, mutate database records, or block user interaction with study material.
2. **Local Persistence**:
   - Position coordinates (`{ x, y }`), enabled state (`true/false`), active pet ID, and active skin ID are stored in `localStorage` under `studyloop_pet_settings`.
   - On screen resize, the pet's position is clamped within viewport bounds.
3. **Non-Intrusive Workflow**:
   - The pet must never obscure study cards or force user attention.
   - Speech bubbles automatically dismiss after 4 seconds and do not block underlying interactions.

---

## Interaction Design & State Separation

### 1. Click / Petting vs Ambient Idle Behaviors
- **On Click (Direct Petting Interaction)**:
  - **Action**: Plays a clean, physical squash-and-bounce animation (`cheer` hop or `wiggle` ear twitch).
  - **Clean Feedback**: No heart emoji spam popping up across the screen.
  - **Thought / Speech Bubble**: ~20% chance on tap to display an encouraging study tip bubble if one is not already open.
- **On Idle Timer (Ambient Passive Behaviors)**:
  - Every 12–20 seconds while studying, Mochi spontaneously performs natural desk companion behaviors:
    - **`blink`**: Natural eye blink.
    - **`wiggle`**: Gentle ear twitch.
    - **`coffee`**: Sips from a steaming coffee mug.
    - **`sleep`**: Closed-eye cozy nap.
    - **`idle`**: Subtle breathing movement.

### 2. Speech Bubble Mechanics
- **Content**: Sourced from `PET_MESSAGES` in `frontend/src/config/pets.js` (anti-repeating randomized selection).
- **Triggers**:
  - Occasional user pet clicks.
  - Milestone / study encouragement events.
  - Dev panel manual trigger ("💬 Say").
- **Dismissal**:
  - Auto-dismisses after 4 seconds via timer.
  - Instantly dismissable by clicking directly on the bubble.
  - Does not capture pointer events from background drag operations.

---

## How to Add New Pets (e.g., Dog, Owl, Capybara)

Adding new pets is designed to be simple and modular:

1. **Register the pet in `frontend/src/config/pets.js`**:
   ```javascript
   {
     id: 'dog',
     name: 'Buster',
     description: 'A loyal study puppy who wags his tail when you learn.',
     defaultSkin: 'golden',
     skins: [
       { id: 'golden', name: 'Golden', primaryColor: '#F59E0B', secondaryColor: '#FEF3C7', accentColor: '#D97706', isFree: true },
       { id: 'husky', name: 'Husky', primaryColor: '#475569', secondaryColor: '#F1F5F9', accentColor: '#334155', isFree: false, priceCoins: 150 }
     ],
     actions: ['idle', 'blink', 'bark', 'wiggle', 'cheer', 'sleep']
   }
   ```
2. **Add the SVG Avatar in `frontend/src/components/FloatingPet.vue`**:
   - Wrap the cat SVG with `v-if="petState.activePetId === 'cat'"` and create the corresponding Dog SVG with `v-else-if="petState.activePetId === 'dog'"`.
3. **No extra plumbing required**:
   - `usePet.js`, dragging, bounds clamping, settings modal, skin selection, and persistence work automatically for all registered pets.
