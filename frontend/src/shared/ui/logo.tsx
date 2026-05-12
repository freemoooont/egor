import { reatomComponent } from '@reatom/react';

import { cn } from '@/shared/lib/index.ts';
import { APP_NAME } from '@/shared/config/index.ts';

interface LogoProps {
  className?: string;
  /** Hide the wordmark; show only the icon. */
  iconOnly?: boolean;
}

/**
 * Brand logo — orange flame-shaped mark + "Micocards" wordmark.
 * Approximates the Figma raster (1:680 / 1:725) — the auth screens render it
 * inline so the whole card has no remote-asset dependency.
 */
export const Logo = reatomComponent<LogoProps>(({ className, iconOnly = false }) => {
  return (
    <span className={cn('inline-flex items-center gap-2', className)}>
      <svg
        width="36"
        height="26"
        viewBox="0 0 36 26"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
        className="shrink-0"
      >
        <path
          d="M3 13c0-6 4-11 10-11 5 0 8 3 9 6 1-2 3-3 5-3 4 0 6 3 6 7 0 6-5 11-13 13-9 2-17-3-17-12z"
          fill="var(--color-brand-500)"
        />
        <text
          x="18"
          y="17"
          textAnchor="middle"
          fontFamily="Inter, system-ui, sans-serif"
          fontWeight="700"
          fontSize="9"
          fill="#FFFFFF"
        >
          CoC
        </text>
      </svg>
      {!iconOnly ? (
        <span className="text-[14px] font-bold leading-none text-[var(--color-ink)]">
          {APP_NAME}
        </span>
      ) : null}
    </span>
  );
}, 'Logo');
