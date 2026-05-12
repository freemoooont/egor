import { atom, reatomBoolean, action, withAsync, wrap } from '@reatom/core';
import { z } from 'zod';

import { ApiError, updateMe } from '@/shared/api/index.ts';
import { currentUserAtom, userFromDto } from '@/entities/user/index.ts';

/**
 * Account settings — profile editor atoms.
 *
 * The page maintains two editable fields (display name, email) with a dirty
 * tracking flag and a save action. We don't use a `reatomForm` here because
 * the field UX is row-based with per-field "Редактировать" toggles, not a
 * single form submit; the action validates each field via Zod manually.
 */
export const displayNameDraftAtom = atom('', 'account.profile.displayNameDraft');
export const displayNameEditingAtom = reatomBoolean(false, 'account.profile.displayNameEditing');
export const displayNameErrorAtom = atom<string | null>(null, 'account.profile.displayNameError');

export const emailDraftAtom = atom('', 'account.profile.emailDraft');
export const emailEditingAtom = reatomBoolean(false, 'account.profile.emailEditing');
export const emailErrorAtom = atom<string | null>(null, 'account.profile.emailError');

const displayNameSchema = z
  .string()
  .trim()
  .min(1, 'Введите имя')
  .max(64, 'Не более 64 символов');
const emailSchema = z.string().trim().email('Введите корректный email');

/** Sync drafts with the loaded user. Called from the page mount. */
export const hydrateDraftsAction = action(async () => {
  const user = currentUserAtom.data();
  if (!user) return;
  if (!displayNameEditingAtom()) {
    displayNameDraftAtom.set(user.displayName);
  }
  if (!emailEditingAtom()) {
    emailDraftAtom.set(user.email);
  }
}, 'account.profile.hydrateDrafts');

export const saveDisplayNameAction = action(async () => {
  const value = displayNameDraftAtom().trim();
  const parsed = displayNameSchema.safeParse(value);
  if (!parsed.success) {
    displayNameErrorAtom.set(parsed.error.issues[0]?.message ?? 'Неверное имя');
    return;
  }
  displayNameErrorAtom.set(null);
  try {
    const dto = await wrap(updateMe({ displayName: parsed.data }));
    const user = userFromDto(dto);
    displayNameDraftAtom.set(user.displayName);
    displayNameEditingAtom.setFalse();
    currentUserAtom.retry();
  } catch (err) {
    if (err instanceof ApiError) {
      displayNameErrorAtom.set(err.message || 'Не удалось сохранить');
    } else {
      displayNameErrorAtom.set('Не удалось сохранить');
    }
  }
}, 'account.profile.saveDisplayName').extend(withAsync());

export const saveEmailAction = action(async () => {
  const value = emailDraftAtom().trim();
  const parsed = emailSchema.safeParse(value);
  if (!parsed.success) {
    emailErrorAtom.set(parsed.error.issues[0]?.message ?? 'Неверный email');
    return;
  }
  emailErrorAtom.set(null);
  try {
    const dto = await wrap(updateMe({ email: parsed.data }));
    const user = userFromDto(dto);
    emailDraftAtom.set(user.email);
    emailEditingAtom.setFalse();
    currentUserAtom.retry();
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.status === 409) emailErrorAtom.set('Этот email уже используется');
      else emailErrorAtom.set(err.message || 'Не удалось сохранить');
    } else {
      emailErrorAtom.set('Не удалось сохранить');
    }
  }
}, 'account.profile.saveEmail').extend(withAsync());
