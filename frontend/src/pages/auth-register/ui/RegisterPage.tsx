import { reatomComponent } from '@reatom/react';
import { wrap } from '@reatom/core';
import type { FormEvent } from 'react';

import { AuthCard, AuthField, AuthTabs, SubmitButton } from '@/features/auth-form/index.ts';
import {
  registerDisplayNameField,
  registerEmailField,
  registerForm,
  registerPasswordField,
  registerPasswordVisibleAtom,
} from '../model/registerForm.ts';

/**
 * Register page — Figma node 1:719 (desktop). Same shell as login; the tab
 * router cycles between `/login` and `/register`. Three fields: displayName,
 * email, password (with eye toggle). CTA washes to 40% opacity until all
 * fields are valid per Zod schema.
 */
export const RegisterPage = reatomComponent(() => {
  const submitPending = registerForm.submit.pending() > 0;
  const submitError = registerForm.submit.error();
  const displayNameValue = registerDisplayNameField();
  const emailValue = registerEmailField();
  const passwordValue = registerPasswordField();

  const emailLooksValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailValue.trim());
  const disabled =
    submitPending ||
    displayNameValue.trim().length === 0 ||
    !emailLooksValid ||
    passwordValue.length < 8;

  const fieldErrorsPresent =
    registerDisplayNameField.validation.errors().length > 0 ||
    registerEmailField.validation.errors().length > 0 ||
    registerPasswordField.validation.errors().length > 0;

  return (
    <AuthCard>
      <form
        className="flex flex-1 w-full flex-col gap-7"
        noValidate
        onSubmit={wrap((event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          registerForm.submit().catch(() => {
            /* surfaced via submit.error() / per-field errors */
          });
        })}
      >
        <div className="flex w-full flex-col gap-7">
          <AuthTabs active="register" />
          <div className="flex w-full flex-col gap-[14px]">
            <AuthField
              field={registerDisplayNameField}
              type="text"
              placeholder="Имя профиля"
              autoComplete="nickname"
            />
            <AuthField
              field={registerEmailField}
              type="email"
              placeholder="Email"
              autoComplete="email"
            />
            <AuthField
              field={registerPasswordField}
              type="password"
              placeholder="Пароль"
              autoComplete="new-password"
              visibilityAtom={registerPasswordVisibleAtom}
            />
            {submitError && submitError.message && !fieldErrorsPresent ? (
              <p
                role="alert"
                className="px-1 text-[14px] font-medium text-[var(--color-error)]"
              >
                {submitError.message}
              </p>
            ) : null}
          </div>
        </div>
        <div className="mt-auto w-full">
          <SubmitButton disabled={disabled} pending={submitPending} label="Продолжить" />
        </div>
      </form>
    </AuthCard>
  );
}, 'RegisterPage');
