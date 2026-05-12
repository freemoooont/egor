import { reatomComponent } from '@reatom/react';
import { bindField } from '@reatom/react';
import { atom, type FieldAtom, wrap } from '@reatom/core';
import { Eye, EyeOff } from 'lucide-react';
import type { HTMLInputAutoCompleteAttribute } from 'react';

import { cn } from '@/shared/lib/index.ts';

interface AuthFieldProps {
  field: FieldAtom<string, string>;
  type?: 'text' | 'email' | 'password';
  placeholder: string;
  /**
   * If provided, the toggle visibility eye icon is rendered — used for password
   * fields. The atom owner controls show/hide so the parent screen can re-use
   * one toggle for multiple fields if needed.
   */
  visibilityAtom?: ReturnType<typeof atom<boolean>>;
  autoComplete?: HTMLInputAutoCompleteAttribute;
  /** Optional inline override for layout tweaks. */
  className?: string;
}

/**
 * AuthField — Figma `1:691` / `1:692` (filled & password variants).
 *
 * Pulls value/error from a Reatom `FieldAtom`. Renders the password show/hide
 * eye toggle when `visibilityAtom` is provided. Error state colours border and
 * helper text in `var(--color-error)` (Figma `1:777`).
 */
export const AuthField = reatomComponent<AuthFieldProps>(
  ({ field, type = 'text', placeholder, visibilityAtom, autoComplete, className }) => {
    const bound = bindField(field);
    const error = bound.error;
    const visible = visibilityAtom ? visibilityAtom() : false;
    const effectiveType = type === 'password' && visible ? 'text' : type;

    return (
      <div className={cn('flex w-full flex-col gap-1', className)}>
        <div
          className={cn(
            'flex h-[56px] w-full items-center rounded-[10px] bg-[var(--color-field-bg)] pl-4 pr-3 transition-shadow',
            'focus-within:ring-2 focus-within:ring-brand-500 focus-within:ring-offset-0',
            error && 'border border-[var(--color-error)]',
          )}
        >
          <input
            type={effectiveType}
            autoComplete={autoComplete}
            placeholder={placeholder}
            value={bound.value ?? ''}
            onChange={bound.onChange}
            onBlur={bound.onBlur}
            onFocus={bound.onFocus}
            className={cn(
              'h-full w-full flex-1 bg-transparent text-[16px] font-medium leading-none outline-none',
              error
                ? 'text-[var(--color-error)] placeholder:text-[var(--color-error)]/70'
                : 'text-[var(--color-ink)] placeholder:text-[14px] placeholder:font-medium placeholder:text-[var(--color-ink-placeholder)]',
            )}
          />
          {visibilityAtom ? (
            <button
              type="button"
              aria-label={visible ? 'Скрыть пароль' : 'Показать пароль'}
              onClick={wrap(() => visibilityAtom.set(!visibilityAtom()))}
              className="ml-3 flex size-5 shrink-0 items-center justify-center text-[var(--color-ink-placeholder)] hover:text-[var(--color-ink)]"
            >
              {visible ? <EyeOff size={20} /> : <Eye size={20} />}
            </button>
          ) : null}
        </div>
        {error ? (
          <p
            className="px-1 text-[14px] font-medium text-[var(--color-error)]"
            role="alert"
          >
            {error}
          </p>
        ) : null}
      </div>
    );
  },
  'AuthField',
);
