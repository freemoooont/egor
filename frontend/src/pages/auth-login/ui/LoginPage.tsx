import { reatomComponent } from '@reatom/react';
import { wrap } from '@reatom/core';
import type { FormEvent } from 'react';

import { AuthCard, AuthField, AuthTabs, SubmitButton } from '@/features/auth-form/index.ts';
import {
  loginEmailField,
  loginForm,
  loginPasswordField,
  loginPasswordVisibleAtom,
} from '../model/loginForm.ts';

/**
 * Login page — Figma nodes 1:674 (empty), 1:740 (filled), 1:760 (email error),
 * 1:695 (mobile). Layout uses the shared `AuthCard` shell, `AuthTabs` for
 * routing-driven tab state, and the `AuthField` wrapper that binds Reatom
 * fields with bindField.
 */
export const LoginPage = reatomComponent(() => {
  const submitPending = loginForm.submit.pending() > 0;
  const submitError = loginForm.submit.error();
  const emailValue = loginEmailField();
  const passwordValue = loginPasswordField();

  // CTA active only when email looks well-formed AND password is non-empty —
  // computed inline so the button becomes orange the moment the user finishes
  // typing, without waiting for blur-driven validation. Submission still runs
  // the full Zod schema for safety.
  const emailLooksValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailValue.trim());
  const disabled = submitPending || !emailLooksValid || passwordValue.length === 0;

  return (
    <AuthCard>
      <form
        className="flex flex-1 w-full flex-col gap-7"
        noValidate
        onSubmit={wrap((event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          void loginForm.submit().then(
            () => {},
            () => {
              /* surfaced via submit.error() / per-field errors */
            },
          );
        })}
      >
        <div className="flex w-full flex-col gap-7">
          <AuthTabs active="login" />
          <div className="flex w-full flex-col gap-[14px]">
            <AuthField
              field={loginEmailField}
              type="email"
              placeholder="Email"
              autoComplete="email"
            />
            <AuthField
              field={loginPasswordField}
              type="password"
              placeholder="Пароль"
              autoComplete="current-password"
              visibilityAtom={loginPasswordVisibleAtom}
            />
            {submitError &&
            submitError.message &&
            loginEmailField.validation.errors().length === 0 &&
            loginPasswordField.validation.errors().length === 0 ? (
              <p
                role="alert"
                className="px-1 text-[14px] font-medium text-[var(--color-error)]"
              >
                {submitError.message}
              </p>
            ) : null}
          </div>
        </div>
        {/* CTA footer — pinned to the bottom of the card on desktop and to the bottom of the viewport on mobile (Figma 1:695). */}
        <div className="mt-auto w-full">
          <SubmitButton disabled={disabled} pending={submitPending} label="Продолжить" />
        </div>
      </form>
    </AuthCard>
  );
}, 'LoginPage');
