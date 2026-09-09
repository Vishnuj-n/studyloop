<template>
  <article class="panel form-grid">
    <div class="header-row">
      <h2>Account &amp; Subscription</h2>
      <span v-if="activeProfileName" class="profile-pill">
        Profile: <strong>{{ activeProfileName }}</strong>
      </span>
    </div>

    <!-- Authentication Error Alert -->
    <div v-if="clerkAuth.authError.value" class="login-error-message animate-fade-in">
      {{ clerkAuth.authError.value }}
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
            PRO PLAN
          </span>
          <span v-else class="plan-pill free">
            FREE PLAN
          </span>
        </div>
      </div>

      <div class="account-actions">
        <!-- ponytail: 1 button is enough; upgrade when free, manage when pro -->
        <button
          v-if="!clerkAuth.isPro.value"
          type="button"
          class="upgrade-btn"
          @click="onBillingClick"
        >
          ★ Support Dev &bull; Get Early Access
        </button>
        <button
          v-else
          type="button"
          class="billing-btn"
          @click="onBillingClick"
        >
          Manage Billing
        </button>
        <button
          type="button"
          class="danger-btn"
          @click="onSignOutClick"
        >
          Sign Out
        </button>
      </div>
    </div>

    <!-- Signed Out / Free Plan State -->
    <div v-else class="signed-out-box">
      <p class="field-hint">
        Sign in to your StudyLoop account to activate your Pro subscription, unlock custom extensions, and access cloud backup.
      </p>

      <div class="signed-out-plan-card">
        <div class="plan-info">
          <span class="plan-pill free">CURRENT: FREE PLAN</span>
          <p class="plan-desc">Access to core study queue, Reader, Quiz, FSRS flashcards, and Free extensions.</p>
        </div>
        <button type="button" class="sign-in-btn" @click="onSignInClick">
          Sign In / Create Account
        </button>
      </div>
    </div>
  </article>
</template>

<script setup>
import { onMounted } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'

defineProps({
  activeProfileName: { type: String, default: '' },
})

const clerkAuth = useClerkAuth()

function onSignInClick() {
  clerkAuth.signIn()
}

function onBillingClick() {
  clerkAuth.openBilling()
}

function onSignOutClick() {
  clerkAuth.signOut()
}

onMounted(() => {
  console.log('[SETTINGS_ACCOUNT] Mounted: isSignedIn =', clerkAuth.isSignedIn.value, 'isPro =', clerkAuth.isPro.value)
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
  border: 1px solid var(--outline-variant);
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
  border: 1px solid var(--outline-variant);
}

h2 {
  font-size: 20px;
  margin: 0;
  font-weight: 700;
}

.field-hint {
  color: var(--muted-text);
  font-size: 13px;
  line-height: 1.5;
  margin: 0;
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
  border: 1px solid var(--outline-variant);
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

.login-error-message {
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid var(--outline-variant);
  color: #f87171;
  padding: 0.75rem 1rem;
  border-radius: 8px;
  font-size: 0.85rem;
}

.animate-fade-in {
  animation: fadeIn 0.2s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>

