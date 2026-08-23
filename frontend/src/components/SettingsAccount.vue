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

    <!-- School & Classroom Cloud Sync Section (Hidden under expandable button) -->
    <div class="school-section">
      <!-- Case 1: Already connected to a school/classroom account -->
      <div v-if="settings.cloud_student_id || settings.cloud_session_token" class="school-connected-box">
        <div class="school-header">
          <div class="status-indicator">
            <span class="pulse-dot active"></span>
            <strong>School &amp; Classroom Connected</strong>
          </div>
          <button type="button" class="danger-btn-sm" :disabled="disabled" @click="$emit('logout')">
            Disconnect
          </button>
        </div>
        <div class="school-info-grid">
          <div class="school-info-item">
            <span class="info-label">Student ID</span>
            <span class="info-val">{{ settings.cloud_student_id || 'Active' }}</span>
          </div>
          <div v-if="settings.cloud_classroom_code" class="school-info-item">
            <span class="info-label">Classroom</span>
            <span class="info-val code-badge">{{ settings.cloud_classroom_code }}</span>
          </div>
        </div>
      </div>

      <!-- Case 2: Not connected -> Collapsible button to reveal school login -->
      <div v-else class="school-expandable-card">
        <button
          type="button"
          class="school-toggle-btn"
          @click="showSchoolLogin = !showSchoolLogin"
        >
          <div class="school-btn-label">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M22 10v6M2 10l10-5 10 5-10 5z"></path>
              <path d="M6 12v5c3 3 9 3 12 0v-5"></path>
            </svg>
            <span>School or Classroom? <strong>Connect School Account</strong></span>
          </div>
          <span class="toggle-icon">{{ showSchoolLogin ? '▲ Hide' : '▼ Expand' }}</span>
        </button>

        <div v-if="showSchoolLogin" class="school-form-body animate-fade-in">
          <div class="auth-toggle-bar">
            <button
              type="button"
              class="auth-tab"
              :class="{ active: !isSignUpMode }"
              @click="isSignUpMode && $emit('toggle-mode')"
            >
              Sign In
            </button>
            <button
              type="button"
              class="auth-tab"
              :class="{ active: isSignUpMode }"
              @click="!isSignUpMode && $emit('toggle-mode')"
            >
              Register with Class Code
            </button>
          </div>

          <p class="field-hint">
            {{
              isSignUpMode
                ? 'Register a student account using the 6-character code provided by your teacher.'
                : 'Sign in with your student credentials to sync assignments with your teacher.'
            }}
          </p>

          <div v-if="loginError" class="login-error-message">
            {{ loginError }}
          </div>

          <template v-if="!isSignUpMode">
            <div class="form-group">
              <label for="student-username">Student Username / ID</label>
              <input
                id="student-username"
                :value="loginUsername"
                type="text"
                placeholder="e.g. john_doe"
                :disabled="loggingIn"
                @input="$emit('update:login-username', $event.target.value)"
              />
            </div>

            <div class="form-group">
              <label for="student-password">Password</label>
              <input
                id="student-password"
                :value="loginPassword"
                type="password"
                placeholder="••••••••"
                :disabled="loggingIn"
                @input="$emit('update:login-password', $event.target.value)"
              />
            </div>

            <button
              type="button"
              class="school-submit-btn"
              :disabled="loggingIn"
              @click="$emit('login')"
            >
              {{ loggingIn ? 'Signing In...' : 'Sign In & Connect' }}
            </button>
          </template>

          <template v-else>
            <div class="form-group">
              <label for="signup-username">Student Username / ID</label>
              <input
                id="signup-username"
                :value="signupUsername"
                type="text"
                placeholder="e.g. john_doe"
                :disabled="loggingIn"
                @input="$emit('update:signup-username', $event.target.value)"
              />
            </div>

            <div class="form-group">
              <label for="signup-password">Create Password</label>
              <input
                id="signup-password"
                :value="signupPassword"
                type="password"
                placeholder="Min. 6 characters"
                :disabled="loggingIn"
                @input="$emit('update:signup-password', $event.target.value)"
              />
            </div>

            <div class="form-group">
              <label for="signup-classroom">Classroom Code</label>
              <input
                id="signup-classroom"
                :value="signupClassroomCode"
                type="text"
                placeholder="e.g. BCD601"
                style="text-transform: uppercase"
                :disabled="loggingIn"
                @input="$emit('update:signup-classroom-code', $event.target.value)"
              />
            </div>

            <button
              type="button"
              class="school-submit-btn"
              :disabled="loggingIn"
              @click="$emit('signup')"
            >
              {{ loggingIn ? 'Creating Account...' : 'Sign Up & Connect' }}
            </button>
          </template>
        </div>
      </div>
    </div>
  </article>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useClerkAuth, initClerk } from '../services/clerkAuth'

