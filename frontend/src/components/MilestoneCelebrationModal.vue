<template>
  <Teleport to="body">
    <div v-if="visible" class="milestone-backdrop" @click.self="closeModal">
      <div class="milestone-card" :class="[tierClass]">
        <button class="close-btn" type="button" aria-label="Close" @click="closeModal">✕</button>

        <div class="milestone-header">
          <div class="milestone-badge-pill">
            <BaseIcon name="sparkles" size="14" />
            <span>MILESTONE ASCENSION</span>
            <BaseIcon name="sparkles" size="14" />
          </div>
          <h2 class="milestone-title">Prestige Ascended!</h2>
          <p class="milestone-sub">Your study momentum has elevated you to a higher realm of mastery.</p>
        </div>

        <!-- Emblem Showcase -->
        <div class="emblem-showcase">
          <div class="emblem-pedestal">
            <span class="emblem-avatar">{{ emoji }}</span>
          </div>
          <h3 class="title-text">{{ milestoneTitle }}</h3>
          <span class="xp-threshold-badge">Achieved at {{ xpFormatted }} XP</span>
        </div>

        <!-- Perks List -->
        <div class="perks-container">
          <div class="perk-item">
            <div class="perk-icon">
              <BaseIcon name="shield" size="18" />
            </div>
            <div class="perk-text">
              <span class="perk-title">Title Prestige</span>
              <span class="perk-desc">Showcased across your study profile</span>
            </div>
          </div>
          <div class="perk-item">
            <div class="perk-icon">
              <BaseIcon name="gift" size="18" />
            </div>
            <div class="perk-text">
              <span class="perk-title">Vault Drops</span>
              <span class="perk-desc">Higher probability of gold & mythic chests</span>
            </div>
          </div>
        </div>

        <button class="celebrate-btn" type="button" @click="closeModal">
          <span>Claim Honors & Continue</span>
          <BaseIcon name="arrow-right" size="16" />
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { getTitleEmoji, getTierThemeClass } from '../utils/gamification'
import { playChestOpenFanfare } from '../utils/audioJuice'

const props = defineProps({
  title: {
    type: String,
    default: 'The Archivist',
  },
  xp: {
    type: Number,
    default: 9000,
  },
})

const emit = defineEmits(['close'])
const visible = ref(true)

const milestoneTitle = computed(() => props.title || 'The Archivist')
const xpFormatted = computed(() => Number(props.xp || 0).toLocaleString())
const emoji = computed(() => getTitleEmoji(milestoneTitle.value))
const tierClass = computed(() => getTierThemeClass(milestoneTitle.value))

function closeModal() {
  visible.value = false
  emit('close')
}

onMounted(() => {
  try {
    playChestOpenFanfare()
  } catch (err) {
    console.debug('Fanfare audio skipped:', err)
  }
})
</script>

<style scoped>
.milestone-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(10px);
  z-index: 99999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
}

.milestone-card {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 24px;
  width: 100%;
  max-width: 440px;
  padding: 2.25rem 2rem;
  text-align: center;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  color: var(--on-surface);
}

.close-btn {
  position: absolute;
  top: 16px;
  right: 18px;
  background: none;
  border: none;
  color: var(--muted-text);
  font-size: 1.25rem;
  cursor: pointer;
  border-radius: 8px;
  padding: 4px 8px;
  transition: color 0.15s ease;
}

.close-btn:hover {
  color: var(--on-surface);
}

.milestone-badge-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  padding: 4px 12px;
  border-radius: 999px;
  margin-bottom: 0.75rem;
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  border: 1px solid color-mix(in srgb, var(--primary) 35%, transparent);
  color: var(--primary);
}

.milestone-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.65rem;
  font-weight: 800;
  margin: 0 0 0.25rem;
}

.milestone-sub {
  font-size: 0.85rem;
  color: var(--muted-text);
  margin: 0 0 1.5rem;
  line-height: 1.4;
}

.emblem-showcase {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 1.5rem;
}

.emblem-pedestal {
  width: 90px;
  height: 90px;
  border-radius: 24px;
  background: var(--surface-container);
  border: 2px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  margin-bottom: 0.75rem;
}

.emblem-avatar {
  font-size: 3.2rem;
}

.title-text {
  font-family: 'Manrope', sans-serif;
  font-size: 1.35rem;
  font-weight: 800;
  margin: 0;
  color: var(--on-surface);
}

.xp-threshold-badge {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--muted-text);
  margin-top: 2px;
}

.perks-container {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 1rem 1.25rem;
  margin-bottom: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 10px;
  text-align: left;
}

.perk-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.perk-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--surface-container-low);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.1rem;
}

.perk-text {
  display: flex;
  flex-direction: column;
}

.perk-title {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--on-surface);
}

.perk-desc {
  font-size: 0.75rem;
  color: var(--muted-text);
}

.celebrate-btn {
  width: 100%;
  padding: 0.8rem;
  font-weight: 800;
  font-size: 0.95rem;
  border-radius: 12px;
  cursor: pointer;
  border: none;
  background: var(--primary);
  color: var(--on-primary);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--primary) 35%, transparent);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.celebrate-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px color-mix(in srgb, var(--primary) 50%, transparent);
}
</style>
