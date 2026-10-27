/**
 * useShortcuts — Unified, priority-scoped keyboard shortcut controller.
 *
 * Architecture:
 *   - A single global 'keydown' listener is lazily installed on the window
 *     when the first scope is pushed, and removed when the last scope pops.
 *   - Scopes are kept in a LIFO stack. On each keydown event the dispatcher
 *     walks from the top of the stack (highest priority) downward.
 *   - Once a binding in a scope handles the event (calls e.preventDefault or
 *     returns true), evaluation stops so lower scopes never see the key.
 *   - Form-element typing guard: single-character / non-modifier shortcuts are
 *     silently skipped when the focused element is an input, textarea, select,
 *     or contentEditable node — unless the binding opts in with allowInInputs.
 *   - Cleanup is automatic: each useShortcuts call returns a remove() function
 *     and also registers it with Vue's onScopeDispose / onUnmounted.
 */
import { onMounted, onUnmounted, getCurrentInstance } from 'vue'

// ─── Module-level singleton state ────────────────────────────────────────────

/** @type {{ id: string, bindings: NormalizedBinding[] }[]} */
const scopeStack = []
let listenerInstalled = false

/**
 * @typedef {Object} Binding
 * @property {string}   [key]          - Key value (e.g. ' ', 'ArrowDown', 'p')
 * @property {string}   [code]         - Physical key code (e.g. 'Digit1', 'Numpad1', 'KeyP')
 * @property {Function} handler        - Called with the KeyboardEvent
 * @property {boolean}  [ctrl]         - Require Ctrl (Windows) or Cmd (Mac)
 * @property {boolean}  [shift]        - Require Shift
 * @property {boolean}  [alt]          - Require Alt
 * @property {boolean}  [meta]         - Require Meta (Windows key / Cmd)
 * @property {boolean}  [allowInInputs]- Fire even inside inputs/textareas
 * @property {boolean}  [preventDefault]- Automatically call e.preventDefault()
 */

/** @typedef {Binding} NormalizedBinding */

// ─── Helpers ─────────────────────────────────────────────────────────────────

/** Returns true when the user is actively typing in a form element. */
function isTyping(e) {
  const t = e.target
  if (!t) return false
  const tag = t.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || !!t.isContentEditable
}

/**
 * Normalise a raw binding definition.
 * Normalises 'Space' → ' ' etc.
 * @param {Binding} binding
 * @returns {NormalizedBinding}
 */
function normalizeBinding(binding) {
  let key = binding.key ?? ''
  // Accept 'Space' as an alias for ' '
  if (key.toLowerCase() === 'space') key = ' '
  return { ...binding, key }
}

/** Test whether a keyboard event matches a normalised binding. */
function matchesBinding(e, b) {
  const ctrlOrMeta = e.ctrlKey || e.metaKey
  if (b.ctrl && !ctrlOrMeta) return false
  if (b.shift !== undefined && b.shift !== e.shiftKey) return false
  if (b.alt !== undefined && b.alt !== e.altKey) return false
  if (b.meta !== undefined && b.meta !== e.metaKey) return false

  // If code is specified, allow match by physical code (e.g. Numpad1 vs Digit1)
  if (b.code && e.code === b.code) return true

  // Key comparison (case-sensitive for letters like 'S' vs 's')
  return Boolean(b.key && e.key === b.key)
}

// ─── Global dispatcher ───────────────────────────────────────────────────────

function globalDispatcher(e) {
  const typing = isTyping(e)

  // Walk scope stack from top (highest priority) to bottom
  for (let i = scopeStack.length - 1; i >= 0; i--) {
    const scope = scopeStack[i]
    for (const b of scope.bindings) {
      if (!matchesBinding(e, b)) continue
      // Skip form-guard unless binding explicitly allows it
      if (typing && !b.allowInInputs) continue

      if (b.preventDefault !== false) e.preventDefault()
      b.handler(e)
      return // First match wins; stop stack traversal
    }
  }
}

function installListener() {
  if (listenerInstalled) return
  window.addEventListener('keydown', globalDispatcher, true)
  listenerInstalled = true
}

function removeListener() {
  if (!listenerInstalled) return
  window.removeEventListener('keydown', globalDispatcher, true)
  listenerInstalled = false
}

// ─── Public API ──────────────────────────────────────────────────────────────

let _nextScopeId = 0

/**
 * Register a keyboard shortcut scope.
 *
 * @param {Object} options
 * @param {string}    [options.scope]    Human-readable scope name (for debugging)
 * @param {Binding[]} [options.bindings] Array of key bindings
 * @param {import('vue').Ref<boolean>} [options.enabled] Reactive enable flag
 *
 * When called inside a Vue component `setup()`, cleanup is automatic via
 * onMounted/onUnmounted. Call the returned `remove()` if used outside a
 * component.
 *
 * @returns {{ remove: Function }}
 */
export function useShortcuts({ scope = 'default', bindings = [] } = {}) {
  const id = `${scope}-${_nextScopeId++}`
  const normalizedBindings = bindings.map(normalizeBinding)

  function push() {
    scopeStack.push({ id, bindings: normalizedBindings })
    installListener()
  }

  function remove() {
    const idx = scopeStack.findIndex((s) => s.id === id)
    if (idx !== -1) scopeStack.splice(idx, 1)
    if (scopeStack.length === 0) removeListener()
  }

  // Auto-lifecycle when called inside a component setup
  try {
    const instance = getCurrentInstance()
    if (instance) {
      onMounted(push)
      onUnmounted(remove)
    } else {
      // Called outside a component (e.g. app-level App.vue script setup — still in setup)
      push()
    }
  } catch {
    // Fallback: just push immediately
    push()
  }

  return { remove }
}
