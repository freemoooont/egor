import type { PracticeResultsDto, PracticeProgressDto } from '@/shared/api/index.ts';

/** Aggregated session results (post-finish). */
export interface PracticeResults {
  deckId: string;
  knowCount: number;
  learningCount: number;
  dontKnowCount: number;
  total: number;
  completedAt: string | null;
}

export function resultsFromDto(dto: PracticeResultsDto): PracticeResults {
  return {
    deckId: dto.deckId,
    knowCount: dto.knowCount,
    learningCount: dto.learningCount,
    dontKnowCount: dto.dontKnowCount,
    total: dto.total,
    completedAt: dto.completedAt,
  };
}

export interface DeckProgress {
  deckId: string;
  knowCount: number;
  learningCount: number;
  dontKnowCount: number;
  total: number;
}

export function progressFromDto(dto: PracticeProgressDto): DeckProgress {
  return {
    deckId: dto.deckId,
    knowCount: dto.knowCount,
    learningCount: dto.learningCount,
    dontKnowCount: dto.dontKnowCount,
    total: dto.total,
  };
}
