import { reatomComponent, bindField } from '@reatom/react';
import { urlAtom, wrap } from '@reatom/core';
import { Plus, Sparkles, Trash2, GripVertical, ChevronUp, ChevronDown } from 'lucide-react';

import { ROUTES } from '@/shared/config/index.ts';
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Input,
} from '@/shared/ui/index.ts';

import {
  addCardRowAction,
  aiDialogOpenAtom,
  aiErrorAtom,
  aiTopicField,
  cardRowsAtom,
  generateAiDeckAction,
  moveCardRowAction,
  removeCardRowAction,
  resetDeckCreateAction,
  submitDeckCreateAction,
  submitErrorAtom,
  titleErrorAtom,
  titleField,
  type CardRow,
} from '../model/deckCreateForm.ts';

const TitleInput = reatomComponent(() => {
  const error = titleErrorAtom();
  return (
    <div className="rounded-2xl border border-[var(--color-card-border)] bg-[var(--color-card-surface)] p-2">
      <Input
        {...bindField(titleField)}
        placeholder="Название колоды"
        maxLength={120}
        className="h-12 border-0 bg-[var(--color-field-bg)] text-[15px] focus-visible:ring-0 focus-visible:ring-offset-0 rounded-xl px-4"
      />
      {error ? (
        <p role="alert" className="px-2 pt-1 text-[12px] text-[var(--color-error)]">
          {error}
        </p>
      ) : null}
    </div>
  );
}, 'DeckCreateTitleInput');

const AiButton = reatomComponent(() => {
  return (
    <button
      type="button"
      onClick={wrap(() => {
        aiErrorAtom.set(null);
        aiDialogOpenAtom.setTrue();
      })}
      className="inline-flex h-11 items-center gap-2 self-start rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-5 text-sm font-semibold text-[var(--color-ink)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
    >
      <Sparkles className="h-4 w-4 text-[var(--color-brand-500)]" />
      Сгенерировать через ИИ
    </button>
  );
}, 'DeckCreateAiButton');

const CardRowItem = reatomComponent<{ row: CardRow; ordinal: number; total: number }>(
  ({ row, ordinal, total }) => {
    return (
      <div className="flex flex-col gap-3 rounded-2xl border border-[var(--color-card-border)] bg-[var(--color-card-surface)] p-4">
        <div className="flex items-center justify-between">
          <span className="text-[13px] font-bold text-[var(--color-ink)]">{ordinal}</span>
          <div className="flex items-center gap-1 text-[var(--color-ink-muted)]">
            <button
              type="button"
              aria-label="Поднять вверх"
              disabled={ordinal === 1}
              onClick={wrap(() => {
                moveCardRowAction(row.id, 'up');
              })}
              className="rounded-full p-1 transition-colors hover:bg-[var(--color-field-bg)] disabled:opacity-30"
            >
              <ChevronUp className="h-4 w-4" />
            </button>
            <button
              type="button"
              aria-label="Опустить вниз"
              disabled={ordinal === total}
              onClick={wrap(() => {
                moveCardRowAction(row.id, 'down');
              })}
              className="rounded-full p-1 transition-colors hover:bg-[var(--color-field-bg)] disabled:opacity-30"
            >
              <ChevronDown className="h-4 w-4" />
            </button>
            <span aria-hidden className="px-1 text-[var(--color-ink-muted)]">
              <GripVertical className="h-4 w-4" />
            </span>
            <button
              type="button"
              aria-label="Удалить карточку"
              onClick={wrap(() => {
                removeCardRowAction(row.id);
              })}
              className="rounded-full p-1 text-[var(--color-ink-muted)] transition-colors hover:bg-[var(--color-field-bg)] hover:text-[var(--color-error)]"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-1">
            <Input
              {...bindField(row.term)}
              placeholder="Термин"
              maxLength={512}
              className="h-11 border-0 bg-[var(--color-field-bg)] text-[14px] focus-visible:ring-0 focus-visible:ring-offset-0 rounded-xl px-4"
            />
            <span className="px-2 text-[11px] font-medium uppercase tracking-wide text-[var(--color-brand-500)]">
              Термин
            </span>
          </div>
          <div className="flex flex-col gap-1">
            <Input
              {...bindField(row.definition)}
              placeholder="Определение"
              maxLength={512}
              className="h-11 border-0 bg-[var(--color-field-bg)] text-[14px] focus-visible:ring-0 focus-visible:ring-offset-0 rounded-xl px-4"
            />
            <span className="px-2 text-[11px] font-medium uppercase tracking-wide text-[var(--color-brand-500)]">
              Определение
            </span>
          </div>
        </div>
      </div>
    );
  },
  'CardRowItem',
);

