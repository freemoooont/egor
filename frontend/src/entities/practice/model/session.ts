import type { PracticeMode, PracticeSessionDto } from '@/shared/api/index.ts';

/** Domain session value type. */
export interface PracticeSession {
  id: string;
  deckId: string;
  startedAt: string;
  mode: PracticeMode;
}

export function sessionFromDto(dto: PracticeSessionDto): PracticeSession {
  return {
    id: dto.id,
    deckId: dto.deckId,
    startedAt: dto.startedAt,
    mode: dto.mode,
  };
}
