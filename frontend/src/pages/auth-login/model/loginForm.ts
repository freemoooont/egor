import { atom, reatomField, reatomForm, urlAtom, wrap } from '@reatom/core';
import { z } from 'zod';

import { ApiError, login } from '@/shared/api/index.ts';
import { persistTokens } from '@/shared/auth/index.ts';
import { ROUTES } from '@/app/routes.ts';

/**
 * Login form — Reatom v1000 `reatomForm` + `reatomField` with a Zod schema.
 *
 * Validation:
 *  - email: must be a non-empty valid email.
 *  - password: ≥1 char (legacy users may have arbitrary passwords; ADR
 *    `0003` enforces strength only on register).
 *
 * Submit calls `POST /api/auth/login` (`@/shared/api/auth.login`) and on
 * success persists tokens and routes to `/decks`. On 401 it surfaces a
 * field-level error on email; on 422 it maps the offending field via the
 * server's error envelope.
 */
export const loginEmailField = reatomField('', { name: 'auth.login.emailField' });
export const loginPasswordField = reatomField('', { name: 'auth.login.passwordField' });
export const loginPasswordVisibleAtom = atom(false, 'auth.login.passwordVisible');

const loginSchema = z.object({
  email: z.string().email('Введите корректный email'),
  password: z.string().min(1, 'Введите пароль'),
});

export const loginForm = reatomForm(
  {
    email: loginEmailField,
    password: loginPasswordField,
  },
  {
    name: 'auth.login.form',
    schema: loginSchema,
    validateOnBlur: true,
    onSubmit: async (values) => {
      try {
        const result = await wrap(login({ email: values.email, password: values.password }));
        await wrap(persistTokens(result.accessToken, result.refreshToken));
        urlAtom.go(ROUTES.decks);
        return result;
      } catch (err) {
        if (err instanceof ApiError) {
          if (err.status === 401) {
            loginEmailField.validation.errors.unshift({
              source: 'submission',
              message: 'Неверный email или пароль',
            });
          } else if (err.status === 422) {
            const target = err.code === 'invalid_email' ? loginEmailField : loginPasswordField;
            target.validation.errors.unshift({
              source: 'submission',
              message:
                err.code === 'invalid_email'
                  ? 'Введите корректный email'
                  : 'Не удалось войти, попробуйте ещё раз',
            });
          }
        }
        throw err;
      }
    },
  },
);
