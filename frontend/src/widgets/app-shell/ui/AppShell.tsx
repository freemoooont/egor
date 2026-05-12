import { reatomComponent } from '@reatom/react';
import { urlAtom, wrap } from '@reatom/core';
import { ChevronDown, LogOut, Plus, Settings, User as UserIcon } from 'lucide-react';
import type { ReactNode } from 'react';

import { ROUTES } from '@/shared/config/index.ts';
import { clearSession } from '@/shared/auth/index.ts';
import { Link } from '@/shared/router/index.ts';
import {
  Avatar,
  AvatarFallback,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Logo,
} from '@/shared/ui/index.ts';
import { currentUserAtom, userInitial } from '@/entities/user/index.ts';

interface AppShellProps {
  children: ReactNode;
}

const CHROMELESS_PATHS = new Set(['/login', '/register']);

/**
 * AppShell — outer chrome reused on every routed screen.
 *
 * Auth routes (`/login`, `/register`) render *without* a header — they paint
 * the full viewport themselves via the centered `AuthCard`. All other routes
 * get a header with the brand logo, an orange "+ create deck" CTA, and an
 * avatar dropdown that links to /account and signs the user out.
 */
export const AppShell = reatomComponent<AppShellProps>(({ children }) => {
  const path = urlAtom().pathname;
  if (CHROMELESS_PATHS.has(path)) {
    return <>{children}</>;
  }
  const user = currentUserAtom.data() ?? null;
  const initial = userInitial(user);

  return (
    <div className="min-h-full bg-[var(--color-app-bg)] text-foreground flex flex-col">
      <header className="bg-background">
        <div className="mx-auto flex w-full max-w-[1320px] items-center justify-between px-4 py-3 sm:px-6">
          <Link to={ROUTES.decks} className="inline-flex items-center" aria-label="На главную">
            <Logo iconOnly />
          </Link>
          <div className="flex items-center gap-3">
            <Link
              to={ROUTES.deckNew}
              aria-label="Создать колоду"
              className="inline-flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-[var(--color-brand-400)] to-[var(--color-brand-600)] text-white shadow-[0_4px_12px_rgba(255,143,45,0.35)] transition-transform hover:scale-105"
            >
              <Plus className="h-5 w-5" strokeWidth={3} />
            </Link>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className="inline-flex items-center gap-1 rounded-full p-1 text-[var(--color-ink)] outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  aria-label="Меню профиля"
                >
                  <Avatar className="h-8 w-8">
                    <AvatarFallback className="bg-[var(--color-field-bg)] text-[var(--color-ink)] text-xs font-semibold">
                      {initial}
                    </AvatarFallback>
                  </Avatar>
                  <ChevronDown className="h-4 w-4 text-[var(--color-ink-muted)]" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuItem
                  onSelect={wrap(() => {
                    urlAtom.go(ROUTES.account);
                  })}
                >
                  <Settings className="h-4 w-4" />
                  Настройки аккаунта
                </DropdownMenuItem>
                <DropdownMenuItem
                  onSelect={wrap(() => {
                    urlAtom.go(ROUTES.decks);
                  })}
                >
                  <UserIcon className="h-4 w-4" />
                  Мои колоды
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onSelect={wrap(() => {
                    clearSession();
                    urlAtom.go(ROUTES.login);
                  })}
                  className="text-[var(--color-error)] focus:text-[var(--color-error)]"
                >
                  <LogOut className="h-4 w-4" />
                  Выйти из аккаунта
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>
      <main className="flex-1 w-full max-w-[1320px] mx-auto px-4 sm:px-6 pb-10">{children}</main>
    </div>
  );
}, 'AppShell');
