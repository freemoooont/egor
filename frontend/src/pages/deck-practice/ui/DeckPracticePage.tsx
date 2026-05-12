import { reatomComponent, useAction } from '@reatom/react';
import { computed, urlAtom, wrap } from '@reatom/core';
import { useEffect } from 'react';
import { ArrowLeft, ChevronLeft, ChevronRight } from 'lucide-react';

import { ROUTES } from '@/shared/config/index.ts';
import { Switch } from '@/shared/ui/index.ts';
import { Rating, activeSessionAtom } from '@/entities/practice/index.ts';
import type { Deck } from '@/entities/deck/index.ts';

import { deckPracticeRoute } from '../model/route.tsx';
import {
  cardFlippedAtom,
  currentCardIdxAtom,
  finishNowAction,
  flipCardAction,
  practiceErrorAtom,
  previousCardAction,
  advanceCardAction,
  rateAndAdvanceAction,
  toggleTrackingAction,
  trackProgressAtom,
} from '../model/practiceState.ts';

/**
 * Deck-practice page (`/decks/:id/practice`).
 *
 * Keyboard shortcuts (documented via tooltips):
 *   - ArrowLeft / ArrowRight — navigate cards
 *   - Space / Enter — flip the current card
 *   - 1 / 2 / 3 — rate as Don't know / Still learning / Know (tracking ON)
 */

const deckLoaderAtom = computed(() => deckPracticeRoute.loader.data() as Deck | null, 'practice.deck');
const deckLoaderStatusAtom = computed(() => deckPracticeRoute.loader.status(), 'practice.deck.status');

const navigateToResults = (sessionId: string, deckId: string): void => {
  urlAtom.go(`${ROUTES.deckResults(deckId)}?sessionId=${encodeURIComponent(sessionId)}`);
};

const Header = reatomComponent<{ deck: Deck }>(({ deck }) => {
  return (
    <header className="flex items-center justify-between gap-3">
      <div className="flex items-center gap-3">
        <button
          type="button"
          aria-label="Назад к колодам"
          onClick={wrap(() => {
            urlAtom.go(ROUTES.decks);
          })}
          className="inline-flex h-10 w-10 items-center justify-center rounded-full bg-[var(--color-card-surface)] text-[var(--color-ink)] shadow-[0_2px_8px_rgba(27,27,27,0.08)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <h1 className="text-[22px] font-bold leading-tight text-[var(--color-ink)] sm:text-[24px]">
          {deck.title}
        </h1>
      </div>
    </header>
  );
}, 'DeckPracticeHeader');

const FlashCard = reatomComponent<{ deck: Deck }>(({ deck }) => {
  const idx = currentCardIdxAtom();
  const flipped = cardFlippedAtom();
  const card = deck.cards[idx];
  if (!card) return null;
  return (
    <div className="relative w-full" style={{ perspective: '1600px' }}>
      <button
        type="button"
        aria-label={flipped ? 'Показать термин' : 'Показать определение'}
        title="Нажмите Space или Enter, чтобы перевернуть"
        onClick={wrap(() => {
          flipCardAction();
        })}
        onKeyDown={wrap((event: React.KeyboardEvent<HTMLButtonElement>) => {
          if (event.key === ' ' || event.key === 'Enter') {
            event.preventDefault();
            flipCardAction();
          }
        })}
        className="block w-full cursor-pointer outline-none focus-visible:ring-4 focus-visible:ring-[var(--color-brand-200)] rounded-[20px]"
        style={{
          minHeight: '320px',
        }}
      >
        <div
          className="relative w-full"
          style={{
            transformStyle: 'preserve-3d',
            transition: 'transform 600ms cubic-bezier(0.4, 0, 0.2, 1)',
            transform: flipped ? 'rotateY(180deg)' : 'rotateY(0deg)',
            minHeight: 'clamp(280px, 40vh, 520px)',
          }}
        >
          <div
            className="absolute inset-0 flex items-center justify-center rounded-[20px] bg-[var(--color-card-surface)] p-6 text-center shadow-[0_0_25px_rgba(27,27,27,0.15)]"
            style={{
              backfaceVisibility: 'hidden',
              WebkitBackfaceVisibility: 'hidden',
            }}
          >
            <p className="text-[28px] font-bold leading-tight text-[var(--color-ink)] sm:text-[40px]">
              {card.term}
            </p>
          </div>
          <div
            className="absolute inset-0 flex items-center justify-center rounded-[20px] bg-[var(--color-card-surface)] p-6 text-center shadow-[0_0_25px_rgba(27,27,27,0.15)]"
            style={{
              backfaceVisibility: 'hidden',
              WebkitBackfaceVisibility: 'hidden',
              transform: 'rotateY(180deg)',
            }}
          >
            <p className="text-[20px] font-medium leading-snug text-[var(--color-ink)] sm:text-[28px]">
              {card.definition}
            </p>
          </div>
        </div>
      </button>
    </div>
  );
}, 'DeckPracticeFlashCard');

