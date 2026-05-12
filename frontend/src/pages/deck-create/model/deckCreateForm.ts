import { atom, action, reatomBoolean, reatomField, withAsync, wrap } from '@reatom/core';
import { z } from 'zod';

import { ApiError } from '@/shared/api/index.ts';
import { createDeckAction, generateDeckAction } from '@/entities/deck/index.ts';

/**
 * Deck-create state model.
 *
 * Each card row owns its own pair of `reatomField`s plus a stable `id` for
 * keying React lists and reorder operations. The list is held in an atom so
 * we can add/remove/move rows reactively. The whole shape is validated on
 * submit (title 1..120, every term/definition 1..512).
 */

export interface CardRow {
  id: string;
  term: ReturnType<typeof reatomField<string>>;
  definition: ReturnType<typeof reatomField<string>>;
}

let cardSeq = 0;
function nextCardId(): string {
  cardSeq += 1;
  return `c-${Date.now().toString(36)}-${cardSeq.toString(36)}`;
}

function makeCardRow(initialTerm = '', initialDefinition = ''): CardRow {
  const id = nextCardId();
  return {
    id,
    term: reatomField(initialTerm, { name: `deckCreate.cards#${id}.term` }),
    definition: reatomField(initialDefinition, { name: `deckCreate.cards#${id}.definition` }),
  };
}

export const titleField = reatomField('', { name: 'deckCreate.title' });
export const titleErrorAtom = atom<string | null>(null, 'deckCreate.titleError');
export const submitErrorAtom = atom<string | null>(null, 'deckCreate.submitError');

export const cardRowsAtom = atom<CardRow[]>([makeCardRow(), makeCardRow()], 'deckCreate.cardRows');

export const addCardRowAction = action(() => {
  cardRowsAtom.set((rows) => [...rows, makeCardRow()]);
}, 'deckCreate.addCardRow');

export const removeCardRowAction = action((id: string) => {
  cardRowsAtom.set((rows) => (rows.length <= 1 ? rows : rows.filter((r) => r.id !== id)));
}, 'deckCreate.removeCardRow');

export const moveCardRowAction = action((id: string, direction: 'up' | 'down') => {
  cardRowsAtom.set((rows) => {
    const idx = rows.findIndex((r) => r.id === id);
    if (idx < 0) return rows;
    const swap = direction === 'up' ? idx - 1 : idx + 1;
    if (swap < 0 || swap >= rows.length) return rows;
    const next = rows.slice();
    const [a, b] = [next[idx]!, next[swap]!];
    next[idx] = b;
    next[swap] = a;
    return next;
  });
}, 'deckCreate.moveCardRow');

export const aiDialogOpenAtom = reatomBoolean(false, 'deckCreate.aiDialogOpen');
export const aiTopicField = reatomField('', { name: 'deckCreate.aiTopic' });
export const aiErrorAtom = atom<string | null>(null, 'deckCreate.aiError');

const titleSchema = z.string().trim().min(1, 'Введите название колоды').max(120, 'Не более 120 символов');
const cardTextSchema = z.string().trim().min(1).max(512);

interface SubmitOptions {
  practice: boolean;
  /** Called when the deck has been created. The handler is wrapped before invocation. */
  onCreated: (deckId: string) => void;
}

export const submitDeckCreateAction = action(async ({ practice, onCreated }: SubmitOptions) => {
  submitErrorAtom.set(null);
  // Validate title
  const titleParsed = titleSchema.safeParse(titleField());
  if (!titleParsed.success) {
    titleErrorAtom.set(titleParsed.error.issues[0]?.message ?? 'Неверное название');
    return;
  }
  titleErrorAtom.set(null);

  // Validate cards (all term + definition must be 1..512)
  const rows = cardRowsAtom();
  const cards = rows.map((row) => ({
    term: row.term().trim(),
    definition: row.definition().trim(),
  }));
  const allValid = cards.every(
    (c) => cardTextSchema.safeParse(c.term).success && cardTextSchema.safeParse(c.definition).success,
  );
  if (!allValid) {
    submitErrorAtom.set('Заполните термин и определение для каждой карточки.');
    return;
  }

  try {
    const deck = await wrap(
      createDeckAction({
        title: titleParsed.data,
        cards,
      }),
    );
    onCreated(deck.id);
  } catch (err) {
    if (err instanceof ApiError) {
      submitErrorAtom.set(err.message || 'Не удалось создать колоду');
    } else {
      submitErrorAtom.set('Не удалось создать колоду');
    }
    void practice; // referenced via onCreated semantics
    throw err;
  }
}, 'deckCreate.submit').extend(withAsync());

export const generateAiDeckAction = action(async () => {
  aiErrorAtom.set(null);
  const topic = aiTopicField().trim();
  if (topic.length === 0) {
    aiErrorAtom.set('Введите тему');
    return;
  }
  try {
    const result = await wrap(generateDeckAction({ topic }));
    cardRowsAtom.set(
      result.cards.map((card) => makeCardRow(card.term, card.definition)),
    );
    if (titleField().trim().length === 0) {
      titleField.set(topic);
    }
    aiDialogOpenAtom.setFalse();
    aiTopicField.set('');
  } catch (err) {
    if (err instanceof ApiError && err.status === 501) {
      aiErrorAtom.set('ИИ-генерация недоступна');
    } else {
      aiErrorAtom.set('Не удалось сгенерировать колоду');
    }
  }
}, 'deckCreate.generateAi').extend(withAsync());

export const resetDeckCreateAction = action(() => {
  titleField.set('');
  titleErrorAtom.set(null);
  submitErrorAtom.set(null);
  cardRowsAtom.set([makeCardRow(), makeCardRow()]);
  aiDialogOpenAtom.setFalse();
  aiTopicField.set('');
  aiErrorAtom.set(null);
}, 'deckCreate.reset');
