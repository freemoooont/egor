# Frontend scaffold notes

This file records every deviation from the spec or the `kot` reference made
while bootstrapping `frontend/`. Future agents and the verifier read it to
understand what is "as designed" vs "drift to fix".

## Status snapshot

- **Date**: 2026-05-07
- **Toolchain**: pnpm + vite-plus (`vp`).
- **Vite+ probe**: `vite-plus@latest` + `npm:@voidzero-dev/vite-plus-core@latest` resolve cleanly via pnpm; `vp` binary lands in `node_modules/.bin/`. No fallback to plain Vite needed.

## Deviations from the spec

1. **`pnpm` instead of `vp add` / `vp remove`.** The spec language ("Use `pnpm` directly … the reference repo uses vite-plus as a real npm package; mirror that") explicitly allows this. We invoke Vite+ via package.json scripts (`pnpm dev` → `vp dev`, etc.) but we manage dependencies with raw `pnpm install`.
2. **Placeholder PWA icons.** `public/icons/icon-192.png`, `icon-512.png`, `apple-touch-icon.png` are 1×1 transparent PNGs (69 bytes each). Replace with real branded artwork during the Figma pass. The PWA manifest references them; build still emits `manifest.webmanifest` and the SW.
3. **Brand orange is `#F97316` for now.** Documented in `tailwind.config.ts`, `src/app/styles/global.css`, `src/shared/config/brand.ts`, and the Vite manifest. Tighten the exact hex from Figma when implementing the first screen.
4. **shadcn/ui components manually authored** (auth screen pass, 2026-05-07). The `pnpm dlx shadcn@latest add <name>` CLI prompts interactively on first run for an init config and we operate unattended. So the auth pass authored the MVP UI primitives (`button`, `input`, `label`, `card`, `alert`) directly under `src/shared/ui/`, mirroring the canonical shadcn registry source. Radix peer-deps + `class-variance-authority` / `clsx` / `tailwind-merge` were already pinned in `package.json`. A future `shadcn init && shadcn add ...` should diff cleanly. Components still pending: switch, avatar, tabs, dialog, dropdown-menu, sonner, separator, skeleton, scroll-area, progress (the Tabs widget on the auth screens is rendered with a small bespoke component because the auth tabs route through the URL rather than local Radix state).
5. **No business logic on screens.** Every page slice exports a `<div className="p-6">` placeholder plus a `reatomRoute`. Reatom forms, async data, and toaster wiring will land in the screen pass.
6. **`features/` and `entities/` are intentionally empty** (FSD pages-first). They have `.gitkeep`s only.

## What works after `pnpm install`

- `pnpm typecheck` (alias for `tsc --noEmit`) — should be green.
- `pnpm build` — should produce `dist/index.html`, `dist/sw.js`, `dist/manifest.webmanifest`.
- `pnpm dev` — verifier-only; agents must not start it.

## Known follow-ups for the screen-implementation pass

- Run `pnpm dlx shadcn@latest init` and `add button input label tabs card switch avatar dialog dropdown-menu sonner separator skeleton scroll-area progress`, then re-export them from `src/shared/ui/index.ts`.
- Replace placeholder PNG icons with real Micocards artwork at 192×192, 512×512, and 180×180.
- Tighten the brand orange hex from the Figma node tokens.
- Wire the auth flow (login/register pages) with `reatomForm` + Zod and `apiFetch`.
- Move `features/auth-login`, `features/auth-register`, etc. out of `pages/` only when 2+ consumers appear.

## Screen-implementation pass — 2026-05-07

Decks-list, account-settings, deck-create implemented. Notes:

