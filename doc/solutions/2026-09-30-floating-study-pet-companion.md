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

## Companion Economy & Pet Sanctuary

1. **Free Starter Companion (Mochi)**:
   - Mochi the Cat is 100% free and unlocked by default.
   - All Mochi classic and aesthetic skins (Calico, Void Black, Matcha Green, Lavender Dusk) are free.

2. **Unlockable Companions & Paid Skins**:
   - **Buster the Dog**: 1,000 Coins. Comes with Golden Retriever default skin; additional paid skins include Shiba Inu (250 coins) and Midnight Husky (500 coins).
   - **Ignis the Mythic Dragon**: 25,000 Coins. Comes with Ruby Crimson default skin; additional legendary skins include Celestial Gold (5,000 coins) and Abyssal Void (10,000 coins).

3. **Dedicated "Pet Sanctuary" Tab in Rewards**:
   - Located as a dedicated 5th tab on the **Rewards & Milestones** page.
   - Features a live interactive preview stage with behavior animations (`idle`, `blink`, `coffee`, `wiggle`, `cheer`, `sleep`), speech bubble tester, adoption catalog, and skin wardrobe atelier.
   - Purchases securely validate and deduct coins via `UnlockCosmeticItem` backend binding and synchronize with local storage.

---

## How to Add New Pets (e.g., Dog, Owl, Capybara)

Adding new pets is designed to be simple and modular:

1. **Register the pet in `frontend/src/config/pets.js`**:
   ```javascript
   {
     id: 'dog',
     name: 'Buster',
     title: 'Loyal Study Pup',
     description: 'A loyal study puppy who wags his tail when you learn.',
     isFree: false,
     priceCoins: 1000,
     defaultSkin: 'golden',
     skins: [
       { id: 'golden', name: 'Golden Retriever', primaryColor: '#F59E0B', secondaryColor: '#FEF3C7', accentColor: '#D97706', isFree: true, priceCoins: 0 },
       { id: 'husky', name: 'Midnight Husky', primaryColor: '#334155', secondaryColor: '#F1F5F9', accentColor: '#0F172A', isFree: false, priceCoins: 500 }
     ],
     actions: ['idle', 'blink', 'coffee', 'wiggle', 'cheer', 'sleep']
   }
   ```
2. **Add the SVG Avatar in `frontend/src/components/FloatingPet.vue` & `frontend/src/components/PetSanctuaryView.vue`**:
   - Render the corresponding vector paths and ears/tail/wings based on `activePetId` or `selectedPet.id`.
3. **Register item codes in `internal/db/gamification_repo.go`**:
   - Add `pet:<id>` and `skin:<id>:<skin_id>` to `getCosmeticCatalog()` for server-side coin validation.
4. **Zero extra plumbing required**:
   - `usePet.js`, dragging, bounds clamping, sanctuary shop, and persistence work automatically.
