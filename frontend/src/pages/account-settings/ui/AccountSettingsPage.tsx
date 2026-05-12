import { reatomComponent, bindField, useAction } from '@reatom/react';
import { urlAtom, wrap } from '@reatom/core';
import { Plus } from 'lucide-react';
import { useEffect } from 'react';

import { ROUTES } from '@/shared/config/index.ts';
import { clearSession } from '@/shared/auth/index.ts';
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Input,
} from '@/shared/ui/index.ts';
import { currentUserAtom } from '@/entities/user/index.ts';

import {
  displayNameDraftAtom,
  displayNameEditingAtom,
  displayNameErrorAtom,
  emailDraftAtom,
  emailEditingAtom,
  emailErrorAtom,
  hydrateDraftsAction,
  saveDisplayNameAction,
  saveEmailAction,
} from '../model/profileForm.ts';
import {
  changePasswordConfirmField,
  changePasswordForm,
  changePasswordNewField,
  changePasswordOldField,
  changePasswordOpenAtom,
} from '../model/changePasswordForm.ts';

const AvatarUploader = reatomComponent(() => {
  return (
    <div className="rounded-2xl border border-[var(--color-card-border)] bg-[var(--color-card-surface)] p-5">
      <p className="text-[15px] font-semibold text-[var(--color-ink)]">Аватар профиля</p>
      <div className="mt-6 flex items-center justify-center pb-2">
        <button
          type="button"
          disabled
          aria-disabled="true"
          title="Скоро"
          className="inline-flex h-12 w-12 items-center justify-center rounded-full border-2 border-[var(--color-card-border)] text-[var(--color-ink-muted)] opacity-70"
        >
          <Plus className="h-5 w-5" strokeWidth={2.5} />
        </button>
      </div>
    </div>
  );
}, 'AvatarUploader');

const ProfileFieldRow = reatomComponent<{
  label: string;
  value: string;
  editing: boolean;
  draft: string;
  pending: boolean;
  error: string | null;
  inputType?: 'text' | 'email';
  onEdit: () => void;
  onChange: (next: string) => void;
  onSave: () => void;
  onCancel: () => void;
}>(
  ({
    label,
    value,
    editing,
    draft,
    pending,
    error,
    inputType = 'text',
    onEdit,
    onChange,
    onSave,
    onCancel,
  }) => {
    return (
      <div className="flex flex-col gap-1 py-2">
        <p className="text-[13px] font-semibold text-[var(--color-ink)]">{label}</p>
        {editing ? (
          <div className="flex flex-col gap-2">
            <Input
              type={inputType}
              value={draft}
              onChange={wrap((event: React.ChangeEvent<HTMLInputElement>) => {
                onChange(event.currentTarget.value);
              })}
              autoFocus
            />
            {error ? (
              <p role="alert" className="text-[12px] text-[var(--color-error)]">
                {error}
              </p>
            ) : null}
            <div className="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                onClick={wrap(() => {
                  onSave();
                })}
                disabled={pending}
              >
                {pending ? 'Сохранение…' : 'Сохранить'}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                onClick={wrap(() => {
                  onCancel();
                })}
                disabled={pending}
              >
                Отмена
              </Button>
            </div>
          </div>
        ) : (
          <div className="flex items-center justify-between gap-3">
            <p className="text-[14px] text-[var(--color-ink)]">{value}</p>
            <button
              type="button"
              onClick={wrap(() => {
                onEdit();
              })}
              className="text-[13px] font-medium text-[var(--color-ink-muted)] outline-none transition-colors hover:text-[var(--color-ink)] focus-visible:underline"
            >
              Редактировать
            </button>
          </div>
        )}
      </div>
    );
  },
  'ProfileFieldRow',
);

