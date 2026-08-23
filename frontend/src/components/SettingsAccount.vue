<template>
  <article class="panel form-grid">
    <div class="header-row">
      <h2>Account &amp; Subscription</h2>
      <span v-if="activeProfileName" class="profile-pill">
        Profile: <strong>{{ activeProfileName }}</strong>
      </span>
    </div>

    <!-- Signed In with Clerk -->
    <div v-if="clerkAuth.isSignedIn.value" class="signed-in-box">
      <div class="status-indicator">
        <span class="pulse-dot active"></span>
        <strong>Connected Account</strong>
      </div>

      <div class="user-details">
        <p class="user-email">
          <strong>Email:</strong> {{ clerkAuth.user.value?.email || 'User' }}
        </p>
        <div class="plan-row">
          <span><strong>Current Plan:</strong></span>
          <span v-if="clerkAuth.isPro.value" class="plan-pill pro">
            ★ PRO PLAN
          </span>
          <span v-else class="plan-pill free">
            FREE PLAN
          </span>
        </div>
      </div>

      <div class="account-actions">
        <button
          v-if="!clerkAuth.isPro.value"
          type="button"
          class="upgrade-btn"
          @click="clerkAuth.openBilling()"
        >
          Upgrade to Pro
        </button>
        <button
          type="button"
          class="billing-btn"
          @click="clerkAuth.openBilling()"
        >
          Manage Billing
        </button>
        <button
          type="button"
          class="danger-btn"
          @click="clerkAuth.signOut()"
        >
          Sign Out
        </button>
      </div>
    </div>

    <!-- Signed Out / Anonymous State -->
    <div v-else class="signed-out-box">
      <p class="field-hint">
        Sign in to your StudyLoop account to activate your Pro subscription, unlock custom extensions, and access cloud backup.
      </p>

      <div class="signed-out-plan-card">
        <div class="plan-info">
          <span class="plan-pill free">CURRENT: FREE PLAN</span>
          <p class="plan-desc">Access to core study queue, Reader, Quiz, FSRS flashcards, and Free extensions.</p>
        </div>
        <button type="button" class="sign-in-btn" @click="clerkAuth.signIn()">
          Sign In / Create Account
        </button>
      </div>
    </div>

    <!-- Local Dev / Testing Helper -->
    <div v-if="isDev" class="dev-section">
      <label class="dev-label">
        Developer Testing
        <span class="dev-badge">DEV</span>
      </label>
      <div class="dev-toggle-row">
        <span>Simulate Pro Subscription:</span>
        <button
          type="button"
          class="dev-toggle-btn"
          :class="{ active: clerkAuth.isPro.value }"
          @click="clerkAuth.setMockPro(!clerkAuth.isPro.value)"
        >
          {{ clerkAuth.isPro.value ? 'Pro Simulated (Active)' : 'Free Mode (Click to Toggle)' }}
        </button>
      </div>
    </div>
  </article>
</template>

<script setup>
import { onMounted } from 'vue'
import { useClerkAuth, initClerk } from '../services/clerkAuth'

defineProps({
  settings: { type: Object, default: () => ({}) },
  isDev: { type: Boolean, default: false },
  activeProfileName: { type: String, default: '' },
})

const clerkAuth = useClerkAuth()

onMounted(async () => {
  await initClerk()
})
</script>

<style scoped>
.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.profile-pill {
  font-size: 0.8rem;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--surface-container-high);
  color: var(--on-surface-variant);
  border: 1px solid color-mix(in srgb, var(--outline-variant) 30%, transparent);
}

.signed-in-box {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #10b981;
  font-size: 14px;
}

.pulse-dot.active {
  width: 8px;
  height: 8px;
  background: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
  }
  70% {
    transform: scale(1);
    box-shadow: 0 0 0 6px rgba(16, 185, 129, 0);
  }
  100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0);
  }
}

.user-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
}

.plan-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.plan-pill {
  font-size: 11px;
  font-weight: 700;
  padding: 3px 10px;
  border-radius: 6px;
  letter-spacing: 0.05em;
}

.plan-pill.free {
  background: var(--surface-container-highest);
  color: var(--on-surface-variant);
}

.plan-pill.pro {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #ffffff;
}

.account-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.upgrade-btn {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #ffffff;
  border: none;
  padding: 10px 18px;
  border-radius: 10px;
  font-weight: 700;
  font-size: 13.5px;
  cursor: pointer;
}

.billing-btn {
  background: var(--surface-container-highest);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  padding: 10px 18px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 13.5px;
  cursor: pointer;
}

.danger-btn {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  padding: 10px 18px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 13.5px;
  cursor: pointer;
  margin-left: auto;
}

.signed-out-box {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.signed-out-plan-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.plan-desc {
  font-size: 13px;
  color: var(--muted-text);
  margin: 6px 0 0 0;
}

.sign-in-btn {
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  padding: 11px 20px;
  border-radius: 10px;
  font-weight: 700;
  font-size: 14px;
  cursor: pointer;
  white-space: nowrap;
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 28px;
  border: 1px solid color-mix(in srgb, var(--outline-variant) 20%, transparent);
}

.field-hint {
  color: var(--muted-text);
  font-size: 13px;
  line-height: 1.5;
  margin: 0;
}

.dev-section {
  margin-top: 10px;
  border-top: 1px solid var(--outline-variant);
  padding-top: 16px;
}

.dev-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--muted-text);
  display: block;
  margin-bottom: 8px;
}

.dev-badge {
  display: inline-block;
  margin-left: 6px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.dev-toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13.5px;
}

.dev-toggle-btn {
  padding: 6px 14px;
  border-radius: 8px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  font-size: 12.5px;
  cursor: pointer;
}

.dev-toggle-btn.active {
  background: #f59e0b;
  color: #000;
  font-weight: 700;
}
</style>
