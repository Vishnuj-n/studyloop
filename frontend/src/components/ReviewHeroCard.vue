<template>
  <article class="card review-hero-card">
    <div class="review-hero-header">
      <span class="review-hero-tag">HIGH PRIORITY</span>
      <span class="review-hero-estimate">{{ task.estimate_minutes }} min review</span>
    </div>
    <div class="review-hero-body">
      <h2>Today's Reviews</h2>
      <div class="review-hero-stats">
        <div class="review-hero-stat">
          <span class="stat-num">{{ dueReviewCards }}</span>
          <span class="stat-lbl">Due Today</span>
        </div>
        <div class="review-hero-stat">
          <span class="stat-num">{{
            totalDueReviewCards > dueReviewCards ? totalDueReviewCards - dueReviewCards : 0
          }}</span>
          <span class="stat-lbl">Remaining Overdue</span>
        </div>
      </div>
      <p class="review-hero-meta">{{ task.meta }}</p>
    </div>
    <button type="button" class="primary-btn review-hero-btn" @click="$emit('start', task)">
      Start Review
    </button>
  </article>
</template>

<script setup>
defineProps({
  task: { type: Object, required: true },
  dueReviewCards: { type: Number, default: 0 },
  totalDueReviewCards: { type: Number, default: 0 },
})

defineEmits(['start'])
</script>

<style scoped>
.card {
  background: var(--surface-container-lowest, #ffffff);
  border: none;
  border-radius: 16px;
}

.review-hero-card {
  background: var(--surface-container-lowest, #ffffff);
  box-shadow: 0 4px 20px rgba(45, 51, 56, 0.04);
  padding: 28px 32px;
  display: grid;
  grid-template-columns: 1fr;
  gap: 20px;
  position: relative;
  overflow: hidden;
  border-radius: 16px;
  transition:
    transform 0.2s cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 0.2s;
}

.review-hero-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 28px rgba(45, 51, 56, 0.08);
}

@media (min-width: 768px) {
  .review-hero-card {
    grid-template-columns: 1fr auto;
    align-items: center;
    gap: 32px;
  }
}

.review-hero-header {
  grid-column: 1 / -1;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.review-hero-tag {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: var(--primary, #005bc1);
  background: var(--surface-container, #ebeef2);
  padding: 4px 10px;
  border-radius: 6px;
  text-transform: uppercase;
}

.review-hero-estimate {
  font-size: 12px;
  font-weight: 500;
  color: var(--muted-text, #64707d);
}

.review-hero-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.review-hero-body h2 {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 24px;
  font-weight: 700;
  color: var(--on-surface, #2d3338);
}

.review-hero-stats {
  display: flex;
  gap: 24px;
  margin-top: 4px;
}

.review-hero-stat {
  display: flex;
  flex-direction: column;
}

.stat-num {
  font-family: 'Manrope', sans-serif;
  font-size: 32px;
  font-weight: 700;
  color: var(--on-surface, #2d3338);
  line-height: 1;
}

.stat-lbl {
  font-size: 11px;
  font-weight: 600;
  color: var(--muted-text, #64707d);
  text-transform: uppercase;
  margin-top: 4px;
}

.review-hero-meta {
  margin: 0;
  font-size: 14px;
  color: var(--muted-text, #64707d);
}

.primary-btn {
  background: linear-gradient(15deg, var(--primary, #005bc1) 0%, var(--primary-dim, #004faa) 100%);
  color: var(--on-primary, #ffffff);
  border: none;
  border-radius: 0.75rem;
  padding: 12px 24px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(45, 51, 56, 0.08);
  transition:
    transform 0.15s cubic-bezier(0.16, 1, 0.3, 1),
    opacity 0.15s;
}

.primary-btn:hover {
  opacity: 0.94;
  transform: translateY(-1px);
}

.primary-btn:active {
  transform: scale(0.98);
}

.review-hero-btn {
  justify-self: stretch;
  height: auto;
}

@media (min-width: 768px) {
  .review-hero-btn {
    justify-self: end;
  }
}
</style>
