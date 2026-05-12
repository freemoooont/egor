export { apiFetch, ApiError } from './client.ts';
export type { ApiFetchInit } from './client.ts';
export {
  register,
  login,
  refresh,
  me,
  userDtoSchema,
  authResultSchema,
  refreshResultSchema,
} from './auth.ts';
export type {
  RegisterInput,
  LoginInput,
  AuthResult,
  RefreshResult,
  UserDto,
} from './auth.ts';
export {
  decks,
  listDecks,
  getDeck,
  createDeck,
  updateDeck,
  deleteDeck,
  generateDeck,
  deckSchema,
  cardSchema,
  deckListSchema,
  generatedDeckSchema,
} from './decks.ts';
export type {
  DeckDto,
  CardDto,
  DeckListDto,
  CreateDeckInput,
  UpdateDeckInput,
  DeckCardInput,
  GenerateDeckInput,
  GeneratedDeckDto,
} from './decks.ts';
export {
  meApi,
  getMe,
  updateMe,
  changePassword,
  changePasswordResultSchema,
} from './me.ts';
export type {
  UpdateMeInput,
  ChangePasswordInput,
  ChangePasswordResult,
} from './me.ts';
export {
  practice,
  startPracticeSession,
  ratePracticeCard,
  finishPracticeSession,
  getPracticeResults,
  getDeckProgress,
  PRACTICE_RATING,
  practiceSessionSchema,
  practiceFinishSchema,
  practiceResultsSchema,
  practiceProgressSchema,
} from './practice.ts';
export type {
  PracticeMode,
  PracticeRating,
  PracticeSessionDto,
  PracticeFinishDto,
  PracticeResultsDto,
  PracticeProgressDto,
  StartSessionInput,
} from './practice.ts';