const AiDialog = reatomComponent(() => {
  const open = aiDialogOpenAtom();
  const pending = generateAiDeckAction.pending() > 0;
  const error = aiErrorAtom();
  return (
    <Dialog
      open={open}
      onOpenChange={wrap((next: boolean) => {
        if (next) aiDialogOpenAtom.setTrue();
        else aiDialogOpenAtom.setFalse();
      })}
    >
      <DialogContent className="rounded-2xl">
        <DialogHeader>
          <DialogTitle>Сгенерировать карточки</DialogTitle>
          <DialogDescription>
            Опишите тему — мы предложим 5 карточек, которые вы сможете отредактировать.
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-3"
          noValidate
          onSubmit={wrap((event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            generateAiDeckAction();
          })}
        >
          <Input {...bindField(aiTopicField)} placeholder="Например: Основы Python" autoFocus />
          {error ? (
            <p role="alert" className="text-[12px] text-[var(--color-error)]">
              {error}
            </p>
          ) : null}
          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={wrap(() => {
                aiDialogOpenAtom.setFalse();
              })}
            >
              Отмена
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? 'Генерация…' : 'Сгенерировать'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}, 'DeckCreateAiDialog');

const ActionButtons = reatomComponent(() => {
  const submitting = submitDeckCreateAction.pending() > 0;
  return (
    <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center">
      <button
        type="button"
        disabled={submitting}
        onClick={wrap(() => {
          submitDeckCreateAction({
            practice: true,
            onCreated: (deckId) => {
              resetDeckCreateAction();
              urlAtom.go(ROUTES.deckPractice(deckId));
            },
          });
        })}
        className="order-1 inline-flex h-11 items-center justify-center rounded-full bg-gradient-to-r from-[var(--color-brand-400)] to-[var(--color-brand-600)] px-5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(255,143,45,0.35)] outline-none transition-transform hover:scale-[1.02] disabled:opacity-60 sm:order-2"
      >
        Создать и практиковать
      </button>
      <button
        type="button"
        disabled={submitting}
        onClick={wrap(() => {
          submitDeckCreateAction({
            practice: false,
            onCreated: () => {
              resetDeckCreateAction();
              urlAtom.go(ROUTES.decks);
            },
          });
        })}
        className="order-2 inline-flex h-11 items-center justify-center rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-5 text-sm font-semibold text-[var(--color-ink)] outline-none transition-colors hover:bg-[var(--color-field-bg)] disabled:opacity-60 sm:order-1"
      >
        {submitting ? 'Сохранение…' : 'Создать'}
      </button>
    </div>
  );
}, 'DeckCreateActions');

export const DeckCreatePage = reatomComponent(() => {
  const rows = cardRowsAtom();
  const submitErr = submitErrorAtom();

  return (
    <section className="flex flex-col gap-5 py-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-[22px] font-bold leading-tight text-[var(--color-ink)]">
          Создание колоды
        </h1>
        <ActionButtons />
      </header>

      <TitleInput />
      <AiButton />

      <div className="flex flex-col gap-4">
        {rows.map((row, idx) => (
          <CardRowItem key={row.id} row={row} ordinal={idx + 1} total={rows.length} />
        ))}
      </div>

      <div className="flex justify-center pt-1">
        <button
          type="button"
          onClick={wrap(() => {
            addCardRowAction();
          })}
          className="inline-flex h-11 items-center gap-2 rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-5 text-sm font-semibold text-[var(--color-ink)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
        >
          <Plus className="h-4 w-4 text-[var(--color-brand-500)]" strokeWidth={3} />
          Добавить карточку
        </button>
      </div>

      {submitErr ? (
        <p role="alert" className="text-sm text-[var(--color-error)]">
          {submitErr}
        </p>
      ) : null}

      <AiDialog />
    </section>
  );
}, 'DeckCreatePage');
