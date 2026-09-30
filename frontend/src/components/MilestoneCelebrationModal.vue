<template>
  <Teleport to="body">
    <div v-if="visible" class="milestone-backdrop" @click.self="closeModal">
      <div class="milestone-card" :class="[tierClass]">
        <button class="close-btn" type="button" aria-label="Close" @click="closeModal"><BaseIcon name="x" size="18" /></button>

        <!-- Ambient glow / Sunburst ray backdrop -->
        <div class="card-ambient-glow"></div>

        <div class="milestone-header">
          <div class="milestone-badge-pill">
            <BaseIcon name="sparkles" size="14" />
            <span>MILESTONE ASCENSION</span>
            <BaseIcon name="sparkles" size="14" />
          </div>
          <h2 class="milestone-title">Prestige Ascended!</h2>
          <p class="milestone-sub">Your study momentum has elevated you to a higher realm of mastery.</p>
        </div>

        <!-- Emblem Showcase with Halo & Sparkle Aura -->
        <div class="emblem-showcase">
          <div class="emblem-aura-container">
            <div class="sunburst-rays"></div>
            <div class="halo-pulse"></div>
            <div class="emblem-pedestal">
              <span class="emblem-avatar">{{ emoji }}</span>
            </div>
            <!-- Twinkling Sparkles (Vector Icons) -->
            <span class="sparkle-dot sparkle-1"><BaseIcon name="sparkles" size="16" /></span>
            <span class="sparkle-dot sparkle-2"><BaseIcon name="star" size="14" /></span>
            <span class="sparkle-dot sparkle-3"><BaseIcon name="sparkles" size="14" /></span>
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { getTitleEmoji, getTierThemeClass } from '../utils/gamification'
import { playChestOpenFanfare } from '../utils/audioJuice'
import { triggerConfettiCelebration } from '../utils/confettiCelebration'

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
let confettiCleanup = null

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

  try {
    confettiCleanup = triggerConfettiCelebration({
      tier: tierClass.value,
      particleCount: 120,
    })
  } catch (err) {
    console.debug('Confetti celebration skipped:', err)
  }
})

onUnmounted(() => {
  if (confettiCleanup) {
    confettiCleanup()
  }
})
</script>

<style scoped>
.milestone-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(12px);
  z-index: 99999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
  animation: backdropFadeIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes backdropFadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.milestone-card {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 28px;
  width: 100%;
  max-width: 440px;
  padding: 2.25rem 2rem;
  text-align: center;
  box-shadow: 0 30px 60px -12px rgba(0, 0, 0, 0.15);
  color: var(--on-surface);
  overflow: hidden;
  animation: cardScaleUp 0.45s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes cardScaleUp {
  0% {
    opacity: 0;
    transform: scale(0.85) translateY(20px);
  }
  100% {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.card-ambient-glow {
  position: absolute;
  top: -60px;
  left: 50%;
  transform: translateX(-50%);
  width: 280px;
  height: 280px;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--primary) 28%, transparent) 0%, transparent 70%);
  pointer-events: none;
  z-index: 0;
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
  z-index: 2;
}

.close-btn:hover {
  color: var(--on-surface);
}

.milestone-header {
  position: relative;
  z-index: 1;
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
  border: 1px solid var(--outline-variant);
  color: var(--primary);
  animation: pulsePill 2.5s infinite ease-in-out;
}

@keyframes pulsePill {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.04); }
}

.milestone-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.65rem;
  font-weight: 800;
  margin: 0 0 0.25rem;
  letter-spacing: -0.02em;
}

.milestone-sub {
  font-size: 0.85rem;
  color: var(--muted-text);
  margin: 0 0 1.25rem;
  line-height: 1.4;
}

.emblem-showcase {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 1.5rem;
  z-index: 1;
}

.emblem-aura-container {
  position: relative;
  width: 120px;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 0.75rem;
}

.sunburst-rays {
  position: absolute;
  inset: -20px;
  background: conic-gradient(
    from 0deg,
    transparent 0deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 20deg,
    transparent 40deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 60deg,
    transparent 80deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 100deg,
    transparent 120deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 140deg,
    transparent 160deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 180deg,
    transparent 200deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 220deg,
    transparent 240deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 260deg,
    transparent 280deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 300deg,
    transparent 320deg,
    color-mix(in srgb, var(--primary) 18%, transparent) 340deg,
    transparent 360deg
  );
  border-radius: 50%;
  animation: rotateRays 14s linear infinite;
  mask-image: radial-gradient(circle, black 40%, transparent 70%);
  -webkit-mask-image: radial-gradient(circle, black 40%, transparent 70%);
  pointer-events: none;
}

@keyframes rotateRays {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.halo-pulse {
  position: absolute;
  inset: 4px;
  border-radius: 28px;
  background: radial-gradient(circle, color-mix(in srgb, var(--primary) 30%, transparent) 0%, transparent 80%);
  animation: haloPulseAnim 2.2s infinite ease-in-out;
  pointer-events: none;
}

@keyframes haloPulseAnim {
  0%, 100% { transform: scale(0.95); opacity: 0.6; }
  50% { transform: scale(1.15); opacity: 1; }
}

.emblem-pedestal {
  position: relative;
  width: 96px;
  height: 96px;
  border-radius: 26px;
  background: var(--surface-container);
  border: 2px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.14);
  z-index: 1;
  animation: pedestalBounce 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

@keyframes pedestalBounce {
  0% { transform: scale(0.4) rotate(-10deg); }
  100% { transform: scale(1) rotate(0deg); }
}

.emblem-avatar {
  font-size: 3.4rem;
  filter: drop-shadow(0 4px 10px rgba(0, 0, 0, 0.14));
  animation: floatAvatar 3s ease-in-out infinite;
}

@keyframes floatAvatar {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}

.sparkle-dot {
  position: absolute;
  pointer-events: none;
  font-size: 1rem;
  z-index: 2;
  animation: sparkleTwinkle 2s infinite ease-in-out;
}

.sparkle-1 {
  top: 0;
  right: 6px;
  animation-delay: 0.2s;
}

.sparkle-2 {
  bottom: 8px;
  left: 2px;
  font-size: 0.85rem;
  animation-delay: 0.8s;
}

.sparkle-3 {
  top: 12px;
  left: 4px;
  font-size: 0.9rem;
  animation-delay: 1.4s;
}

@keyframes sparkleTwinkle {
  0%, 100% { transform: scale(0.6) rotate(0deg); opacity: 0.3; }
  50% { transform: scale(1.2) rotate(45deg); opacity: 1; }
}

.title-text {
  font-family: 'Manrope', sans-serif;
  font-size: 1.4rem;
  font-weight: 800;
  margin: 0;
  color: var(--on-surface);
}

.xp-threshold-badge {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--muted-text);
  margin-top: 3px;
}

.perks-container {
  position: relative;
  z-index: 1;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
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
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
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
  position: relative;
  z-index: 1;
  width: 100%;
  padding: 0.85rem 1rem;
  font-weight: 800;
  font-size: 0.95rem;
  border-radius: 14px;
  cursor: pointer;
  border: none;
  background: var(--primary);
  color: var(--on-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  box-shadow: 0 6px 20px color-mix(in srgb, var(--primary) 35%, transparent);
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1), box-shadow 0.2s ease;
}

.celebrate-btn:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 8px 25px color-mix(in srgb, var(--primary) 50%, transparent);
}

.celebrate-btn:active {
  transform: translateY(0) scale(0.98);
}
</style>

