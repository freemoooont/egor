/**
 * Brand tokens — single source of truth for the Micocards orange.
 *
 * These constants exist for non-CSS contexts (PWA manifest, head meta tags,
 * canvas drawing). For component styling reach for Tailwind tokens via the
 * `bg-brand-500` / `text-brand-500` etc. utility classes (configured in
 * `tailwind.config.ts` and `src/app/styles/global.css`).
 */
export const APP_NAME = 'Micocards';
export const APP_SHORT_NAME = 'Micocards';

export const BRAND_ORANGE = {
  /** Tightened from Figma node 1:759 (filled "Продолжить" CTA). */
  hex: '#FF8F2D',
  hsl: '27 100% 59%',
} as const;

export const BRAND_NEUTRAL = {
  white: '#FFFFFF',
  black: '#0F172A',
} as const;
