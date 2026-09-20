<template>
  <div class="dashboard-banners">
    <StatusBanner
      v-if="showGitHubStarToast"
      variant="star"
      icon=""
      title="Support Studyloop on GitHub"
      subtitle="You've completed 5+ study sessions! If you find Studyloop helpful, consider starring the repo on GitHub to support open-source development."
      action-label="⭐ Star on GitHub ↗"
      dismissable
      @action="emit('star-github')"
      @dismiss="emit('dismiss-star-github')"
    >
      <template #icon>
        <BaseIcon name="github" size="20" />
      </template>
    </StatusBanner>

    <StatusBanner
      v-if="streakSavedEvent"
      variant="info"
      icon="shield"
      title="Your streak was saved!"
      :subtitle="`We used 1 Streak Freeze to protect your ${streakSavedEvent.streak_length}-day streak yesterday. You have ${streakSavedEvent.freezes_remaining} freeze(s) remaining.`"
      action-label="Dismiss"
      @action="emit('dismiss-streak-saved')"
    />

    <StatusBanner
      v-if="pendingIngestionBook"
      variant="warning"
      icon="zap"
      :title="pendingIngestionBannerTitle"
      :subtitle="`${pendingIngestionBook.title} is ready for chapter extraction and ingestion.`"
      action-label="Ingest Book"
      @action="emit('ingest-book', pendingIngestionBook.id)"
    />

    <StatusBanner
      v-if="skipToReadingActive"
      variant="info"
      icon="zap"
      title='"Skip to Reading" Escape Hatch Active'
      subtitle="Review tasks have been pushed to the background so you can focus on reading new chapters."
    />

    <StatusBanner
      v-if="hasSocraticRescueTask"
      variant="rescue"
      icon="shield"
      title="Concept Rescue Active"
      subtitle="Your study queue is locked because you failed the quiz twice on this topic. You must complete the Socratic tutor rescue session to unblock your timeline."
    />

    <StatusBanner
      v-if="flashcardNotice"
      variant="success"
      icon="sparkles"
      title="Flashcards Ready"
      :subtitle="flashcardNotice"
    />

    <StatusBanner
      v-if="flashcardsJustCreated"
      variant="success"
      icon="check"
      :title="'Flashcards generated successfully!'"
      :subtitle="`${flashcardsJustCreated} cards scheduled for spaced repetition.`"
    />

    <StatusBanner
      v-if="actionError"
      variant="error"
      icon="alert-triangle"
      title="Error starting task"
      :subtitle="actionError"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import StatusBanner from './StatusBanner.vue'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  showGitHubStarToast: { type: Boolean, default: false },
  streakSavedEvent: { type: Object, default: null },
  pendingIngestionBook: { type: Object, default: null },
  isCloudAccount: { type: Boolean, default: false },
  skipToReadingActive: { type: Boolean, default: false },
  hasSocraticRescueTask: { type: Boolean, default: false },
  flashcardNotice: { type: String, default: '' },
  flashcardsJustCreated: { type: Number, default: 0 },
  actionError: { type: String, default: '' },
})

const emit = defineEmits([
  'star-github',
  'dismiss-star-github',
  'dismiss-streak-saved',
  'ingest-book',
])

const pendingIngestionBannerTitle = computed(() => {
  return props.isCloudAccount
    ? 'New Assignment — Ingestion Needed'
    : 'New Book — Ingestion Needed'
})
</script>

<style scoped>
.dashboard-banners {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
