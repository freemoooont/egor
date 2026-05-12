import { reatomComponent } from '@reatom/react';

import { Button } from '@/shared/ui/index.ts';

interface SubmitButtonProps {
  disabled: boolean;
  pending: boolean;
  label: string;
}

/**
 * Big "Продолжить" CTA — Figma `1:759` (active) / `1:693` (disabled).
 * Single source of truth for the auth submit button's appearance & states.
 */
export const SubmitButton = reatomComponent<SubmitButtonProps>(
  ({ disabled, pending, label }) => {
    return (
      <Button
        type="submit"
        variant="brandCta"
        size="full"
        disabled={disabled || pending}
        aria-busy={pending}
      >
        {pending ? 'Отправляем…' : label}
      </Button>
    );
  },
  'SubmitButton',
);
