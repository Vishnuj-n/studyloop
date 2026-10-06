<template>
  <svg
    viewBox="0 0 100 100"
    class="pet-avatar-svg"
    :class="[`pet-${resolvedPetId}`, `action-${action}`]"
    xmlns="http://www.w3.org/2000/svg"
  >
    <!-- Ground Shadow (Optional) -->
    <ellipse
      v-if="showShadow"
      cx="50"
      cy="88"
      rx="28"
      ry="6"
      class="pet-ground-shadow"
    />

    <!-- ==================== CAT (Mochi) ==================== -->
    <g v-if="resolvedPetId === 'cat'" class="pet-model-cat">
      <!-- Tail -->
      <path
        d="M 24 72 C 14 70 10 58 12 50 C 14 54 20 62 26 66"
        :fill="resolvedSkin.accentColor"
        class="pet-tail"
      />

      <!-- Body -->
      <path
        d="M 26 52 C 26 36 36 28 50 28 C 64 28 74 36 74 52 C 74 68 70 82 50 82 C 30 82 26 68 26 52 Z"
        :fill="resolvedSkin.primaryColor"
        class="pet-body"
      />

      <!-- Belly Patch -->
      <ellipse
        cx="50"
        cy="62"
        rx="15"
        ry="14"
        :fill="resolvedSkin.secondaryColor"
        class="pet-belly"
      />

      <!-- Ears -->
      <polygon
        points="30,34 38,14 48,30"
        :fill="resolvedSkin.primaryColor"
        class="pet-ear ear-left"
      />
      <polygon
        points="33,31 38,18 45,28"
        :fill="resolvedSkin.secondaryColor"
        class="pet-ear-inner ear-left-inner"
      />

      <polygon
        points="70,34 62,14 52,30"
        :fill="resolvedSkin.primaryColor"
        class="pet-ear ear-right"
      />
      <polygon
        points="67,31 62,18 55,28"
        :fill="resolvedSkin.secondaryColor"
        class="pet-ear-inner ear-right-inner"
      />

      <!-- Cat Eyes -->
      <g v-if="action === 'sleep'" class="eyes-sleeping">
        <path d="M 38 46 Q 42 50 46 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
        <path d="M 54 46 Q 58 50 62 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
      </g>
      <g v-else-if="action === 'blink'" class="eyes-blink">
        <line x1="38" y1="46" x2="46" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
        <line x1="54" y1="46" x2="62" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
      </g>
      <g v-else class="eyes-open">
        <ellipse cx="42" cy="45" rx="3.5" ry="4.5" fill="#1E293B" />
        <circle cx="43.5" cy="43.5" r="1.5" fill="#FFFFFF" />
        <ellipse cx="58" cy="45" rx="3.5" ry="4.5" fill="#1E293B" />
        <circle cx="59.5" cy="43.5" r="1.5" fill="#FFFFFF" />
      </g>

      <!-- Nose & Mouth -->
      <polygon points="48,51 52,51 50,54" fill="#F43F5E" />
      <path
        d="M 46 55 Q 50 58 50 55 Q 50 58 54 55"
        fill="none"
        stroke="#1E293B"
        stroke-width="1.5"
        stroke-linecap="round"
      />

      <!-- Whiskers -->
      <line x1="28" y1="48" x2="36" y2="50" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
      <line x1="28" y1="54" x2="36" y2="53" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
      <line x1="64" y1="50" x2="72" y2="48" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
      <line x1="64" y1="53" x2="72" y2="54" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
    </g>

    <!-- ==================== DOG (Buster) ==================== -->
    <g v-else-if="resolvedPetId === 'dog'" class="pet-model-dog">
      <!-- Tail -->
      <path
        d="M 74 72 C 84 68 88 56 86 48 C 84 52 78 60 72 66"
        :fill="resolvedSkin.accentColor"
        class="pet-tail"
      />

      <!-- Body -->
      <path
        d="M 26 52 C 26 36 36 28 50 28 C 64 28 74 36 74 52 C 74 68 70 82 50 82 C 30 82 26 68 26 52 Z"
        :fill="resolvedSkin.primaryColor"
        class="pet-body"
      />

      <!-- Floppy Ears -->
      <path
        d="M 30 32 C 20 34 16 46 18 56 C 20 60 25 60 27 54 C 29 48 31 38 33 34 Z"
        :fill="resolvedSkin.accentColor"
        class="pet-ear ear-left"
      />
      <path
        d="M 70 32 C 80 34 84 46 82 56 C 80 60 75 60 73 54 C 71 48 69 38 67 34 Z"
        :fill="resolvedSkin.accentColor"
        class="pet-ear ear-right"
      />

      <!-- Belly & Muzzle Patch -->
      <ellipse
        cx="50"
        cy="63"
        rx="16"
        ry="14"
        :fill="resolvedSkin.secondaryColor"
        class="pet-belly"
      />
      <ellipse
        cx="50"
        cy="52"
        rx="12"
        ry="9"
        :fill="resolvedSkin.secondaryColor"
      />

      <!-- Dog Eyes -->
      <g v-if="action === 'sleep'" class="eyes-sleeping">
        <path d="M 38 45 Q 42 49 46 45" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
        <path d="M 54 45 Q 58 49 62 45" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
      </g>
      <g v-else-if="action === 'blink'" class="eyes-blink">
        <line x1="38" y1="45" x2="46" y2="45" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
        <line x1="54" y1="45" x2="62" y2="45" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
      </g>
      <g v-else class="eyes-open">
        <circle cx="41" cy="44" r="4" fill="#1E293B" />
        <circle cx="42.5" cy="42.5" r="1.5" fill="#FFFFFF" />
        <circle cx="59" cy="44" r="4" fill="#1E293B" />
        <circle cx="60.5" cy="42.5" r="1.5" fill="#FFFFFF" />
      </g>

      <!-- Nose & Mouth -->
      <ellipse cx="50" cy="50" rx="4.5" ry="3.5" fill="#1E293B" />
      <path
        d="M 46 54 Q 50 57 50 54 Q 50 57 54 54"
        fill="none"
        stroke="#1E293B"
        stroke-width="1.6"
        stroke-linecap="round"
      />
    </g>

    <!-- ==================== FOX (Rusty) ==================== -->
    <g v-else-if="resolvedPetId === 'fox'" class="pet-model-fox">
      <!-- Bushy Fox Tail with White Plume Tip -->
      <g class="pet-tail">
        <path
          d="M 28 72 C 14 70 2 56 4 40 C 5 28 16 20 25 22 C 31 24 32 32 28 42 C 25 50 28 60 32 68 Z"
          :fill="resolvedSkin.primaryColor"
        />
        <path
          d="M 12 40 C 8 32 12 24 20 22 C 25 22 26 26 26 30 C 23 31 21 34 20 32 C 18 37 15 36 13 40 Z"
          :fill="resolvedSkin.secondaryColor"
        />
      </g>

      <!-- Body -->
      <path
        d="M 26 52 C 26 36 36 28 50 28 C 64 28 74 36 74 52 C 74 68 70 82 50 82 C 30 82 26 68 26 52 Z"
        :fill="resolvedSkin.primaryColor"
        class="pet-body"
      />

      <!-- Large Pointed Fox Ears with Dark Tips and Fluffy Inners -->
      <g class="pet-ear ear-left">
        <path
          d="M 28 36 C 24 26 26 12 36 8 C 42 16 46 26 48 34 Z"
          :fill="resolvedSkin.primaryColor"
        />
        <path
          d="M 30 18 C 33 12 36 8 36 8 C 38 10 40 14 42 18 C 38 19 34 19 30 18 Z"
          :fill="resolvedSkin.accentColor"
        />
        <path
          d="M 32 32 C 30 24 33 16 36 14 C 39 18 41 24 42 30 C 39 29 36 31 32 32 Z"
          :fill="resolvedSkin.secondaryColor"
          class="pet-ear-inner"
        />
      </g>

      <g class="pet-ear ear-right">
        <path
          d="M 72 36 C 76 26 74 12 64 8 C 58 16 54 26 52 34 Z"
          :fill="resolvedSkin.primaryColor"
        />
        <path
          d="M 70 18 C 67 12 64 8 64 8 C 62 10 60 14 58 18 C 62 19 66 19 70 18 Z"
          :fill="resolvedSkin.accentColor"
        />
        <path
          d="M 68 32 C 70 24 67 16 64 14 C 61 18 59 24 58 30 C 61 29 64 31 68 32 Z"
          :fill="resolvedSkin.secondaryColor"
          class="pet-ear-inner"
        />
      </g>

      <!-- White Cheek / Muzzle Fluff & Belly Mask -->
      <path
        d="M 27 52 C 27 60 34 66 41 64 C 40 70 43 78 50 78 C 57 78 60 70 59 64 C 66 66 73 60 73 52 C 69 56 62 55 57 52 C 53 50 50 54 50 54 C 50 54 47 50 43 52 C 38 55 31 56 27 52 Z"
        :fill="resolvedSkin.secondaryColor"
        class="pet-belly"
      />

      <!-- Fox Eyes (Sharp & Intelligent with Double Highlights) -->
      <g v-if="action === 'sleep'" class="eyes-sleeping">
        <path d="M 37 45 Q 41 49 45 45" fill="none" stroke="#1E293B" stroke-width="2.4" stroke-linecap="round" />
        <path d="M 55 45 Q 59 49 63 45" fill="none" stroke="#1E293B" stroke-width="2.4" stroke-linecap="round" />
      </g>
      <g v-else-if="action === 'blink'" class="eyes-blink">
        <line x1="37" y1="45" x2="45" y2="45" stroke="#1E293B" stroke-width="2.4" stroke-linecap="round" />
        <line x1="55" y1="45" x2="63" y2="45" stroke="#1E293B" stroke-width="2.4" stroke-linecap="round" />
      </g>
      <g v-else class="eyes-open">
        <ellipse cx="41" cy="44" rx="3.8" ry="4.4" fill="#1E293B" />
        <circle cx="42.2" cy="42.5" r="1.5" fill="#FFFFFF" />
        <circle cx="39.8" cy="45.5" r="0.75" fill="#FFFFFF" />
        <ellipse cx="59" cy="44" rx="3.8" ry="4.4" fill="#1E293B" />
        <circle cx="60.2" cy="42.5" r="1.5" fill="#FFFFFF" />
        <circle cx="57.8" cy="45.5" r="0.75" fill="#FFFFFF" />
      </g>

      <!-- Fox Snout & Smile -->
      <ellipse cx="50" cy="51.5" rx="3.2" ry="2.2" fill="#1E293B" />
      <path
        d="M 46.5 54.5 Q 50 57 50 54.5 Q 50 57 53.5 54.5"
        fill="none"
        stroke="#1E293B"
        stroke-width="1.5"
        stroke-linecap="round"
      />
    </g>

    <!-- Coffee Mug (Shown during 'coffee' action) -->
    <g v-if="action === 'coffee'" class="pet-coffee-mug">
      <rect x="44" y="66" width="12" height="11" rx="2" fill="#E2E8F0" stroke="#475569" stroke-width="1" />
      <path d="M 56 68 Q 60 71 56 74" fill="none" stroke="#475569" stroke-width="1" />
      <path d="M 48 63 Q 50 60 50 58" fill="none" stroke="#94A3B8" stroke-width="1" stroke-linecap="round" class="steam-line" />
    </g>

    <!-- Front Paws -->
    <ellipse cx="40" cy="78" rx="5" ry="4" :fill="resolvedSkin.secondaryColor" class="paw paw-left" />
    <ellipse cx="60" cy="78" rx="5" ry="4" :fill="resolvedSkin.secondaryColor" class="paw paw-right" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'
