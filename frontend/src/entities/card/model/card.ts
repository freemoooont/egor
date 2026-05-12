import type { CardDto } from '@/shared/api/index.ts';

/**
 * Domain card value type — used on both the deck-create and deck-practice
 * screens. Stays serializable; mutable bits (focus, selection, dnd order) are
 * lifted into atoms by the consumer.
 */
export interface Card {
  id: string;
  term: string;
  definition: string;
  ordinal: number;
}

export function cardFromDto(dto: CardDto): Card {
  return {
    id: dto.id,
    term: dto.term,
    definition: dto.definition,
    ordinal: dto.ordinal,
  };
}

/** Compare cards by their `ordinal` field — stable ordering for any consumer. */
export function cardOrdinalCompare(a: Card, b: Card): number {
  return a.ordinal - b.ordinal;
}
