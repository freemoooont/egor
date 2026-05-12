import { z } from 'zod';

import { apiFetch } from './client.ts';

/**
 * Auth endpoints — typed Zod-parsed wrappers. Mirrors `docs/backend/openapi.yaml`
 * operations under `/api/auth/...` and `/api/me`. See use-cases:
 * RegisterUser, LoginUser, RefreshAccessToken, GetCurrentUser.
 */

export const userDtoSchema = z.object({
  id: z.string(),
  email: z.string(),
  displayName: z.string(),
  avatarRef: z.string().nullable().optional(),
  registeredAt: z.string().optional(),
});

export type UserDto = z.infer<typeof userDtoSchema>;

const tokensFragment = {
  accessToken: z.string(),
  refreshToken: z.string(),
  accessTokenExpiresAt: z.string().optional(),
  refreshTokenExpiresAt: z.string().optional(),
};

export const authResultSchema = z.object({
  ...tokensFragment,
  user: userDtoSchema,
});
export type AuthResult = z.infer<typeof authResultSchema>;

export const refreshResultSchema = z.object(tokensFragment);
export type RefreshResult = z.infer<typeof refreshResultSchema>;

export interface RegisterInput {
  email: string;
  password: string;
  displayName: string;
}

export interface LoginInput {
  email: string;
  password: string;
}

export async function register(
  input: RegisterInput,
  options: { idempotencyKey?: string } = {},
): Promise<AuthResult> {
  const raw = await apiFetch('/auth/register', {
    method: 'POST',
    body: JSON.stringify(input),
    anonymous: true,
    idempotencyKey: options.idempotencyKey,
  });
  return authResultSchema.parse(raw);
}

export async function login(input: LoginInput): Promise<AuthResult> {
  const raw = await apiFetch('/auth/login', {
    method: 'POST',
    body: JSON.stringify(input),
    anonymous: true,
  });
  return authResultSchema.parse(raw);
}

export async function refresh(refreshToken: string): Promise<RefreshResult> {
  const raw = await apiFetch('/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refreshToken }),
    anonymous: true,
    skipRefresh: true,
  });
  return refreshResultSchema.parse(raw);
}

export async function me(): Promise<UserDto> {
  const raw = await apiFetch('/me', { method: 'GET' });
  return userDtoSchema.parse(raw);
}
