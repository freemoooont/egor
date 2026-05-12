import type { DeckDto } from '@/shared/api/index.ts';
import { cardFromDto, type Card } from '@/entities/card/index.ts';

/**
 * Domain deck — the immutable shape consumed by widgets and pages.
 * Atomized state (selected, expanded, drag-active) is layered on top by
 * page/feature slices, never normalized into a sidecar list.
 */
export interface Deck {
  id: string;
  title: string;
  authorName: string;
  createdAt: string;
  termsCount: number;
  lessonsCount: number;
  cards: Card[];
}

export function deckFromDto(dto: DeckDto): Deck {
  // The live API only emits the canonical {id,title,cards|cardCount,createdAt}
  // shape; legacy / mock layers added optional `authorName` / `termsCount` /
  // `lessonsCount`. Default cosmetic fields so widgets keep rendering.
  const cards = (dto.cards ?? []).map(cardFromDto);
  return {
    id: dto.id,
    title: dto.title,
    authorName: dto.authorName ?? '',
    createdAt: dto.createdAt,
    termsCount: dto.termsCount ?? dto.cardCount ?? cards.length,
    lessonsCount: dto.lessonsCount ?? 0,
    cards,
  };
}