defineProps({
  settings: { type: Object, required: true },
  isDev: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  activeProfileName: { type: String, default: '' },
  isSignUpMode: { type: Boolean, default: false },
  loginUsername: { type: String, default: '' },
  loginPassword: { type: String, default: '' },
  signupUsername: { type: String, default: '' },
  signupPassword: { type: String, default: '' },
  signupClassroomCode: { type: String, default: '' },
  loginError: { type: String, default: '' },
  loggingIn: { type: Boolean, default: false },
})

defineEmits([
  'login',
  'signup',
  'logout',
  'toggle-mode',
  'update:login-username',
  'update:login-password',
  'update:signup-username',
  'update:signup-password',
  'update:signup-classroom-code',
])

const clerkAuth = useClerkAuth()
const showSchoolLogin = ref(false)

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

.dev-section {
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

/* School & Classroom Section */
.school-section {
  border-top: 1px solid var(--outline-variant);
  padding-top: 18px;
}

.school-connected-box {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.school-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.danger-btn-sm {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  padding: 5px 12px;
  border-radius: 8px;
  font-weight: 600;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.danger-btn-sm:hover {
  background: rgba(239, 68, 68, 0.2);
}

.school-info-grid {
  display: flex;
  gap: 20px;
}

.school-info-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-label {
  font-size: 11px;
  color: var(--muted-text);
  font-weight: 600;
  text-transform: uppercase;
}

.info-val {
  font-size: 14px;
  font-weight: 600;
  color: var(--on-surface);
}

.code-badge {
  font-family: monospace;
  letter-spacing: 0.05em;
  background: var(--surface-container-highest);
  padding: 2px 6px;
  border-radius: 4px;
}

.school-expandable-card {
  border: 1px dashed var(--outline-variant);
  border-radius: 12px;
  overflow: hidden;
  background: var(--surface-container-lowest);
}

.school-toggle-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--surface-container-low);
  border: none;
  padding: 12px 18px;
  cursor: pointer;
  color: var(--on-surface);
  transition: background 0.2s ease;
}

.school-toggle-btn:hover {
  background: var(--surface-container-high);
}

.school-btn-label {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13.5px;
  color: var(--on-surface);
}

.toggle-icon {
  font-size: 12px;
  color: var(--muted-text);
  font-weight: 600;
}

.school-form-body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  background: var(--surface-container-lowest);
}

.auth-toggle-bar {
  display: flex;
  gap: 8px;
  background: var(--surface-container-low);
  padding: 4px;
  border-radius: 10px;
}

.auth-tab {
  flex: 1;
  border: none;
  background: transparent;
  color: var(--muted-text);
  padding: 8px 12px;
  font-size: 13px;
  font-weight: 600;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.auth-tab.active {
  background: var(--surface-container-highest);
  color: var(--primary);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

label {
  font-weight: 600;
  font-size: 13px;
  color: var(--on-surface);
}

input[type='text'],
input[type='password'] {
  border: 1px solid color-mix(in srgb, var(--outline-variant) 20%, transparent);
  border-radius: 10px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  padding: 10px 12px;
  font-size: 13.5px;
  font-family: inherit;
  transition: border-color 0.2s ease;
}

input:focus {
  border-color: var(--primary);
  outline: none;
}

.school-submit-btn {
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 11px 18px;
  color: var(--primary);
  font-weight: 700;
  font-size: 13.5px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.school-submit-btn:hover {
  background: var(--surface-container-high);
}

.login-error-message {
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
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

