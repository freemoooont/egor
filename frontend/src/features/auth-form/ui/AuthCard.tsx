import { reatomComponent } from '@reatom/react';
import type { ReactNode } from 'react';

import { cn } from '@/shared/lib/index.ts';
import { Logo } from '@/shared/ui/index.ts';

interface AuthCardProps {
  children: ReactNode;
  className?: string;
}

/**
 * Centered auth card — mirrors Figma container `1:675` / `1:720`.
 *
 * - Desktop: 464×504 white card on a peach gradient background, centered.
 * - Mobile: full-width frame, gradient hidden, top-aligned.
 *
 * Logo lives at the top of the card on desktop and at the top of the page on
 * mobile (via the same `<Logo />` slot inside the card; on mobile we drop the
 * card chrome).
 */
export const AuthCard = reatomComponent<AuthCardProps>(({ children, className }) => {
  return (
    <div
      className={cn(
        // mobile (default): white viewport, content top-aligned, full bleed
        'relative flex min-h-[100dvh] w-full flex-col bg-white px-4 py-6',
        // desktop: peach gradient + centered card layout
        'sm:items-center sm:justify-center sm:bg-[var(--color-background)] sm:px-6',
      )}
      style={{
        // gradient applied via inline style so we can keep it desktop-only via @media query in CSS
        // (Tailwind v4 has no first-class arbitrary background-image with breakpoint variants)
      }}
    >
      <style>{`@media (min-width: 640px) { .auth-bg { background-image: linear-gradient(153.21deg, rgb(247, 217, 192) 13.09%, rgb(255, 243, 233) 86.48%); } }`}</style>
      <div className="auth-bg pointer-events-none absolute inset-0 z-0" aria-hidden="true" />
      <div
        className={cn(
          // shared — column with logo on top, children stretching to fill height
          'relative z-10 flex w-full max-w-[464px] flex-1 flex-col gap-7',
          // mobile: no card chrome, no padding (page already provides it)
          'p-0',
          // desktop: white card, padding 24px, radii 35/26, soft drop shadow,
          // fixed height (504px), `flex-none` to prevent it stretching the
          // outer flex container.
          'sm:h-[504px] sm:flex-none sm:rounded-t-[35px] sm:rounded-b-[26px] sm:bg-white sm:p-6 sm:shadow-[0px_0px_25px_0px_rgba(27,27,27,0.1)]',
          className,
        )}
      >
        <div className="flex w-full items-center">
          <Logo className="h-[32px]" />
        </div>
        {children}
      </div>
    </div>
  );
}, 'AuthCard');
