import { reatomComponent } from '@reatom/react';
import { wrap } from '@reatom/core';
import { urlAtom } from '@reatom/core';

import { ROUTES } from '@/app/routes.ts';
import { cn } from '@/shared/lib/index.ts';

interface AuthTabsProps {
  active: 'login' | 'register';
}

/**
 * Tabs «Вход / Регистрация» — Figma node `1:780`.
 * Switches via the router (no local state) so each tab is its own page.
 */
export const AuthTabs = reatomComponent<AuthTabsProps>(({ active }) => {
  return (
    <div
      role="tablist"
      aria-label="Auth"
      className="flex h-[48px] w-full items-stretch overflow-hidden rounded-[10px] border border-[var(--color-tab-border)] bg-white"
    >
      <button
        type="button"
        role="tab"
        aria-selected={active === 'login'}
        onClick={wrap(() => urlAtom.go(ROUTES.login))}
        className={cn(
          'flex-1 rounded-[10px] px-4 text-[16px] font-semibold transition-colors',
          active === 'login'
            ? 'bg-[var(--color-tab-active-bg)] text-[var(--color-ink)]'
            : 'text-[var(--color-ink-muted)] hover:text-[var(--color-ink)]',
        )}
      >
        Вход
      </button>
      <button
        type="button"
        role="tab"
        aria-selected={active === 'register'}
        onClick={wrap(() => urlAtom.go(ROUTES.register))}
        className={cn(
          'flex-1 rounded-[10px] px-4 text-[16px] font-semibold transition-colors',
          active === 'register'
            ? 'bg-[var(--color-tab-active-bg)] text-[var(--color-ink)]'
            : 'text-[var(--color-ink-muted)] hover:text-[var(--color-ink)]',
        )}
      >
        Регистрация
      </button>
    </div>
  );
}, 'AuthTabs');
