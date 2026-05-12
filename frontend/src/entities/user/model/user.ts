import type { UserDto } from '@/shared/api/index.ts';

/**
 * Domain user value type — the immutable shape consumed by widgets, the app
 * shell, and the account-settings page. Mirrors the UserDto that lands from
 * `/api/me` but lives in the entities layer so consumers don't deep-import
 * shared/api.
 */
export interface User {
  id: string;
  email: string;
  displayName: string;
  avatarRef: string | null;
  registeredAt: string | null;
}

export function userFromDto(dto: UserDto): User {
  return {
    id: dto.id,
    email: dto.email,
    displayName: dto.displayName,
    avatarRef: dto.avatarRef ?? null,
    registeredAt: dto.registeredAt ?? null,
  };
}

/** First letter of the display name (or `?`) — used for `<AvatarFallback />`. */
export function userInitial(user: User | null): string {
  const dn = user?.displayName?.trim();
  if (dn && dn.length > 0) return dn[0]!.toUpperCase();
  const email = user?.email?.trim();
  if (email && email.length > 0) return email[0]!.toUpperCase();
  return '?';
}
