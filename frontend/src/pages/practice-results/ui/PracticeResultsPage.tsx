import { reatomComponent } from '@reatom/react';
import { computed, urlAtom, wrap } from '@reatom/core';

import { ROUTES } from '@/shared/config/index.ts';
import {
  reatomPracticeResults,
  type PracticeResults,
} from '@/entities/practice/index.ts';

import { practiceResultsRoute } from '../model/route.tsx';
import { DonutChart } from './DonutChart.tsx';

/**
 * Practice-results page (`/decks/:id/results?sessionId=...`).
 *
 * - Reads the sessionId from the route's typed search.
 * - 404 state if no sessionId or the API returns 404.
 * - Heading + circular progress chart + legend + two action buttons.
 */

const sessionIdAtom = computed((): string | null => {
  const params = practiceResultsRoute() as { id?: string; sessionId?: string } | null;
  if (!params) return null;
  const sessionId = params.sessionId ?? null;
  return sessionId && sessionId.length > 0 ? sessionId : null;
}, 'practiceResults.sessionId');

const deckIdAtom = computed((): string | null => {
  const params = practiceResultsRoute() as { id?: string; sessionId?: string } | null;
  return params?.id ?? null;
}, 'practiceResults.deckId');

const resultsAtom = reatomPracticeResults(() => sessionIdAtom());

const NotFound = reatomComponent(() => {
  return (
    <div className="flex flex-col items-center justify-center gap-4 rounded-[20px] border border-dashed border-[var(--color-card-border)] bg-[var(--color-card-surface)] py-16 text-center">
      <h2 className="text-base font-semibold text-[var(--color-ink)]">Сессия не найдена</h2>
      <p className="max-w-md text-sm text-[var(--color-ink-muted)]">
        Не удалось загрузить результаты. Возможно, ссылка устарела.
      </p>
      <a
        href={ROUTES.decks}
        className="inline-flex h-10 items-center gap-2 rounded-full bg-gradient-to-br from-[var(--color-brand-400)] to-[var(--color-brand-600)] px-5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(255,143,45,0.35)]"
      >
        Назад к колодам
      </a>
    </div>
  );
}, 'PracticeResultsNotFound');

interface LegendItemProps {
  label: string;
  count: number;
  bg: string;
  fg: string;
}

const LegendItem = reatomComponent<LegendItemProps>(({ label, count, bg, fg }) => {
  return (
    <div
      className="flex items-center justify-between rounded-[20px] px-4 py-2.5 text-[14px] font-semibold sm:text-[16px]"
      style={{ backgroundColor: bg, color: fg }}
    >
      <span>{label}</span>
      <span>{count}</span>
    </div>
  );
}, 'PracticeResultsLegendItem');

const ResultsCard = reatomComponent<{ results: PracticeResults; deckId: string }>(
  ({ results, deckId }) => {
    const onPracticeAgain = (): void => {
      urlAtom.go(ROUTES.deckPractice(deckId));
    };
    const onHome = (): void => {
      urlAtom.go(ROUTES.decks);
    };
    return (
      <div className="rounded-[20px] bg-[var(--color-card-surface)] px-5 py-7 shadow-[0_0_25px_rgba(27,27,27,0.1)] sm:px-7">
        <h2 className="text-[16px] font-semibold text-[var(--color-ink)] sm:text-[18px]">
          Ваши успехи
        </h2>
        <div className="mt-6 flex flex-col items-center gap-8 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex shrink-0 items-center justify-center">
            <DonutChart
              knowCount={results.knowCount}
              learningCount={results.learningCount}
              dontKnowCount={results.dontKnowCount}
            />
          </div>
          <div className="flex w-full max-w-[320px] flex-col gap-3 lg:max-w-[260px]">
            <LegendItem
              label="Знаю"
              count={results.knowCount}
              bg="rgba(255, 143, 45, 0.18)"
              fg="var(--color-brand-700)"
            />
            <LegendItem
              label="Ещё изучаю"
              count={results.learningCount}
              bg="rgba(253, 186, 116, 0.25)"
              fg="#a87e21"
            />
            <LegendItem
              label="Не знаю"
              count={results.dontKnowCount}
              bg="rgba(120, 113, 108, 0.15)"
              fg="var(--color-ink-muted)"
            />
          </div>
          <div className="flex w-full flex-col gap-4 lg:w-[240px] lg:items-stretch lg:self-center">
            <button
              type="button"
              onClick={wrap(() => {
                onPracticeAgain();
              })}
              className="inline-flex h-12 items-center justify-center rounded-full bg-gradient-to-br from-[var(--color-brand-400)] to-[var(--color-brand-600)] px-6 text-sm font-bold text-white shadow-[0_4px_12px_rgba(255,143,45,0.35)] outline-none transition-transform hover:scale-[1.02] focus-visible:ring-2 focus-visible:ring-ring"
            >
              Пройти карточки заново
            </button>
            <button
              type="button"
              onClick={wrap(() => {
                onHome();
              })}
              className="inline-flex h-12 items-center justify-center rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-6 text-sm font-bold text-[var(--color-ink-on-brand)] shadow-[0_2px_8px_rgba(27,27,27,0.06)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
            >
              На главную
            </button>
          </div>
        </div>
      </div>
    );
  },
  'PracticeResultsCard',
);

export const PracticeResultsPage = reatomComponent(() => {
  const sessionId = sessionIdAtom();
  const deckId = deckIdAtom();
  const status = resultsAtom.status();
  const results = resultsAtom.data();

  if (sessionId === null || deckId === null) {
    return (
      <section className="flex flex-col gap-6 py-6">
        <NotFound />
      </section>
    );
  }

  if (status.isPending && !status.isFulfilled) {
    return (
      <section className="flex flex-col gap-6 py-6">
        <div className="h-8 w-72 animate-pulse rounded-md bg-[var(--color-field-bg)]" />
        <div className="h-[352px] w-full animate-pulse rounded-[20px] bg-[var(--color-field-bg)]" />
      </section>
    );
  }

  if (results === null) {
    return (
      <section className="flex flex-col gap-6 py-6">
        <NotFound />
      </section>
    );
  }

  return (
    <section className="flex flex-col gap-6 py-6">
      <h1 className="text-[20px] font-bold leading-tight text-[var(--color-ink)] sm:text-[24px]">
        Так держать! Продолжайте в том же духе!
      </h1>
      <ResultsCard results={results} deckId={deckId} />
    </section>
  );
}, 'PracticeResultsPage');