const ChangePasswordDialog = reatomComponent(() => {
  const open = changePasswordOpenAtom();
  const submitPending = changePasswordForm.submit.pending() > 0;
  const oldErrors = changePasswordOldField.validation.errors();
  const newErrors = changePasswordNewField.validation.errors();
  const confirmErrors = changePasswordConfirmField.validation.errors();

  return (
    <Dialog
      open={open}
      onOpenChange={wrap((next: boolean) => {
        if (next) changePasswordOpenAtom.setTrue();
        else changePasswordOpenAtom.setFalse();
      })}
    >
      <DialogContent className="rounded-2xl">
        <DialogHeader>
          <DialogTitle>Сменить пароль</DialogTitle>
          <DialogDescription>Минимум 8 символов.</DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          noValidate
          onSubmit={wrap((event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            changePasswordForm.submit().catch(() => {});
          })}
        >
          <div className="flex flex-col gap-1">
            <label className="text-[13px] font-semibold text-[var(--color-ink)]">
              Текущий пароль
            </label>
            <Input type="password" autoComplete="current-password" {...bindField(changePasswordOldField)} />
            {oldErrors.length > 0 ? (
              <p className="text-[12px] text-[var(--color-error)]">{oldErrors[0]!.message}</p>
            ) : null}
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-[13px] font-semibold text-[var(--color-ink)]">
              Новый пароль
            </label>
            <Input type="password" autoComplete="new-password" {...bindField(changePasswordNewField)} />
            {newErrors.length > 0 ? (
              <p className="text-[12px] text-[var(--color-error)]">{newErrors[0]!.message}</p>
            ) : null}
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-[13px] font-semibold text-[var(--color-ink)]">
              Подтверждение
            </label>
            <Input type="password" autoComplete="new-password" {...bindField(changePasswordConfirmField)} />
            {confirmErrors.length > 0 ? (
              <p className="text-[12px] text-[var(--color-error)]">{confirmErrors[0]!.message}</p>
            ) : null}
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={wrap(() => {
                changePasswordOpenAtom.setFalse();
              })}
            >
              Отмена
            </Button>
            <Button type="submit" disabled={submitPending}>
              {submitPending ? 'Сохранение…' : 'Сохранить'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}, 'ChangePasswordDialog');

export const AccountSettingsPage = reatomComponent(() => {
  const user = currentUserAtom.data();
  const dnEditing = displayNameEditingAtom();
  const dnDraft = displayNameDraftAtom();
  const dnError = displayNameErrorAtom();
  const dnPending = saveDisplayNameAction.pending() > 0;

  const emEditing = emailEditingAtom();
  const emDraft = emailDraftAtom();
  const emError = emailErrorAtom();
  const emPending = saveEmailAction.pending() > 0;

  // Hydrate drafts when user loads / changes. `useAction` binds the action to
  // the React frame so it can be called from a useEffect without losing the
  // reatom async stack.
  const hydrateDrafts = useAction(hydrateDraftsAction);
  useEffect(() => {
    if (user) {
      hydrateDrafts();
    }
  }, [user?.id, user?.email, user?.displayName, hydrateDrafts]);

  return (
    <section className="flex flex-col gap-6 py-6">
      <h1 className="text-[22px] font-bold leading-tight text-[var(--color-ink)]">
        Настройки аккаунта
      </h1>

      <AvatarUploader />

      <div className="rounded-2xl border border-[var(--color-card-border)] bg-[var(--color-card-surface)] p-5">
        <ProfileFieldRow
          label="Имя профиля"
          value={user?.displayName ?? '—'}
          editing={dnEditing}
          draft={dnDraft}
          pending={dnPending}
          error={dnError}
          onEdit={() => {
            displayNameErrorAtom.set(null);
            displayNameDraftAtom.set(user?.displayName ?? '');
            displayNameEditingAtom.setTrue();
          }}
          onChange={(next) => {
            displayNameDraftAtom.set(next);
          }}
          onSave={() => {
            saveDisplayNameAction();
          }}
          onCancel={() => {
            displayNameErrorAtom.set(null);
            displayNameDraftAtom.set(user?.displayName ?? '');
            displayNameEditingAtom.setFalse();
          }}
        />
        <div className="my-1 h-px w-full bg-[var(--color-card-border)]" />
        <ProfileFieldRow
          label="Электронная почта"
          value={user?.email ?? '—'}
          editing={emEditing}
          draft={emDraft}
          pending={emPending}
          error={emError}
          inputType="email"
          onEdit={() => {
            emailErrorAtom.set(null);
            emailDraftAtom.set(user?.email ?? '');
            emailEditingAtom.setTrue();
          }}
          onChange={(next) => {
            emailDraftAtom.set(next);
          }}
          onSave={() => {
            saveEmailAction();
          }}
          onCancel={() => {
            emailErrorAtom.set(null);
            emailDraftAtom.set(user?.email ?? '');
            emailEditingAtom.setFalse();
          }}
        />
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <button
          type="button"
          onClick={wrap(() => {
            clearSession();
            urlAtom.go(ROUTES.login);
          })}
          className="inline-flex h-11 items-center justify-center rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-5 text-sm font-semibold text-[var(--color-error)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
        >
          Выйти из аккаунта
        </button>
        <button
          type="button"
          onClick={wrap(() => {
            changePasswordOpenAtom.setTrue();
          })}
          className="inline-flex h-11 items-center justify-center rounded-full border border-[var(--color-card-border)] bg-[var(--color-card-surface)] px-5 text-sm font-semibold text-[var(--color-ink)] outline-none transition-colors hover:bg-[var(--color-field-bg)] focus-visible:ring-2 focus-visible:ring-ring"
        >
          Сменить пароль
        </button>
      </div>

      <ChangePasswordDialog />
    </section>
  );
}, 'AccountSettingsPage');
