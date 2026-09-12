<template>
  <svg
    class="gamification-icon"
    :width="size"
    :height="size"
    viewBox="0 0 64 64"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <defs>
      <!-- Chest Gradients -->
      <linearGradient id="bronze-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stop-color="#CD7F32" />
        <stop offset="100%" stop-color="#7A4210" />
      </linearGradient>

      <linearGradient id="silver-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stop-color="#E6ECF5" />
        <stop offset="100%" stop-color="#8A99AD" />
      </linearGradient>

      <linearGradient id="gold-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stop-color="#FFDF00" />
        <stop offset="100%" stop-color="#D4AF37" />
      </linearGradient>

      <linearGradient id="mythic-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stop-color="#C084FC" />
        <stop offset="50%" stop-color="#818CF8" />
        <stop offset="100%" stop-color="#4F46E5" />
      </linearGradient>

      <linearGradient id="metal-trim" x1="0%" y1="0%" x2="0%" y2="100%">
        <stop offset="0%" stop-color="#FCD34D" />
        <stop offset="100%" stop-color="#B45309" />
      </linearGradient>
    </defs>

    <!-- CHEST BRONZE -->
    <g v-if="name === 'chest-bronze'">
      <rect x="8" y="24" width="48" height="30" rx="4" fill="url(#bronze-gradient)" stroke="#4A2608" stroke-width="2" />
      <path d="M6 24 C6 16 18 10 32 10 C46 10 58 16 58 24 Z" fill="url(#bronze-gradient)" stroke="#4A2608" stroke-width="2" />
      <rect x="18" y="10" width="6" height="44" fill="url(#metal-trim)" />
      <rect x="40" y="10" width="6" height="44" fill="url(#metal-trim)" />
      <rect x="27" y="26" width="10" height="12" rx="2" fill="#FEF08A" stroke="#B45309" stroke-width="2" />
      <circle cx="32" cy="30" r="2" fill="#78350F" />
    </g>

    <!-- CHEST SILVER -->
    <g v-else-if="name === 'chest-silver'">
      <rect x="8" y="24" width="48" height="30" rx="4" fill="url(#silver-gradient)" stroke="#334155" stroke-width="2" />
      <path d="M6 24 C6 16 18 10 32 10 C46 10 58 16 58 24 Z" fill="url(#silver-gradient)" stroke="#334155" stroke-width="2" />
      <rect x="18" y="10" width="6" height="44" fill="#94A3B8" />
      <rect x="40" y="10" width="6" height="44" fill="#94A3B8" />
      <rect x="27" y="26" width="10" height="12" rx="2" fill="#E2E8F0" stroke="#475569" stroke-width="2" />
      <circle cx="32" cy="30" r="2" fill="#0F172A" />
    </g>

    <!-- CHEST GOLD -->
    <g v-else-if="name === 'chest-gold'">
      <rect x="8" y="24" width="48" height="30" rx="4" fill="url(#gold-gradient)" stroke="#78350F" stroke-width="2" />
      <path d="M6 24 C6 16 18 10 32 10 C46 10 58 16 58 24 Z" fill="url(#gold-gradient)" stroke="#78350F" stroke-width="2" />
      <rect x="18" y="10" width="6" height="44" fill="url(#metal-trim)" />
      <rect x="40" y="10" width="6" height="44" fill="url(#metal-trim)" />
      <rect x="26" y="25" width="12" height="14" rx="3" fill="#FEF08A" stroke="#78350F" stroke-width="2" />
      <polygon points="32,27 36,32 32,37 28,32" fill="#EF4444" />
    </g>

    <!-- CHEST MYTHIC -->
    <g v-else-if="name === 'chest-mythic'">
      <rect x="8" y="24" width="48" height="30" rx="4" fill="url(#mythic-gradient)" stroke="#312E81" stroke-width="2" />
      <path d="M6 24 C6 16 18 10 32 10 C46 10 58 16 58 24 Z" fill="url(#mythic-gradient)" stroke="#312E81" stroke-width="2" />
      <rect x="18" y="10" width="6" height="44" fill="#F472B6" />
      <rect x="40" y="10" width="6" height="44" fill="#F472B6" />
      <rect x="25" y="24" width="14" height="16" rx="3" fill="#FDE047" stroke="#831843" stroke-width="2" />
      <polygon points="32,26 36,33 32,38 28,33" fill="#A855F7" />
    </g>

    <!-- DEFAULT FALLBACK -->
    <g v-else>
      <circle cx="32" cy="32" r="24" fill="#64748B" />
      <text x="32" y="38" font-size="20" fill="#FFFFFF" text-anchor="middle">?</text>
    </g>
  </svg>
</template>

<script setup>
defineProps({
  name: {
    type: String,
    required: true,
  },
  size: {
    type: [Number, String],
    default: 32,
  },
})
</script>

<style scoped>
.gamification-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.2));
  transition: transform 0.2s ease, filter 0.2s ease;
}

.gamification-icon:hover {
  transform: scale(1.06);
  filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.35));
}
</style>
