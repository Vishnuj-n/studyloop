import { createRouter, createWebHashHistory } from 'vue-router'

import Dashboard from '../pages/Dashboard.vue'
import Reader from '../pages/Reader.vue'
import Quiz from '../pages/Quiz.vue'
import Flashcards from '../pages/Flashcards.vue'
import Socratic from '../pages/Socratic.vue'
import SocraticRescue from '../pages/SocraticRescue.vue'
import Settings from '../pages/Settings.vue'
import Notebook from '../pages/Notebook.vue'
import Onboarding from '../pages/Onboarding.vue'
import { isOnboarded, logFrontendEvent } from '../services/appApi'

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', name: 'dashboard', component: Dashboard },
  { path: '/reader', name: 'reader', component: Reader },
  {
    path: '/notes',
    name: 'notes',
    component: () => import('../pages/Notes.vue'),
  },
  { path: '/quiz', name: 'quiz', component: Quiz },
  { path: '/flashcards', name: 'flashcards', component: Flashcards },
  {
    path: '/examiner',
    name: 'examiner',
    component: () => import('../pages/WrittenAssessment.vue'),
  },
  { path: '/tutor', name: 'tutor', component: Socratic },
  { path: '/socratic', redirect: '/tutor' },
  { path: '/socratic-rescue', name: 'socratic-rescue', component: SocraticRescue },
  { path: '/notebooks', name: 'notebooks', component: Notebook },
  {
    path: '/extensions',
    name: 'extensions',
    component: () => import('../pages/Extensions.vue'),
  },
  {
    path: '/simplify',
    name: 'simplify',
    component: () => import('../pages/Simplify.vue'),
  },
  {
    path: '/rewards',
    name: 'rewards',
    component: () => import('../pages/Rewards.vue'),
  },
  {
    path: '/quiz-analysis',
    name: 'quiz-analysis',
    component: () => import('../pages/QuizAnalysis.vue'),
  },
  { path: '/settings', name: 'settings', component: Settings },
  { path: '/onboarding', name: 'onboarding', component: Onboarding },
  {
    path: '/select-profile',
    name: 'select-profile',
    component: () => import('../pages/ProfileSelect.vue'),
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach(async (to, from, next) => {
  console.log(`[Router Navigation] from "${from.fullPath || from.path}" -> to "${to.fullPath || to.path}"`)
  if (to.path === '/onboarding') {
    next()
    return
  }
  try {
    const res = await isOnboarded()
    if (res && res.error) {
      logFrontendEvent(
        'warn',
        'Router',
        'onboarding_check_backend_error',
        JSON.stringify({ target: to.path, error: res.error })
      )
      next()
    } else if (res && res.onboarded === false) {
      logFrontendEvent(
        'info',
        'Router',
        'onboarding_redirect',
        JSON.stringify({ target: to.path, res })
      )
      console.log('[Router Navigation] Redirecting to /onboarding (not onboarded)')
      next('/onboarding')
    } else {
      // Check if startup profile selection is needed on initial launch
      if (to.path !== '/select-profile') {
        const askStartup = localStorage.getItem('studyloop_ask_profile_on_startup')
        const alreadyShown = sessionStorage.getItem('studyloop_profile_picker_shown')

        if (askStartup !== 'false' && !alreadyShown && (!from.name || from.path === '/')) {
          try {
            const { getProfiles, getUserSettings } = await import('../services/appApi')
            const [profilesRes, settingsRes] = await Promise.all([
              getProfiles(),
              getUserSettings(),
            ])
            const profiles = Array.isArray(profilesRes?.profiles) ? profilesRes.profiles : []
            if (profiles.length > 1 && settingsRes?.ask_profile_on_startup !== false) {
              sessionStorage.setItem('studyloop_profile_picker_shown', 'true')
              console.log('[Router Navigation] Routing to /select-profile on startup')
              next('/select-profile')
              return
            }
          } catch {
            // Fail open to standard navigation if profile check fails
          }
        }
        // Mark as shown once user is actively navigating within app
        sessionStorage.setItem('studyloop_profile_picker_shown', 'true')
      }
      console.log(`[Router Navigation] Proceeding to ${to.path}`)
      next()
    }
  } catch (err) {
    logFrontendEvent(
      'error',
      'Router',
      'onboarding_check_error',
      JSON.stringify({ target: to.path, error: String(err) })
    )
    console.error('[Router Navigation] Error during route guard:', err)
    next()
  }
})

router.afterEach((to, from, failure) => {
  if (failure) {
    console.error('[Router afterEach] Navigation failed/aborted:', failure)
  } else {
    console.log(`[Router afterEach] Successfully navigated to: ${to.fullPath || to.path}`)
  }
})

router.onError((error) => {
  console.error('[Router onError] Uncaught navigation error:', error)
})

export default router
