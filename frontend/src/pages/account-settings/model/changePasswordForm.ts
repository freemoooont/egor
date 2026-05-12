import { reatomField, reatomForm, reatomBoolean, wrap } from '@reatom/core';
import { z } from 'zod';

import { ApiError, changePassword } from '@/shared/api/index.ts';

/**
 * Change-password dialog form. Uses `reatomForm` with a Zod schema so
 * validation errors land at the right field paths (per `llms/reatom.md`).
 */
export const changePasswordOpenAtom = reatomBoolean(false, 'account.changePassword.open');
export const changePasswordOldField = reatomField('', { name: 'account.changePassword.old' });
export const changePasswordNewField = reatomField('', { name: 'account.changePassword.new' });
export const changePasswordConfirmField = reatomField('', {
  name: 'account.changePassword.confirm',
  validate({ state }) {
    if (state.length === 0) return undefined;
    if (state !== changePasswordNewField()) return 'Пароли не совпадают';
    return undefined;
  },
});

const schema = z.object({
  oldPassword: z.string().min(1, 'Введите текущий пароль'),
  newPassword: z.string().min(8, 'Не менее 8 символов'),
  confirmPassword: z.string().min(1, 'Подтвердите пароль'),
});

export const changePasswordForm = reatomForm(
  {
    oldPassword: changePasswordOldField,
    newPassword: changePasswordNewField,
    confirmPassword: changePasswordConfirmField,
  },
  {
    name: 'account.changePassword.form',
    schema,
    validateOnBlur: true,
    onSubmit: async (values) => {
      try {
        await wrap(
          changePassword({
            oldPassword: values.oldPassword,
            newPassword: values.newPassword,
          }),
        );
        changePasswordOpenAtom.setFalse();
        changePasswordOldField.set('');
        changePasswordNewField.set('');
        changePasswordConfirmField.set('');
        return { ok: true } as const;
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          changePasswordOldField.validation.errors.unshift({
            source: 'submission',
            message: 'Неверный текущий пароль',
          });
        }
        throw err;
      }
    },
  },
);