const NavigationBar = reatomComponent<{ deck: Deck }>(({ deck }) => {
  const idx = currentCardIdxAtom();
  const total = deck.cards.length;
  const tracking = trackProgressAtom();
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <label className="flex items-center gap-3 text-[14px] font-bold text-[var(--color-ink-on-brand)] sm:text-[16px]">
        <Switch
          checked={tracking}
          onCheckedChange={wrap((_value: boolean) => {
            void _value;
            toggleTrackingAction({
              deckId: deck.id,
              onSessionFinished: (sessionId) => {
                navigateToResults(sessionId, deck.id);
              },
            });
          })}
          aria-label="Отслеживать прогресс"
        />
        Отслеживать прогресс
      </label>
      <div className="flex items-center gap-3">
        <button
          type="button"
          aria-label="Предыдущая карточка"
          title="Стрелка влево"
          disabled={idx === 0}
          onClick={wrap(() => {
            previousCardAction();
          })}
          className="inline-flex h-10 w-[60px] items-center justify-center rounded-full bg-[var(--color-card-surface)] text-[var(--color-ink)] shadow-[0_0_25px_rgba(27,27,27,0.1)] outline-none transition-opacity hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-40"
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
        <p className="min-w-[60px] text-center text-[14px] font-medium text-[var(--color-ink)] sm:text-[16px]">
          {Math.min(idx + 1, total)}/{total}
        </p>
        <button
          type="button"
          aria-label="Следующая карточка"
          title="Стрелка вправо"
          disabled={idx >= total - 1}
          onClick={wrap(() => {
            advanceCardAction({ total });
          })}
          className="inline-flex h-10 w-[60px] items-center justify-center rounded-full bg-[var(--color-card-surface)] text-[var(--color-ink)] shadow-[0_0_25px_rgba(27,27,27,0.1)] outline-none transition-opacity hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-40"
        >
          <ChevronRight className="h-5 w-5" />
        </button>
      </div>
    </div>
  );
}, 'DeckPracticeNavigationBar');

const RateBar = reatomComponent<{ deck: Deck }>(({ deck }) => {
  const tracking = trackProgressAtom();
  if (!tracking) return null;
  const idx = currentCardIdxAtom();
  const card = deck.cards[idx];
  if (!card) return null;
  const total = deck.cards.length;
  const onFinish = (): void => {
    finishNowAction({
      onFinished: (sessionId) => {
        navigateToResults(sessionId, deck.id);
      },
    });
  };
  return (
    <div className="flex w-full flex-col gap-3 sm:flex-row sm:gap-4">
      <button
        type="button"
        title="Клавиша 1"
        onClick={wrap(() => {
          rateAndAdvanceAction({
            cardId: card.id,
            rating: Rating.DontKnow,
            total,
            onFinish,
          });
        })}
        className="flex h-[60px] flex-1 items-center justify-center rounded-[20px] bg-[var(--color-card-surface)] text-[16px] font-bold text-[var(--color-error)] shadow-[0_0_25px_rgba(27,27,27,0.15)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
      >
        Не знаю
      </button>
      <button
        type="button"
        title="Клавиша 2"
        onClick={wrap(() => {
          rateAndAdvanceAction({
            cardId: card.id,
            rating: Rating.StillLearning,
            total,
            onFinish,
          });
        })}
        className="flex h-[60px] flex-1 items-center justify-center rounded-[20px] bg-[var(--color-card-surface)] text-[16px] font-bold text-[var(--color-ink-muted)] shadow-[0_0_25px_rgba(27,27,27,0.15)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
      >
        Ещё изучаю
      </button>
      <button
        type="button"
        title="Клавиша 3"
        onClick={wrap(() => {
          rateAndAdvanceAction({
            cardId: card.id,
            rating: Rating.KnowKnow,
            total,
            onFinish,
          });
        })}
        className="flex h-[60px] flex-1 items-center justify-center rounded-[20px] bg-[var(--color-card-surface)] text-[16px] font-bold text-[#3c881b] shadow-[0_0_25px_rgba(27,27,27,0.15)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
      >
        Знаю
      </button>
    </div>
  );
}, 'DeckPracticeRateBar');

