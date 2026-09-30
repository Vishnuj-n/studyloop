import { describe, it, expect, beforeEach } from 'vitest'
import { usePet } from './usePet'

describe('usePet Composable', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('initializes with default enabled state and default pet/skin', () => {
    const { petState, currentPet, currentSkin } = usePet()
    expect(petState.value.enabled).toBe(true)
    expect(currentPet.value.id).toBe('cat')
    expect(currentSkin.value.id).toBe('calico')
  })

  it('toggles pet enabled state', () => {
    const { petState, togglePet } = usePet()
    togglePet(false)
    expect(petState.value.enabled).toBe(false)
    togglePet(true)
    expect(petState.value.enabled).toBe(true)
  })

  it('updates skin correctly', () => {
    const { petState, currentSkin, setSkin } = usePet()
    setSkin('void')
    expect(petState.value.activeSkinId).toBe('void')
    expect(currentSkin.value.id).toBe('void')
  })

  it('updates and resets screen position', () => {
    const { petState, setPosition, resetPosition } = usePet()
    setPosition(120, 240)
    expect(petState.value.position).toEqual({ x: 120, y: 240 })

    resetPosition()
    expect(petState.value.position).toEqual({ x: null, y: null })
  })
})
