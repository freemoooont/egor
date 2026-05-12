import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * `cn` — canonical Tailwind class merger used by every shadcn/ui component.
 * Resolves class conflicts (e.g. `p-2 p-4` -> `p-4`) and lets variants compose
 * without manual deduplication.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
