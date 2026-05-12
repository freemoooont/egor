import { reatomComponent } from '@reatom/react';
import type { ReactNode } from 'react';
import { Toaster } from 'sonner';

interface ProvidersProps {
  children: ReactNode;
}

/**
 * Cross-cutting providers that wrap the routed content.
 * - Sonner toaster for transient feedback (saved, error, etc).
 * - Future: error boundary, theme provider, etc.
 */
export const Providers = reatomComponent<ProvidersProps>(({ children }) => {
  return (
    <>
      {children}
      <Toaster position="top-right" richColors closeButton />
    </>
  );
}, 'Providers');
