import { z } from 'zod';

import { apiFetch } from './client.ts';
import { userDtoSchema, type UserDto } from './auth.ts';

/**
 * Current-user mutation endpoints — `PUT /api/me` and
 * `POST /api/auth/change-password`. The MSW handlers persist their state to
 * `localStorage` so reloads keep edits.
 */

export interface UpdateMeInput {
  email?: string;
  displayName?: string;
}

export interface ChangePasswordInput {
  oldPassword: string;
  newPassword: string;
}

export const changePasswordResultSchema = z.object({
  ok: z.literal(true),
});
export type ChangePasswordResult = z.infer<typeof changePasswordResultSchema>;

export async function updateMe(input: UpdateMeInput): Promise<UserDto> {
  const raw = await apiFetch('/me', {
    method: 'PUT',
    body: JSON.stringify(input),
  });
  return userDtoSchema.parse(raw);
}

export async function changePassword(input: ChangePasswordInput): Promise<ChangePasswordResult> {
  const raw = await apiFetch('/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(input),
  });
  return changePasswordResultSchema.parse(raw);
}

export async function getMe(): Promise<UserDto> {
  const raw = await apiFetch('/me', { method: 'GET' });
  return userDtoSchema.parse(raw);
}

export const meApi = {
  get: getMe,
  update: updateMe,
  changePassword,
};
