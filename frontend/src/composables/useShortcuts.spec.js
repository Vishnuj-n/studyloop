import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { useShortcuts } from './useShortcuts'

// ─── Helpers ──────────────────────────────────────────────────────────────────

function dispatchKey(opts = {}) {
  const evt = new KeyboardEvent('keydown', {
    bubbles: true,
    cancelable: true,
    ...opts,
  })
  window.dispatchEvent(evt)
  return evt
}

/** Wrap useShortcuts in a mounted component for proper lifecycle. */
function makeComponent(bindings) {
  return defineComponent({
    setup() {
      useShortcuts({ scope: 'test', bindings })
    },
    render() {
      return h('div')
    },
  })
}

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('useShortcuts', () => {
  beforeEach(() => {
    // Reset listener state between tests by unmounting all components
  })

  it('calls handler when the matching key is pressed', async () => {
    const handler = vi.fn()
    const wrapper = mount(makeComponent([{ key: 'a', handler }]))
    dispatchKey({ key: 'a' })
    expect(handler).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('does NOT call handler when a different key is pressed', () => {
    const handler = vi.fn()
    const wrapper = mount(makeComponent([{ key: 'a', handler }]))
    dispatchKey({ key: 'b' })
    expect(handler).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('matches Ctrl+Shift combos', () => {
    const handler = vi.fn()
    const wrapper = mount(
      makeComponent([{ key: 'p', ctrl: true, shift: true, allowInInputs: true, handler }])
    )
    dispatchKey({ key: 'p', ctrlKey: true, shiftKey: true })
    expect(handler).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('does NOT fire Ctrl+Shift binding without modifiers', () => {
    const handler = vi.fn()
    const wrapper = mount(
      makeComponent([{ key: 'p', ctrl: true, shift: true, handler }])
    )
    dispatchKey({ key: 'p' })
    expect(handler).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('suppresses single-key shortcuts when typing in an INPUT', () => {
    const handler = vi.fn()
    const wrapper = mount(
      makeComponent([{ key: '1', handler }])
    )
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()

    const evt = new KeyboardEvent('keydown', { key: '1', bubbles: true, cancelable: true })
    Object.defineProperty(evt, 'target', { value: input, writable: false })
    window.dispatchEvent(evt)

    expect(handler).not.toHaveBeenCalled()
    document.body.removeChild(input)
    wrapper.unmount()
  })

  it('fires allowInInputs binding even inside an INPUT', () => {
    const handler = vi.fn()
    const wrapper = mount(
      makeComponent([{ key: 'Escape', allowInInputs: true, handler }])
    )
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()

    const evt = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true })
    Object.defineProperty(evt, 'target', { value: input, writable: false })
    window.dispatchEvent(evt)

    expect(handler).toHaveBeenCalledOnce()
    document.body.removeChild(input)
    wrapper.unmount()
  })

  it('higher-priority scope (later mounted) handles first; lower scope does NOT', () => {
    const lowHandler = vi.fn()
    const highHandler = vi.fn()

    // Low-priority scope (mounted first)
    const LowComp = makeComponent([{ key: 'q', handler: lowHandler }])
    // High-priority scope (mounted second → top of stack)
    const HighComp = makeComponent([{ key: 'q', handler: highHandler }])

    const wrapperLow = mount(LowComp)
    const wrapperHigh = mount(HighComp)

    dispatchKey({ key: 'q' })

    expect(highHandler).toHaveBeenCalledOnce()
    expect(lowHandler).not.toHaveBeenCalled()

    wrapperHigh.unmount()
    wrapperLow.unmount()
  })

  it('after unmount the binding is no longer active', () => {
    const handler = vi.fn()
    const wrapper = mount(makeComponent([{ key: 'z', handler }]))

    dispatchKey({ key: 'z' })
    expect(handler).toHaveBeenCalledOnce()

    wrapper.unmount()
    handler.mockReset()

    dispatchKey({ key: 'z' })
    expect(handler).not.toHaveBeenCalled()
  })

  it('normalises "Space" alias to the space key', () => {
    const handler = vi.fn()
    const wrapper = mount(makeComponent([{ key: 'Space', handler }]))
    dispatchKey({ key: ' ' })
    expect(handler).toHaveBeenCalledOnce()
    wrapper.unmount()
  })
})
