import { computed, withAsyncData, wrap } from '@reatom/core';

import { ApiError, getMe } from '@/shared/api/index.ts';
import { tokenStore } from '@/shared/auth/index.ts';

import { userFromDto, type User } from './user.ts';

/**
 * `currentUser` — async-data computed for `GET /api/me`.
 *
 * Reads from `tokenStore` directly so it picks up tokens written outside the
 * Reatom session (e.g. dev MSW seeding via raw fetch). Unauthenticated reads
 * short-circuit to `null` so the AppShell header can render without flashing
 * an error during the very first paint after logout.
 */
export const currentUserAtom = computed(async (): Promise<User | null> => {
  if (tokenStore.getAccess() === null && tokenStore.getRefresh() === null) return null;
  try {
    const dto = await wrap(getMe());
    return userFromDto(dto);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      return null;
    }
    throw err;
  }
}, 'entities.currentUser').extend(withAsyncData({ initState: null as User | null, status: true }));