import { PET_REGISTRY } from '../config/pets'

const props = defineProps({
  petId: {
    type: String,
    default: 'cat',
  },
  skin: {
    type: Object,
    default: null,
  },
  skinId: {
    type: String,
    default: null,
  },
  action: {
    type: String,
    default: 'idle',
  },
  showShadow: {
    type: Boolean,
    default: true,
  },
})

const resolvedPetId = computed(() => {
  const found = PET_REGISTRY.find((p) => p.id === props.petId)
  return found ? found.id : 'cat'
})

const resolvedSkin = computed(() => {
  if (props.skin && props.skin.primaryColor) {
    return props.skin
  }
  const pet = PET_REGISTRY.find((p) => p.id === resolvedPetId.value) || PET_REGISTRY[0]
  if (props.skinId) {
    const foundSkin = pet.skins.find((s) => s.id === props.skinId)
    if (foundSkin) return foundSkin
  }
  return pet.skins[0]
})
</script>

<style scoped>
.pet-avatar-svg {
  width: 100%;
  height: 100%;
  overflow: visible;
  display: block;
}

.pet-ground-shadow {
  fill: rgba(0, 0, 0, 0.14);
}

/* Base Animations */
.action-idle .pet-body {
  animation: petBreathe 3s ease-in-out infinite;
}

