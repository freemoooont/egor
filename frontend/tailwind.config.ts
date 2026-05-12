import type { Config } from 'tailwindcss';

/**
 * Tailwind v4 — design tokens for Micocards.
 *
 * Brand orange is the load-bearing colour from `docs/design.md`.
 * Until the actual screens are implemented and Figma tokens captured,
 * we use the canonical Tailwind `orange-500` (#F97316) as the brand
 * primary — this can be tightened to the exact hex after design pass.
 *
 * NOTE: With Tailwind v4 the bulk of token wiring lives in CSS
 * (`@theme` block in `src/app/styles/global.css`). This file is kept
 * for IDE intellisense and to surface the brand intent in code.
 */
const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#FFF7ED',
          100: '#FFEDD5',
          200: '#FED7AA',
          300: '#FDBA74',
          400: '#FF8F2D',
          500: '#FF8F2D',
          600: '#EA580C',
          700: '#C2410C',
          800: '#9A3412',
          900: '#7C2D12',
        },
      },
      borderRadius: {
        sm: 'calc(var(--radius) - 4px)',
        md: 'calc(var(--radius) - 2px)',
        lg: 'var(--radius)',
        xl: 'calc(var(--radius) + 4px)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
};

export default config;
