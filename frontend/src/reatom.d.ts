import type { JSX } from 'react';

declare module '@reatom/core' {
  interface RouteChild extends JSX.Element {}
}