.action-wiggle .pet-ear.ear-left {
  animation: earTwitchLeft 0.5s ease-in-out infinite;
}

.action-wiggle .pet-ear.ear-right {
  animation: earTwitchRight 0.5s ease-in-out infinite;
}

.action-cheer {
  animation: petBounce 0.4s ease-in-out infinite alternate;
}

.action-cheer .pet-tail {
  animation: tailWag 0.2s ease-in-out infinite;
}

.steam-line {
  animation: steamRise 1.2s ease-out infinite;
}

@keyframes petBreathe {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-2px); }
}

@keyframes petBounce {
  0% { transform: translateY(0); }
  100% { transform: translateY(-8px); }
}

@keyframes earTwitchLeft {
  0%, 100% { transform: rotate(0deg); transform-origin: 38px 25px; }
  50% { transform: rotate(-12deg); transform-origin: 38px 25px; }
}

@keyframes earTwitchRight {
  0%, 100% { transform: rotate(0deg); transform-origin: 62px 25px; }
  50% { transform: rotate(12deg); transform-origin: 62px 25px; }
}

@keyframes tailWag {
  0%, 100% { transform: rotate(0deg); transform-origin: 26px 66px; }
  50% { transform: rotate(-20deg); transform-origin: 26px 66px; }
}

@keyframes steamRise {
  0% { opacity: 0; transform: translateY(0); }
  50% { opacity: 0.8; }
  100% { opacity: 0; transform: translateY(-4px); }
}
</style>
