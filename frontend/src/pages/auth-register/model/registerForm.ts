import { atom, reatomField, reatomForm, urlAtom, wrap } from '@reatom/core';
import { z } from 'zod';

import { ApiError, register } from '@/shared/api/index.ts';
import { persistTokens } from '@/shared/auth/index.ts';
import { ROUTES } from '@/app/routes.ts';

/**
 * Register form — Reatom v1000 `reatomForm` + `reatomField` with a Zod schema.
 *
 * Fields:
 *  - displayName: 1..64 chars (User_DisplayNameLengthBetween1And64).
 *  - email: must be a non-empty valid email.
 *  - password: ≥8 chars (ADR 0003 + User_PasswordHashMustComeFromBcryptHasher).
 *
 * Submits to `POST /api/auth/register` (`@/shared/api/auth.register`) with an
 * Idempotency-Key (ADR 0005). On success persists tokens and routes to /decks.
 * On 409 (`email_taken`) / 422 surfaces field-level errors.
 */
export const registerDisplayNameField = reatomField('', {
  name: 'auth.register.displayNameField',
});
export const registerEmailField = reatomField('', { name: 'auth.register.emailField' });
export const registerPasswordField = reatomField('', { name: 'auth.register.passwordField' });
export const registerPasswordVisibleAtom = atom(false, 'auth.register.passwordVisible');

const registerSchema = z.object({
  displayName: z
    .string()
    .min(1, 'Введите имя профиля')
    .max(64, 'Имя профиля до 64 символов'),
  email: z.string().email('Введите корректный email'),
  password: z.string().min(8, 'Пароль не короче 8 символов'),
});

const newIdempotencyKey = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
};

export const registerForm = reatomForm(
  {
    displayName: registerDisplayNameField,
    email: registerEmailField,
    password: registerPasswordField,
  },
  {
    name: 'auth.register.form',
    schema: registerSchema,
    validateOnBlur: true,
    onSubmit: async (values) => {
      try {
        const result = await wrap(
          register(
            {
              displayName: values.displayName.trim(),
              email: values.email.trim(),
              password: values.password,
            },
            { idempotencyKey: newIdempotencyKey() },
          ),
        );
        await wrap(persistTokens(result.accessToken, result.refreshToken));
        urlAtom.go(ROUTES.decks);
        return result;
      } catch (err) {
        if (err instanceof ApiError) {
          if (err.status === 409 && err.code === 'email_taken') {
            registerEmailField.validation.errors.unshift({
              source: 'submission',
              message: 'Email уже зарегистрирован',
            });
          } else if (err.status === 422) {
            const map: Record<string, ReturnType<typeof reatomField<string>>> = {
              invalid_email: registerEmailField,
              invalid_display_name: registerDisplayNameField,
              password_too_weak: registerPasswordField,
            };
            const target = err.code ? map[err.code] : undefined;
            if (target) {
              target.validation.errors.unshift({
                source: 'submission',
                message:
                  err.code === 'invalid_email'
                    ? 'Введите корректный email'
                    : err.code === 'password_too_weak'
                      ? 'Пароль не короче 8 символов'
                      : 'Имя профиля от 1 до 64 символов',
              });
            }
          }
        }
        throw err;
      }
    },
  },
);
