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

  it('manages pet and skin unlocks correctly', () => {
    const {
      petState,
      isPetUnlocked,
      isSkinUnlocked,
      unlockPet,
      unlockSkin,
      setPet,
      setSkin,
    } = usePet()

    // Mochi is free by default
    expect(isPetUnlocked('cat')).toBe(true)
    expect(isSkinUnlocked('cat', 'calico')).toBe(true)
    expect(isSkinUnlocked('cat', 'void')).toBe(true)

    // Dog is locked initially until unlocked
    expect(isPetUnlocked('dog')).toBe(false)
    setPet('dog')
    // Should NOT switch to locked pet
    expect(petState.value.activePetId).toBe('cat')

    // Unlock dog
    unlockPet('dog')
    expect(isPetUnlocked('dog')).toBe(true)
    setPet('dog')
    expect(petState.value.activePetId).toBe('dog')

    // Dog paid skin (husky) is locked initially
    expect(isSkinUnlocked('dog', 'husky')).toBe(false)
    setSkin('husky')
    expect(petState.value.activeSkinId).not.toBe('husky')

    // Unlock skin
    unlockSkin('dog', 'husky')
    expect(isSkinUnlocked('dog', 'husky')).toBe(true)
    setSkin('husky')
    expect(petState.value.activeSkinId).toBe('husky')
  })
})
