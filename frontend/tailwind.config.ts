import type { Config } from 'tailwindcss'

export default {
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'surface': {
          DEFAULT: '#0b1326',
          dim: '#0b1326',
          bright: '#31394d',
          'container-lowest': '#060e20',
          'container-low': '#131b2e',
          container: '#171f33',
          'container-high': '#222a3d',
          'container-highest': '#2d3449',
          variant: '#2d3449',
        },
        'primary': {
          DEFAULT: '#8ed5ff',
          container: '#38bdf8',
        },
        'tertiary': {
          DEFAULT: '#ffc176',
          container: '#f1a02b',
        },
        'error': {
          DEFAULT: '#ffb4ab',
          container: '#93000a',
        },
        'on-surface': '#dae2fd',
        'on-surface-variant': '#bdc8d1',
        'on-primary': '#00354a',
        'on-primary-container': '#004965',
        'on-tertiary': '#472a00',
        'on-tertiary-container': '#613b00',
        'outline': '#87929a',
        'outline-variant': '#3e484f',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
} satisfies Config
