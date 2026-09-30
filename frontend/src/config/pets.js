// ponytail: single registry array; new pets and skins require zero template refactoring
export const PET_REGISTRY = [
  {
    id: 'cat',
    name: 'Mochi',
    description: 'A cozy feline companion who purrs when you study.',
    defaultSkin: 'calico',
    skins: [
      {
        id: 'calico',
        name: 'Calico (Classic)',
        primaryColor: '#F97316',
        secondaryColor: '#FFEDD5',
        accentColor: '#EA580C',
        isFree: true,
      },
      {
        id: 'void',
        name: 'Void Black',
        primaryColor: '#1E293B',
        secondaryColor: '#475569',
        accentColor: '#0F172A',
        isFree: false,
        priceCoins: 100,
      },
      {
        id: 'matcha',
        name: 'Matcha Green',
        primaryColor: '#16A34A',
        secondaryColor: '#DCFCE7',
        accentColor: '#15803D',
        isFree: false,
        priceCoins: 200,
      },
      {
        id: 'lavender',
        name: 'Lavender Dusk',
        primaryColor: '#8B5CF6',
        secondaryColor: '#EDE9FE',
        accentColor: '#7C3AED',
        isFree: false,
        priceCoins: 300,
      },
    ],
    actions: ['idle', 'blink', 'coffee', 'wiggle', 'cheer', 'sleep'],
  },
]

export const PET_MESSAGES = [
  "You're doing great! Keep going!",
  'One flashcard at a time!',
  'Remember to stay hydrated!',
  'Focus mode active.',
  'Deep work pays off. Keep it up!',
  'Consistent practice beats cramming!',
  'Nice momentum! Stay locked in.',
]
