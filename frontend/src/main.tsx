import './setup'; // must be imported before any other reatom code!

import { createRoot } from 'react-dom/client';
import { reatomContext } from '@reatom/react';

import { App } from '@/app/App.tsx';
import { rootFrame } from '@/setup.ts';
import '@/app/styles/global.css';

async function bootstrap() {
  if (import.meta.env.DEV && import.meta.env.VITE_USE_MOCKS === 'true') {
    const { worker } = await import('@/mocks/browser.ts');
    await worker.start({ onUnhandledRequest: 'bypass' });
  }

  createRoot(document.getElementById('root')!).render(
    <reatomContext.Provider value={rootFrame}>
      <App />
    </reatomContext.Provider>,
  );
}

bootstrap();
