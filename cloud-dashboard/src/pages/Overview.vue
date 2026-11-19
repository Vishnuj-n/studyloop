<template>
  <div class="overview-page fade-in">
    <!-- Welcome & Classroom Banner -->
    <section class="card animate-fade-in welcome-banner" style="margin-bottom: 1.5rem; padding: 1.5rem 1.75rem;">
      <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
        <div style="display: flex; align-items: center; gap: 1.25rem;">
          <div class="teacher-avatar">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
              <circle cx="12" cy="7" r="4"></circle>
            </svg>
          </div>
          <div>
            <div class="micro-label" style="letter-spacing: 0.08em; margin-bottom: 0.2rem;">TEACHER PORTAL</div>
            <h2 style="margin: 0; font-size: 1.35rem; font-weight: 700; color: var(--ds-fg); display: flex; align-items: center; gap: 0.5rem;">
              <span>Welcome back<span v-if="currentUsername">, <span style="color: var(--ds-primary);">{{ currentUsername }}</span></span>!</span>
            </h2>
            <p class="muted" style="margin: 0.35rem 0 0 0; font-size: 0.85rem;">
              Managing workspace for classroom
              <span class="classroom-pill" :title="'Classroom Code: ' + classroomCode">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 4px;">
                  <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
                  <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
                </svg>
                {{ classroomCode || 'NOT SET' }}
              </span>
            </p>
          </div>
        </div>

        <div style="display: flex; gap: 0.6rem; align-items: center;">
          <button class="btn-ghost" @click="copyClassCode" :title="'Click to copy classroom code ' + classroomCode">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            <span>{{ copied ? 'Copied Code!' : 'Share Code' }}</span>
          </button>
          <RouterLink to="/students" class="btn" style="text-decoration: none;">
            <span>View Students</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-left: 4px;">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </RouterLink>
        </div>
      </div>
    </section>

    <section class="stats-grid">
      <!-- Stat 1: Students Syncing -->
      <div class="stat-card animate-fade-in" style="animation-delay: 0ms">
        <div class="stat-header">
          <span class="stat-title">Syncing Students</span>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--ds-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
            <circle cx="9" cy="7" r="4"></circle>
            <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
          </svg>
        </div>
        <div style="display: flex; justify-content: space-between; align-items: flex-end;">
          <div class="stat-value">{{ stats.studentsCount }}</div>
          <div v-if="students.length > 0" class="avatar-stack">
            <div v-for="student in students.slice(0, 4)" :key="student.token" class="avatar-stacked" :title="student.token">
              {{ student.token.substring(0, 2).toUpperCase() }}
            </div>
            <div v-if="students.length > 4" class="avatar-stacked" style="background: var(--ds-surface-hi); color: var(--ds-muted);">
              +{{ students.length - 4 }}
            </div>
          </div>
        </div>
        <div class="stat-desc">Active study profiles connected</div>
      </div>

      <!-- Stat 2: FSRS Reviews -->
      <div class="stat-card animate-fade-in" style="animation-delay: 60ms">
        <div class="stat-header">
          <span class="stat-title">FSRS Reviews</span>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--ds-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
            <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
          </svg>
        </div>
        <div class="stat-value">{{ stats.totalLogs }}</div>
        <div class="stat-desc">Avg. {{ stats.studentsCount > 0 ? Math.round(stats.totalLogs / stats.studentsCount) : 0 }} reviews per student</div>
      </div>

      <!-- Stat 3: Recall Pass Rate -->
      <div class="stat-card animate-fade-in" style="animation-delay: 120ms">
        <div class="stat-header">
          <span class="stat-title">Recall Pass Rate</span>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--ds-success)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
          </svg>
        </div>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <div class="stat-value">{{ stats.passRate }}%</div>
          <svg width="36" height="36" viewBox="0 0 36 36">
            <circle stroke="var(--ds-border-hi)" stroke-width="3" fill="transparent" r="13" cx="18" cy="18"/>
            <circle
              :stroke="stats.passRate > 75 ? 'var(--ds-success)' : stats.passRate > 55 ? 'var(--ds-warning)' : 'var(--ds-danger)'"
              stroke-width="3"
              stroke-linecap="round"
              fill="transparent"
              r="13"
              cx="18"
              cy="18"
              :style="{ strokeDasharray: '81.68', strokeDashoffset: 81.68 - (81.68 * stats.passRate / 100), transform: 'rotate(-90deg)', transformOrigin: '50% 50%', transition: 'stroke-dashoffset 0.5s ease' }"
            />
          </svg>
        </div>
        <div class="stat-desc">Fraction of reviews rated Good or Easy</div>
      </div>

      <!-- Stat 4: Red Alerts -->
      <div class="stat-card animate-fade-in" style="animation-delay: 180ms" :style="{ borderColor: stats.alertsCount > 0 ? 'var(--ds-danger-ring)' : 'var(--ds-border)' }">
        <div class="stat-header">
          <span class="stat-title">Teacher Intervention Alerts</span>
          <span v-if="stats.alertsCount > 0" class="pulse-dot" style="background-color: var(--ds-danger);"></span>
          <svg v-else width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--ds-muted)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
        </div>
        <div class="stat-value" :style="{ color: stats.alertsCount > 0 ? 'var(--ds-danger)' : 'var(--ds-fg)' }">
          {{ stats.alertsCount }}
        </div>
        <div class="stat-desc">
          {{ stats.alertsCount > 0 ? 'Students requiring teacher intervention' : 'All students progressing smoothly' }}
        </div>
      </div>
    </section>

    <!-- FSRS Rating Distribution Breakdown Card -->
    <section class="card animate-fade-in" style="animation-delay: 240ms; margin-bottom: 1.75rem;">
      <div class="section-title" style="display: flex; justify-content: space-between; align-items: center;">
        <span>FSRS Rating Breakdown</span>
        <span class="micro-label">CLASSROOM AGGREGATE</span>
      </div>

      <div style="display: flex; flex-direction: column; gap: 1rem; margin-top: 0.5rem;">
        <div style="display: flex; align-items: center; gap: 1rem;">
          <span style="width: 50px; font-size: 0.78rem; font-weight: 600; color: var(--ds-primary);">Easy</span>
          <div style="flex: 1; height: 10px; background: var(--ds-surface-hi); border-radius: 999px; overflow: hidden;">
            <div :style="{ width: (stats.ratingBreakdown?.easyPct || 0) + '%' }" style="height: 100%; background: var(--ds-primary); border-radius: 999px; transition: width 0.5s ease;"></div>
          </div>
          <span style="width: 40px; text-align: right; font-size: 0.78rem; font-family: var(--ds-mono); color: var(--ds-muted);">{{ stats.ratingBreakdown?.easyPct || 0 }}%</span>
        </div>

        <div style="display: flex; align-items: center; gap: 1rem;">
          <span style="width: 50px; font-size: 0.78rem; font-weight: 600; color: var(--ds-success);">Good</span>
          <div style="flex: 1; height: 10px; background: var(--ds-surface-hi); border-radius: 999px; overflow: hidden;">
            <div :style="{ width: (stats.ratingBreakdown?.goodPct || 0) + '%' }" style="height: 100%; background: var(--ds-success); border-radius: 999px; transition: width 0.5s ease;"></div>
          </div>
          <span style="width: 40px; text-align: right; font-size: 0.78rem; font-family: var(--ds-mono); color: var(--ds-muted);">{{ stats.ratingBreakdown?.goodPct || 0 }}%</span>
        </div>

        <div style="display: flex; align-items: center; gap: 1rem;">
          <span style="width: 50px; font-size: 0.78rem; font-weight: 600; color: var(--ds-warning);">Hard</span>
          <div style="flex: 1; height: 10px; background: var(--ds-surface-hi); border-radius: 999px; overflow: hidden;">
            <div :style="{ width: (stats.ratingBreakdown?.hardPct || 0) + '%' }" style="height: 100%; background: var(--ds-warning); border-radius: 999px; transition: width 0.5s ease;"></div>
          </div>
          <span style="width: 40px; text-align: right; font-size: 0.78rem; font-family: var(--ds-mono); color: var(--ds-muted);">{{ stats.ratingBreakdown?.hardPct || 0 }}%</span>
        </div>

        <div style="display: flex; align-items: center; gap: 1rem;">
          <span style="width: 50px; font-size: 0.78rem; font-weight: 600; color: var(--ds-danger);">Fail</span>
          <div style="flex: 1; height: 10px; background: var(--ds-surface-hi); border-radius: 999px; overflow: hidden;">
            <div :style="{ width: (stats.ratingBreakdown?.failPct || 0) + '%' }" style="height: 100%; background: var(--ds-danger); border-radius: 999px; transition: width 0.5s ease;"></div>
          </div>
          <span style="width: 40px; text-align: right; font-size: 0.78rem; font-family: var(--ds-mono); color: var(--ds-muted);">{{ stats.ratingBreakdown?.failPct || 0 }}%</span>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { RouterLink } from 'vue-router';
import { useDashboard } from '../composables/useDashboard';

const { stats, students, classroomCode, currentUsername } = useDashboard();
const copied = ref(false);

function copyClassCode() {
  if (!classroomCode.value) return;
  navigator.clipboard.writeText(classroomCode.value).then(() => {
    copied.value = true;
    setTimeout(() => {
      copied.value = false;
    }, 2500);
  }).catch(() => {});
}
</script>

<style scoped>
.welcome-banner {
  background: linear-gradient(135deg, var(--ds-surface) 0%, var(--ds-surface-hi) 100%);
  border: 1px solid var(--ds-border-hi);
  position: relative;
  overflow: hidden;
}

.teacher-avatar {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: var(--ds-primary-glow);
  color: var(--ds-primary);
  border: 1px solid var(--ds-primary-ring);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.classroom-pill {
  display: inline-flex;
  align-items: center;
  font-family: var(--ds-mono);
  font-weight: 600;
  font-size: 0.8rem;
  color: var(--ds-primary);
  background: var(--ds-surface-hi);
  border: 1px solid var(--ds-border-hi);
  padding: 0.15rem 0.5rem;
  border-radius: 6px;
  margin-left: 0.35rem;
}
</style>
