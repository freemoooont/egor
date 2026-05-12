import { PRACTICE_RATING, type PracticeRating } from '@/shared/api/index.ts';

/**
 * Domain rating enum — re-exports the wire format under a name that reads
 * cleanly in domain code. The enum values match the backend wire encoding:
 *   0 — DontKnow
 *   1 — StillLearning
 *   2 — KnowKnow
 */
export const Rating = PRACTICE_RATING;
export type Rating = PracticeRating;

export type RatingLabel = 'dontKnow' | 'stillLearning' | 'know';

export function ratingToLabel(rating: Rating): RatingLabel {
  if (rating === Rating.KnowKnow) return 'know';
  if (rating === Rating.StillLearning) return 'stillLearning';
  return 'dontKnow';
}

/** Display copy in Russian (matches the Figma source). */
export const RATING_COPY: Record<RatingLabel, string> = {
  dontKnow: 'Не знаю',
  stillLearning: 'Ещё изучаю',
  know: 'Знаю',
};
