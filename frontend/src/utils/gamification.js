/**
 * Central source of truth for gamification title tiers, emojis, and styling tokens.
 */
export const GAMIFICATION_TIERS = [
  {
    baseTitle: 'The Apprentice',
    minXP: 0,
    emoji: '🧑‍🎓',
    description: 'Beginning the study loop journey. Mastering initial habits and flashcards.',
    themeClass: 'tier-bronze',
  },
  {
    baseTitle: 'The Scholar',
    minXP: 1500,
    emoji: '🦁',
    description: 'Demonstrating routine study consistency and subject retention.',
    themeClass: 'tier-silver',
  },
  {
    baseTitle: 'The Inquisitor',
    minXP: 4500,
    emoji: '🦚',
    description: 'Relentlessly probing deeper into notes, concepts, and quizzes.',
    themeClass: 'tier-silver',
  },
  {
    baseTitle: 'The Archivist',
    minXP: 9000,
    emoji: '🦄',
    description: 'Curating a vast vault of connected knowledge and study streaks.',
    themeClass: 'tier-archivist',
  },
  {
    baseTitle: 'The Polymath',
    minXP: 16000,
    emoji: '🦖',
    description: 'Interconnecting multiple subjects and achieving mastery across disciplines.',
    themeClass: 'tier-archivist',
  },
  {
    baseTitle: 'The Grandmaster',
    minXP: 26000,
    emoji: '🐦‍🔥',
    description: 'Elite mastery of recall, problem solving, and long-term retention.',
    themeClass: 'tier-gold',
  },
  {
    baseTitle: 'The Paragon',
    minXP: 40000,
    emoji: '🤴',
    description: 'A beacon of academic dedication and unstoppable momentum.',
    themeClass: 'tier-gold',
  },
  {
    baseTitle: 'The Luminary',
    minXP: 60000,
    emoji: '🦸',
    description: 'Enlightened intellectual prowess. Unrivaled consistency.',
    themeClass: 'tier-gold',
  },
  {
    baseTitle: 'The Mythic Sage',
    minXP: 90000,
    emoji: '🐉',
    description: 'The pinnacle of academic ascension. A legendary scholar of the ages.',
    themeClass: 'tier-mythic',
  },
]

export function getTitleEmoji(title) {
  const t = (title || '').toLowerCase()
  if (t.includes('mythic') || t.includes('sage')) return '🐉'
  if (t.includes('luminary')) return '🦸'
  if (t.includes('paragon')) return '🤴'
  if (t.includes('grandmaster')) return '🐦‍🔥'
  if (t.includes('polymath')) return '🦖'
  if (t.includes('archivist')) return '🦄'
  if (t.includes('inquisitor')) return '🦚'
  if (t.includes('scholar')) return '🦁'
  if (t.includes('apprentice') || t.includes('novice')) return '🧑‍🎓'
  return '🧑‍🎓'
}

export function getTierThemeClass(title) {
  const t = (title || '').toLowerCase()
  if (t.includes('mythic') || t.includes('sage')) return 'tier-mythic'
  if (t.includes('grandmaster') || t.includes('paragon') || t.includes('luminary')) return 'tier-gold'
  if (t.includes('archivist') || t.includes('polymath')) return 'tier-archivist'
  if (t.includes('inquisitor') || t.includes('scholar')) return 'tier-silver'
  return 'tier-bronze'
}