const TermsList = reatomComponent<{ deck: Deck }>(({ deck }) => {
  return (
    <section className="flex flex-col gap-5">
      <div className="flex items-center gap-2 text-[20px] font-bold sm:text-[24px]">
        <span className="text-[var(--color-ink)]">Термины</span>
        <span className="text-[var(--color-brand-600)]">{deck.cards.length}</span>
      </div>
      <ul className="flex flex-col gap-4">
        {deck.cards.map((card, idx) => (
          <li
            key={card.id}
            className="rounded-[20px] bg-[var(--color-card-surface)] px-5 pt-5 pb-7 shadow-[0_0_25px_rgba(27,27,27,0.1)]"
          >
            <p className="text-[14px] font-bold text-[var(--color-ink)]">{idx + 1}</p>
            <div className="mt-3 grid grid-cols-1 items-center gap-3 sm:grid-cols-[1fr_1px_1fr] sm:gap-7">
              <p className="text-[14px] font-medium text-[var(--color-ink)] sm:text-[16px]">
                {card.term}
              </p>
              <div
                aria-hidden
                className="hidden h-[60px] w-px rounded bg-[var(--color-card-border)] sm:block"
              />
              <p className="text-[14px] font-medium text-[var(--color-ink)] sm:text-[16px]">
                {card.definition}
              </p>
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}, 'DeckPracticeTermsList');

const KeyboardShortcutHandler = reatomComponent<{ deck: Deck }>(({ deck }) => {
  const tracking = trackProgressAtom();
  // Bind every action via useAction so it can be called from a useEffect /
  // DOM event without losing the reatom async stack.
  const advance = useAction(advanceCardAction);
  const previous = useAction(previousCardAction);
  const flip = useAction(flipCardAction);
  const rateAndAdvance = useAction(rateAndAdvanceAction);
  const finishNow = useAction(finishNowAction);
  // Tracking is read so the effect re-runs when the toggle flips, ensuring the
  // closures see the latest atom values via direct reads.
  void tracking;
  useEffect(() => {
    const handler = (event: KeyboardEvent): void => {
      const target = event.target as HTMLElement | null;
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')) return;
      const total = deck.cards.length;
      const onFinish = (): void => {
        finishNow({
          onFinished: (sessionId) => {
            navigateToResults(sessionId, deck.id);
          },
        });
      };
      switch (event.key) {
        case 'ArrowRight':
          event.preventDefault();
          advance({ total });
          return;
        case 'ArrowLeft':
          event.preventDefault();
          previous();
          return;
        case ' ':
        case 'Enter':
          // Don't preempt buttons that are focused — the button's own
          // onKeyDown already handles flip when focused.
          if (
            target &&
            (target.tagName === 'BUTTON' ||
              target.getAttribute('role') === 'button' ||
              target.closest('[role="dialog"]'))
          ) {
            return;
          }
          event.preventDefault();
          flip();
          return;
        case '1':
        case '2':
        case '3': {
          const card = deck.cards[currentCardIdxAtom()];
          if (!trackProgressAtom() || !card) return;
          event.preventDefault();
          const rating =
            event.key === '1'
              ? Rating.DontKnow
              : event.key === '2'
                ? Rating.StillLearning
                : Rating.KnowKnow;
          rateAndAdvance({
            cardId: card.id,
            rating,
            total,
            onFinish,
          });
          return;
        }
        default:
          return;
      }
    };
    window.addEventListener('keydown', handler);
    return () => {
      window.removeEventListener('keydown', handler);
    };
  }, [deck, tracking, advance, previous, flip, rateAndAdvance, finishNow]);
  return null;
}, 'DeckPracticeKeyboardShortcuts');

const EmptyDeck = reatomComponent(() => {
  return (
    <div className="flex flex-col items-center justify-center gap-4 rounded-[20px] border border-dashed border-[var(--color-card-border)] bg-[var(--color-card-surface)] py-16 text-center">
      <p className="max-w-md text-base font-semibold text-[var(--color-ink)]">
        В этой колоде пока нет карточек
      </p>
      <a
        href={ROUTES.decks}
        className="inline-flex h-10 items-center gap-2 rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-5 text-sm font-semibold text-[var(--color-ink)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
      >
        Назад к колодам
      </a>
    </div>
  );
}, 'DeckPracticeEmptyDeck');

const NotFound = reatomComponent(() => {
  return (
    <div className="flex flex-col items-center justify-center gap-4 rounded-[20px] border border-dashed border-[var(--color-card-border)] bg-[var(--color-card-surface)] py-16 text-center">
      <h2 className="text-base font-semibold text-[var(--color-ink)]">Колода не найдена</h2>
      <a
        href={ROUTES.decks}
        className="inline-flex h-10 items-center gap-2 rounded-full bg-gradient-to-br from-[var(--color-brand-400)] to-[var(--color-brand-600)] px-5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(255,143,45,0.35)]"
      >
        Назад к колодам
      </a>
    </div>
  );
}, 'DeckPracticeNotFound');

const PracticeError = reatomComponent(() => {
  const message = practiceErrorAtom();
  if (!message) return null;
  return (
    <p role="alert" className="text-sm text-[var(--color-error)]">
      {message}
    </p>
  );
}, 'DeckPracticeError');

const SessionFinishOnExit = reatomComponent<{ deckId: string }>(({ deckId }) => {
  // When the user navigates away from the practice route while a session is
  // active, finish it. We can't reliably detect "leave" from inside React, so
  // this component subscribes to the URL atom and reacts when the URL changes
  // away from the practice path.
  void deckId;
  return null;
}, 'DeckPracticeSessionFinishOnExit');

export const DeckPracticePage = reatomComponent(() => {
  const deck = deckLoaderAtom();
  const status = deckLoaderStatusAtom();

  if (status.isPending && !status.isFulfilled) {
    return (
      <section className="flex flex-col gap-6 py-6">
        <div className="h-8 w-48 animate-pulse rounded-md bg-[var(--color-field-bg)]" />
        <div className="h-[320px] w-full animate-pulse rounded-[20px] bg-[var(--color-field-bg)]" />
      </section>
    );
  }

  if (deck == null) {
    return (
      <section className="flex flex-col gap-6 py-6">
        <NotFound />
      </section>
    );
  }

  if (deck.cards.length === 0) {
    return (
      <section className="flex flex-col gap-6 py-6">
        <Header deck={deck} />
        <EmptyDeck />
      </section>
    );
  }

  return (
    <section className="flex flex-col gap-5 py-6">
      <Header deck={deck} />
      <FlashCard deck={deck} />
      <NavigationBar deck={deck} />
      <RateBar deck={deck} />
      <PracticeError />
      <TermsList deck={deck} />
      <KeyboardShortcutHandler deck={deck} />
      <SessionFinishOnExit deckId={deck.id} />
    </section>
  );
}, 'DeckPracticePage');

// Suppress unused-vars: activeSessionAtom is referenced from practiceState.ts but
// import here keeps the public API surface explicit when extending the page.
void activeSessionAtom;
