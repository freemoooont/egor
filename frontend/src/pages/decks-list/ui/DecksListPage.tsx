import { reatomComponent } from '@reatom/react';
import { urlAtom, wrap } from '@reatom/core';
import { MoreVertical, Plus } from 'lucide-react';

import { ROUTES } from '@/shared/config/index.ts';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/shared/ui/index.ts';
import { decksListAtom, deleteDeckAction, type Deck } from '@/entities/deck/index.ts';

const DeckCard = reatomComponent<{ deck: Deck }>(({ deck }) => {
  return (
    <div className="group relative flex h-[180px] flex-col justify-between rounded-2xl border border-[var(--color-card-border)] bg-[var(--color-card-surface)] p-5 shadow-[0_2px_4px_var(--color-card-shadow)] transition-shadow hover:shadow-[0_4px_12px_rgba(53,29,22,0.08)]">
      <div className="flex items-start justify-between gap-2">
        <h3 className="line-clamp-2 text-[15px] font-semibold leading-tight text-[var(--color-ink)]">
          {deck.title}
        </h3>
      </div>
      <div className="flex items-end justify-between">
        <div className="flex items-center gap-2 text-[12px] text-[var(--color-ink-muted)]">
          <span>{deck.termsCount} Терминов</span>
          <span aria-hidden>•</span>
          <span>{deck.authorName}</span>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              aria-label="Действия с колодой"
              className="rounded-full p-1 text-[var(--color-ink-muted)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
            >
              <MoreVertical className="h-4 w-4" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-44">
            <DropdownMenuItem
              onSelect={wrap(() => {
                urlAtom.go(ROUTES.deckPractice(deck.id));
              })}
            >
              Изучать
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={wrap(() => {
                deleteDeckAction(deck.id);
              })}
              className="text-[var(--color-error)] focus:text-[var(--color-error)]"
            >
              Удалить
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}, 'DeckCard');

const EmptyState = reatomComponent(() => {
  return (
    <div className="flex flex-col items-center justify-center gap-4 rounded-2xl border border-dashed border-[var(--color-card-border)] bg-[var(--color-card-surface)] py-16 text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-[var(--color-brand-400)] to-[var(--color-brand-600)] text-white shadow-[0_4px_12px_rgba(255,143,45,0.35)]">
        <Plus className="h-7 w-7" strokeWidth={3} />
      </div>
      <div className="max-w-md space-y-1">
        <p className="text-base font-semibold text-[var(--color-ink)]">У вас пока нет колод</p>
        <p className="text-sm text-[var(--color-ink-muted)]">Создайте первую колоду, чтобы начать учиться.</p>
      </div>
      <a
        href={ROUTES.deckNew}
        className="inline-flex h-10 items-center gap-2 rounded-full bg-gradient-to-br from-[var(--color-brand-400)] to-[var(--color-brand-600)] px-5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(255,143,45,0.35)] transition-transform hover:scale-[1.02]"
      >
        <Plus className="h-4 w-4" strokeWidth={3} />
        Создать колоду
      </a>
    </div>
  );
}, 'DecksEmptyState');

export const DecksListPage = reatomComponent(() => {
  const status = decksListAtom.status();
  const decks: Deck[] = decksListAtom.data() ?? [];
  const isFirstPending = status.isPending && !status.isFulfilled;
  const errorMessage =
    'isRejected' in status && status.isRejected ? 'error' in status ? status.error : null : null;

  return (
    <section className="flex flex-col gap-6 py-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-[22px] font-bold leading-tight text-[var(--color-ink)]">
          Мои колоды{' '}
          <span className="text-[var(--color-brand-500)]">{decks.length}</span>
        </h1>
        <a
          href={ROUTES.deckNew}
          className="hidden sm:inline-flex h-10 items-center gap-2 rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-4 text-sm font-semibold text-[var(--color-ink)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
        >
          <Plus className="h-4 w-4 text-[var(--color-brand-500)]" strokeWidth={3} />
          Создать колоду
        </a>
      </header>

      {isFirstPending ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="h-[180px] animate-pulse rounded-2xl border border-[var(--color-card-border)] bg-[var(--color-card-surface)]"
            />
          ))}
        </div>
      ) : decks.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {decks.map((deck) => (
            <DeckCard key={deck.id} deck={deck} />
          ))}
        </div>
      )}

      {/* Mobile-only floating create button when list is non-empty */}
      {decks.length > 0 ? (
        <a
          href={ROUTES.deckNew}
          className="fixed bottom-6 right-6 z-30 inline-flex h-14 items-center gap-2 rounded-full bg-gradient-to-br from-[var(--color-brand-400)] to-[var(--color-brand-600)] px-5 text-sm font-semibold text-white shadow-[0_8px_24px_rgba(255,143,45,0.45)] sm:hidden"
        >
          <Plus className="h-5 w-5" strokeWidth={3} />
          Создать
        </a>
      ) : null}

      {errorMessage ? (
        <p role="alert" className="text-sm text-[var(--color-error)]">
          Не удалось загрузить колоды. Попробуйте обновить страницу.
        </p>
      ) : null}
    </section>
  );
}, 'DecksListPage');
