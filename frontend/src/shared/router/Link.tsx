import { urlAtom, wrap } from '@reatom/core';
import type { AnchorHTMLAttributes, MouseEvent, ReactNode } from 'react';

/**
 * Router-aware link.
 *
 * Renders a real `<a href={to}>` so middle-click, screen-readers and
 * cmd/ctrl-click "open in new tab" still work. Primary-button clicks without
 * modifier keys are intercepted: `event.preventDefault()` plus
 * `urlAtom.go(to)` keeps navigation in-SPA, which is critical because every
 * full reload wipes the in-memory access token and forces a refresh round-trip
 * (see Bug #2 in `.agent/tasks/micocards-mvp/evidence.md`).
 */
export interface LinkProps
  extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'href' | 'children'> {
  to: string;
  /** When true, replace the current history entry instead of pushing. */
  replace?: boolean;
  children?: ReactNode;
}

export function Link({ to, replace, onClick, children, ...rest }: LinkProps): ReactNode {
  const handleClick = (event: MouseEvent<HTMLAnchorElement>): void => {
    onClick?.(event);
    if (event.defaultPrevented) return;
    // Honour the user's intent to open in a new tab/window.
    if (
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey ||
      // External targets and explicit downloads stay native.
      (rest.target && rest.target !== '_self') ||
      rest.download != null
    ) {
      return;
    }
    event.preventDefault();
    wrap(() => {
      urlAtom.go(to, replace);
    })();
  };

  return (
    <a {...rest} href={to} onClick={handleClick}>
      {children}
    </a>
  );
}
