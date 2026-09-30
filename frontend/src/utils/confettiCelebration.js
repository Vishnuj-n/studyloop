/**
 * Zero-dependency celebratory confetti & particle burst utility for Vue 3 UI.
 * Spawns an ephemeral full-screen canvas with physics-driven particles, shimmer, and ribbons.
 */

export function triggerConfettiCelebration(options = {}) {
  const canvas = document.createElement('canvas')
  canvas.className = 'celebration-confetti-canvas'
  canvas.style.position = 'fixed'
  canvas.style.inset = '0'
  canvas.style.width = '100vw'
  canvas.style.height = '100vh'
  canvas.style.pointerEvents = 'none'
  canvas.style.zIndex = '999999'

  document.body.appendChild(canvas)
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    canvas.remove()
    return () => {}
  }

  let width = (canvas.width = window.innerWidth)
  let height = (canvas.height = window.innerHeight)

  const handleResize = () => {
    width = canvas.width = window.innerWidth
    height = canvas.height = window.innerHeight
  }
  window.addEventListener('resize', handleResize)

  const defaultPalettes = {
    bronze: ['#f59e0b', '#d97706', '#b45309', '#fef3c7', '#fbbf24'],
    silver: ['#38bdf8', '#0284c7', '#94a3b8', '#e2e8f0', '#0ea5e9'],
    archivist: ['#a855f7', '#8b5cf6', '#ec4899', '#f472b6', '#c084fc'],
    gold: ['#eab308', '#f59e0b', '#fbbf24', '#fef08a', '#ffffff', '#ca8a04'],
    mythic: ['#10b981', '#06b6d4', '#6366f1', '#ec4899', '#f59e0b', '#a855f7'],
    default: ['#6366f1', '#a855f7', '#ec4899', '#3b82f6', '#10b981', '#f59e0b'],
  }

  const tier = (options.tier || 'default').toLowerCase()
  let palette = defaultPalettes.default
  if (tier.includes('mythic') || tier.includes('sage')) palette = defaultPalettes.mythic
  else if (tier.includes('gold') || tier.includes('grandmaster') || tier.includes('paragon') || tier.includes('luminary')) palette = defaultPalettes.gold
  else if (tier.includes('archivist') || tier.includes('polymath')) palette = defaultPalettes.archivist
  else if (tier.includes('silver') || tier.includes('inquisitor') || tier.includes('scholar')) palette = defaultPalettes.silver
  else if (tier.includes('bronze') || tier.includes('apprentice')) palette = defaultPalettes.bronze

  const particles = []
  const count = options.particleCount || 100
  const originX = width / 2
  const originY = height * 0.42

  for (let i = 0; i < count; i++) {
    const angle = Math.random() * Math.PI * 2
    const speed = 4 + Math.random() * 12
    const size = 5 + Math.random() * 6
    const isRibbon = Math.random() > 0.65

    particles.push({
      x: originX,
      y: originY,
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed - 5 - Math.random() * 4,
      size,
      color: palette[Math.floor(Math.random() * palette.length)],
      rotation: Math.random() * 360,
      rotationSpeed: (Math.random() - 0.5) * 12,
      isRibbon,
      width: isRibbon ? size * 0.5 : size,
      height: isRibbon ? size * 2.2 : size,
      alpha: 1,
      decay: 0.008 + Math.random() * 0.008,
      gravity: 0.28,
      wobble: Math.random() * 10,
      wobbleSpeed: 0.1 + Math.random() * 0.1,
    })
  }

  let animationFrameId
  let active = true

  function render() {
    if (!active) return
    ctx.clearRect(0, 0, width, height)

    let aliveCount = 0

    for (let i = 0; i < particles.length; i++) {
      const p = particles[i]
      if (p.alpha <= 0.01) continue

      aliveCount++
      p.x += p.vx
      p.y += p.vy
      p.vy += p.gravity
      p.vx *= 0.98
      p.rotation += p.rotationSpeed
      p.wobble += p.wobbleSpeed
      p.alpha -= p.decay

      ctx.save()
      ctx.globalAlpha = Math.max(0, p.alpha)
      ctx.translate(p.x, p.y)
      ctx.rotate((p.rotation * Math.PI) / 180)

      const scaleX = Math.cos(p.wobble)
      ctx.scale(scaleX, 1)

      ctx.fillStyle = p.color
      ctx.fillRect(-p.width / 2, -p.height / 2, p.width, p.height)
      ctx.restore()
    }

    if (aliveCount > 0) {
      animationFrameId = requestAnimationFrame(render)
    } else {
      cleanup()
    }
  }

  function cleanup() {
    active = false
    if (animationFrameId) cancelAnimationFrame(animationFrameId)
    window.removeEventListener('resize', handleResize)
    if (canvas && canvas.parentNode) {
      canvas.remove()
    }
  }

  animationFrameId = requestAnimationFrame(render)

  // Safety fallback cleanup after 5 seconds
  const timer = setTimeout(cleanup, 5000)

  return () => {
    clearTimeout(timer)
    cleanup()
  }
}