7. **shadcn primitives authored manually** (avatar, dialog, dropdown-menu, separator, skeleton, textarea — `progress`, `switch`, `scroll-area` are pending). Mirrors registry sources; future `pnpm dlx shadcn@latest add <name>` runs should diff cleanly.
8. **Dev access token persists in `sessionStorage`.** Production keeps the access token in memory only; dev MSW persists it under `micocards.auth.access` so a navigate doesn't trigger a refresh round-trip on every page load. `tokenStore.setAccess` writes both to memory and (in dev) to sessionStorage.
9. **MSW handlers persist to `localStorage`.** Users, decks, and refresh tokens are saved under `micocards.mocks.users` / `micocards.mocks.decks` / `micocards.mocks.refreshTokens` so a refresh keeps state. Each handler hydrates from storage on every call to survive HMR module resets.
10. **Drag-and-drop is up/down chevrons.** The Figma deck-create design shows a grip icon for reordering; we render the icon non-interactively and provide chevron-up / chevron-down buttons next to it. No `@dnd-kit/core` dependency added. Visual fidelity matches; documented in `.agent/tasks/micocards-mvp/raw/screenshots/decks.notes.md`.
11. **Avatar uploader is disabled.** Per spec, the upload backend is not wired; the `+` button is `disabled` with `title="Скоро"`.
12. **AI-generation always succeeds in dev MSW** with 5 hardcoded sample cards (Reatom-themed). The real backend may return 501 → page surfaces "ИИ-генерация недоступна".
13. **`apiFetch` body wrapped in a single `wrap`.** The original implementation chained multiple `await wrap(...)` calls inside the function; that violates the wrap rule (chaining after a wrap loses the async stack) and crashed every read with "missing async stack". Fixed by wrapping the whole request/response body in one outer `wrap(promise)`.
14. **`decksListAtom` / `currentUserAtom` read `tokenStore` directly** rather than the `isAuthenticatedAtom`. The atom only mirrors the boot-time token-store state; reads via raw fetch (during dev seeding) don't propagate. Reading the store directly inside the computed makes the entry/exit gate honest. A `decksRefreshTickAtom` is bumped after mutations so the computed re-runs.
15. **`decksListRoute` gates on `self.exact()`.** Reatom matches `path: 'decks'` as a prefix (so `/decks/new` would also render the list). The render returns `<></>` when not exact so the deck-create render is the only visible slot.
16. **`ROUTES` moved from `app/routes.ts` to `shared/config/routes.ts`.** The widget AppShell imports it; pulling from `app/routes` would create a circular import (widget → app/routes → page → shared/router → widget).
17. **`useAction(hydrateDraftsAction)` in account settings.** Calling actions directly from a `useEffect` fails the wrap rule (no async stack). Reatom's `useAction` binds the action to the React frame; the call is then safe.
18. **`AppShell.outlet()` filters null slots.** The previous `.at(0)` returned the first registered child whether or not it matched, masking deeper matches. Fixed.

## Practice screens pass — 2026-05-07

deck-practice and practice-results implemented. Notes:

19. **shadcn primitives `switch`, `progress`, `scroll-area` authored manually** (the trio called out in the practice spec). Mirrors Radix + registry sources; future `pnpm dlx shadcn@latest add <name>` runs should diff cleanly. Radix peer-deps `@radix-ui/react-switch`, `@radix-ui/react-progress`, `@radix-ui/react-scroll-area` were already pinned in `package.json` from the bootstrap pass.
20. **`entities/practice` exposes `reatomPracticeResults`/`reatomDeckProgress` as factory functions** — the page slices call them with a getter for the search-param sessionId so the computed re-runs when the URL changes. Mirrors the v1000 "factory pattern in loaders" idiom (`llms/reatom.md` § Routing/Full SPA example).
21. **MSW persists practice sessions and ratings to `localStorage`** (`micocards.mocks.practiceSessions`, `micocards.mocks.practiceRatings`) so a page refresh survives. The result aggregator keeps the LAST rating per `cardId` to avoid double-counting double taps.
22. **Practice page keyboard shortcuts**: ArrowLeft / ArrowRight to navigate, Space / Enter to flip the card, 1 / 2 / 3 to rate (when tracking is on). Documented via `title=` tooltips on each rate button. The global `keydown` handler skips events whose target is a focused button or an editable input so it doesn't double-fire with the on-card flip handler.
23. **AI integration is mock-only** — no real LLM calls were added. The existing AI port stub in `entities/deck` is unchanged.
24. **Donut chart is bespoke SVG** (`pages/practice-results/ui/DonutChart.tsx`) — three concentric strokes via `strokeDasharray`/`strokeDashoffset`. `role="img"` + `aria-label` summarises counts. No charting library dependency added.
25. **Toggle ON starts a tracked session via `POST /api/practice/sessions`**; toggle OFF (or reaching the last rated card) calls `POST /api/practice/sessions/:id/finish` and routes to `/decks/:id/results?sessionId=<id>`. Untracked mode never touches the API.
